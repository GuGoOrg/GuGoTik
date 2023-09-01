package relation

import (
	"GuGoTik/src/constant/config"
	"GuGoTik/src/constant/strings"
	"GuGoTik/src/extra/tracing"
	"GuGoTik/src/rpc/relation"
	grpc2 "GuGoTik/src/utils/grpc"
	"GuGoTik/src/utils/logging"
	"GuGoTik/src/web/models"
	"GuGoTik/src/web/utils"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"net/http"
)

var Client relation.RelationServiceClient

func init() {
	conn := grpc2.Connect(config.RelationRpcServerName)
	Client = relation.NewRelationServiceClient(conn)
}

// ActionRelationHandler todo: frontend interface   relation/action
func ActionRelationHandler(c *gin.Context) {

	var req models.RelationActionReq
	_, span := tracing.Tracer.Start(c.Request.Context(), "ActionRelationHandler")
	defer span.End()
	logging.SetSpanWithHostname(span)
	logger := logging.LogService("GateWay.ActionRelation").WithContext(c.Request.Context())

	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusOK, models.RelationActionRes{
			StatusCode: strings.GateWayParamsErrorCode,
			StatusMsg:  strings.GateWayParamsError,
		})
		return
	}

	var res *relation.RelationActionResponse
	var err error
	if req.ActionType == 1 {
		res, err = Client.Follow(c.Request.Context(), &relation.RelationActionRequest{
			ActorId: uint32(req.ActorId),
			UserId:  uint32(req.UserId),
		})
	} else if req.ActionType == 2 {
		res, err = Client.Unfollow(c.Request.Context(), &relation.RelationActionRequest{
			ActorId: uint32(req.ActorId),
			UserId:  uint32(req.UserId),
		})
	} else {
		c.JSON(http.StatusOK, models.ActionCommentRes{
			StatusCode: strings.GateWayParamsErrorCode,
			StatusMsg:  strings.GateWayParamsError,
		})
		return
	}

	if err != nil {
		logger.WithFields(logrus.Fields{
			"actor_id": req.ActorId,
			"user_id":  req.UserId,
		}).Warnf("RelationActionService returned an error response: %v", err)
		c.Render(http.StatusOK, utils.CustomJSON{Data: res, Context: c})
		return
	}

	logger.WithFields(logrus.Fields{
		"actor_id": req.ActorId,
		"user_id":  req.UserId,
	}).Infof("RelationAction success")

	c.Render(http.StatusOK, utils.CustomJSON{Data: res, Context: c})
}

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
		}).Warnf("FollowService returned an error response: %v", err)
		c.Render(http.StatusOK, utils.CustomJSON{Data: res, Context: c})
		return
	}

	logger.WithFields(logrus.Fields{
		"actor_id": req.ActorId,
		"user_id":  req.UserId,
	}).Infof("Follow success")

	c.Render(http.StatusOK, utils.CustomJSON{Data: res, Context: c})

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
		}).Warnf("UnFollowService returned an error response: %v", err)
		c.Render(http.StatusOK, utils.CustomJSON{Data: res, Context: c})
		return
	}

	logger.WithFields(logrus.Fields{
		"actor_id": req.ActorId,
		"user_id":  req.UserId,
	}).Infof("Unfollow success")

	c.Render(http.StatusOK, utils.CustomJSON{Data: res, Context: c})
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
		}).Warnf("GetFollowListService returned an error response: %v", err)
		c.Render(http.StatusOK, utils.CustomJSON{Data: res, Context: c})
		return
	}

	logger.WithFields(logrus.Fields{
		"actor_id": req.ActorId,
		"user_id":  req.UserId,
	}).Infof("GetFollowList success")

	c.Render(http.StatusOK, utils.CustomJSON{Data: res, Context: c})

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
		}).Warnf("CountFollowListService returned an error response: %v", err)
		c.Render(http.StatusOK, utils.CustomJSON{Data: res, Context: c})
		return
	}

	logger.WithFields(logrus.Fields{
		"user_id": req.UserId,
	}).Infof("Count follow success")
	c.Render(http.StatusOK, utils.CustomJSON{Data: res, Context: c})

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
		}).Warnf("GetFollowerListService returned an error response: %v", err)
		c.Render(http.StatusOK, utils.CustomJSON{Data: res, Context: c})
		return
	}

	logger.WithFields(logrus.Fields{
		"actor_id": req.ActorId,
		"user_id":  req.UserId,
	}).Infof("GetFollowerList success")

	c.Render(http.StatusOK, utils.CustomJSON{Data: res, Context: c})

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
		}).Warnf("CountFollowerListService returned an error response: %v", err)
		c.Render(http.StatusOK, utils.CustomJSON{Data: res, Context: c})
		return
	}

	logger.WithFields(logrus.Fields{
		"user_id": req.UserId,
	}).Infof("Count follower success")
	c.Render(http.StatusOK, utils.CustomJSON{Data: res, Context: c})
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
		}).Warnf("GetFriendListService returned an error response: %v", err)
		c.Render(http.StatusOK, utils.CustomJSON{Data: res, Context: c})
		return
	}

	logger.WithFields(logrus.Fields{
		"actor_id": req.ActorId,
		"user_id":  req.UserId,
	}).Infof("GetFriendList success")

	c.Render(http.StatusOK, utils.CustomJSON{Data: res, Context: c})

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
		}).Warnf("IsFollowService returned an error response: %v", err)
		c.Render(http.StatusOK, utils.CustomJSON{Data: res, Context: c})
		return
	}

	logger.WithFields(logrus.Fields{
		"actor_id": req.ActorId,
		"user_id":  req.UserId,
	}).Infof("IsFollow success")
	c.Render(http.StatusOK, utils.CustomJSON{Data: res, Context: c})
}
