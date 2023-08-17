package main

import (
	"GuGoTik/src/constant/config"
	"GuGoTik/src/constant/strings"
	"GuGoTik/src/extra/tracing"
	"GuGoTik/src/models"
	"GuGoTik/src/rpc/comment"
	"GuGoTik/src/rpc/user"
	"GuGoTik/src/storage/cached"
	"GuGoTik/src/storage/database"
	"GuGoTik/src/storage/redis"
	grpc2 "GuGoTik/src/utils/grpc"
	"GuGoTik/src/utils/logging"
	"context"
	"fmt"
	"github.com/go-redis/redis_rate/v10"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/trace"
	"strconv"
)

var userClient user.UserServiceClient

var actionCommentLimitKeyPrefix = config.EnvCfg.RedisPrefix + "comment_freq_limit"

const actionCommentMaxQPS = 3 // Maximum ActionComment query amount of an actor per second

// Return redis key to record the amount of ActionComment query of an actor, e.g., comment_freq_limit-1-1669524458
func actionCommentLimitKey(userId uint32) string {
	return fmt.Sprintf("%s-%d", actionCommentLimitKeyPrefix, userId)
}

type CommentServiceImpl struct {
	comment.CommentServiceServer
}

func init() {
	userRpcConn := grpc2.Connect(config.UserRpcServerName)
	userClient = user.NewUserServiceClient(userRpcConn)
}

// ActionComment implements the CommentServiceImpl interface.
func (c CommentServiceImpl) ActionComment(ctx context.Context, request *comment.ActionCommentRequest) (resp *comment.ActionCommentResponse, err error) {
	ctx, span := tracing.Tracer.Start(ctx, "ActionCommentService")
	defer span.End()
	logger := logging.LogService("CommentService.ActionComment").WithContext(ctx)
	logger.WithFields(logrus.Fields{
		"user_id":      request.ActorId,
		"video_id":     request.VideoId,
		"action_type":  request.ActionType,
		"comment_text": request.GetCommentText(),
		"comment_id":   request.GetCommentId(),
	}).Debugf("Process start")

	var pCommentText string
	var pCommentID uint32

	switch request.ActionType {
	case comment.ActionCommentType_ACTION_COMMENT_TYPE_ADD:
		pCommentText = request.GetCommentText()
	case comment.ActionCommentType_ACTION_COMMENT_TYPE_DELETE:
		pCommentID = request.GetCommentId()
	case comment.ActionCommentType_ACTION_COMMENT_TYPE_UNSPECIFIED:
		fallthrough
	default:
		logger.Warnf("Invalid action type")
		resp = &comment.ActionCommentResponse{
			StatusCode: strings.ActionCommentTypeInvalidCode,
			StatusMsg:  strings.ActionCommentTypeInvalid,
		}
		return
	}

	// Rate limiting
	limiter := redis_rate.NewLimiter(redis.Client)
	limiterKey := actionCommentLimitKey(request.ActorId)
	limiterRes, err := limiter.Allow(ctx, limiterKey, redis_rate.PerSecond(actionCommentMaxQPS))
	if err != nil {
		logger.WithFields(logrus.Fields{
			"err":     err,
			"ActorId": request.ActorId,
		}).Errorf("ActionComment limiter error")
		logging.SetSpanError(span, err)

		resp = &comment.ActionCommentResponse{
			StatusCode: strings.UnableToCreateCommentErrorCode,
			StatusMsg:  strings.UnableToCreateCommentError,
		}

		return
	}
	if limiterRes.Allowed == 0 {
		logger.WithFields(logrus.Fields{
			"err":     err,
			"ActorId": request.ActorId,
		}).Infof("Action comment query too frequently by user %d", request.ActorId)

		resp = &comment.ActionCommentResponse{
			StatusCode: strings.ActionCommentLimitedCode,
			StatusMsg:  strings.ActionCommentLimited,
		}

		return
	}

	// Get target user
	userResponse, err := userClient.GetUserInfo(ctx, &user.UserRequest{
		UserId:  request.ActorId,
		ActorId: request.ActorId,
	})

	if err != nil || userResponse.StatusCode != strings.ServiceOKCode {
		logger.WithFields(logrus.Fields{
			"err":     err,
			"ActorId": request.ActorId,
		}).Errorf("User service error")
		logging.SetSpanError(span, err)

		resp = &comment.ActionCommentResponse{
			StatusCode: strings.UnableToQueryUserErrorCode,
			StatusMsg:  strings.UnableToQueryUserError,
		}
		return
	}

	pUser := userResponse.User

	switch request.ActionType {
	case comment.ActionCommentType_ACTION_COMMENT_TYPE_ADD:
		resp, err = addComment(ctx, logger, span, pUser, request.VideoId, pCommentText)
	case comment.ActionCommentType_ACTION_COMMENT_TYPE_DELETE:
		resp, err = deleteComment(ctx, logger, span, pUser, request.VideoId, pCommentID)
	}

	if err != nil {
		return
	}

	countCommentKey := fmt.Sprintf("CommentCount-%d", request.VideoId)
	cached.TagDelete(ctx, countCommentKey)

	logger.WithFields(logrus.Fields{
		"response": resp,
	}).Debugf("Process done.")

	return
}

