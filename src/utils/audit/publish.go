package audit

import (
	"GuGoTik/src/constant/strings"
	"GuGoTik/src/extra/tracing"
	models2 "GuGoTik/src/models"
	"GuGoTik/src/utils/logging"
	"GuGoTik/src/utils/rabbitmq"
	"context"
	"encoding/json"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/sirupsen/logrus"
)

func exitOnError(err error) {
	if err != nil {
		panic(err)
	}
}

func DeclareAuditExchange(channel *amqp.Channel) {
	err := channel.ExchangeDeclare(
		strings.AuditExchange,
		"direct",
		true,
		false,
		false,
		false,
		nil,
	)
	exitOnError(err)
}

func PublishAuditEvent(ctx context.Context, action *models2.Action, channel *amqp.Channel) {
	ctx, span := tracing.Tracer.Start(ctx, "AuditEventPublisher")
	defer span.End()
	logging.SetSpanWithHostname(span)
	logger := logging.LogService("AuditEventPublisher").WithContext(ctx)

	data, err := json.Marshal(action)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"err": err,
		}).Errorf("Error when marshal the action model")
		logging.SetSpanError(span, err)
		return
	}

	headers := rabbitmq.InjectAMQPHeaders(ctx)

	err = channel.PublishWithContext(ctx,
		strings.AuditExchange,
		strings.AuditPublishEvent,
		false,
		false,
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        data,
			Headers:     headers,
		},
	)

	if err != nil {
		logger.WithFields(logrus.Fields{
			"err": err,
		}).Errorf("Error when publishing the action model")
		logging.SetSpanError(span, err)
		return
	}

}
