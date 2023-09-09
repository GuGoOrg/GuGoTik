package main

import (
	"GuGoTik/src/constant/config"
	"GuGoTik/src/constant/strings"
	"GuGoTik/src/extra/tracing"
	"GuGoTik/src/models"
	"GuGoTik/src/rpc/chat"
	"GuGoTik/src/storage/database"
	grpc2 "GuGoTik/src/utils/grpc"
	"GuGoTik/src/utils/logging"
	"GuGoTik/src/utils/rabbitmq"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	url2 "net/url"
	"sync"

	"fmt"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/sashabaranov/go-openai"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/trace"
)

var chatClient chat.ChatServiceClient
var conn *amqp.Connection
var channel *amqp.Channel

func failOnError(err error, msg string) {
	//打日志
	if err != nil {
		logging.Logger.WithFields(logrus.Fields{
			"err": err,
		}).Errorf(msg)
	}
}

var delayTime = int32(2 * 60 * 1000) //2 minutes
var maxRetries = int32(3)

var openaiClient *openai.Client

func init() {
	cfg := openai.DefaultConfig(config.EnvCfg.ChatGPTAPIKEYS)
	url, err := url2.Parse(config.EnvCfg.ChatGptProxy)
	if err != nil {
		panic(err)
	}
	cfg.HTTPClient = &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(url),
		},
	}
	openaiClient = openai.NewClientWithConfig(cfg)
}

func CloseMQConn() {
	if err := conn.Close(); err != nil {
		panic(err)
	}

	if err := channel.Close(); err != nil {
		panic(err)
	}
}

func main() {
	chatRpcConn := grpc2.Connect(config.MessageRpcServerName)
	chatClient = chat.NewChatServiceClient(chatRpcConn)

	var err error
	conn, err = amqp.Dial(rabbitmq.BuildMQConnAddr())
	failOnError(err, "Failed to connect to RabbitMQ")

	tp, err := tracing.SetTraceProvider(config.MsgConsumer)
	if err != nil {
		logging.Logger.WithFields(logrus.Fields{
			"err": err,
		}).Panicf("Error to set the trace")
	}
	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			logging.Logger.WithFields(logrus.Fields{
				"err": err,
			}).Errorf("Error to set the trace")
		}
	}()

	channel, err = conn.Channel()
	if err != nil {
		failOnError(err, "Failed to open a channel")
	}

	err = channel.ExchangeDeclare(
		strings.MessageExchange,
		"x-delayed-message",
		true, false, false, false,
		amqp.Table{
			"x-delayed-type": "topic",
		},
	)
	failOnError(err, "Failed to get exchange")

	err = channel.ExchangeDeclare(
		strings.AuditExchange,
		"direct",
		true, false, false, false,
		nil,
	)
	failOnError(err, fmt.Sprintf("Failed to get %s exchange", strings.AuditExchange))

	_, err = channel.QueueDeclare(
		strings.MessageCommon,
		true, false, false, false,
		nil,
	)
	failOnError(err, "Failed to define queue")

	_, err = channel.QueueDeclare(
		strings.MessageGPT,
		true, false, false, false,
		nil,
	)

	failOnError(err, "Failed to define queue")
	_, err = channel.QueueDeclare(
		strings.MessageES,
		true, false, false, false,
		nil,
	)

	failOnError(err, "Failed to define queue")

	_, err = channel.QueueDeclare(
		strings.AuditPicker,
		true, false, false, false,
		nil,
	)
	failOnError(err, fmt.Sprintf("Failed to define %s queue", strings.AuditPicker))

	err = channel.QueueBind(
		strings.MessageCommon,
		"message.#",
		strings.MessageExchange,
		false,
		nil,
	)
	failOnError(err, "Failed to bind queue to exchange")

	err = channel.QueueBind(
		strings.MessageES,
		"message.#",
		strings.MessageExchange,
		false,
		nil,
	)
	failOnError(err, "Failed to bind queue to exchange")

	err = channel.QueueBind(
		strings.MessageGPT,
		strings.MessageGptActionEvent,
		strings.MessageExchange,
		false,
		nil,
	)
	failOnError(err, "Failed to bind queue to exchange")

	err = channel.QueueBind(
		strings.AuditPicker,
		strings.AuditPublishEvent,
		strings.AuditExchange,
		false,
		nil,
	)
	failOnError(err, fmt.Sprintf("Failed to bind %s queue to %s exchange", strings.AuditPicker, strings.AuditExchange))

	go saveMessage(channel)
	logger := logging.LogService("MessageSend")
	logger.Infof(strings.MessageActionEvent + " is running now")

	go chatWithGPT(channel)
	logger = logging.LogService("MessageGPTSend")
	logger.Infof(strings.MessageGptActionEvent + " is running now")

	go saveAuditAction(channel)
	logger = logging.LogService("AuditPublish")
	logger.Infof(strings.AuditPublishEvent + " is running now")

	go esSaveMessage(channel)
	logger = logging.LogService("esSaveMessage")
	logger.Infof(strings.VideoPicker + " is running now")

	defer CloseMQConn()

	wg := sync.WaitGroup{}
	wg.Add(1)
	wg.Wait()
}

