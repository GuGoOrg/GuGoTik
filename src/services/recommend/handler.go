package main

import (
	"GuGoTik/src/constant/config"
	"GuGoTik/src/constant/strings"
	"GuGoTik/src/extra/gorse"
	"GuGoTik/src/extra/tracing"
	"GuGoTik/src/rpc/recommend"
	"GuGoTik/src/storage/redis"
	"GuGoTik/src/utils/logging"
	"context"
	"fmt"
	redis2 "github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"strconv"
)

type RecommendServiceImpl struct {
	recommend.RecommendServiceServer
}

func (a RecommendServiceImpl) New() {
	gorseClient = gorse.NewGorseClient(config.EnvCfg.GorseAddr, config.EnvCfg.GorseApiKey)
}

var gorseClient *gorse.GorseClient

func (a RecommendServiceImpl) GetRecommendInformation(ctx context.Context, request *recommend.RecommendRequest) (resp *recommend.RecommendResponse, err error) {
	ctx, span := tracing.Tracer.Start(ctx, "GetRecommendService")
	defer span.End()
	logger := logging.LogService("RecommendService.GetRecommend").WithContext(ctx)

	var offset int
	if request.Offset == -1 {
		res := redis.Client.Get(ctx, fmt.Sprintf("%s-RecommendUserOffset-%d", config.EnvCfg.RedisPrefix, request.UserId))
		if res.Err() != nil {
			if res.Err() == redis2.Nil {
				redis.Client.Set(ctx, fmt.Sprintf("%s-RecommendUserOffset-%d", config.EnvCfg.RedisPrefix, request.UserId), 0, redis2.KeepTTL)
				offset = 0
			} else {
				logger.WithFields(logrus.Fields{
					"err": err,
				}).Errorf("Error when operate redis")
				resp = &recommend.RecommendResponse{
					StatusCode: strings.RecommendServiceInnerErrorCode,
					StatusMsg:  strings.RecommendServiceInnerError,
					VideoList:  nil,
				}
				return
			}
		} else {
			offset, err = strconv.Atoi(res.Val())
			if err != nil {
				logger.WithFields(logrus.Fields{
					"err": err,
				}).Errorf("Error when operate redis")
				resp = &recommend.RecommendResponse{
					StatusCode: strings.RecommendServiceInnerErrorCode,
					StatusMsg:  strings.RecommendServiceInnerError,
					VideoList:  nil,
				}
				return
			}
			offset += int(request.Number)
			redis.Client.Set(ctx, fmt.Sprintf("%s-RecommendUserOffset-%d", config.EnvCfg.RedisPrefix, request.UserId), offset, redis2.KeepTTL)
		}
	} else {
		offset = int(request.Offset)
	}

	videos, err :=
		gorseClient.GetItemRecommend(ctx, strconv.Itoa(int(request.UserId)), []string{}, "read", "5m", int(request.Number), offset)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"err": err,
		}).Errorf("Error when getting recommend user item")
		resp = &recommend.RecommendResponse{
			StatusCode: strings.RecommendServiceInnerErrorCode,
			StatusMsg:  strings.RecommendServiceInnerError,
			VideoList:  nil,
		}
		return
	}

	var videoIds []uint32
	for _, id := range videos {
		parseUint, err := strconv.ParseUint(id, 10, 32)
		if err != nil {
			resp = &recommend.RecommendResponse{
				StatusCode: strings.RecommendServiceInnerErrorCode,
				StatusMsg:  strings.RecommendServiceInnerError,
				VideoList:  nil,
			}
			return resp, err
		}
		videoIds = append(videoIds, uint32(parseUint))
	}
	resp = &recommend.RecommendResponse{
		StatusCode: strings.ServiceOKCode,
		StatusMsg:  strings.ServiceOK,
		VideoList:  videoIds,
	}
	return
}

func (a RecommendServiceImpl) RegisterRecommendUser(ctx context.Context, request *recommend.RecommendRegisterRequest) (resp *recommend.RecommendRegisterResponse, err error) {
	ctx, span := tracing.Tracer.Start(ctx, "RegisterRecommendService")
	defer span.End()
	logger := logging.LogService("RecommendService.RegisterRecommend").WithContext(ctx)

	_, err = gorseClient.InsertUsers(ctx, []gorse.User{
		{
			UserId:  strconv.Itoa(int(request.UserId)),
			Comment: strconv.Itoa(int(request.UserId)),
		},
	})

	if err != nil {
		logger.WithFields(logrus.Fields{
			"err": err,
		}).Errorf("Error when creating recommend user")
		resp = &recommend.RecommendRegisterResponse{
			StatusCode: strings.RecommendServiceInnerErrorCode,
			StatusMsg:  strings.RecommendServiceInnerError,
		}
		return
	}

	resp = &recommend.RecommendRegisterResponse{
		StatusCode: strings.ServiceOKCode,
		StatusMsg:  strings.ServiceOK,
	}
	return
}
