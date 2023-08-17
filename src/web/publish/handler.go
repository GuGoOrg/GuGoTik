package publish

import (
	"GuGoTik/src/constant/config"
	"GuGoTik/src/constant/strings"
	"GuGoTik/src/extra/tracing"
	"GuGoTik/src/rpc/publish"
	grpc2 "GuGoTik/src/utils/grpc"
	"GuGoTik/src/utils/logging"
	"GuGoTik/src/web/models"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"net/http"
)

var Client publish.PublishServiceClient

func init() {
	conn := grpc2.Connect(config.PublishRpcServerName)
	Client = publish.NewPublishServiceClient(conn)
}

func ListPublishHandle(c *gin.Context) {
	_, span := tracing.Tracer.Start(c.Request.Context(), "Publish-ListHandle")
	defer span.End()
	logger := logging.LogService("GateWay.PublishList").WithContext(c.Request.Context())
	var req models.ListPublishReq
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusOK, models.ListPublishRes{
			StatusCode: strings.GateWayParamsErrorCode,
			StatusMsg:  strings.GateWayParamsError,
			VideoList:  nil,
		})
	}

	res, err := Client.ListVideo(c.Request.Context(), &publish.ListVideoRequest{
		ActorId: req.ActorId,
		UserId:  req.UserId,
	})
	if err != nil {
		logger.WithFields(logrus.Fields{
			"UserId": req.UserId,
		}).Warnf("Error when trying to connect with PublishService")
		c.JSON(http.StatusOK, res)
		return
	}
	userId := req.UserId
	logger.WithFields(logrus.Fields{
		"UserId": userId,
	}).Infof("Publish List videos")

	c.JSON(http.StatusOK, res)
}
