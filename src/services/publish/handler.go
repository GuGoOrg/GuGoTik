package main

import (
	"GuGoTik/src/constant/strings"
	"GuGoTik/src/extra/tracing"
	"GuGoTik/src/models"
	"GuGoTik/src/rpc/publish"
	"GuGoTik/src/utils/logging"
	"GuGoTik/src/utils/pathgen"
	"GuGoTik/src/utils/rabbitmq"
	"context"
	"encoding/json"
	"github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
)

type PublishServiceImpl struct {
	publish.PublishServiceServer
}

var conn *amqp.Connection

var channel *amqp.Channel

func exitOnError(err error) {
	if err != nil {
		panic(err)
	}
}

func init() {
	var err error

	conn, err = amqp.Dial(rabbitmq.BuildMQConnAddr())
	exitOnError(err)

	channel, err = conn.Channel()
	exitOnError(err)

	err = channel.ExchangeDeclare(
		strings.VideoExchange,
		"fanout",
		true,
		false,
		false,
		false,
		nil,
	)
	exitOnError(err)

	_, err = channel.QueueDeclare(
		strings.VideoPicker, //视频信息采集(封面/水印)
		true,
		false,
		false,
		false,
		nil,
	)
	exitOnError(err)

	_, err = channel.QueueDeclare(
		strings.VideoSummary,
		true,
		false,
		false,
		false,
		nil,
	)
	exitOnError(err)

	err = channel.QueueBind(
		strings.VideoPicker,
		"",
		strings.VideoExchange,
		false,
		nil,
	)
	exitOnError(err)

	err = channel.QueueBind(
		strings.VideoSummary,
		"",
		strings.VideoExchange,
		false,
		nil,
	)
	exitOnError(err)
}

func CloseMQConn() {
	if err := conn.Close(); err != nil {
		panic(err)
	}

	if err := channel.Close(); err != nil {
		panic(err)
	}
}

func (a PublishServiceImpl) CreateVideo(ctx context.Context, request *publish.CreateVideoRequest) (resp *publish.CreateVideoResponse, err error) {
	ctx, span := tracing.Tracer.Start(ctx, "CreateVideoService")
	defer span.End()
	logger := logging.LogService("PublishService.CreateVideo").WithContext(ctx)

	logger.WithFields(logrus.Fields{
		"ActorId": request.ActorId,
		"Title":   request.Title,
	}).Infof("Create video requested.")

	raw := &models.RawVideo{
		ActorId:  request.ActorId,
		Title:    request.Title,
		FileName: pathgen.GenerateRawVideoName(request.ActorId, request.Title),
	}

	bytes, err := json.Marshal(raw)

	if err != nil {
		resp = &publish.CreateVideoResponse{
			StatusCode: strings.VideoServiceInnerErrorCode,
			StatusMsg:  strings.VideoServiceInnerError,
		}
		return
	}

	// Context 注入到 RabbitMQ 中
	headers := rabbitmq.InjectAMQPHeaders(ctx)

	err = channel.Publish(strings.VideoExchange, "", false, false,
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType:  "text/plain",
			Body:         bytes,
			Headers:      headers,
		})

	resp = &publish.CreateVideoResponse{
		StatusCode: strings.ServiceOKCode,
		StatusMsg:  strings.ServiceOK,
	}
	return
}
