package message

import (
	"GuGoTik/src/constant/config"
	"GuGoTik/src/constant/strings"
	"GuGoTik/src/extra/tracing"
	"GuGoTik/src/rpc/chat"
	grpc2 "GuGoTik/src/utils/grpc"
	"GuGoTik/src/utils/logging"
	"GuGoTik/src/web/models"

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
	logger := logging.LogService("GateWay.ActionChat").WithContext(c.Request.Context())

	if err := c.ShouldBindQuery(&req); err != nil {
		logger.WithFields(logrus.Fields{
			//"CreateTime": req.Create_time,
			"user_id": req.ActorId,
			"from_id": req.UserId,
			"err":     err,
		}).Errorf("Error when trying to bind query")

		c.JSON(http.StatusOK, models.ActionCommentRes{
			StatusCode: strings.GateWayParamsErrorCode,
			StatusMsg:  strings.GateWayParamsError,
		})
		return
	}

	var res *chat.ActionResponse
	var err error

	res, err = Client.ChatAction(c.Request.Context(), &chat.ActionRequest{
		ActorId:    uint32(req.ActorId),
		UserId:     uint32(req.UserId),
		ActionType: uint32(req.Action_type),
		Content:    req.Content,
	})

	if err != nil {
		logger.WithFields(logrus.Fields{
			"actor_id": req.ActorId,
			"content":  req.Content,
		}).Error("Error when trying to connect with ActionMessageHandler")

		c.JSON(http.StatusBadRequest, res)
		return
	}
	logger.WithFields(logrus.Fields{
		"actor_id": req.ActorId,
		"content":  req.Content,
	}).Infof("Action send message success")

	c.JSON(http.StatusOK, res)
}

func ListMessageHandler(c *gin.Context) {
	var req models.ListMessageReq
	_, span := tracing.Tracer.Start(c.Request.Context(), "ListMessageHandler")
	defer span.End()
	logger := logging.LogService("GateWay.ListMessage").WithContext(c.Request.Context())

	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusOK, models.ListCommentRes{
			StatusCode: strings.GateWayParamsErrorCode,
			StatusMsg:  strings.GateWayParamsError,
		})
		return
	}

	res, err := Client.Chat(c.Request.Context(), &chat.ChatRequest{
		ActorId:    req.ActorId,
		UserId:     req.UserId,
		PreMsgTime: req.PreMsgTime,
	})

	if err != nil {
		logger.WithFields(logrus.Fields{
			"actor_id": req.ActorId,
			"user_id":  req.UserId,
		}).Error("Error when trying to connect with ListMessageHandler")
		c.JSON(http.StatusOK, res)
		return
	}

	logger.WithFields(logrus.Fields{
		"actor_id": req.ActorId,
		"user_id":  req.UserId,
	}).Infof("List comment success")

	c.JSON(http.StatusOK, res)
}
