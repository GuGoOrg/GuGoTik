package feed

import (
	"GuGoTik/src/constant/config"
	"GuGoTik/src/constant/strings"
	"GuGoTik/src/extra/tracing"
	"GuGoTik/src/rpc/feed"
	grpc2 "GuGoTik/src/utils/grpc"
	"GuGoTik/src/utils/logging"
	"GuGoTik/src/web/models"
	"github.com/gin-gonic/gin"
	_ "github.com/mbobakov/grpc-consul-resolver"
	"github.com/sirupsen/logrus"
	"net/http"
)

var Client feed.FeedServiceClient

func ListVideosHandle(c *gin.Context) {
	var req models.ListVideosReq
	_, span := tracing.Tracer.Start(c.Request.Context(), "Feed-ListVideoHandle")
	defer span.End()
	logger := logging.LogService("GateWay.Videos").WithContext(c.Request.Context())

	if err := c.ShouldBindQuery(&req); err != nil {
		logger.WithFields(logrus.Fields{
			"latestTime": req.LatestTime,
			"err":        err,
		}).Warnf("Error when trying to bind query")
		c.JSON(http.StatusOK, models.ListVideosRes{
			StatusCode: strings.GateWayParamsErrorCode,
			StatusMsg:  strings.GateWayParamsError,
			NextTime:   nil,
			VideoList:  nil,
		})
	}

	latestTime := req.LatestTime
	res, err := Client.ListVideos(c.Request.Context(), &feed.ListFeedRequest{
		LatestTime: &latestTime,
	})
	if err != nil {
		logger.WithFields(logrus.Fields{
			"LatestTime": latestTime,
		}).Warnf("Error when trying to connect with FeedService")
		c.JSON(http.StatusOK, models.ListVideosRes{
			StatusCode: strings.FeedServiceInnerErrorCode,
			StatusMsg:  strings.FeedServiceInnerError,
			NextTime:   nil,
			VideoList:  nil,
		})
		return
	}

	logger.WithFields(logrus.Fields{
		"LatestTime": latestTime,
		"res":        res,
	}).Infof("Feed List videos")
	c.JSON(http.StatusOK, res)
}

func init() {
	conn := grpc2.Connect(config.FeedRpcServerName)
	Client = feed.NewFeedServiceClient(conn)
}
