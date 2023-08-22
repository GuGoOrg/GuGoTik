package main

import (
	"GuGoTik/src/constant/config"
	strings2 "GuGoTik/src/constant/strings"
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
	"github.com/sashabaranov/go-openai"
	"github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/gorm/clause"
	"os/exec"
	"strings"
)

var openaiClient = openai.NewClient(config.EnvCfg.ChatGPTAPIKEYS)

func errorHandler(d amqp.Delivery, requeue bool, logger *logrus.Entry, span *trace.Span) {
	err := d.Nack(false, requeue)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"err": err,
		}).Errorf("Error when resending the video to queue...")
		logging.SetSpanError(*span, err)
	}
}

func SummaryConsume(channel *amqp.Channel) {
	msg, err := channel.Consume(strings2.VideoSummary, "", false, false, false, false, nil)
	if err != nil {
		panic(err)
	}

	for d := range msg {
		//解包 Otel Context
		ctx := rabbitmq.ExtractAMQPHeaders(context.Background(), d.Headers)
		ctx, span := tracing.Tracer.Start(ctx, "VideoSummaryService")
		logger := logging.LogService("VideoSummary").WithContext(ctx)

		var raw models.RawVideo
		logger.WithFields(logrus.Fields{
			"body": d.Body,
		}).Debugf("Message body")
		if err := json.Unmarshal(d.Body, &raw); err != nil {
			logger.WithFields(logrus.Fields{
				"err": err,
			}).Errorf("Error when unmarshaling the prepare json body.")
			logging.SetSpanError(span, err)

			errorHandler(d, false, logger, &span)
			span.End()
			continue
		}

		// Video -> Audio
		audioFileName, err := video2Audio(ctx, raw.FileName)
		if err != nil {
			logger.WithFields(logrus.Fields{
				"err":             err,
				"video_file_name": raw.FileName,
			}).Errorf("Failed to transform video to audio")
			logging.SetSpanError(span, err)

			errorHandler(d, false, logger, &span)
			span.End()
			continue
		}

		// Audio -> Transcript
		transcript, err := speech2Text(ctx, audioFileName)
		if err != nil {
			logger.WithFields(logrus.Fields{
				"err":             err,
				"audio_file_name": audioFileName,
			}).Errorf("Failed to get transcript of an audio from ChatGPT")
			logging.SetSpanError(span, err)

			errorHandler(d, true, logger, &span)
			span.End()
			continue
		}

		var (
			summary  string
			keywords string
		)

		// Transcript -> Summary
		summaryChannel := make(chan string)
		summaryErrChannel := make(chan error)
		go text2Summary(ctx, transcript, &summaryChannel, &summaryErrChannel)

		// Transcript -> Keywords
		keywordsChannel := make(chan string)
		keywordsErrChannel := make(chan error)
		go text2Keywords(ctx, transcript, &keywordsChannel, &keywordsErrChannel)

		select {
		case summary = <-summaryChannel:
		case err = <-summaryErrChannel:
			logger.WithFields(logrus.Fields{
				"err":             err,
				"audio_file_name": audioFileName,
			}).Errorf("Failed to get summary of an audio from ChatGPT")
			logging.SetSpanError(span, err)
			summary = ""

			errorHandler(d, true, logger, &span)
			span.End()
			continue
		}

		select {
		case keywords = <-keywordsChannel:
		case err = <-keywordsErrChannel:
			logger.WithFields(logrus.Fields{
				"err":             err,
				"audio_file_name": audioFileName,
			}).Errorf("Failed to get keywords of an audio from ChatGPT")
			logging.SetSpanError(span, err)
			keywords = ""

			errorHandler(d, true, logger, &span)
			span.End()
			continue
		}

		// Update summary information to database
		video := &models.Video{
			ID:            raw.VideoId,
			AudioFileName: audioFileName,
			Transcript:    transcript,
			Summary:       summary,
			Keywords:      keywords,
		}
		result := database.Client.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "id"}},
			DoUpdates: clause.AssignmentColumns([]string{"audio_file_name", "transcript", "summary", "keywords"}),
		}).Create(&video)
		if result.Error != nil {
			logger.WithFields(logrus.Fields{
				"Err":           result.Error,
				"ID":            raw.VideoId,
				"AudioFileName": audioFileName,
				"Transcript":    transcript,
				"Summary":       summary,
				"Keywords":      keywords,
			}).Errorf("Error when updating summary information to database")
			logging.SetSpanError(span, result.Error)
			errorHandler(d, true, logger, &span)
			span.End()
			continue
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

func video2Audio(ctx context.Context, videoFileName string) (audioFileName string, err error) {
	ctx, span := tracing.Tracer.Start(ctx, "Video2Audio")
	defer span.End()
	logger := logging.LogService("VideoSummary.Video2Audio").WithContext(ctx)
	logger.WithFields(logrus.Fields{
		"video_file_name": videoFileName,
	}).Debugf("Transforming video to audio")

	videoFilePath := file.GetLocalPath(ctx, videoFileName)
	cmdArgs := []string{
		"-i", videoFilePath, "-q:a", "0", "-map", "a", "-f", "mp3", "-",
	}
	cmd := exec.Command("ffmpeg", cmdArgs...)
	var buf bytes.Buffer
	cmd.Stdout = &buf

	err = cmd.Run()
	if err != nil {
		logger.WithFields(logrus.Fields{
			"VideoFileName": videoFileName,
		}).Errorf("cmd %s failed with %s", "ffmpeg "+strings.Join(cmdArgs, " "), err)
		logging.SetSpanError(span, err)
		return
	}

	audioFileName = pathgen.GenerateAudioName(videoFileName)

	_, err = file.Upload(ctx, audioFileName, bytes.NewReader(buf.Bytes()))
	if err != nil {
		logger.WithFields(logrus.Fields{
			"VideoFileName": videoFileName,
			"AudioFileName": audioFileName,
		}).Errorf("Failed to upload audio file")
		logging.SetSpanError(span, err)
		return
	}
	return
}

func speech2Text(ctx context.Context, audioFileName string) (transcript string, err error) {
	ctx, span := tracing.Tracer.Start(ctx, "Speech2Text")
	defer span.End()
	logger := logging.LogService("VideoSummary.Speech2Text").WithContext(ctx)
	logger.WithFields(logrus.Fields{
		"AudioFileName": audioFileName,
	}).Debugf("Transforming audio to transcirpt")

	audioFilePath := file.GetLocalPath(ctx, audioFileName)

	req := openai.AudioRequest{
		Model:    openai.Whisper1,
		FilePath: audioFilePath,
	}
	resp, err := openaiClient.CreateTranscription(ctx, req)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"AudioFileName": audioFileName,
			"err":           err,
		}).Errorf("Failed to get transcript from ChatGPT")
		logging.SetSpanError(span, err)
		return
	}

	transcript = resp.Text
	logger.WithFields(logrus.Fields{
		"Transcript": transcript,
	}).Debugf("Successful to get transcript from ChatGPT")

	return
}

