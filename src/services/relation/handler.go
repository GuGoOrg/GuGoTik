package main

import (
	"GuGoTik/src/constant/config"
	"GuGoTik/src/constant/strings"
	"GuGoTik/src/extra/tracing"
	"GuGoTik/src/models"
	"GuGoTik/src/rpc/relation"
	"GuGoTik/src/rpc/user"
	"GuGoTik/src/storage/database"
	grpc2 "GuGoTik/src/utils/grpc"
	"GuGoTik/src/utils/logging"
	"context"
	"github.com/sirupsen/logrus"
)

var UserClient user.UserServiceClient

type RelationServiceImpl struct {
	relation.RelationServiceServer
}

func init() {
	userRpcConn := grpc2.Connect(config.UserRpcServerName)
	UserClient = user.NewUserServiceClient(userRpcConn)
}

func (r RelationServiceImpl) Follow(ctx context.Context, request *relation.RelationActionRequest) (resp *relation.RelationActionResponse, err error) {
	ctx, span := tracing.Tracer.Start(ctx, "FollowService")
	defer span.End()
	logger := logging.LogService("RelationService.Follow").WithContext(ctx)

	//检查登陆状态 todo

	userResponse, err := UserClient.GetUserInfo(ctx, &user.UserRequest{
		UserId:  request.UserId,
		ActorId: request.ActorId,
	})

	if err != nil || userResponse.StatusCode != strings.ServiceOKCode {
		logger.WithFields(logrus.Fields{
			"err":     err,
			"ActorId": request.ActorId,
		}).Errorf("User service error")
		logging.SetSpanError(span, err)

		return &relation.RelationActionResponse{
			StatusCode: strings.UnableToQueryUserErrorCode,
			StatusMsg:  strings.UnableToQueryUserError,
		}, nil
	}

	rRelation := &models.Relation{
		ActorId: request.ActorId, // 关注者的 ID
		UserId:  request.UserId,  // 被关注者的 ID
	}

	result := database.Client.WithContext(ctx).Create(&rRelation)

	if result.Error != nil {
		resp = &relation.RelationActionResponse{
			StatusCode: strings.UnableToFollowErrorCode,
			StatusMsg:  strings.UnableToFollowError,
		}
		return
	}

	//更新关注数和粉丝数
	UpdateFollowerCount()
	UpdateFollowCount()

	resp = &relation.RelationActionResponse{
		StatusCode: strings.ServiceOKCode,
		StatusMsg:  strings.ServiceOK,
	}

	return
}

func (r RelationServiceImpl) Unfollow(ctx context.Context, request *relation.RelationActionRequest) (resp *relation.RelationActionResponse, err error) {

	ctx, span := tracing.Tracer.Start(ctx, "UnfollowService")
	defer span.End()
	logger := logging.LogService("RelationService.Unfollow").WithContext(ctx)

	userResponse, err := UserClient.GetUserInfo(ctx, &user.UserRequest{
		UserId:  request.UserId,
		ActorId: request.ActorId,
	})

	if err != nil || userResponse.StatusCode != strings.ServiceOKCode {
		logger.WithFields(logrus.Fields{
			"err":     err,
			"ActorId": request.ActorId,
		}).Errorf("User service error")
		logging.SetSpanError(span, err)

		return &relation.RelationActionResponse{
			StatusCode: strings.UnableToQueryUserErrorCode,
			StatusMsg:  strings.UnableToQueryUserError,
		}, nil
	}

	rRelation := models.Relation{
		ActorId: request.ActorId,
		UserId:  request.UserId,
	}

	result := database.Client.WithContext(ctx).
		Where(&rRelation).
		Delete(&rRelation)

	if result.Error != nil {
		resp = &relation.RelationActionResponse{
			StatusCode: strings.UnableToUnFollowErrorCode,
			StatusMsg:  strings.UnableToUnFollowError,
		}
		return
	}

	//更新关注数和粉丝数
	UpdateFollowerCount()
	UpdateFollowCount()

	resp = &relation.RelationActionResponse{
		StatusCode: strings.ServiceOKCode,
		StatusMsg:  strings.ServiceOK,
	}
	return
}

func (r RelationServiceImpl) GetFollowList(ctx context.Context, request *relation.FollowerListRequest) (resp *relation.FollowListResponse, err error) {
	ctx, span := tracing.Tracer.Start(ctx, "GetFollowListService")
	defer span.End()
	logger := logging.LogService("RelationService.GetFollowList").WithContext(ctx)

	var followList []models.Relation
	result := database.Client.WithContext(ctx).
		Where("actor_id = ?", request.UserId).
		Order("created_at desc").
		Find(&followList)

	if result.Error != nil {
		logger.WithFields(logrus.Fields{
			"err": result.Error,
		}).Errorf("GetFollowListService list follow failed to response when listing follows")
		logging.SetSpanError(span, err)

		resp = &relation.FollowListResponse{
			StatusCode: strings.UnableToGetFollowListErrorCode,
			StatusMsg:  strings.UnableToGetFollowListError,
		}
		return
	}

	rFollowList := make([]*user.User, 0, result.RowsAffected)
	for _, follow := range followList {
		userResponse, err := UserClient.GetUserInfo(ctx, &user.UserRequest{
			UserId:  follow.UserId,
			ActorId: request.ActorId,
		})
		if err != nil || userResponse.StatusCode != strings.ServiceOKCode {
			logger.WithFields(logrus.Fields{
				"err":    err,
				"follow": follow,
			}).Errorf("Unable to get user info")
			logging.SetSpanError(span, err)
		}
		rFollowList = append(rFollowList, userResponse.User)
	}

	resp = &relation.FollowListResponse{
		StatusCode: strings.ServiceOKCode,
		StatusMsg:  strings.ServiceOK,
		UserList:   rFollowList,
	}
	return
}