// ListComment implements the CommentServiceImpl interface.
func (c CommentServiceImpl) ListComment(ctx context.Context, request *comment.ListCommentRequest) (resp *comment.ListCommentResponse, err error) {
	ctx, span := tracing.Tracer.Start(ctx, "ListCommentService")
	defer span.End()
	logger := logging.LogService("CommentService.ListComment").WithContext(ctx)
	logger.WithFields(logrus.Fields{
		"user_id":  request.ActorId,
		"video_id": request.VideoId,
	}).Debugf("Process start")

	var pCommentList []models.Comment
	result := database.Client.WithContext(ctx).
		Where("video_id = ?", request.VideoId).
		Order("created_at desc").
		Find(&pCommentList)
	if result.Error != nil {
		logger.WithFields(logrus.Fields{
			"err": result.Error,
		}).Errorf("CommentService list comment failed to response when listing comments")
		logging.SetSpanError(span, err)

		resp = &comment.ListCommentResponse{
			StatusCode: strings.UnableToQueryCommentErrorCode,
			StatusMsg:  strings.UnableToQueryCommentError,
		}
		return
	}

	rCommentList := make([]*comment.Comment, 0, result.RowsAffected)
	userList := make(map[uint32]*user.User, result.RowsAffected)
	for _, pComment := range pCommentList {
		curUser, ok := userList[pComment.UserId]
		if !ok {
			userResponse, err := userClient.GetUserInfo(ctx, &user.UserRequest{
				UserId:  pComment.UserId,
				ActorId: request.ActorId,
			})
			if err != nil || userResponse.StatusCode != strings.ServiceOKCode {
				logger.WithFields(logrus.Fields{
					"err":      err,
					"pComment": pComment,
				}).Errorf("Unable to get user info")
				logging.SetSpanError(span, err)
			}
			curUser = userResponse.User
			userList[pComment.UserId] = curUser
		}

		rCommentList = append(rCommentList, &comment.Comment{
			Id:         pComment.ID,
			User:       curUser,
			Content:    pComment.Content,
			CreateDate: pComment.CreatedAt.Format("01-02"),
		})
	}

	resp = &comment.ListCommentResponse{
		StatusCode:  strings.ServiceOKCode,
		StatusMsg:   strings.ServiceOK,
		CommentList: rCommentList,
	}

	logger.WithFields(logrus.Fields{
		"response": resp,
	}).Debugf("Process done.")

	return
}

// CountComment implements the CommentServiceImpl interface.
func (c CommentServiceImpl) CountComment(ctx context.Context, request *comment.CountCommentRequest) (resp *comment.CountCommentResponse, err error) {
	ctx, span := tracing.Tracer.Start(ctx, "CountCommentService")
	defer span.End()
	logger := logging.LogService("CommentService.CountComment").WithContext(ctx)
	logger.WithFields(logrus.Fields{
		"user_id":  request.ActorId,
		"video_id": request.VideoId,
	}).Debugf("Process start")

	countStringKey := fmt.Sprintf("CommentCount-%d", request.VideoId)
	countString := cached.GetWithFunc(ctx, countStringKey,
		func(ctx context.Context, key string) string {
			rCount, err := count(ctx, request.VideoId)
			if err != nil {
				return "error"
			}

			return strconv.FormatInt(rCount, 10)
		})

	rCount, err := strconv.ParseUint(countString, 10, 64)
	if countString == "error" || err != nil {
		cached.TagDelete(ctx, "CommentCount")
		logger.WithFields(logrus.Fields{
			"err":      err,
			"video_id": request.VideoId,
		}).Errorf("Unable to get comment count")
		logging.SetSpanError(span, err)

		resp = &comment.CountCommentResponse{
			StatusCode: strings.UnableToQueryCommentErrorCode,
			StatusMsg:  strings.UnableToQueryCommentError,
		}
		return
	}

	resp = &comment.CountCommentResponse{
		StatusCode:   strings.ServiceOKCode,
		StatusMsg:    strings.ServiceOK,
		CommentCount: uint32(rCount),
	}
	logger.WithFields(logrus.Fields{
		"response": resp,
	}).Debugf("Process done.")
	return
}

