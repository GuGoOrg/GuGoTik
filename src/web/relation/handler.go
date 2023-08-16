package relation

import (
	"GuGoTik/src/constant/config"
	"GuGoTik/src/constant/strings"
	"GuGoTik/src/extra/tracing"
	"GuGoTik/src/rpc/relation"
	grpc2 "GuGoTik/src/utils/grpc"
	"GuGoTik/src/utils/logging"
	"GuGoTik/src/web/models"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"net/http"
)

var Client relation.RelationServiceClient

func init() {
	conn := grpc2.Connect(config.RelationRpcServerName)
	Client = relation.NewRelationServiceClient(conn)
}

//todo: frontend interface   relation/action
//func ActionHandler(c *gin.Context) {
//	actiontype := c.Query("")
//}

func FollowHandler(c *gin.Context) {

	var req models.RelationActionReq
	_, span := tracing.Tracer.Start(c.Request.Context(), "FollowHandler")
	defer span.End()
	logger := logging.LogService("GateWay.Follow").WithContext(c.Request.Context())

	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusOK, models.RelationActionRes{
			StatusCode: strings.GateWayParamsErrorCode,
			StatusMsg:  strings.GateWayParamsError,
		})
		return
	}

	res, err := Client.Follow(c.Request.Context(), &relation.RelationActionRequest{
		ActorId: uint32(req.ActorId),
		UserId:  uint32(req.UserId),
	})

	if err != nil {
		logger.WithFields(logrus.Fields{
			"actor_id": req.ActorId,
			"user_id":  req.UserId,
		}).Warnf("Error when trying to connect with FollowService")
		c.JSON(http.StatusOK, res)
		return
	}

	logger.WithFields(logrus.Fields{
		"actor_id": req.ActorId,
		"user_id":  req.UserId,
	}).Infof("Follow success")

	c.JSON(http.StatusOK, res)

}

func UnfollowHandler(c *gin.Context) {
	var req models.RelationActionReq
	_, span := tracing.Tracer.Start(c.Request.Context(), "UnFollowHandler")
	defer span.End()
	logger := logging.LogService("GateWay.UnFollow").WithContext(c.Request.Context())

	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusOK, models.RelationActionRes{
			StatusCode: strings.GateWayParamsErrorCode,
			StatusMsg:  strings.GateWayParamsError,
		})
		return
	}

	res, err := Client.Unfollow(c.Request.Context(), &relation.RelationActionRequest{
		ActorId: uint32(req.ActorId),
		UserId:  uint32(req.UserId),
	})

	if err != nil {
		logger.WithFields(logrus.Fields{
			"actor_id": req.ActorId,
			"user_id":  req.UserId,
		}).Warnf("Error when trying to connect with UnfollowService")
		c.JSON(http.StatusOK, res)
		return
	}

	logger.WithFields(logrus.Fields{
		"actor_id": req.ActorId,
		"user_id":  req.UserId,
	}).Infof("Unfollow success")

	c.JSON(http.StatusOK, res)
}

func GetFollowListHandler(c *gin.Context) {
	var req models.FollowListReq
	_, span := tracing.Tracer.Start(c.Request.Context(), "GetFollowListHandler")
	defer span.End()
	logger := logging.LogService("GateWay.GetFollowList").WithContext(c.Request.Context())

	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusOK, models.FollowListRes{
			StatusCode: strings.GateWayParamsErrorCode,
			StatusMsg:  strings.GateWayParamsError,
		})
		return
	}

	res, err := Client.GetFollowList(c.Request.Context(), &relation.FollowListRequest{
		ActorId: uint32(req.ActorId),
		UserId:  uint32(req.UserId),
	})
	if err != nil {
		logger.WithFields(logrus.Fields{
			"actor_id": req.ActorId,
			"user_id":  req.UserId,
		}).Warnf("Error when trying to connect with GetFollowListService")
		c.JSON(http.StatusOK, res)
		return
	}

	logger.WithFields(logrus.Fields{
		"actor_id": req.ActorId,
		"user_id":  req.UserId,
	}).Infof("GetFollowList success")

	c.JSON(http.StatusOK, res)

}

func CountFollowHandler(c *gin.Context) {
	var req models.CountFollowListReq
	_, span := tracing.Tracer.Start(c.Request.Context(), "CountFollowHandler")
	defer span.End()
	logger := logging.LogService("GateWay.CountFollow").WithContext(c.Request.Context())

	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusOK, models.CountFollowListRes{
			StatusCode: strings.GateWayParamsErrorCode,
			StatusMsg:  strings.GateWayParamsError,
		})
		return
	}

	res, err := Client.CountFollowList(c.Request.Context(), &relation.CountFollowListRequest{
		UserId: uint32(req.UserId),
	})
	if err != nil {
		logger.WithFields(logrus.Fields{
			"user_id": req.UserId,
		}).Warnf("Error when trying to connect with CountFollowListService")
		c.JSON(http.StatusOK, res)
		return
	}

	logger.WithFields(logrus.Fields{
		"user_id": req.UserId,
	}).Infof("Count follow success")
	c.JSON(http.StatusOK, res)

}

