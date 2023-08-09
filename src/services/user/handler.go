package main

import (
	"GuGoTik/src/constant/strings"
	"GuGoTik/src/extra/tracing"
	"GuGoTik/src/models"
	"GuGoTik/src/rpc/user"
	"GuGoTik/src/utils/logging"
	"context"
	"github.com/sirupsen/logrus"
)

type UserServiceImpl struct {
	user.UserServiceServer
}

func (a UserServiceImpl) GetUserInfo(ctx context.Context, request *user.UserRequest) (resp *user.UserResponse, err error) {
	ctx, span := tracing.Tracer.Start(ctx, "UserService-GetUserInfo")
	defer span.End()
	logger := logging.LogService("UserService.GetUserInfo").WithContext(ctx)

	var userModel models.User
	userModel.ID = request.UserId
	err = userModel.FillFromRedis(ctx)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"err":     err,
			"UserId":  request.UserId,
			"ActorId": request.ActorId,
		}).Errorf("Error when get user info")
		logging.SetSpanError(span, err)
		resp = &user.UserResponse{
			StatusCode: strings.UserServiceInnerErrorCode,
			StatusMsg:  strings.UserServiceInnerError,
			User:       nil,
		}
		return
	}

	if userModel.UserName == "" {
		resp = &user.UserResponse{
			StatusCode: strings.UserNotExistedCode,
			StatusMsg:  strings.UserNotExisted,
			User:       nil,
		}
	}

	//TODO 等待其他服务写完后接入
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
	return
}
