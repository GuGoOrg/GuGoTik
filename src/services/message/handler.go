package main

import (
	"GuGoTik/src/constant/config"
	"GuGoTik/src/constant/strings"
	"GuGoTik/src/extra/tracing"
	"GuGoTik/src/models"
	"GuGoTik/src/rpc/chat"
	"GuGoTik/src/rpc/user"
	"GuGoTik/src/storage/database"
	"GuGoTik/src/storage/redis"
	grpc2 "GuGoTik/src/utils/grpc"
	"GuGoTik/src/utils/logging"
	"GuGoTik/src/utils/rabbitmq"
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-redis/redis_rate/v10"
	"github.com/streadway/amqp"

	"github.com/sirupsen/logrus"
)

var UserClient user.UserServiceClient

type MessageServiceImpl struct {
	chat.ChatServiceServer
}

// 连接
var conn *amqp.Connection
var channel *amqp.Channel

//输出

func failOnError(err error, msg string) {
	//打日志
	logging.Logger.Errorf("err %s", msg)

}

func (c MessageServiceImpl) New() {
	userRpcConn := grpc2.Connect(config.UserRpcServerName)
	UserClient = user.NewUserServiceClient(userRpcConn)
	var err error
	conn, err = amqp.Dial(rabbitmq.BuildMQConnAddr())
	if err != nil {
		failOnError(err, "Fialed to conenct to RabbitMQ")
	}
	channel, err = conn.Channel()
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

}

func CloseMQConn() {
	if err := channel.Close(); err != nil {
		failOnError(err, "close channel error")
	}
	if err := conn.Close(); err != nil {
		failOnError(err, "close conn error")
	}
}

//发送消息

var chatActionLimitKeyPrefix = config.EnvCfg.RedisPrefix + "chat_freq_limit"

const chatActionMaxQPS = 3

func chatActionLimitKey(userId uint32) string {
	return fmt.Sprintf("%s-%d", chatActionLimitKeyPrefix, userId)
}

func (c MessageServiceImpl) ChatAction(ctx context.Context, request *chat.ActionRequest) (res *chat.ActionResponse, err error) {
	ctx, span := tracing.Tracer.Start(ctx, "ChatActionService")
	defer span.End()
	logger := logging.LogService("ChatService.ActionMessage").WithContext(ctx)

	logger.WithFields(logrus.Fields{
		"ActorId":      request.ActorId,
		"user_id":      request.UserId,
		"action_type":  request.ActionType,
		"content_text": request.Content,
	}).Debugf("Process start")

	// Rate limiting
	limiter := redis_rate.NewLimiter(redis.Client)
	limiterKey := chatActionLimitKey(request.ActorId)
	limiterRes, err := limiter.Allow(ctx, limiterKey, redis_rate.PerSecond(chatActionMaxQPS))
	if err != nil {
		logger.WithFields(logrus.Fields{
			"ActorId":      request.ActorId,
			"user_id":      request.UserId,
			"action_type":  request.ActionType,
			"content_text": request.Content,
		}).Errorf("ChatAction limiter error")

		res = &chat.ActionResponse{
			StatusCode: strings.UnableToAddMessageErrorCode,
			StatusMsg:  strings.UnableToAddMessageError,
		}
		return
	}
	if limiterRes.Allowed == 0 {
		logger.WithFields(logrus.Fields{
			"ActorId":      request.ActorId,
			"user_id":      request.UserId,
			"action_type":  request.ActionType,
			"content_text": request.Content,
		}).Errorf("Chat action query too frequently by user %d", request.ActorId)

		res = &chat.ActionResponse{
			StatusCode: strings.ChatActionLimitedCode,
			StatusMsg:  strings.ChatActionLimitedError,
		}
		return
	}

	/* 	userResponse, err := UserClient.GetUserExistInformation(ctx, &user.UserExistRequest{
	   		UserId: request.UserId,
	   	})

	   	if err != nil || userResponse.StatusCode != strings.ServiceOKCode {
	   		logger.WithFields(logrus.Fields{
	   			"err":          err,
	   			"ActorId":      request.ActorId,
	   			"user_id":      request.UserId,
	   			"action_type":  request.ActionType,
	   			"content_text": request.Content,
	   		}).Errorf("User service error")
	   		logging.SetSpanError(span, err)

	   		return &chat.ActionResponse{
	   			StatusCode: strings.UnableToAddMessageErrorCode,
	   			StatusMsg:  strings.UnableToAddMessageError,
	   		}, err
	   	} */

	res, err = addMessage(ctx, request.ActorId, request.UserId, request.Content)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"err":          err,
			"user_id":      request.UserId,
			"action_type":  request.ActionType,
			"content_text": request.Content,
		}).Errorf("database insert  error")
		logging.SetSpanError(span, err)
		return res, err
	}

	logger.WithFields(logrus.Fields{
		"response": res,
	}).Debugf("Process done.")

	return res, err
}

