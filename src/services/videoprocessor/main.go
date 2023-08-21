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
	"bytes"
	"context"
	"encoding/json"
	"github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
	"os/exec"
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
	exitOnError(err)

	defer func(ch *amqp.Channel) {
		err := ch.Close()
		exitOnError(err)
	}(ch)

	err = ch.ExchangeDeclare(
		strings.VideoExchange,
		"fanout",
		true,
		false,
		false,
		false,
		nil,
	)
	exitOnError(err)

	_, err = ch.QueueDeclare(
		strings.VideoPicker, //视频信息采集(封面/水印)
		true,
		false,
		false,
		false,
		nil,
	)
	exitOnError(err)

	_, err = ch.QueueDeclare(
		strings.VideoSummary,
		true,
		false,
		false,
		false,
		nil,
	)
	exitOnError(err)

	err = ch.QueueBind(
		strings.VideoPicker,
		"",
		strings.VideoExchange,
		false,
		nil,
	)
	exitOnError(err)

	err = ch.QueueBind(
		strings.VideoSummary,
		"",
		strings.VideoExchange,
		false,
		nil,
	)
	exitOnError(err)

	err = ch.Qos(1, 0, false)
	exitOnError(err)

	go Consume(ch)
	logger := logging.LogService("VideoPicker")
	logger.Infof(strings.VideoPicker + " is running now")

	go SummaryConsume(ch)
	logger = logging.LogService("VideoSummary")
	logger.Infof(strings.VideoSummary + " is running now")

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
	cmdArgs := []string{
		"-i", RawFilePath, "-vframes", "1", "-an", "-f", "image2pipe", "-",
	}
	cmd := exec.Command("ffmpeg", cmdArgs...)
	// Create a bytes.Buffer to capture stdout
	var buf bytes.Buffer
	cmd.Stdout = &buf

	err := cmd.Run()
	if err != nil {
		logger.WithFields(logrus.Fields{
			"err": err,
		}).Warnf("cmd.Run() failed with %s\n", err)
		logging.SetSpanError(span, err)
		return err
	}
	// buf.Bytes() now contains the image data. You can use it to write to a file or send it to an output stream.
	_, err = file.Upload(ctx, CoverFileName, bytes.NewReader(buf.Bytes()))
	if err != nil {
		logger.WithFields(logrus.Fields{
			"err": err,
		}).Warnf("failed to upload video cover")
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
	cmdArgs := []string{
		"-i", RawFilePath,
		"-vf", "drawtext=text=" + video.Title + ":x=(w-text_w-10):y=10:fontsize=24:fontcolor=white",
		"-c:v", "copy",
		"-f", "mp4",
		"-",
	}

	cmd := exec.Command("ffmpeg", cmdArgs...)
	var buf bytes.Buffer
	cmd.Stdout = &buf

	// Execute the command
	err := cmd.Run()
	if err != nil {
		logger.WithFields(logrus.Fields{
			"err": err,
		}).Warnf("cmd.Run() failed with %s\n", err)
		logging.SetSpanError(span, err)
	}

	// Write the captured stdout to a file
	_, err = file.Upload(ctx, FinalFileName, bytes.NewReader(buf.Bytes()))
	if err != nil {
		logger.WithFields(logrus.Fields{
			"err": err,
		}).Warnf("failed to upload video with watermark")
		logging.SetSpanError(span, err)
		return err
	}
	video.FileName = FinalFileName
	return nil
}
