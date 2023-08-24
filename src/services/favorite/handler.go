package main

import (
	"GuGoTik/src/constant/strings"
	"GuGoTik/src/extra/tracing"
	"GuGoTik/src/rpc/favorite"
	"GuGoTik/src/rpc/feed"
	redis2 "GuGoTik/src/storage/redis"
	"GuGoTik/src/utils/logging"
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

var feedClient feed.FeedServiceClient

type FavoriteServiceServerImpl struct {
	favorite.FavoriteServiceServer
}

func (a FavoriteServiceServerImpl) New() {

}

// func (c FavoriteServiceServerImpl) New() {
// 	userRpcConn := grpc2.Connect(config.FeedRpcServerName)
// 	Client = feed.NewFeedServiceClient(userRpcConn)
// }

func (c FavoriteServiceServerImpl) FavoriteAction(ctx context.Context, req *favorite.FavoriteRequest) (resp *favorite.FavoriteResponse, err error) {
	ctx, span := tracing.Tracer.Start(ctx, "FavoriteServiceServerImpl")
	defer span.End()
	logger := logging.LogService("FavoriteService.FavoriteAction").WithContext(ctx)

	logger.WithFields(logrus.Fields{
		"ActorId":     req.ActorId,
		"video_id":    req.VideoId,
		"action_type": req.ActionType, //点赞 1 2 取消点赞
	}).Debugf("Process start")

	// VideosRes, err := feedClient.QueryVideos(ctx, &feed.QueryVideosRequest{
	// 	ActorId:  req.ActorId,
	// 	VideoIds: []uint32{req.VideoId},
	// })

	// if err != nil || VideosRes.StatusCode != strings.ServiceOKCode {
	// 	logger.WithFields(logrus.Fields{
	// 		"ActorId":     req.ActorId,
	// 		"video_id":    req.VideoId,
	// 		"action_type": req.ActionType, //点赞 1 2 取消点赞
	// 	}).Errorf("FavoriteAction call feed Service error")
	// 	logging.SetSpanError(span, err)

	// 	return &favorite.FavoriteResponse{
	// 		StatusCode: strings.FavorivateServiceErrorCode,
	// 		StatusMsg:  strings.FavorivateServiceError,
	// 	}, err
	// }

	// user_liked := VideosRes.VideoList[0].Author.Id
	user_liked := 2

	_, err = redis2.Client.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
		var val int64
		if req.ActionType == 1 {
			val = 1
		} else {
			val = -1
		}
		videoId := fmt.Sprintf("video_like_%d", req.VideoId)      // redis 视频key
		user_like_Id := fmt.Sprintf("user_like_%d", req.ActorId)  // redis 用户喜欢 key
		user_liked_id := fmt.Sprintf("user_liked_%d", user_liked) //用户被喜欢key
		pipe.IncrBy(ctx, videoId, val)
		pipe.IncrBy(ctx, user_liked_id, val)
		if req.ActionType == 2 {
			pipe.ZRem(ctx, user_like_Id, req.VideoId)
		} else {
			pipe.ZAdd(ctx, user_like_Id, redis.Z{Score: float64(time.Now().Unix()), Member: req.VideoId})
		}
		return nil
	})

	if err != nil {
		logger.WithFields(logrus.Fields{
			"ActorId":     req.ActorId,
			"video_id":    req.VideoId,
			"action_type": req.ActionType, //点赞 1 2 取消点赞
		}).Errorf("redis Service error")
		logging.SetSpanError(span, err)

		return &favorite.FavoriteResponse{
			StatusCode: strings.FavorivateServiceErrorCode,
			StatusMsg:  strings.FavorivateServiceError,
		}, err
	}
	resp = &favorite.FavoriteResponse{
		StatusCode: strings.ServiceOKCode,
		StatusMsg:  strings.ServiceOK,
	}
	logger.WithFields(logrus.Fields{
		"response": resp,
	}).Debugf("Process done.")

	return
}

