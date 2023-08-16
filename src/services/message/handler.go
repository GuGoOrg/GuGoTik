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

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/trace"
)

var UserClient user.UserServiceClient

type MessageServiceImpl struct {
	chat.ChatServiceServer
}

func init() {
	userRpcConn := grpc2.Connect(config.UserRpcServerName)
	UserClient = user.NewUserServiceClient(userRpcConn)
}

func (c MessageServiceImpl) ChatAction(ctx context.Context, request *chat.ActionRequest) (res *chat.ActionResponse, err error) {
	ctx, span := tracing.Tracer.Start(ctx, "MessageActionService")
	defer span.End()
	logger := logging.LogService("MessageService.ActionMessage").WithContext(ctx)

	logger.WithFields(logrus.Fields{
		"actor_id":     request.ActorId,
		"user_id":      request.UserId,
		"action_type":  request.ActionType,
		"Content_text": request.Content,
	})
	logger.Debugf("Process start")

	userResponse, err := UserClient.GetUserInfo(ctx, &user.UserRequest{
		UserId:  request.ActorId,
		ActorId: request.ActorId,
	})

	if err != nil || userResponse.StatusCode != strings.ServiceOKCode {
		logger.WithFields(logrus.Fields{
			"err":     err,
			"ActorId": request.ActorId,
		}).Errorf("User service error")
		logging.SetSpanError(span, err)

		return &chat.ActionResponse{
			StatusCode: strings.UnableToQueryUserErrorCode,
			StatusMsg:  strings.UnableToQueryUserError,
		}, nil
	}

	pUser := userResponse.User

	res, err = addMessage(ctx, logger, span, pUser, request.UserId, request.Content)

	if err != nil {
		return res, err
	}

	logger.WithFields(logrus.Fields{
		"response": res,
	}).Debugf("Process done.")

	return res, err
}

// Chat(context.Context, *ChatRequest) (*ChatResponse, error)
func (c MessageServiceImpl) Chat(ctx context.Context, request *chat.ChatRequest) (resp *chat.ChatResponse, err error) {
	ctx, span := tracing.Tracer.Start(ctx, "MessageService")
	defer span.End()
	logger := logging.LogService("MessageService.chat").WithContext(ctx)
	logger.WithFields(logrus.Fields{
		"user_id": request.UserId,
		"from_id": request.ActorId,
	})
	logger.Debugf("Process start")

	var rMessageList []*chat.Message
	result := database.Client.WithContext(ctx).Where("to_user_id=?", request.UserId).Find(&rMessageList)

	if result.Error != nil {
		logger.WithFields(logrus.Fields{
			"err": result.Error,
		}).Errorf("MessageServiceImpl list comment failed to response when listing message")
		logging.SetSpanError(span, err)

		resp = &chat.ChatResponse{
			StatusCode: strings.UnableToQueryCommentErrorCode,
			StatusMsg:  strings.UnableToQueryCommentError,
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

func addMessage(ctx context.Context, logger *logrus.Entry, span trace.Span, pUser *user.User, to_user_id uint32, Context string) (resp *chat.ActionResponse, err error) {
	message := models.Message{
		To_user_id:   to_user_id,
		From_user_id: pUser.Id,
		Content:      Context,
	}

	result := database.Client.WithContext(ctx).Create(&message)

	if result.Error != nil {
		//TO_DO 错误 替换
		resp = &chat.ActionResponse{
			StatusCode: 400,
			StatusMsg:  "发生错误了",
		}
		return
	}

	resp = &chat.ActionResponse{
		StatusCode: strings.ServiceOKCode,
		StatusMsg:  strings.ServiceOK,
	}
	return

}
