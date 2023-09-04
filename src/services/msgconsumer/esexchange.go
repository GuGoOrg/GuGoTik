package main

import (
	"GuGoTik/src/constant/strings"
	"GuGoTik/src/extra/tracing"
	"GuGoTik/src/models"
	"GuGoTik/src/storage/es"
	"GuGoTik/src/utils/logging"
	"GuGoTik/src/utils/rabbitmq"
	"bytes"
	"context"
	"encoding/json"

	"github.com/elastic/go-elasticsearch/esapi"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/sirupsen/logrus"
)

func esSaveMessage(channel *amqp.Channel) {

	msg, err := channel.Consume(strings.MessageES, "",
		false, false, false, false, nil,
	)
	failOnError(err, "Failed to Consume")

	var message models.Message
	for body := range msg {
		ctx := rabbitmq.ExtractAMQPHeaders(context.Background(), body.Headers)
		ctx, span := tracing.Tracer.Start(ctx, "MessageSendService")
		logger := logging.LogService("MessageSend").WithContext(ctx)

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

		EsMessage := models.EsMessage{
			ToUserId:       message.ToUserId,
			FromUserId:     message.FromUserId,
			ConversationId: message.ConversationId,
			Content:        message.Content,
			CreateTime:     message.CreatedAt,
		}
		data, _ := json.Marshal(EsMessage)

		req := esapi.IndexRequest{
			Index:   "message",
			Refresh: "true",
			Body:    bytes.NewReader(data),
		}
		//返回值close
		res, err := req.Do(ctx, es.EsClient)

		if err != nil {
			logger.WithFields(logrus.Fields{
				"from_id": message.FromUserId,
				"to_id":   message.ToUserId,
				"content": message.Content,
				"err":     err,
			}).Errorf("Error when insert message to database.")
			logging.SetSpanError(span, err)

			span.End()
			continue
		}
		res.Body.Close()

		err = body.Ack(false)

		if err != nil {
			logger.WithFields(logrus.Fields{
				"err": err,
			}).Errorf("Error when dealing with the message...3")
			logging.SetSpanError(span, err)
		}

	}
}
