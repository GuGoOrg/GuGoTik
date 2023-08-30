package comment

import (
	"GuGoTik/src/constant/config"
	"GuGoTik/src/constant/strings"
	"GuGoTik/src/extra/tracing"
	"GuGoTik/src/rpc/comment"
	grpc2 "GuGoTik/src/utils/grpc"
	"GuGoTik/src/utils/logging"
	"GuGoTik/src/web/models"
	"GuGoTik/src/web/utils"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"net/http"
)

var Client comment.CommentServiceClient

func init() {
	conn := grpc2.Connect(config.CommentRpcServerName)
	Client = comment.NewCommentServiceClient(conn)
}

func ActionCommentHandler(c *gin.Context) {
	var req models.ActionCommentReq
	_, span := tracing.Tracer.Start(c.Request.Context(), "ActionCommentHandler")
	defer span.End()
	logging.SetSpanWithHostname(span)
	logger := logging.LogService("GateWay.ActionComment").WithContext(c.Request.Context())

	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusOK, models.ActionCommentRes{
			StatusCode: strings.GateWayParamsErrorCode,
			StatusMsg:  strings.GateWayParamsError,
		})
		return
	}

	var res *comment.ActionCommentResponse
	var err error
	if req.ActionType == 1 {
		res, err = Client.ActionComment(c.Request.Context(), &comment.ActionCommentRequest{
			ActorId:    uint32(req.ActorId),
			VideoId:    uint32(req.VideoId),
			ActionType: comment.ActionCommentType_ACTION_COMMENT_TYPE_ADD,
			Action:     &comment.ActionCommentRequest_CommentText{CommentText: req.CommentText},
		})
	} else if req.ActionType == 2 {
		res, err = Client.ActionComment(c.Request.Context(), &comment.ActionCommentRequest{
			ActorId:    uint32(req.ActorId),
			VideoId:    uint32(req.VideoId),
			ActionType: comment.ActionCommentType_ACTION_COMMENT_TYPE_DELETE,
			Action:     &comment.ActionCommentRequest_CommentId{CommentId: uint32(req.CommentId)},
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
			"video_id": req.VideoId,
			"actor_id": req.ActorId,
		}).Warnf("Error when trying to connect with ActionCommentService")
		c.Render(http.StatusOK, utils.CustomJSON{Data: res, Context: c})
		return
	}

	logger.WithFields(logrus.Fields{
		"video_id": req.VideoId,
		"actor_id": req.ActorId,
	}).Infof("Action comment success")

	c.Render(http.StatusOK, utils.CustomJSON{Data: res, Context: c})
}

func ListCommentHandler(c *gin.Context) {
	var req models.ListCommentReq
	_, span := tracing.Tracer.Start(c.Request.Context(), "ListCommentHandler")
	defer span.End()
	logger := logging.LogService("GateWay.ListComment").WithContext(c.Request.Context())

	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusOK, models.ListCommentRes{
			StatusCode: strings.GateWayParamsErrorCode,
			StatusMsg:  strings.GateWayParamsError,
		})
		return
	}

	res, err := Client.ListComment(c.Request.Context(), &comment.ListCommentRequest{
		ActorId: uint32(req.ActorId),
		VideoId: uint32(req.VideoId),
	})
	if err != nil {
		logger.WithFields(logrus.Fields{
			"video_id": req.VideoId,
			"actor_id": req.ActorId,
		}).Warnf("Error when trying to connect with ListCommentService")
		c.Render(http.StatusOK, utils.CustomJSON{Data: res, Context: c})
		return
	}

	logger.WithFields(logrus.Fields{
		"video_id": req.VideoId,
		"actor_id": req.ActorId,
	}).Infof("List comment success")

	c.Render(http.StatusOK, utils.CustomJSON{Data: res, Context: c})
}

func CountCommentHandler(c *gin.Context) {
	var req models.CountCommentReq
	_, span := tracing.Tracer.Start(c.Request.Context(), "CountCommentHandler")
	defer span.End()
	logger := logging.LogService("GateWay.CountComment").WithContext(c.Request.Context())

	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusOK, models.CountCommentRes{
			StatusCode: strings.GateWayParamsErrorCode,
			StatusMsg:  strings.GateWayParamsError,
		})
		return
	}

	res, err := Client.CountComment(c.Request.Context(), &comment.CountCommentRequest{
		ActorId: uint32(req.ActorId),
		VideoId: uint32(req.VideoId),
	})
	if err != nil {
		logger.WithFields(logrus.Fields{
			"video_id": req.VideoId,
			"actor_id": req.ActorId,
		}).Warnf("Error when trying to connect with CountCommentService")
		c.Render(http.StatusOK, utils.CustomJSON{Data: res, Context: c})
		return
	}

	logger.WithFields(logrus.Fields{
		"video_id": req.VideoId,
		"actor_id": req.ActorId,
	}).Infof("Count comment success")

	c.Render(http.StatusOK, utils.CustomJSON{Data: res, Context: c})
}
