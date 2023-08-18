package main

import (
	"GuGoTik/src/constant/config"
	"GuGoTik/src/constant/strings"
	"GuGoTik/src/extra/tracing"
	"GuGoTik/src/models"
	"GuGoTik/src/storage/database"
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
		coverPath, err := ExtractVideoCover(ctx, raw)
		if err != nil {
			logger.WithFields(logrus.Fields{
				"err":       err,
				"coverPath": coverPath,
			}).Errorf("Error when extracting video cover.")
		}
		// 添加水印逻辑
		watermarkedVideo, err := addWatermarkToVideo(ctx, raw)
		if err != nil {
			logger.WithFields(logrus.Fields{
				"err":              err,
				"watermarkedVideo": watermarkedVideo,
			}).Errorf("Error when adding watermark to video.")
		}

		span.End()
		err = d.Ack(false)
		if err != nil {
			logger.WithFields(logrus.Fields{
				"err": err,
			}).Errorf("Error when dealing with the video...")
		}
	}
}

func ExtractVideoCover(ctx context.Context, video models.RawVideo) (string, error) {
	ctx, span := tracing.Tracer.Start(ctx, "PublishServiceImpl.CountVideo")
	defer span.End()
	logger := logging.LogService("VideoPicker.Picker").WithContext(ctx)
	logger.Debug("Extracting video cover...")
	cmdArgs := []string{
		"-i", video.FilePath,
		"-ss", "00:00:01",
		"-vframes", "1",
		video.CoverPath,
	}
	cmd := exec.Command("ffmpeg", cmdArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	coverPath := pathgen.GenerateRawVideoName(video.ActorId, video.Title)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"err": err,
		}).Warnf("failed to get video cover")
		return coverPath, err
	}
	err = database.Client.Create(&video).Error
	if err != nil {
		logger.WithFields(logrus.Fields{
			"err": err,
		}).Warnf("failed to save processed video to database")
		return coverPath, err
	}

	return coverPath, nil
}

func addWatermarkToVideo(ctx context.Context, video models.RawVideo) (models.RawVideo, error) {
	ctx, span := tracing.Tracer.Start(ctx, "PublishServiceImpl.CountVideo")
	defer span.End()
	logger := logging.LogService("VideoPicker.Picker").WithContext(ctx)
	logger.Debug("Adding watermark to video...")

	cmdArgs := []string{
		"-i", video.FilePath,
		"-vf", fmt.Sprintf("drawtext=text='%s':x=(w-text_w-10):y=10:fontsize=24:fontcolor=white", video.Title),
		video.FilePath,
	}
	cmd := exec.Command("ffmpeg", cmdArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		logger.WithFields(logrus.Fields{
			"err": err,
		}).Warnf("failed to add video watermark")
		return video, err
	}
	return video, nil
}
