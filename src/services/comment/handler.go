package main

import (
	"GuGoTik/src/constant/config"
	"GuGoTik/src/constant/strings"
	"GuGoTik/src/extra/tracing"
	"GuGoTik/src/models"
	"GuGoTik/src/rpc/comment"
	"GuGoTik/src/rpc/user"
	"GuGoTik/src/storage/database"
	"GuGoTik/src/utils/consul"
	grpc2 "GuGoTik/src/utils/grpc"
	"GuGoTik/src/utils/logging"
	"context"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/trace"
)

var UserClient user.UserServiceClient

type CommentServiceImpl struct {
	comment.CommentServiceServer
}

func init() {
	service, err := consul.ResolveService(config.UserRpcServerName)
	if err != nil {
		logging.Logger.WithFields(logrus.Fields{
			"err": err,
		}).Fatalf("Cannot find user rpc server")
	}

	logging.Logger.Debugf("Found service %v in port %v", service.ServiceID, service.ServicePort)

	conn, err := grpc2.Connect(service)
	if err != nil {
		logging.Logger.WithFields(logrus.Fields{
			"err": err,
		}).Fatalf("Cannot find user rpc server")
	}
	UserClient = user.NewUserServiceClient(conn)
}

// ActionComment implements the CommentServiceImpl interface.
func (c CommentServiceImpl) ActionComment(ctx context.Context, request *comment.ActionCommentRequest) (resp *comment.ActionCommentResponse, err error) {
	ctx, span := tracing.Tracer.Start(ctx, "CommentService-ActionComment")
	defer span.End()
	logger := logging.LogService("CommentService.ActionComment").WithContext(ctx)
	logger.WithFields(logrus.Fields{
		"user_id":      request.ActorId,
		"video_id":     request.VideoId,
		"action_type":  request.ActionType,
		"comment_text": request.GetCommentText(),
		"comment_id":   request.GetCommentId(),
	})
	logger.Debugf("Process start")

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
		return &comment.ActionCommentResponse{
			StatusCode: strings.ActionCommentTypeInvalidCode,
			StatusMsg:  strings.ActionCommentTypeInvalid,
		}, nil
	}

	// TODO: Video check: check if the qVideo exists || check if creator is the same as actor

	// Get target user
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

		return &comment.ActionCommentResponse{
			StatusCode: strings.UnableToQueryUserErrorCode,
			StatusMsg:  strings.UnableToQueryUserError,
		}, nil
	}

	pUser := userResponse.User

	switch request.ActionType {
	case comment.ActionCommentType_ACTION_COMMENT_TYPE_ADD:
		resp, err = addComment(ctx, logger, span, pUser, request.VideoId, pCommentText)
	case comment.ActionCommentType_ACTION_COMMENT_TYPE_DELETE:
		resp, err = deleteComment(ctx, logger, span, pUser, request.VideoId, pCommentID)
	}

	if err != nil {
		return resp, err
	}

	logger.WithFields(logrus.Fields{
		"response": resp,
	}).Debugf("Process done.")

	return resp, err
}

// ListComment TODO
func (c CommentServiceImpl) ListComment(ctx context.Context, request *comment.ListCommentRequest) (resp *comment.ListCommentResponse, err error) {

	return
}

// CountComment TODO
func (c CommentServiceImpl) CountComment(ctx context.Context, request *comment.CountCommentRequest) (resp *comment.CountCommentResponse, err error) {
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
		logging.SetSpanError(span, err)

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
	return
}

func count(ctx context.Context, videoID uint32) (count int64, err error) {
	result := database.Client.Model(&models.Comment{}).WithContext(ctx).Where("video_id = ?", videoID).Count(&count)
	return count, result.Error
}