func (c FavoriteServiceServerImpl) FavoriteList(ctx context.Context, req *favorite.FavoriteListRequest) (resp *favorite.FavoriteListResponse, err error) {

	ctx, span := tracing.Tracer.Start(ctx, "FavoriteServiceServerImpl")
	defer span.End()
	logger := logging.LogService("FavoriteService.FavoriteList").WithContext(ctx)

	logger.WithFields(logrus.Fields{
		"ActorId": req.ActorId,
		"user_id": req.UserId,
	}).Debugf("Process start")
	userId := fmt.Sprintf("user_like_%d", req.ActorId)
	arr, err := redis2.Client.ZRevRange(ctx, userId, 0, -1).Result()
	if err != nil {
		logger.WithFields(logrus.Fields{
			"ActorId": req.ActorId,
			"user_id": req.UserId,
		}).Errorf("redis Service error")
		logging.SetSpanError(span, err)

		return &favorite.FavoriteListResponse{
			StatusCode: strings.FavorivateServiceErrorCode,
			StatusMsg:  strings.FavorivateServiceError,
		}, err
	}
	if len(arr) == 0 {
		resp = &favorite.FavoriteListResponse{
			StatusCode: strings.ServiceOKCode,
			StatusMsg:  strings.ServiceOK,
			VideoList:  nil,
		}
		return resp, nil
	}

	res := make([]uint32, len(arr))
	for index, val := range arr {
		num, _ := strconv.Atoi(val)
		res[index] = uint32(num)

	}

	var VideoList []*feed.Video
	value, err := feedClient.QueryVideos(ctx, &feed.QueryVideosRequest{
		ActorId:  req.ActorId,
		VideoIds: res,
	})
	if err != nil || value.StatusCode != strings.ServiceOKCode {
		logger.WithFields(logrus.Fields{
			"ActorId": req.ActorId,
			"user_id": req.UserId,
		}).Errorf("feed Service error")
		logging.SetSpanError(span, err)
		return &favorite.FavoriteListResponse{
			StatusCode: strings.FavorivateServiceErrorCode,
			StatusMsg:  strings.FavorivateServiceError,
		}, err
	}

	VideoList = value.VideoList

	resp = &favorite.FavoriteListResponse{
		StatusCode: strings.ServiceOKCode,
		StatusMsg:  strings.ServiceOK,
		VideoList:  VideoList,
	}
	return resp, nil
}

func (c FavoriteServiceServerImpl) IsFavorite(ctx context.Context, req *favorite.IsFavoriteRequest) (resp *favorite.IsFavoriteResponse, err error) {
	ctx, span := tracing.Tracer.Start(ctx, "FavoriteServiceServerImpl")
	defer span.End()
	logger := logging.LogService("FavoriteService.IsFavorite").WithContext(ctx)

	logger.WithFields(logrus.Fields{
		"ActorId":  req.ActorId,
		"video_id": req.VideoId,
	}).Debugf("Process start")

	userId := fmt.Sprintf("user_like_%d", req.ActorId)
	videoId := fmt.Sprintf("%d", req.VideoId)
	//等下单步跟下 返回值
	ok, err := redis2.Client.ZScore(ctx, userId, videoId).Result()
	if err != nil {
		logger.WithFields(logrus.Fields{
			"ActorId":  req.ActorId,
			"video_id": req.VideoId,
		}).Errorf("redis Service error")
		logging.SetSpanError(span, err)

		return &favorite.IsFavoriteResponse{
			StatusCode: strings.FavorivateServiceErrorCode,
			StatusMsg:  strings.FavorivateServiceError,
		}, err
	}

	if ok != 0 {
		resp = &favorite.IsFavoriteResponse{
			StatusCode: strings.ServiceOKCode,
			StatusMsg:  strings.ServiceOK,
			Result:     true,
		}
	} else {
		resp = &favorite.IsFavoriteResponse{
			StatusCode: strings.ServiceOKCode,
			StatusMsg:  strings.ServiceOK,
			Result:     false,
		}
	}
	logger.WithFields(logrus.Fields{
		"response": resp,
	}).Debugf("Process done.")
	return
}