func GetFollowerListHandler(c *gin.Context) {
	var req models.FollowerListReq
	_, span := tracing.Tracer.Start(c.Request.Context(), "GetFollowerListHandler")
	defer span.End()
	logger := logging.LogService("GateWay.GetFollowerList").WithContext(c.Request.Context())

	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusOK, models.FollowerListRes{
			StatusCode: strings.GateWayParamsErrorCode,
			StatusMsg:  strings.GateWayParamsError,
		})
		return
	}

	res, err := Client.GetFollowerList(c.Request.Context(), &relation.FollowerListRequest{
		ActorId: uint32(req.ActorId),
		UserId:  uint32(req.UserId),
	})
	if err != nil {
		logger.WithFields(logrus.Fields{
			"actor_id": req.ActorId,
			"user_id":  req.UserId,
		}).Warnf("Error when trying to connect with GetFollowerListService")
		c.JSON(http.StatusOK, res)
		return
	}

	logger.WithFields(logrus.Fields{
		"actor_id": req.ActorId,
		"user_id":  req.UserId,
	}).Infof("GetFollowerList success")

	c.JSON(http.StatusOK, res)

}

func CountFollowerHandler(c *gin.Context) {
	var req models.CountFollowerListReq
	_, span := tracing.Tracer.Start(c.Request.Context(), "CounterFollowHandler")
	defer span.End()
	logger := logging.LogService("GateWay.CountFollower").WithContext(c.Request.Context())

	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusOK, models.CountFollowerListRes{
			StatusCode: strings.GateWayParamsErrorCode,
			StatusMsg:  strings.GateWayParamsError,
		})
		return
	}

	res, err := Client.CountFollowerList(c.Request.Context(), &relation.CountFollowerListRequest{
		UserId: uint32(req.UserId),
	})
	if err != nil {
		logger.WithFields(logrus.Fields{
			"user_id": req.UserId,
		}).Warnf("Error when trying to connect with CountFollowerListService")
		c.JSON(http.StatusOK, res)
		return
	}

	logger.WithFields(logrus.Fields{
		"user_id": req.UserId,
	}).Infof("Count follower success")
	c.JSON(http.StatusOK, res)
}

func GetFriendListHandler(c *gin.Context) {

	var req models.FriendListReq
	_, span := tracing.Tracer.Start(c.Request.Context(), "GetFriendListHandler")
	defer span.End()
	logger := logging.LogService("GateWay.GetFriendList").WithContext(c.Request.Context())

	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusOK, models.FriendListRes{
			StatusCode: strings.GateWayParamsErrorCode,
			StatusMsg:  strings.GateWayParamsError,
		})
		return
	}

	res, err := Client.GetFriendList(c.Request.Context(), &relation.FriendListRequest{
		ActorId: uint32(req.ActorId),
		UserId:  uint32(req.UserId),
	})
	if err != nil {
		logger.WithFields(logrus.Fields{
			"actor_id": req.ActorId,
			"user_id":  req.UserId,
		}).Warnf("Error when trying to connect with GetFriendListService")
		c.JSON(http.StatusOK, res)
		return
	}

	logger.WithFields(logrus.Fields{
		"actor_id": req.ActorId,
		"user_id":  req.UserId,
	}).Infof("GetFriendList success")

	c.JSON(http.StatusOK, res)

}

func IsFollowHandler(c *gin.Context) {

	var req models.IsFollowReq
	_, span := tracing.Tracer.Start(c.Request.Context(), "IsFollowHandler")
	defer span.End()
	logger := logging.LogService("GateWay.IsFollow").WithContext(c.Request.Context())

	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusOK, models.IsFollowRes{
			Result: false,
		})
		return
	}

	res, err := Client.IsFollow(c.Request.Context(), &relation.IsFollowRequest{
		ActorId: uint32(req.ActorId),
		UserId:  uint32(req.UserId),
	})
	if err != nil {
		logger.WithFields(logrus.Fields{
			"actor_id": req.ActorId,
			"user_id":  req.UserId,
		}).Warnf("Error when trying to connect with IsFollowService")
		c.JSON(http.StatusOK, res)
		return
	}

	logger.WithFields(logrus.Fields{
		"actor_id": req.ActorId,
		"user_id":  req.UserId,
	}).Infof("IsFollow success")
	c.JSON(http.StatusOK, res)
}
