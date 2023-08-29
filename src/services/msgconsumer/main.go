package main

import (
	"GuGoTik/src/constant/strings"
	"GuGoTik/src/extra/tracing"
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
		true, false, false, false,
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

	logger := logging.LogService("VideoPicker")
	logger.Infof(strings.VideoPicker + " is running now")
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
					"err":     err,
				}).Errorf("Error when unmarshaling the prepare json body.")
				logging.SetSpanError(span, err)
				return
			}

			result := database.Client.WithContext(ctx).Create(&message)
			if result.Error != nil {
				logger.WithFields(logrus.Fields{
					"from_id": message.FromUserId,
					"to_id":   message.ToUserId,
					"err":     result.Error,
				}).Errorf("Error when insert message to database.")
				logging.SetSpanError(span, err)
				return
			}
		}
	}()
	<-foreever

}
