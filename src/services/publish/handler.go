package main

import (
	"GuGoTik/src/constant/config"
	"GuGoTik/src/constant/strings"
	"GuGoTik/src/extra/tracing"
	"GuGoTik/src/models"
	"GuGoTik/src/rpc/feed"
	"GuGoTik/src/rpc/publish"
	"GuGoTik/src/storage/database"
	"GuGoTik/src/storage/file"
	grpc2 "GuGoTik/src/utils/grpc"
	"GuGoTik/src/utils/logging"
	"GuGoTik/src/utils/pathgen"
	"GuGoTik/src/utils/rabbitmq"
	"bytes"
	"context"
	"encoding/json"
	"github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
	"math/rand"
	"net/http"
	"time"
)

type PublishServiceImpl struct {
	publish.PublishServiceServer
}

var conn *amqp.Connection

var channel *amqp.Channel

var queue amqp.Queue

var FeedClient feed.FeedServiceClient

func init() {
	FeedRpcConn := grpc2.Connect(config.FeedRpcServerName)
	FeedClient = feed.NewFeedServiceClient(FeedRpcConn)
	var err error
	conn, err = amqp.Dial(rabbitmq.BuildMQConnAddr())
	if err != nil {
		panic(err)
	}

	channel, err = conn.Channel()
	if err != nil {
		panic(err)
	}

	queue, err = channel.QueueDeclare(
		strings.VideoPicker, //视频信息采集(封面/水印)
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		panic(err)
	}
}

func (a PublishServiceImpl) ListVideo(ctx context.Context, req *publish.ListVideoRequest) (resp *publish.ListVideoResponse, err error) {
	ctx, span := tracing.Tracer.Start(ctx, "ListVideoService")
	defer span.End()
	logger := logging.LogService("PublishServiceImpl.ListVideo").WithContext(ctx)

	var videos []models.Video
	err = database.Client.WithContext(ctx).
		Where("user_id = ?", req.UserId).
		Order("created_at DESC").
		Find(&videos).Error
	if err != nil {
		logger.WithFields(logrus.Fields{
			"err": err,
		}).Warnf("failed to query video")
		logging.SetSpanError(span, err)
		resp = &publish.ListVideoResponse{
			StatusCode: strings.PublishServiceInnerErrorCode,
			StatusMsg:  strings.PublishServiceInnerError,
		}
		return
	}
	// todo: 使用协程完成，开 go func
	videoIds := make([]uint32, 0, len(videos))
	for _, video := range videos {
		videoIds = append(videoIds, video.ID)
	}

	queryVideoResp, err := FeedClient.QueryVideos(ctx, &feed.QueryVideosRequest{
		ActorId:  req.ActorId,
		VideoIds: videoIds,
	})

	logger.WithFields(logrus.Fields{
		"response": resp,
	}).Debug("all process done, ready to launch response")
	return &publish.ListVideoResponse{
		StatusCode: strings.ServiceOKCode,
		StatusMsg:  strings.ServiceOK,
		VideoList:  queryVideoResp.VideoList,
	}, nil
}

func (a PublishServiceImpl) CountVideo(ctx context.Context, req *publish.CountVideoRequest) (resp *publish.CountVideoResponse, err error) {
	ctx, span := tracing.Tracer.Start(ctx, "CountVideoService")
	defer span.End()
	logger := logging.LogService("PublishServiceImpl.CountVideo").WithContext(ctx)
	var count int64
	err = database.Client.WithContext(ctx).Where("user_id = ?", req.UserId).Count(&count).Error
	if err != nil {
		logger.WithFields(logrus.Fields{
			"err": err,
		}).Warnf("failed to count video")
		resp = &publish.CountVideoResponse{
			StatusCode: strings.PublishServiceInnerErrorCode,
			StatusMsg:  strings.PublishServiceInnerError,
		}
		logging.SetSpanError(span, err)
		return
	}

	resp = &publish.CountVideoResponse{
		StatusCode: strings.ServiceOKCode,
		StatusMsg:  strings.ServiceOK,
		Count:      uint32(count),
	}
	return
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
	// 检测视频格式
	detectedContentType := http.DetectContentType(request.Data)
	if detectedContentType != "video/mp4" {
		logger.WithFields(logrus.Fields{
			"content_type": detectedContentType,
		}).Debug("invalid content type")
		resp = &publish.CreateVideoResponse{
			StatusCode: strings.InvalidContentTypeCode,
			StatusMsg:  strings.InvalidContentType,
		}
		return
	}
	// byte[] -> reader
	reader := bytes.NewReader(request.Data)

	// 创建一个新的随机数生成器
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	videoId := r.Uint32()
	fileName := pathgen.GenerateRawVideoName(request.ActorId, request.Title, videoId)
	coverName := pathgen.GenerateCoverName(request.ActorId, request.Title, videoId)
	// 上传视频
	_, err = file.Upload(ctx, fileName, reader)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"file_name": fileName,
			"err":       err,
		}).Debug("failed to upload video")
		resp = &publish.CreateVideoResponse{
			StatusCode: strings.VideoServiceInnerErrorCode,
			StatusMsg:  strings.VideoServiceInnerError,
		}
		return
	}
	logger.WithFields(logrus.Fields{
		"file_name": fileName,
	}).Debug("uploaded video")

	raw := &models.RawVideo{
		ActorId:   request.ActorId,
		VideoId:   videoId,
		Title:     request.Title,
		FileName:  fileName,
		CoverName: coverName,
	}

	marshal, err := json.Marshal(raw)
	if err != nil {
		resp = &publish.CreateVideoResponse{
			StatusCode: strings.VideoServiceInnerErrorCode,
			StatusMsg:  strings.VideoServiceInnerError,
		}
		return
	}

	// Context 注入到 RabbitMQ 中
	headers := rabbitmq.InjectAMQPHeaders(ctx)

	err = channel.Publish("", queue.Name, false, false,
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType:  "text/plain",
			Body:         marshal,
			Headers:      headers,
		})

	resp = &publish.CreateVideoResponse{
		StatusCode: strings.ServiceOKCode,
		StatusMsg:  strings.ServiceOK,
	}
	return
}
