package main

import (
	"GuGoTik/src/constant/config"
	"GuGoTik/src/constant/strings"
	"GuGoTik/src/extra/tracing"
	"GuGoTik/src/models"
	"GuGoTik/src/storage/database"
	"GuGoTik/src/utils/logging"
	"GuGoTik/src/utils/rabbitmq"
	"context"
	"encoding/json"
	"errors"
	"net/http"

	url2 "net/url"

	"github.com/sashabaranov/go-openai"
	"github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
	"go.opentelemetry.io/otel/trace"
)

func failOnError(err error, msg string) {
	//打日志
	logging.Logger.Errorf("err %s", msg)

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

func main() {
	conn, err := amqp.Dial(rabbitmq.BuildMQConnAddr())
	if err != nil {
		failOnError(err, "Fialed to conenct to RabbitMQ")
	}
	defer func(conn *amqp.Connection) {
		err := conn.Close()
		failOnError(err, "Fialed to close conn")
	}(conn)

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

	channel, err := conn.Channel()
	if err != nil {
		failOnError(err, "Failed to open a channel")
	}

	defer func(channel *amqp.Channel) {
		err := channel.Close()
		failOnError(err, "Fialed to close channel")
	}(channel)

	err = channel.ExchangeDeclare(
		strings.MessageExchange,
		"x-delayed-message",
		true, false, false, false,
		amqp.Table{
			"x-delayed-type": "direct",
		},
	)
	if err != nil {
		failOnError(err, "Failed to get exchange")
	}

	_, err = channel.QueueDeclare(
		strings.MessageActionEvent,
		false, false, false, false,
		nil,
	)
	if err != nil {
		failOnError(err, "Failed to define queue")
	}

	_, err = channel.QueueDeclare(
		strings.MessageGptActionEvent,
		false, false, false, false,
		nil,
	)

	if err != nil {
		failOnError(err, "Failed to define queue")
	}

	err = channel.QueueBind(
		strings.MessageActionEvent,
		strings.MessageActionEvent,
		strings.MessageExchange,
		false,
		nil,
	)
	if err != nil {
		failOnError(err, "Failed to bind queue to exchange")
	}

	err = channel.QueueBind(
		strings.MessageGptActionEvent,
		strings.MessageGptActionEvent,
		strings.MessageExchange,
		false,
		nil,
	)
	if err != nil {
		failOnError(err, "Failed to bind queue  to exchange")
	}

	msg, err := channel.Consume(
		strings.MessageActionEvent,
		"",
		false, false, false, false,
		nil,
	)
	if err != nil {
		failOnError(err, "Failed to Consume")
	}

	var foreever chan struct{}

	logger := logging.LogService("msgConsumer")
	logger.Infof(strings.MessageActionEvent + " is running now")
	go func() {
		var message models.Message
		for body := range msg {
			ctx := rabbitmq.ExtractAMQPHeaders(context.Background(), body.Headers)

			ctx, span := tracing.Tracer.Start(ctx, "message_send Service")
			logger := logging.LogService("message_send").WithContext(ctx)

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
			logger.Info(pmessage)
			//可能会重新插入数据 开启事务 晚点改
			result := database.Client.WithContext(context.Background()).Create(&pmessage)
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
			err = body.Ack(true)

			if err != nil {
				logger.WithFields(logrus.Fields{
					"err": err,
				}).Errorf("Error when dealing with the message...")
				logging.SetSpanError(span, err)
			}
		}
	}()

	go ss(channel)

	<-foreever

}

func ss(channel *amqp.Channel) {
	gptmsg, err := channel.Consume(
		strings.MessageGptActionEvent,
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
		ctx, span := tracing.Tracer.Start(ctx, "message_send Service")
		logger := logging.LogService("message_send").WithContext(ctx)

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
		//可能会重新插入数据 开启事务 晚点改
		result := database.Client.WithContext(context.Background()).Create(&pmessage)
		//发一份消息到openai api
		if result.Error != nil {
			logger.WithFields(logrus.Fields{
				"from_id": message.FromUserId,
				"to_id":   message.ToUserId,
				"err":     result.Error,
			}).Errorf("Error when insert message to database.")
			logging.SetSpanError(span, err)
			//重试?
			continue
		}

		req := openai.ChatCompletionRequest{
			Model: openai.GPT3Dot5Turbo,
			Messages: []openai.ChatCompletionMessage{{
				Role:    openai.ChatMessageRoleUser,
				Content: message.Content,
			},
			},
		}

		resp, err := openaiClient.CreateChatCompletion(
			context.Background(),
			req,
		)

		if err != nil {
			logger.WithFields(logrus.Fields{
				"Err":     err,
				"from_id": message.FromUserId,
				"context": message.Content,
			}).Errorf("Failed to get keywords from ChatGPT")

			logging.SetSpanError(span, err)
			//重试
			errorHandler(channel, body, true, logger, &span)
		}

		text := resp.Choices[0].Message.Content
		pmessage = models.Message{
			ToUserId:       message.FromUserId,
			FromUserId:     message.ToUserId,
			ConversationId: message.ConversationId,
			Content:        text,
			// Content: "111",
		}

		result = database.Client.WithContext(context.Background()).Create(&pmessage)

		if result.Error != nil {
			logger.WithFields(logrus.Fields{
				"from_id": message.FromUserId,
				"to_id":   message.ToUserId,
				"err":     result.Error,
			}).Errorf("Error when insert message to database.")
			logging.SetSpanError(span, err)
			//重试?
			continue
		}

		err = body.Ack(true)

		if err != nil {
			logger.WithFields(logrus.Fields{
				"err": err,
			}).Errorf("Error when dealing with the message...")
			logging.SetSpanError(span, err)
		}

	}
}

func errorHandler(channel *amqp.Channel, d amqp.Delivery, requeue bool, logger *logrus.Entry, span *trace.Span) {
	if !requeue { // Nack the message
		err := d.Nack(false, false)
		if err != nil {
			logger.WithFields(logrus.Fields{
				"err": err,
			}).Errorf("Error when nacking the video...")
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
				}).Errorf("Error when dealing with the video...")
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
				}).Errorf("Error when dealing with the video...")
			}

			logger.Debugf("Retrying %d times", curRetry)

			err = channel.Publish(
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
				}).Errorf("Error when re-publishing the video to queue...")
				logging.SetSpanError(*span, err)
			}
		}
	}
}
