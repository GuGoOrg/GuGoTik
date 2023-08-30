package main

import (
	"GuGoTik/src/constant/strings"
	"GuGoTik/src/models"
	"GuGoTik/src/storage/database"
	"GuGoTik/src/utils/logging"
	"GuGoTik/src/utils/rabbitmq"
	"context"
	"encoding/json"

	"github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
)

func failOnError(err error, msg string) {
	//打日志
	logging.Logger.Errorf("err %s", msg)

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
	channel, err := conn.Channel()
	if err != nil {
		failOnError(err, "Failed to open a channel")
	}

	_, err = channel.QueueDeclare(
		strings.MessageActionEvent,
		false, false, false, false,
		nil,
	)
	if err != nil {
		failOnError(err, "Failed to define queue")
	}

	msg, err := channel.Consume(
		strings.MessageActionEvent,
		"",
		false, false, false, false,
		nil,
	)
	if err != nil {
		failOnError(err, "Failed to define queue")
	}

	var foreever chan struct{}

	logger := logging.LogService("msgConsumer")
	logger.Infof(strings.MessageActionEvent + " is running now")
	go func() {
		var message models.Message
		for body := range msg {
			if err := json.Unmarshal(body.Body, &message); err != nil {
				logger.WithFields(logrus.Fields{
					"from_id": message.FromUserId,
					"to_id":   message.ToUserId,
					"err":     err,
				}).Errorf("Error when unmarshaling the prepare json body.")
				return
			}

			/* 	ctx := rabbitmq.ExtractAMQPHeaders(context.Background(), body.Headers)
			ctx, span := tracing.Tracer.Start(ctx, "message_send Service")
			logger := logging.LogService("message_send").WithContext(ctx)

			if err := json.Unmarshal(body.Body, &message); err != nil {
				logger.WithFields(logrus.Fields{
					"from_id": message.FromUserId,
					"to_id":   message.ToUserId,
					"err":     err,
				}).Errorf("Error when unmarshaling the prepare json body.")
				logging.SetSpanError(span, err)
				return
			} */

			pmessage := models.Message{
				ToUserId:       message.ToUserId,
				FromUserId:     message.FromUserId,
				ConversationId: message.ConversationId,
				Content:        message.Content,
			}
			logger.Info(pmessage)
			result := database.Client.WithContext(context.Background()).Create(&pmessage)
			if result.Error != nil {
				logger.WithFields(logrus.Fields{
					"from_id": message.FromUserId,
					"to_id":   message.ToUserId,
					"err":     result.Error,
				}).Errorf("Error when insert message to database.")
				// logging.SetSpanError(span, err)
				return
			}
			err = body.Ack(true)
			if err != nil {
				logger.WithFields(logrus.Fields{
					"err": err,
				}).Errorf("Error when dealing with the ,essage...")
			}
		}
	}()

	<-foreever

}