func text2Summary(ctx context.Context, transcript string, summaryChannel *chan string, errChannel *chan error) {
	ctx, span := tracing.Tracer.Start(ctx, "Text2Summary")
	defer span.End()
	logger := logging.LogService("VideoSummary.Text2Summary").WithContext(ctx)
	logger.WithFields(logrus.Fields{
		"transcript": transcript,
	}).Debugf("Getting transcript summary form ChatGPT")

	req := openai.ChatCompletionRequest{
		Model: openai.GPT3Dot5Turbo,
		Messages: []openai.ChatCompletionMessage{
			{
				Role: openai.ChatMessageRoleSystem,
				Content: "You will be provided with a block of text which is the content of a video, " +
					"and your task is to give 2 Simplified Chinese sentences to summarize the video.",
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: transcript,
			},
		},
	}
	resp, err := openaiClient.CreateChatCompletion(ctx, req)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"Transcript": transcript,
		}).Errorf("Failed to get summary from ChatGPT")
		logging.SetSpanError(span, err)
		*errChannel <- err
		return
	}

	summary := resp.Choices[0].Message.Content
	*summaryChannel <- summary

	logger.WithFields(logrus.Fields{
		"Summary": summary,
	}).Debugf("Successful to get summary from ChatGPT")
}

func text2Keywords(ctx context.Context, transcript string, keywordsChannel *chan string, errChannel *chan error) {
	ctx, span := tracing.Tracer.Start(ctx, "Text2Keywords")
	defer span.End()
	logger := logging.LogService("VideoSummary.Text2Keywords").WithContext(ctx)
	logger.WithFields(logrus.Fields{
		"transcript": transcript,
	}).Debugf("Getting transcript keywords from ChatGPT")

	req := openai.ChatCompletionRequest{
		Model: openai.GPT3Dot5Turbo,
		Messages: []openai.ChatCompletionMessage{
			{
				Role: openai.ChatMessageRoleSystem,
				Content: "You will be provided with a block of text which is the content of a video, " +
					"and your task is to give 5 tags in Simplified Chinese to the video to attract audience. " +
					"For example, 美食 | 旅行 | 阅读",
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: transcript,
			},
		},
	}
	resp, err := openaiClient.CreateChatCompletion(ctx, req)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"Transcript": transcript,
		}).Errorf("Failed to get keywords from ChatGPT")
		logging.SetSpanError(span, err)
		*errChannel <- err
		return
	}

	keywords := resp.Choices[0].Message.Content

	*keywordsChannel <- keywords

	logger.WithFields(logrus.Fields{
		"Keywords": keywords,
	}).Debugf("Successful to get keywords from ChatGPT")
}