func saveMessage(channel *amqp.Channel) {
	msg, err := channel.Consume(
		strings.MessageCommon,
		"",
		false, false, false, false,
		nil,
	)
	failOnError(err, "Failed to Consume")
	var message models.Message
	for body := range msg {
		ctx := rabbitmq.ExtractAMQPHeaders(context.Background(), body.Headers)

		ctx, span := tracing.Tracer.Start(ctx, "MessageSendService")
		logger := logging.LogService("MessageSend").WithContext(ctx)

		// Check if it is a re-publish message
		retry, ok := body.Headers["x-retry"].(int32)
		if ok || retry >= 1 {
			err := body.Ack(false)
			if err != nil {
				logger.WithFields(logrus.Fields{
					"err": err,
				}).Errorf("Error when dealing with the message...")
				logging.SetSpanError(span, err)
			}
			span.End()
			continue
		}

		if err := json.Unmarshal(body.Body, &message); err != nil {
			logger.WithFields(logrus.Fields{
				"from_id": message.FromUserId,
				"to_id":   message.ToUserId,
				"content": message.Content,
				"err":     err,
			}).Errorf("Error when unmarshaling the prepare json body.")
			logging.SetSpanError(span, err)
			err = body.Nack(false, true)
			if err != nil {
				logger.WithFields(
					logrus.Fields{
						"from_id": message.FromUserId,
						"to_id":   message.ToUserId,
						"content": message.Content,
						"err":     err,
					},
				).Errorf("Error when nack the message")
				logging.SetSpanError(span, err)
			}
			span.End()
			continue
		}

		pmessage := models.Message{
			ToUserId:       message.ToUserId,
			FromUserId:     message.FromUserId,
			ConversationId: message.ConversationId,
			Content:        message.Content,
		}
		logger.WithFields(logrus.Fields{
			"message": pmessage,
		}).Debugf("Receive message event")

		//可能会重新插入数据 开启事务 晚点改
		//写入数据库

		result := database.Client.WithContext(ctx).Create(&pmessage)

		if result.Error != nil {
			logger.WithFields(logrus.Fields{
				"from_id": message.FromUserId,
				"to_id":   message.ToUserId,
				"content": message.Content,
				"err":     result.Error,
			}).Errorf("Error when insert message to database.")
			logging.SetSpanError(span, err)
			err = body.Nack(false, true)
			if err != nil {
				logger.WithFields(
					logrus.Fields{
						"from_id": message.FromUserId,
						"to_id":   message.ToUserId,
						"content": message.Content,
						"err":     err,
					}).Errorf("Error when nack the message")
				logging.SetSpanError(span, err)
			}
			span.End()
			continue
		}
		err = body.Ack(false)

		if err != nil {
			logger.WithFields(logrus.Fields{
				"err": err,
			}).Errorf("Error when dealing with the message...")
			logging.SetSpanError(span, err)
		}
		span.End()
	}
}

func chatWithGPT(channel *amqp.Channel) {
	gptmsg, err := channel.Consume(
		strings.MessageGPT,
		"",
		false, false, false, false,
		nil,
	)
	if err != nil {
		failOnError(err, "Failed to Consume")
	}
	var message models.Message

	for body := range gptmsg {
		ctx := rabbitmq.ExtractAMQPHeaders(context.Background(), body.Headers)
		ctx, span := tracing.Tracer.Start(ctx, "MessageGPTSendService")
		logger := logging.LogService("MessageGPTSend").WithContext(ctx)

		if err := json.Unmarshal(body.Body, &message); err != nil {
			logger.WithFields(logrus.Fields{
				"from_id": message.FromUserId,
				"to_id":   message.ToUserId,
				"content": message.Content,
				"err":     err,
			}).Errorf("Error when unmarshaling the prepare json body.")
			logging.SetSpanError(span, err)

			//重试
			errorHandler(channel, body, false, logger, &span)
			span.End()
			continue
		}

		logger.WithFields(logrus.Fields{
			"content": message.Content,
		}).Debugf("Receive ChatGPT message event")

		req := openai.ChatCompletionRequest{
			Model: openai.GPT3Dot5Turbo,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: message.Content,
				},
			},
		}

		resp, err := openaiClient.CreateChatCompletion(ctx, req)

		if err != nil {
			logger.WithFields(logrus.Fields{
				"Err":     err,
				"from_id": message.FromUserId,
				"context": message.Content,
			}).Errorf("Failed to get reply from ChatGPT")

			logging.SetSpanError(span, err)
			//重试
			errorHandler(channel, body, true, logger, &span)
			span.End()
			continue
		}

		text := resp.Choices[0].Message.Content
		logger.WithFields(logrus.Fields{
			"reply": text,
		}).Infof("Successfully get the reply for ChatGPT")

		_, err = chatClient.ChatAction(ctx, &chat.ActionRequest{
			ActorId:    message.ToUserId,
			UserId:     message.FromUserId,
			ActionType: 1,
			Content:    text,
		})
		if err != nil {
			logger.WithFields(logrus.Fields{
				"err": err,
			}).Errorf("Replying to user happens error")
			errorHandler(channel, body, true, logger, &span)
			continue
		}
		logger.Infof("Successfully send the reply to user")

		err = body.Ack(false)

		if err != nil {
			logger.WithFields(logrus.Fields{
				"err": err,
			}).Errorf("Error when dealing with the message...")
			logging.SetSpanError(span, err)
		}
		span.End()
	}
}

