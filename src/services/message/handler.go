package main

import (
	"GuGoTik/src/constant/config"
	"GuGoTik/src/constant/strings"
	"GuGoTik/src/extra/tracing"
	"GuGoTik/src/models"
	"GuGoTik/src/rpc/chat"
	"GuGoTik/src/rpc/user"
	"GuGoTik/src/storage/database"
	grpc2 "GuGoTik/src/utils/grpc"
	"GuGoTik/src/utils/logging"
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
)

var UserClient user.UserServiceClient

type MessageServiceImpl struct {
	chat.ChatServiceServer
}

func (c MessageServiceImpl) New() {
	userRpcConn := grpc2.Connect(config.UserRpcServerName)
	UserClient = user.NewUserServiceClient(userRpcConn)
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

	userResponse, err := UserClient.GetUserInfo(ctx, &user.UserRequest{
		ActorId: request.ActorId,
		UserId:  request.UserId,
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
			StatusMsg:  strings.UnableToAddMessageRrror,
		}, err
	}

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
	toUserId := request.UserId
	fromUserId := request.ActorId

	conversationId := fmt.Sprintf("%d_%d", toUserId, fromUserId)

	if toUserId > fromUserId {
		conversationId = fmt.Sprintf("%d_%d", fromUserId, toUserId)
	}
	//这个地方应该取出多少条消息？
	//TO DO 看怎么需要一下

	var rMessageList []*chat.Message
	result := database.Client.WithContext(ctx).Where("conversation_id=?", conversationId).
		Order("created_at desc").Find(&rMessageList)

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
	result := database.Client.WithContext(ctx).Create(&message)

	if result.Error != nil {

		resp = &chat.ActionResponse{
			StatusCode: strings.UnableToAddMessageErrorCode,
			StatusMsg:  strings.UnableToAddMessageRrror,
		}
		return
	}

	resp = &chat.ActionResponse{
		StatusCode: strings.ServiceOKCode,
		StatusMsg:  strings.ServiceOK,
	}
	return

}