func (r RelationServiceImpl) CountFollowList(ctx context.Context, request *relation.CountFollowListRequest) (resp *relation.CountFollowListResponse, err error) {
	ctx, span := tracing.Tracer.Start(ctx, "CountFollowListService")
	defer span.End()
	logger := logging.LogService("RelationService.CountFollowList").WithContext(ctx)

	var count int64
	result := database.Client.WithContext(ctx).
		Model(&models.Relation{}).
		Where("actor_id = ?", request.UserId).
		Count(&count)

	if result.Error != nil {
		logger.WithFields(logrus.Fields{
			"err": result.Error,
		}).Errorf("CountFollowListService failed to count follows")
		logging.SetSpanError(span, err)

		resp = &relation.CountFollowListResponse{
			StatusCode: strings.UnableToGetFollowListErrorCode,
			StatusMsg:  strings.UnableToGetFollowListError,
		}
		return
	}

	resp = &relation.CountFollowListResponse{
		StatusCode: strings.ServiceOKCode,
		StatusMsg:  strings.ServiceOK,
		Count:      uint32(count),
	}

	return
}

func (r RelationServiceImpl) GetFollowerList(ctx context.Context, request *relation.FollowerListRequest) (resp *relation.FollowerListResponse, err error) {

	ctx, span := tracing.Tracer.Start(ctx, "GetFollowerListService")
	defer span.End()
	logger := logging.LogService("RelationService.GetFollowerList").WithContext(ctx)

	var followerList []models.Relation
	result := database.Client.WithContext(ctx).
		Where("user_id = ?", request.UserId).
		Order("created_at desc").
		Find(&followerList)

	if result.Error != nil {
		logger.WithFields(logrus.Fields{
			"err": result.Error,
		}).Errorf("GetFollowerListService list follower failed to response when listing followers")
		logging.SetSpanError(span, err)

		resp = &relation.FollowerListResponse{
			StatusCode: strings.UnableToGetFollowerListErrorCode,
			StatusMsg:  strings.UnableToGetFollowerListError,
		}
		return
	}

	rFollowerList := make([]*user.User, 0, result.RowsAffected)
	for _, follower := range followerList {
		userResponse, err := UserClient.GetUserInfo(ctx, &user.UserRequest{
			UserId:  follower.UserId,
			ActorId: request.ActorId,
		})
		if err != nil || userResponse.StatusCode != strings.ServiceOKCode {
			logger.WithFields(logrus.Fields{
				"err":      err,
				"follower": follower,
			}).Errorf("Unable to get user info")
			logging.SetSpanError(span, err)
		}
		rFollowerList = append(rFollowerList, userResponse.User)
	}

	resp = &relation.FollowerListResponse{
		StatusCode: strings.ServiceOKCode,
		StatusMsg:  strings.ServiceOK,
		UserList:   rFollowerList,
	}
	return

}

func (r RelationServiceImpl) CountFollowerList(ctx context.Context, request *relation.CountFollowerListRequest) (resp *relation.CountFollowerListResponse, err error) {
	ctx, span := tracing.Tracer.Start(ctx, "CountFollowerListService")
	defer span.End()
	logger := logging.LogService("RelationService.CountFollowerList").WithContext(ctx)

	var count int64
	result := database.Client.WithContext(ctx).
		Model(&models.Relation{}).
		Where("user_id = ?", request.UserId).
		Count(&count)

	if result.Error != nil {
		logger.WithFields(logrus.Fields{
			"err": result.Error,
		}).Errorf("CountFollowerListService failed to count follows")
		logging.SetSpanError(span, err)

		resp = &relation.CountFollowerListResponse{
			StatusCode: strings.UnableToGetFollowerListErrorCode,
			StatusMsg:  strings.UnableToGetFollowerListError,
		}
		return
	}

	resp = &relation.CountFollowerListResponse{
		StatusCode: strings.ServiceOKCode,
		StatusMsg:  strings.ServiceOK,
		Count:      uint32(count),
	}
	return
}

func (r RelationServiceImpl) GetFriendList(ctx context.Context, request *relation.FriendListRequest) (resp *relation.FriendListResponse, err error) {
	return
}

func (r RelationServiceImpl) IsFollow(ctx context.Context, request *relation.IsFollowRequest) (resp *relation.IsFollowResponse, err error) {

	ctx, span := tracing.Tracer.Start(ctx, "isFollowService")
	defer span.End()
	logger := logging.LogService("RelationService.isFollow").WithContext(ctx)

	var count int64
	result := database.Client.WithContext(ctx).
		Model(&models.Relation{}).
		Where("user_id = ? AND actor_id = ?", request.UserId, request.ActorId).
		Count(&count)

	if result.Error != nil {

		logger.WithFields(logrus.Fields{
			"err":     result.Error,
			"ActorId": request.ActorId,
			"UserId":  request.UserId,
		}).Errorf("IsFollowService failed")
		logging.SetSpanError(span, err)

		resp = &relation.IsFollowResponse{
			Result: false,
		}
		return
	}

	if count > 0 {
		resp = &relation.IsFollowResponse{
			Result: true,
		}
	} else {
		resp = &relation.IsFollowResponse{
			Result: false,
		}
	}

	return
}

// UpdateFollowCount 更新关注数和粉丝数
// Todo
func UpdateFollowCount() {

}
func UpdateFollowerCount() {

}