// Chat Chat(context.Context, *ChatRequest) (*ChatResponse, error)
func (c MessageServiceImpl) Chat(ctx context.Context, request *chat.ChatRequest) (resp *chat.ChatResponse, err error) {
	ctx, span := tracing.Tracer.Start(ctx, "ChatService")
	defer span.End()
	logger := logging.LogService("ChatService.chat").WithContext(ctx)
	logger.WithFields(logrus.Fields{
		"user_id":      request.UserId,
		"ActorId":      request.ActorId,
		"pre_msg_time": request.PreMsgTime,
	}).Debugf("Process start")

	/* 	userResponse, err := UserClient.GetUserExistInformation(ctx, &user.UserExistRequest{
	   		UserId: request.UserId,
	   	})

	   	if err != nil || userResponse.StatusCode != strings.ServiceOKCode {
	   		logger.WithFields(logrus.Fields{
	   			"err":     err,
	   			"ActorId": request.ActorId,
	   			"user_id": request.UserId,
	   		}).Errorf("User service error")
	   		logging.SetSpanError(span, err)

	   		resp = &chat.ChatResponse{
	   			StatusCode: strings.UnableToQueryMessageErrorCode,
	   			StatusMsg:  strings.UnableToQueryMessageError,
	   		}
	   		return
	   	}
	*/
	toUserId := request.UserId
	fromUserId := request.ActorId

	conversationId := fmt.Sprintf("%d_%d", toUserId, fromUserId)

	if toUserId > fromUserId {
		conversationId = fmt.Sprintf("%d_%d", fromUserId, toUserId)
	}
	//这个地方应该取出多少条消息？
	//TO DO 看怎么需要改一下

	var pMessageList []models.Message
	result := database.Client.WithContext(ctx).
		Where("conversation_id=?", conversationId).
		Order("created_at desc").
		Find(&pMessageList)

	if result.Error != nil {
		logger.WithFields(logrus.Fields{
			"err":          result.Error,
			"user_id":      request.UserId,
			"ActorId":      request.ActorId,
			"pre_msg_time": request.PreMsgTime,
		}).Errorf("ChatServiceImpl list chat failed to response when listing message,database err")
		logging.SetSpanError(span, err)

		resp = &chat.ChatResponse{
			StatusCode: strings.UnableToQueryMessageErrorCode,
			StatusMsg:  strings.UnableToQueryMessageError,
		}
		return
	}

	rMessageList := make([]*chat.Message, 0, len(pMessageList))
	for _, pMessage := range pMessageList {
		rMessageList = append(rMessageList, &chat.Message{
			Id:         pMessage.ID,
			Content:    pMessage.Content,
			CreateTime: uint32(pMessage.CreatedAt.Unix()),
			FromUserId: &pMessage.FromUserId,
			ToUserId:   &pMessage.ToUserId,
		})
	}

	resp = &chat.ChatResponse{
		StatusCode:  strings.ServiceOKCode,
		StatusMsg:   strings.ServiceOK,
		MessageList: rMessageList,
	}

	logger.WithFields(logrus.Fields{
		"response": resp,
	}).Debugf("Process done.")

	return
}

func addMessage(ctx context.Context, fromUserId uint32, toUserId uint32, Context string) (resp *chat.ActionResponse, err error) {
	conversationId := fmt.Sprintf("%d_%d", toUserId, fromUserId)

	if toUserId > fromUserId {
		conversationId = fmt.Sprintf("%d_%d", fromUserId, toUserId)
	}
	message := models.Message{
		ToUserId:       toUserId,
		FromUserId:     fromUserId,
		Content:        Context,
		ConversationId: conversationId,
	}

	//TO_DO 后面写mq？

	body, err := json.Marshal(message)

	if err != nil {
		resp = &chat.ActionResponse{
			StatusCode: strings.UnableToAddMessageErrorCode,
			StatusMsg:  strings.UnableToAddMessageError,
		}
		return
	}
	headers := rabbitmq.InjectAMQPHeaders(ctx)
	err = channel.Publish("", "strings.MessageActionEvent", false, false,
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType:  "text/json",
			Body:         body,
			Headers:      headers,
		})

	// result := database.Client.WithContext(ctx).Create(&message)

	if err != nil {
		resp = &chat.ActionResponse{
			StatusCode: strings.UnableToAddMessageErrorCode,
			StatusMsg:  strings.UnableToAddMessageError,
		}
		return
	}

	resp = &chat.ActionResponse{
		StatusCode: strings.ServiceOKCode,
		StatusMsg:  strings.ServiceOK,
	}
	return

}
