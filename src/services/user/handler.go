package main

import (
	"GuGoTik/src/constant/config"
	"GuGoTik/src/constant/strings"
	"GuGoTik/src/extra/tracing"
	"GuGoTik/src/models"
	"GuGoTik/src/rpc/publish"
	"GuGoTik/src/rpc/relation"
	"GuGoTik/src/rpc/user"
	"GuGoTik/src/storage/cached"
	grpc2 "GuGoTik/src/utils/grpc"
	"GuGoTik/src/utils/logging"
	"context"
	"sync"

	"github.com/sirupsen/logrus"
)

type UserServiceImpl struct {
	user.UserServiceServer
}

var relationClient relation.RelationServiceClient

var publishClient publish.PublishServiceClient

func (a UserServiceImpl) New() {
	relationConn := grpc2.Connect(config.RelationRpcServerName)
	relationClient = relation.NewRelationServiceClient(relationConn)

	publishConn := grpc2.Connect(config.PublishRpcServerName)
	publishClient = publish.NewPublishServiceClient(publishConn)
}

func (a UserServiceImpl) GetUserInfo(ctx context.Context, request *user.UserRequest) (resp *user.UserResponse, err error) {
	ctx, span := tracing.Tracer.Start(ctx, "UserService-GetUserInfo")
	defer span.End()
	logger := logging.LogService("UserService.GetUserInfo").WithContext(ctx)

	var userModel models.User
	userModel.ID = request.UserId
	ok, err := cached.ScanGet(ctx, "UserInfo", &userModel)

	if err != nil {

		resp = &user.UserResponse{
			StatusCode: strings.AuthServiceInnerErrorCode,
			StatusMsg:  strings.AuthServiceInnerError,
		}
		return
	}

	if !ok {
		resp = &user.UserResponse{
			StatusCode: strings.UserNotExistedCode,
			StatusMsg:  strings.UserNotExisted,
			User:       nil,
		}
		logger.WithFields(logrus.Fields{
			"user": request.UserId,
		})
		return
	}

	resp = &user.UserResponse{
		StatusCode: strings.ServiceOKCode,
		StatusMsg:  strings.ServiceOK,
		User: &user.User{
			Id:              request.UserId,
			Name:            userModel.UserName,
			FollowCount:     nil,
			FollowerCount:   nil,
			IsFollow:        false,
			Avatar:          &userModel.Avatar,
			BackgroundImage: &userModel.BackgroundImage,
			Signature:       &userModel.Signature,
			TotalFavorited:  nil,
			WorkCount:       nil,
			FavoriteCount:   nil,
		},
	}

	var wg sync.WaitGroup
	wg.Add(4)
	isErr := false

	go func() {
		defer wg.Done()
		rResp, err := relationClient.CountFollowList(ctx, &relation.CountFollowListRequest{UserId: request.UserId})
		if err != nil {
			logger.WithFields(logrus.Fields{
				"err":    err,
				"userId": request.UserId,
			}).Errorf("Error when user service get follow list")
			isErr = true
			return
		}

		if rResp != nil && rResp.StatusCode == strings.ServiceOKCode {
			if err != nil {
				logger.WithFields(logrus.Fields{
					"errMsg": rResp.StatusMsg,
					"userId": request.UserId,
				}).Errorf("Error when user service get follow list")
				isErr = true
				return
			}
		}

		resp.User.FollowCount = &rResp.Count
	}()

	go func() {
		defer wg.Done()
		rResp, err := relationClient.CountFollowerList(ctx, &relation.CountFollowerListRequest{UserId: request.UserId})
		if err != nil {
			logger.WithFields(logrus.Fields{
				"err":    err,
				"userId": request.UserId,
			}).Errorf("Error when user service get follower list")
			isErr = true
			return
		}

		if rResp != nil && rResp.StatusCode == strings.ServiceOKCode {
			if err != nil {
				logger.WithFields(logrus.Fields{
					"errMsg": rResp.StatusMsg,
					"userId": request.UserId,
				}).Errorf("Error when user service get follower list")
				isErr = true
				return
			}
		}

		resp.User.FollowerCount = &rResp.Count
	}()

	go func() {
		defer wg.Done()
		rResp, err := relationClient.IsFollow(ctx, &relation.IsFollowRequest{
			ActorId: request.ActorId,
			UserId:  request.UserId,
		})
		if err != nil {
			logger.WithFields(logrus.Fields{
				"err":    err,
				"userId": request.UserId,
			}).Errorf("Error when user service get is follow")
			isErr = true
			return
		}

		resp.User.IsFollow = rResp.Result
	}()

	go func() {
		defer wg.Done()
		rResp, err := publishClient.CountVideo(ctx, &publish.CountVideoRequest{UserId: request.UserId})
		if err != nil {
			logger.WithFields(logrus.Fields{
				"err":    err,
				"userId": request.UserId,
			}).Errorf("Error when user service get published count")
			isErr = true
			return
		}

		if rResp != nil && rResp.StatusCode == strings.ServiceOKCode {
			if err != nil {
				logger.WithFields(logrus.Fields{
					"errMsg": rResp.StatusMsg,
					"userId": request.UserId,
				}).Errorf("Error when user service get published count")
				isErr = true
				return
			}
		}

		resp.User.WorkCount = &rResp.Count
	}()

	wg.Wait()

	if isErr {
		resp = &user.UserResponse{
			StatusCode: strings.AuthServiceInnerErrorCode,
			StatusMsg:  strings.AuthServiceInnerError,
		}
		return
	}

	return
}