func (c FavoriteServiceServerImpl) CountFavorite(ctx context.Context, req *favorite.CountFavoriteRequest) (resp *favorite.CountFavoriteResponse, err error) {

	ctx, span := tracing.Tracer.Start(ctx, "FavoriteServiceServerImpl")
	defer span.End()
	logger := logging.LogService("FavoriteService.CountFavorite").WithContext(ctx)

	logger.WithFields(logrus.Fields{
		"video_id": req.VideoId,
	}).Debugf("Process start")

	videoId := fmt.Sprintf("video_like_%d", req.VideoId)
	value, err := redis2.Client.Get(ctx, videoId).Result()
	if err != nil {
		logger.WithFields(logrus.Fields{
			"video_id": req.VideoId,
		}).Errorf("redis Service error")
		logging.SetSpanError(span, err)

		return &favorite.CountFavoriteResponse{
			StatusCode: strings.FavorivateServiceErrorCode,
			StatusMsg:  strings.FavorivateServiceError,
		}, err
	}

	num, err := strconv.Atoi(value)

	resp = &favorite.CountFavoriteResponse{
		StatusCode: strings.ServiceOKCode,
		StatusMsg:  strings.ServiceOK,
		Count:      uint32(num),
	}
	logger.WithFields(logrus.Fields{
		"response": resp,
	}).Debugf("Process done.")
	return
}

// CountUserFavorite(context.Context, *CountUserFavoriteRequest) (*CountUserFavoriteResponse, error)

func (c FavoriteServiceServerImpl) CountUserFavorite(ctx context.Context, req *favorite.CountUserFavoriteRequest) (resp *favorite.CountUserFavoriteResponse, err error) {

	ctx, span := tracing.Tracer.Start(ctx, "FavoriteServiceServerImpl")
	defer span.End()
	logger := logging.LogService("FavoriteService.CountUserFavorite").WithContext(ctx)

	logger.WithFields(logrus.Fields{
		"user_id": req.UserId,
	}).Debugf("Process start")

	user_like_id := fmt.Sprintf("user_like_%d", req.UserId)
	value, err := redis2.Client.ZCard(ctx, user_like_id).Result()

	if err != nil {
		logger.WithFields(logrus.Fields{
			"user_id": req.UserId,
		}).Errorf("redis Service error")
		logging.SetSpanError(span, err)

		return &favorite.CountUserFavoriteResponse{
			StatusCode: strings.FavorivateServiceErrorCode,
			StatusMsg:  strings.FavorivateServiceError,
		}, err
	}

	resp = &favorite.CountUserFavoriteResponse{
		StatusCode: strings.ServiceOKCode,
		StatusMsg:  strings.ServiceOK,
		Count:      uint32(value),
	}
	logger.WithFields(logrus.Fields{
		"response": resp,
	}).Debugf("Process done.")
	return
}

func (c FavoriteServiceServerImpl) CountUserTotalFavorited(ctx context.Context, req *favorite.CountUserTotalFavoritedRequest) (resp *favorite.CountUserTotalFavoritedResponse, err error) {
	ctx, span := tracing.Tracer.Start(ctx, "FavoriteServiceServerImpl")
	defer span.End()
	logger := logging.LogService("FavoriteService.CountUserTotalFavorited").WithContext(ctx)

	logger.WithFields(logrus.Fields{
		"user_id": req.UserId,
		"ActorId": req.ActorId,
	}).Debugf("Process start")

	user_liked_id := fmt.Sprintf("user_liked_%d", req.UserId)
	value, err := redis2.Client.Get(ctx, user_liked_id).Result()

	if err != nil {
		logger.WithFields(logrus.Fields{
			"err":     err,
			"user_id": req.UserId,
			"ActorId": req.ActorId,
		}).Errorf("redis Service error")
		logging.SetSpanError(span, err)

		return &favorite.CountUserTotalFavoritedResponse{
			StatusCode: strings.FavorivateServiceErrorCode,
			StatusMsg:  strings.FavorivateServiceError,
		}, err
	}
	num, err := strconv.Atoi(value)
	resp = &favorite.CountUserTotalFavoritedResponse{
		StatusCode: strings.ServiceOKCode,
		StatusMsg:  strings.ServiceOK,
		Count:      uint32(num),
	}
	logger.WithFields(logrus.Fields{
		"response": resp,
	}).Debugf("Process done.")
	return

}
