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
	logging.SetSpanWithHostname(span)
	logger := logging.LogService("RecommendService.GetRecommend").WithContext(ctx)

	var offset int
	if request.Offset == -1 {
		ids, err := getVideoIds(ctx, strconv.Itoa(int(request.UserId)), int(request.Number))

		if err != nil {
			logger.WithFields(logrus.Fields{
				"err": err,
			}).Errorf("Error when getting recommend user item with default logic")
			logging.SetSpanError(span, err)
			resp = &recommend.RecommendResponse{
				StatusCode: strings.RecommendServiceInnerErrorCode,
				StatusMsg:  strings.RecommendServiceInnerError,
				VideoList:  nil,
			}
			return resp, err
		}

		resp = &recommend.RecommendResponse{
			StatusCode: strings.ServiceOKCode,
			StatusMsg:  strings.ServiceOK,
			VideoList:  ids,
		}
		return resp, nil

	} else {
		offset = int(request.Offset)
	}

	videos, err :=
		gorseClient.GetItemRecommend(ctx, strconv.Itoa(int(request.UserId)), []string{}, "read", "5m", int(request.Number), offset)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"err": err,
		}).Errorf("Error when getting recommend user item")
		logging.SetSpanError(span, err)
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
			logger.WithFields(logrus.Fields{
				"err": err,
			}).Errorf("Error when getting recommend user item")
			logging.SetSpanError(span, err)
			resp = &recommend.RecommendResponse{
				StatusCode: strings.RecommendServiceInnerErrorCode,
				StatusMsg:  strings.RecommendServiceInnerError,
				VideoList:  nil,
			}
			return resp, err
		}
		videoIds = append(videoIds, uint32(parseUint))
	}

	logger.WithFields(logrus.Fields{
		"offset":   offset,
		"videoIds": videoIds,
	}).Infof("Get recommend with offset")
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
	logging.SetSpanWithHostname(span)
	logger := logging.LogService("RecommendService.RegisterRecommend").WithContext(ctx)

	_, err = gorseClient.InsertUsers(ctx, []gorse.User{
		{
			UserId:  strconv.Itoa(int(request.UserId)),
			Comment: request.Username,
		},
	})

	if err != nil {
		logger.WithFields(logrus.Fields{
			"err": err,
		}).Errorf("Error when creating recommend user")
		logging.SetSpanError(span, err)
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

func getVideoIds(ctx context.Context, actorId string, num int) (ids []uint32, err error) {
	ctx, span := tracing.Tracer.Start(ctx, "GetRecommendAutoService")
	defer span.End()
	logging.SetSpanWithHostname(span)
	logger := logging.LogService("RecommendService.GetRecommendAuto").WithContext(ctx)
	key := fmt.Sprintf("%s-RecommendAutoService-%s", config.EnvCfg.RedisPrefix, actorId)
	offset := 0

	for len(ids) < num {
		vIds, err := gorseClient.GetItemRecommend(ctx, actorId, []string{}, "read", "5m", num, offset)
		logger.WithFields(logrus.Fields{
			"vIds": vIds,
		}).Debugf("Fetch data from Gorse")

		if err != nil {
			logger.WithFields(logrus.Fields{
				"err":     err,
				"actorId": actorId,
				"num":     num,
			}).Errorf("Error when getting item recommend")
			return nil, err
		}

		for _, id := range vIds {
			res := redis.Client.SIsMember(ctx, key, id)
			if res.Err() != nil && res.Err() != redis2.Nil {
				logger.WithFields(logrus.Fields{
					"err":     err,
					"actorId": actorId,
					"num":     num,
				}).Errorf("Error when getting item recommend")
				return nil, err
			}

			logger.WithFields(logrus.Fields{
				"id":  id,
				"res": res,
			}).Debugf("Get id in redis information")

			if !res.Val() {
				uintId, err := strconv.ParseUint(id, 10, 32)
				if err != nil {
					logger.WithFields(logrus.Fields{
						"err":     err,
						"actorId": actorId,
						"num":     num,
						"uint":    id,
					}).Errorf("Error when parsing uint")
					return nil, err
				}
				ids = append(ids, uint32(uintId))
			}
		}

		var idsStr []interface{}

		for _, id := range ids {
			idsStr = append(idsStr, strconv.FormatUint(uint64(id), 10))
		}

		logger.WithFields(logrus.Fields{
			"actorId": actorId,
			"ids":     idsStr,
		}).Infof("Get recommend information")

		if len(idsStr) != 0 {
			res := redis.Client.SAdd(ctx, key, idsStr)
			if res.Err() != nil {
				if err != nil {
					logger.WithFields(logrus.Fields{
						"err":     err,
						"actorId": actorId,
						"num":     num,
						"ids":     idsStr,
					}).Errorf("Error when locking redis ids read state")
					return nil, err
				}
			}
		}

		if len(vIds) != num {
			break
		}
		offset += num
	}
	return
}
