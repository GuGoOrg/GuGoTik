package main

import (
	"GuGoTik/src/constant/config"
	"GuGoTik/src/constant/strings"
	"GuGoTik/src/extra/tracing"
	"GuGoTik/src/models"
	"GuGoTik/src/storage/database"
	"GuGoTik/src/storage/file"
	"GuGoTik/src/utils/logging"
	"GuGoTik/src/utils/pathgen"
	"GuGoTik/src/utils/rabbitmq"
	"context"
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
	"os"
	"os/exec"
	"sync"
)

func main() {
	conn, err := amqp.Dial(rabbitmq.BuildMQConnAddr())
	if err != nil {
		panic(err)
	}

	defer func(conn *amqp.Connection) {
		err := conn.Close()
		if err != nil {
			panic(err)
		}
	}(conn)

	tp, err := tracing.SetTraceProvider(config.VideoPicker)
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
	if err != nil {
		panic(err)
	}

	defer func(ch *amqp.Channel) {
		err := ch.Close()
		if err != nil {
			panic(err)
		}
	}(ch)

	if _, err = ch.QueueDeclare(strings.VideoPicker, true, false, false, false, nil); err != nil {
		panic(err)
	}

	if err = ch.Qos(1, 0, false); err != nil {
		panic(err)
	}

	go Consume(ch)
	logger := logging.LogService("VideoPicker")
	logger.Infof(strings.VideoPicker + "is running now")

	wg := sync.WaitGroup{}
	wg.Add(1)
	wg.Wait()
}

func Consume(channel *amqp.Channel) {
	msg, err := channel.Consume(strings.VideoPicker, "", false, false, false, false, nil)
	if err != nil {
		panic(err)
	}

	for d := range msg {
		//解包 Otel Context
		ctx := rabbitmq.ExtractAMQPHeaders(context.Background(), d.Headers)
		ctx, span := tracing.Tracer.Start(ctx, "VideoPickerService")
		logger := logging.LogService("VideoPicker.Picker").WithContext(ctx)

		var raw models.RawVideo
		if err := json.Unmarshal(d.Body, &raw); err != nil {
			logger.WithFields(logrus.Fields{
				"err": err,
			}).Errorf("Error when unmarshaling the prepare json body.")
		}

		// 截取封面
		err := extractVideoCover(ctx, &raw)
		if err != nil {
			logger.WithFields(logrus.Fields{
				"err": err,
			}).Errorf("Error when extracting video cover.")
			logging.SetSpanError(span, err)
		}

		// 添加水印逻辑
		err = addWatermarkToVideo(ctx, &raw)
		if err != nil {
			logger.WithFields(logrus.Fields{
				"err": err,
			}).Errorf("Error when adding watermark to video.")
			logging.SetSpanError(span, err)
		}
		//todo: update封面

		// 保存到数据库
		err = database.Client.WithContext(ctx).Create(&raw).Error
		if err != nil {
			logger.WithFields(logrus.Fields{
				"file_name":  raw.FileName,
				"cover_name": raw.CoverName,
				"err":        err,
			}).Debug("failed to create db entry")
		}
		logger.WithFields(logrus.Fields{
			"entry": raw,
		}).Debug("saved db entry")

		span.End()
		err = d.Ack(false)
		if err != nil {
			logger.WithFields(logrus.Fields{
				"err": err,
			}).Errorf("Error when dealing with the video...")
		}
	}
}

func extractVideoCover(ctx context.Context, video *models.RawVideo) error {
	ctx, span := tracing.Tracer.Start(ctx, "ExtractVideoCoverService")
	defer span.End()
	logger := logging.LogService("VideoPicker.Picker").WithContext(ctx)
	logger.Debug("Extracting video cover...")
	RawFileName := video.FileName
	CoverFileName := video.CoverName
	RawFilePath := file.GetLocalPath(ctx, RawFileName)
	CoverFilePath := file.GetLocalPath(ctx, CoverFileName)
	cmdArgs := []string{
		"-i", RawFilePath,
		"-ss", "00:00:01",
		"-vframes", "1",
		CoverFilePath,
	}
	cmd := exec.Command("ffmpeg", cmdArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		logger.WithFields(logrus.Fields{
			"err": err,
		}).Warnf("failed to get video cover")
		logging.SetSpanError(span, err)
		return err
	}
	return nil
}

func addWatermarkToVideo(ctx context.Context, video *models.RawVideo) error {
	ctx, span := tracing.Tracer.Start(ctx, "AddWatermarkToVideoService")
	defer span.End()
	logger := logging.LogService("VideoPicker.Picker").WithContext(ctx)
	logger.Debug("Adding watermark to video...")
	RawFileName := video.FileName
	FinalFileName := pathgen.GenerateFinalVideoName(video.ActorId, video.Title, video.VideoId)
	RawFilePath := file.GetLocalPath(ctx, RawFileName)
	FinalFilePath := file.GetLocalPath(ctx, FinalFileName)
	cmdArgs := []string{
		"-i", RawFilePath,
		"-vf", fmt.Sprintf("drawtext=text='%s':x=(w-text_w-10):y=10:fontsize=24:fontcolor=white", video.Title),
		FinalFilePath,
	}
	cmd := exec.Command("ffmpeg", cmdArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		logger.WithFields(logrus.Fields{
			"err": err,
		}).Warnf("failed to add video watermark")
		logging.SetSpanError(span, err)
		return err
	}
	video.FileName = FinalFileName
	return nil
}
