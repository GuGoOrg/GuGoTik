package message

import (
	"GuGoTik/src/constant/config"
	"GuGoTik/src/constant/strings"
	"GuGoTik/src/extra/tracing"
	"GuGoTik/src/rpc/chat"
	grpc2 "GuGoTik/src/utils/grpc"
	"GuGoTik/src/utils/logging"
	"GuGoTik/src/web/models"
	"GuGoTik/src/web/utils"

	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

var Client chat.ChatServiceClient

func init() {
	conn := grpc2.Connect(config.MessageRpcServerName)
	Client = chat.NewChatServiceClient(conn)
}

func ActionMessageHandler(c *gin.Context) {
	var req models.SMessageReq
	_, span := tracing.Tracer.Start(c.Request.Context(), "ActionMessageHandler")
	defer span.End()
	logging.SetSpanWithHostname(span)
	logger := logging.LogService("GateWay.ActionChat").WithContext(c.Request.Context())

	if err := c.ShouldBindQuery(&req); err != nil {
		logger.WithFields(logrus.Fields{
			//"CreateTime": req.Create_time,
			"ActorId":    req.ActorId,
			"ToUserId":   req.ToUserId,
			"ActionType": req.ActionType,
			"Content":    req.Content,
			"err":        err,
		}).Errorf("Error when trying to bind query")

		c.JSON(http.StatusOK, models.ActionCommentRes{
			StatusCode: strings.GateWayParamsErrorCode,
			StatusMsg:  strings.GateWayParamsError,
		})
		return
	}

	var res *chat.ActionResponse
	var err error

	if req.ActionType != 1 {
		c.JSON(http.StatusOK, models.ActionCommentRes{
			StatusCode: strings.GateWayParamsErrorCode,
			StatusMsg:  strings.GateWayParamsError,
		})
		return
	}

	res, err = Client.ChatAction(c.Request.Context(), &chat.ActionRequest{
		ActorId:    uint32(req.ActorId),
		UserId:     uint32(req.ToUserId),
		ActionType: uint32(req.ActionType),
		Content:    req.Content,
	})

	if err != nil {
		logger.WithFields(logrus.Fields{
			"ActorId":    req.ActorId,
			"ToUserId":   req.ToUserId,
			"ActionType": req.ActionType,
			"Content":    req.Content,
			"err":        err,
		}).Error("Error when trying to connect with ActionMessageHandler")

		c.Render(http.StatusBadRequest, utils.CustomJSON{Data: res, Context: c})
		return
	}
	logger.WithFields(logrus.Fields{
		"ActorId":    req.ActorId,
		"ToUserId":   req.ToUserId,
		"ActionType": req.ActionType,
		"Content":    req.Content,
		"err":        err,
	}).Infof("Action send message success")

	c.Render(http.StatusOK, utils.CustomJSON{Data: res, Context: c})
}

func ListMessageHandler(c *gin.Context) {
	var req models.ListMessageReq
	_, span := tracing.Tracer.Start(c.Request.Context(), "ListMessageHandler")
	defer span.End()
	logger := logging.LogService("GateWay.ListMessage").WithContext(c.Request.Context())

	if err := c.ShouldBindQuery(&req); err != nil {
		logger.WithFields(logrus.Fields{
			"ActorId":    req.ActorId,
			"ToUserId":   req.ToUserId,
			"PreMsgTime": req.PreMsgTime,
			"Err":        err,
		}).Error("Error when trying to bind query")
		c.JSON(http.StatusOK, models.ListCommentRes{
			StatusCode: strings.GateWayParamsErrorCode,
			StatusMsg:  strings.GateWayParamsError,
		})
		return
	}

	res, err := Client.Chat(c.Request.Context(), &chat.ChatRequest{
		ActorId:    req.ActorId,
		UserId:     req.ToUserId,
		PreMsgTime: req.PreMsgTime,
	})

	if err != nil {
		logger.WithFields(logrus.Fields{
			"ActorId":    req.ActorId,
			"ToUserId":   req.ToUserId,
			"PreMsgTime": req.PreMsgTime,
			"Err":        err,
		}).Error("Error when trying to connect with ListMessageHandler")
		c.Render(http.StatusOK, utils.CustomJSON{Data: res, Context: c})
		return
	}

	logger.WithFields(logrus.Fields{
		"ActorId": req.ActorId,
		"user_id": req.ToUserId,
	}).Infof("List message success")

	c.Render(http.StatusOK, utils.CustomJSON{Data: res, Context: c})
}
