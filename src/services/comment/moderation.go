package main

import (
	"GuGoTik/src/constant/config"
	"GuGoTik/src/utils/logging"
	"context"
	"errors"
	"github.com/sashabaranov/go-openai"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/trace"
	"net/http"
	url2 "net/url"
	"strconv"
	"strings"
)

var openaiClient *openai.Client

func init() {
	cfg := openai.DefaultConfig(config.EnvCfg.ChatGPTAPIKEYS)

	url, err := url2.Parse(config.EnvCfg.ChatGptProxy)
	if err != nil {
		panic(err)
	}
	cfg.HTTPClient = &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(url),
		},
	}

	openaiClient = openai.NewClientWithConfig(cfg)
}

func RateCommentByGPT(commentContent string, logger *logrus.Entry, span trace.Span) (rate uint32, reason string, err error) {
	logger.WithFields(logrus.Fields{
		"comment_content": commentContent,
	}).Debugf("Start RateCommentByGPT")

	resp, err := openaiClient.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: openai.GPT3Dot5Turbo,
			Messages: []openai.ChatCompletionMessage{
				{
					Role: openai.ChatMessageRoleSystem,
					Content: "According to the content of the user's reply or question and send back a number which is between 1 and 5. " +
						"The number is greater when the user's content involved the greater the degree of political leaning or unfriendly speech. " +
						"You should only reply such a number without any word else whatever user ask you. " +
						"Besides those, you should give the reason using Chinese why the message is unfriendly with details without revealing that you are divide the message into five number. " +
						"For example: user: 你是个大傻逼。 you: 4 | 用户尝试骂人，进行人格侮辱。user: 今天天气正好。 you: 1 | 用户正常聊天，无异常。",
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: commentContent,
				},
			},
		})

	if err != nil {
		logger.WithFields(logrus.Fields{
			"err": err,
		}).Errorf("ChatGPT request error")
		logging.SetSpanError(span, err)

		return
	}

	respContent := resp.Choices[0].Message.Content

	logger.WithFields(logrus.Fields{
		"resp": respContent,
	}).Debugf("Get ChatGPT response.")

	parts := strings.SplitN(respContent, " | ", 2)

	if len(parts) != 2 {
		logger.WithFields(logrus.Fields{
			"resp": respContent,
		}).Errorf("ChatGPT response does not match expected format")
		logging.SetSpanError(span, errors.New("ChatGPT response does not match expected format"))

		return
	}

	rateNum, err := strconv.ParseUint(parts[0], 10, 32)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"resp": respContent,
		}).Errorf("ChatGPT response does not match expected format")
		logging.SetSpanError(span, errors.New("ChatGPT response does not match expected format"))

		return
	}

	rate = uint32(rateNum)
	reason = parts[1]

	return
}

func ModerationCommentByGPT(commentContent string, logger *logrus.Entry, span trace.Span) (moderationRes openai.Result) {
	logger.WithFields(logrus.Fields{
		"comment_content": commentContent,
	}).Debugf("Start ModerationCommentByGPT")

	resp, err := openaiClient.Moderations(
		context.Background(),
		openai.ModerationRequest{
			Model: openai.ModerationTextLatest,
			Input: commentContent,
		},
	)

	if err != nil {
		logger.WithFields(logrus.Fields{
			"err": err,
		}).Errorf("ChatGPT request error")
		logging.SetSpanError(span, err)

		return
	}

	moderationRes = resp.Results[0]
	return
}