func addComment(ctx context.Context, logger *logrus.Entry, span trace.Span, pUser *user.User, pVideoID uint32, pCommentText string) (resp *comment.ActionCommentResponse, err error) {
	rComment := models.Comment{
		VideoId: pVideoID,
		UserId:  pUser.Id,
		Content: pCommentText,
	}

	result := database.Client.WithContext(ctx).Create(&rComment)
	if result.Error != nil {
		logger.WithFields(logrus.Fields{
			"err":        result.Error,
			"comment_id": rComment.ID,
			"video_id":   pVideoID,
		}).Errorf("CommentService add comment action failed to response when adding comment")
		logging.SetSpanError(span, result.Error)

		resp = &comment.ActionCommentResponse{
			StatusCode: strings.UnableToCreateCommentErrorCode,
			StatusMsg:  strings.UnableToCreateCommentError,
		}
		return
	}

	resp = &comment.ActionCommentResponse{
		StatusCode: strings.ServiceOKCode,
		StatusMsg:  strings.ServiceOK,
		Comment: &comment.Comment{
			Id:         rComment.ID,
			User:       pUser,
			Content:    rComment.Content,
			CreateDate: rComment.CreatedAt.Format("01-02"),
		},
	}
	return
}

func deleteComment(ctx context.Context, logger *logrus.Entry, span trace.Span, pUser *user.User, pVideoID uint32, commentID uint32) (resp *comment.ActionCommentResponse, err error) {
	rComment := models.Comment{}
	result := database.Client.WithContext(ctx).
		Where("video_id = ? AND id = ?", pVideoID, commentID).
		First(&rComment)
	if result.Error != nil {
		logger.WithFields(logrus.Fields{
			"err":        result.Error,
			"video_id":   pVideoID,
			"comment_id": commentID,
		}).Errorf("Failed to get the comment")
		logging.SetSpanError(span, result.Error)

		resp = &comment.ActionCommentResponse{
			StatusCode: strings.UnableToQueryCommentErrorCode,
			StatusMsg:  strings.UnableToQueryCommentError,
		}
		return
	}

	if rComment.UserId != pUser.Id {
		logger.Errorf("Comment creator and deletor not match")
		resp = &comment.ActionCommentResponse{
			StatusCode: strings.ActorIDNotMatchErrorCode,
			StatusMsg:  strings.ActorIDNotMatchError,
		}
		return
	}

	result = database.Client.WithContext(ctx).Delete(&models.Comment{}, commentID)
	if result.Error != nil {
		logger.WithFields(logrus.Fields{
			"err": result.Error,
		}).Errorf("Failed to delete comment")
		logging.SetSpanError(span, result.Error)

		resp = &comment.ActionCommentResponse{
			StatusCode: strings.UnableToDeleteCommentErrorCode,
			StatusMsg:  strings.UnableToDeleteCommentError,
		}
		return
	}
	resp = &comment.ActionCommentResponse{
		StatusCode: strings.ServiceOKCode,
		StatusMsg:  strings.ServiceOK,
		Comment:    nil,
	}
	return
}

func count(ctx context.Context, videoId uint32) (count int64, err error) {
	ctx, span := tracing.Tracer.Start(ctx, "CountComment")
	defer span.End()
	logger := logging.LogService("CommentService.CountComment").WithContext(ctx)

	result := database.Client.Model(&models.Comment{}).WithContext(ctx).
		Where("video_id = ?", videoId).
		Count(&count)

	if result.Error != nil {
		logger.WithFields(logrus.Fields{
			"err": err,
		}).Errorf("Faild to count comments")
		logging.SetSpanError(span, err)
	}
	return count, result.Error
}
