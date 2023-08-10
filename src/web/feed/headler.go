package feed

import (
	"GuGoTik/src/constant/config"
	"GuGoTik/src/constant/strings"
	"GuGoTik/src/extra/tracing"
	"GuGoTik/src/rpc/feed"
	"GuGoTik/src/utils/logging"
	"GuGoTik/src/web/models"
	"fmt"
	"net/http"
	"strconv"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"

	"github.com/gin-gonic/gin"
	_ "github.com/mbobakov/grpc-consul-resolver"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var Client feed.FeedServiceClient

func ListVideosHandle(c *gin.Context) {

	var req models.ListVideosReq
	_, span := tracing.Tracer.Start(c.Request.Context(), "Feed-ListVideoHandle")
	defer span.End()
	logger := logging.LogService("GateWay.Videos").WithContext(c.Request.Context())

	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusOK, models.ListVideosRes{
			StatusCode: strings.GateWayParamsErrorCode,
			StatusMsg:  strings.GateWayParamsError,
			NextTime:   nil,
			VideoList:  nil,
		})
		return
	}

	res, err := Client.ListVideos(c.Request.Context(), &feed.ListFeedRequest{})
	if err != nil {
		logger.WithFields(logrus.Fields{
			"LatestTime": req.LatestTime,
		}).Warnf("Error when trying to connect with FeedService")
		c.JSON(http.StatusOK, models.ListVideosRes{
			StatusCode: strings.FeedServiceInnerErrorCode,
			StatusMsg:  strings.FeedServiceInnerError,
			NextTime:   nil,
			VideoList:  nil,
		})
		return
	}
	latestTime := req.LatestTime
	if _, err := strconv.Atoi(latestTime); latestTime != "" && err != nil {
		logger.WithFields(logrus.Fields{
			"LatestTime": req.LatestTime,
		}).Warnf("Error when trying to convert LatestTime to int")
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
	}).Infof("Feed List videos")
	c.JSON(http.StatusOK, res)
}

func init() {
	//Dial creates a client connection to the given target.
	conn, err := grpc.Dial(
		fmt.Sprintf("consul://%s/%s?wait=15s", config.EnvCfg.ConsulAddr, config.FeedRpcServerName),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy": "round_robin"}`),
		grpc.WithUnaryInterceptor(otelgrpc.UnaryClientInterceptor()),
	)

	if err != nil {
		logging.Logger.WithFields(logrus.Fields{
			"err": err,
		}).Errorf("Build FeedService Client meet trouble")
	}
	Client = feed.NewFeedServiceClient(conn)
}
