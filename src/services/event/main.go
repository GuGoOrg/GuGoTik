package main

import (
	"GuGoTik/src/constant/config"
	"GuGoTik/src/constant/strings"
	"GuGoTik/src/extra/tracing"
	"GuGoTik/src/models"
	"GuGoTik/src/utils/logging"
	"GuGoTik/src/utils/rabbitmq"
	"context"
	"encoding/json"
	"github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
	"sync"
)

func exitOnError(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	conn, err := amqp.Dial(rabbitmq.BuildMQConnAddr())
	exitOnError(err)

	defer func(conn *amqp.Connection) {
		err := conn.Close()
		exitOnError(err)
	}(conn)

	tp, err := tracing.SetTraceProvider(config.Event)
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

	ch, err := conn.Channel()
	exitOnError(err)

	defer func(ch *amqp.Channel) {
		err := ch.Close()
		exitOnError(err)
	}(ch)

	err = ch.ExchangeDeclare(
		strings.EventExchange,
		"topic",
		true,
		false,
		false,
		false,
		nil,
	)
	exitOnError(err)

	q, err := ch.QueueDeclare(
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	exitOnError(err)

	err = ch.Qos(1, 0, false)
	exitOnError(err)

	err = ch.QueueBind(
		q.Name,
		strings.FavoriteActionEvent,
		strings.EventExchange,
		false,
		nil)

	exitOnError(err)
	go Consume(ch, q.Name)
	wg := sync.WaitGroup{}
	wg.Add(1)
	wg.Wait()
}

func Consume(ch *amqp.Channel, queueName string) {
	msg, err := ch.Consume(queueName, "", false, false, false, false, nil)
	if err != nil {
		panic(err)
	}

	for d := range msg {
		//解包 Otel Context
		ctx := rabbitmq.ExtractAMQPHeaders(context.Background(), d.Headers)
		ctx, span := tracing.Tracer.Start(ctx, "EventSystem")
		logger := logging.LogService("EventSystem.Recommend").WithContext(ctx)

		var raw models.RecommendEvent
		if err := json.Unmarshal(d.Body, &raw); err != nil {
			logger.WithFields(logrus.Fields{
				"err": err,
			}).Errorf("Error when unmarshaling the prepare json body.")
			logging.SetSpanError(span, err)
			return
		}

	}
}