func saveAuditAction(channel *amqp.Channel) {
	msg, err := channel.Consume(
		strings.AuditPicker,
		"",
		false, false, false, false,
		nil,
	)
	failOnError(err, "Failed to Consume")

	var action models.Action
	for body := range msg {
		ctx := rabbitmq.ExtractAMQPHeaders(context.Background(), body.Headers)

		ctx, span := tracing.Tracer.Start(ctx, "AuditPublishService")
		logger := logging.LogService("AuditPublish").WithContext(ctx)

		if err := json.Unmarshal(body.Body, &action); err != nil {
			logger.WithFields(logrus.Fields{
				"err": err,
			}).Errorf("Error when unmarshaling the prepare json body.")
			logging.SetSpanError(span, err)
			err = body.Nack(false, true)
			if err != nil {
				logger.WithFields(
					logrus.Fields{
						"err":         err,
						"Type":        action.Type,
						"SubName":     action.SubName,
						"ServiceName": action.ServiceName,
					},
				).Errorf("Error when nack the message")
				logging.SetSpanError(span, err)
			}
			span.End()
			continue
		}

		pAction := models.Action{
			Type:         action.Type,
			Name:         action.Name,
			SubName:      action.SubName,
			ServiceName:  action.ServiceName,
			Attached:     action.Attached,
			ActorId:      action.ActorId,
			VideoId:      action.VideoId,
			AffectAction: action.AffectAction,
			AffectedData: action.AffectedData,
			EventId:      action.EventId,
			TraceId:      action.TraceId,
			SpanId:       action.SpanId,
		}
		logger.WithFields(logrus.Fields{
			"action": pAction,
		}).Debugf("Recevie action event")

		result := database.Client.WithContext(ctx).Create(&pAction)
		if result.Error != nil {
			logger.WithFields(
				logrus.Fields{
					"err":         err,
					"Type":        action.Type,
					"SubName":     action.SubName,
					"ServiceName": action.ServiceName,
				},
			).Errorf("Error when nack the message")
			logging.SetSpanError(span, err)
			err = body.Nack(false, true)
			if err != nil {
				logger.WithFields(
					logrus.Fields{
						"err":         err,
						"Type":        action.Type,
						"SubName":     action.SubName,
						"ServiceName": action.ServiceName,
					},
				).Errorf("Error when nack the message")
				logging.SetSpanError(span, err)
			}
			span.End()
			continue
		}
		err = body.Ack(false)

		if err != nil {
			logger.WithFields(logrus.Fields{
				"err": err,
			}).Errorf("Error when dealing with the action...")
			logging.SetSpanError(span, err)
		}
		span.End()
	}
}

func errorHandler(channel *amqp.Channel, d amqp.Delivery, requeue bool, logger *logrus.Entry, span *trace.Span) {
	if !requeue { // Nack the message
		err := d.Nack(false, false)
		if err != nil {
			logger.WithFields(logrus.Fields{
				"err": err,
			}).Errorf("Error when nacking the message event...")
			logging.SetSpanError(*span, err)
		}
	} else { // Re-publish the message
		curRetry, ok := d.Headers["x-retry"].(int32)
		if !ok {
			curRetry = 0
		}
		if curRetry >= maxRetries {
			logger.WithFields(logrus.Fields{
				"body": d.Body,
			}).Errorf("Maximum retries reached for message.")
			logging.SetSpanError(*span, errors.New("maximum retries reached for message"))
			err := d.Ack(false)
			if err != nil {
				logger.WithFields(logrus.Fields{
					"err": err,
				}).Errorf("Error when dealing with the message event...")
			}
		} else {
			curRetry++
			headers := d.Headers
			headers["x-delay"] = delayTime
			headers["x-retry"] = curRetry

			err := d.Ack(false)
			if err != nil {
				logger.WithFields(logrus.Fields{
					"err": err,
				}).Errorf("Error when dealing with the message event...")
			}

			logger.Debugf("Retrying %d times", curRetry)

			err = channel.PublishWithContext(
				context.Background(),
				strings.MessageExchange,
				strings.MessageGptActionEvent,
				false,
				false,
				amqp.Publishing{
					DeliveryMode: amqp.Persistent,
					ContentType:  "text/plain",
					Body:         d.Body,
					Headers:      headers,
				},
			)
			if err != nil {
				logger.WithFields(logrus.Fields{
					"err": err,
				}).Errorf("Error when re-publishing the message event to queue...")
				logging.SetSpanError(*span, err)
			}
		}
	}
}
