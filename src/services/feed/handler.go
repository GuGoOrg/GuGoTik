package main

import (
	"GuGoTik/src/constant/config"
	"GuGoTik/src/constant/strings"
	"GuGoTik/src/extra/tracing"
	"GuGoTik/src/models"
	"GuGoTik/src/rpc/comment"
	"GuGoTik/src/rpc/favorite"
	"GuGoTik/src/rpc/feed"
	"GuGoTik/src/rpc/user"
	"GuGoTik/src/storage/database"
	"GuGoTik/src/storage/file"
	"GuGoTik/src/utils/consul"
	"GuGoTik/src/utils/logging"
	"context"
	"errors"
	"strconv"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

type FeedServiceImpl struct {
	feed.FeedServiceServer
}

var UserClient user.UserServiceClient
var CommentClient comment.CommentServiceClient
var FavoriteClient favorite.FavoriteServiceClient

func init() {
	userErr := consul.RegisterConsul(config.UserRpcServerName, config.UserRpcServerPort)
	if userErr != nil {
		logging.Logger.WithFields(logrus.Fields{
			"err": userErr,
		}).Errorf("User Service meet trouble.")
	}
	commentErr := consul.RegisterConsul(config.CommentRpcServerName, config.CommentRpcServerPort)
	if commentErr != nil {
		logging.Logger.WithFields(logrus.Fields{
			"err": commentErr,
		}).Errorf("Comment Service meet trouble.")
	}
	favoriteErr := consul.RegisterConsul(config.FavoriteRpcServerName, config.FavoriteRpcServerPort)
	if favoriteErr != nil {
		logging.Logger.WithFields(logrus.Fields{
			"err": favoriteErr,
		}).Errorf("Favorite Service meet trouble.")
	}
}

func (s FeedServiceImpl) ListVideos(ctx context.Context, request *feed.ListFeedRequest) (resp *feed.ListFeedResponse, err error) {
	ctx, span := tracing.Tracer.Start(ctx, "ListVideosService")
	defer span.End()
	logger := logging.LogService("FeedService.ListVideos").WithContext(ctx)

	now := uint32(time.Now().UnixMilli())
	if request.LatestTime == nil {
		logger.WithFields(logrus.Fields{
			"LatestTime": *request.LatestTime,
		}).Warnf("request.LatestTime is nil.")
		logging.SetSpanError(span, err)
	}
	if request.LatestTime == nil {
		logger.WithFields(logrus.Fields{
			"request.LatestTime": request.LatestTime,
		}).Warnf("request.LatestTime is nil.")
		logging.SetSpanError(span, err)
	}
	latestTime, err := strconv.ParseInt(*request.LatestTime, 10, 64)

	if err != nil {
		logger.WithFields(logrus.Fields{
			"now": now,
		}).Warnf("strconv.ParseInt meet trouble.")
		//logging.SetSpanError(span, err)
		var numError *strconv.NumError
		if errors.As(err, &numError) {
			latestTime = int64(now)
		}
	}
	find, err := findVideos(ctx, latestTime)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"find": find,
		}).Warnf("func findVideos meet trouble.")
		logging.SetSpanError(span, err)

		resp = &feed.ListFeedResponse{
			StatusCode: strings.FeedServiceInnerErrorCode,
			StatusMsg:  strings.FeedServiceInnerError,
			NextTime:   &now,
			VideoList:  nil,
		}
		return
	}
	if len(find) == 0 {
		resp = &feed.ListFeedResponse{
			StatusCode: strings.ServiceOKCode,
			StatusMsg:  strings.ServiceOK,
			NextTime:   nil,
			VideoList:  nil,
		}
		return
	}
	nextTime := uint32(find[len(find)-1].CreatedAt.Add(time.Duration(-1)).UnixMilli())

	var actorId uint32 = 0
	if request.ActorId != nil {
		actorId = *request.ActorId
	}
	videos := queryDetailed(ctx, logger, actorId, find)
	if videos == nil {
		logger.WithFields(logrus.Fields{
			"videos": videos,
		}).Warnf("func queryDetailed meet trouble.")
		logging.SetSpanError(span, err)
		resp = &feed.ListFeedResponse{
			StatusCode: strings.FeedServiceInnerErrorCode,
			StatusMsg:  strings.FeedServiceInnerError,
			NextTime:   nil,
			VideoList:  nil,
		}
		return
	}
	resp = &feed.ListFeedResponse{
		StatusCode: strings.ServiceOKCode,
		StatusMsg:  strings.ServiceOK,
		NextTime:   &nextTime,
		VideoList:  videos,
	}
	return
}

func (s FeedServiceImpl) QueryVideos(ctx context.Context, req *feed.QueryVideosRequest) (resp *feed.QueryVideosResponse, err error) {
	ctx, span := tracing.Tracer.Start(ctx, "QueryVideosService")
	defer span.End()
	logger := logging.LogService("FeedService.QueryVideos").WithContext(ctx)
	ServiceOK := strings.ServiceOK
	FeedServiceInnerError := strings.FeedServiceInnerError
	rst, err := query(ctx, logger, req.ActorId, req.VideoIds)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"rst": rst,
		}).Warnf("func query meet trouble.")
		logging.SetSpanError(span, err)
		resp = &feed.QueryVideosResponse{
			StatusCode: strings.FeedServiceInnerErrorCode,
			StatusMsg:  FeedServiceInnerError,
			VideoList:  rst,
		}
		return
	}

	resp = &feed.QueryVideosResponse{
		StatusCode: strings.ServiceOKCode,
		StatusMsg:  ServiceOK,
		VideoList:  rst,
	}
	return
}

func findVideos(ctx context.Context, latestTime int64) ([]*models.Video, error) {
	logger := logging.LogService("ListVideos.findVideos").WithContext(ctx)

	var videos []*models.Video
	result := database.Client.Where("created_at <= ?", time.Unix(latestTime, 0)).
		Order("created_at DESC").
		Limit(strings.VideoCount).
		Find(&videos)

	if result.Error != nil {
		logger.WithFields(logrus.Fields{
			"videos": videos,
		}).Warnf("database.Client.Where meet trouble")
		return nil, result.Error
	}
	return videos, nil
}

func queryDetailed(
	ctx context.Context,
	logger *logrus.Entry,
	actorId uint32,
	videos []*models.Video,
) (respVideoList []*feed.Video) {
	//ctx, span := tracing.Tracer.Start(ctx, "queryDetailed")
	//defer span.End()
	logger = logging.LogService("ListVideos.queryDetailed").WithContext(ctx)
	wg := sync.WaitGroup{}
	respVideoList = make([]*feed.Video, len(videos))
	for i, v := range videos {
		respVideoList[i] = &feed.Video{
			Id:     uint32(v.ID),
			Title:  v.Title,
			Author: &user.User{Id: uint32(v.UserId)},
		}
		wg.Add(6)
		// fill author
		go func(i int, v *models.Video) {
			defer wg.Done()
			userResponse, localErr := UserClient.GetUserInfo(ctx, &user.UserRequest{
				UserId:  uint32(v.UserId),
				ActorId: actorId,
			})
			if localErr != nil || userResponse.StatusCode != strings.ServiceOKCode {
				logger.WithFields(logrus.Fields{
					"video_id": v.ID,
					"user_id":  v.UserId,
					"cause":    localErr,
				}).Warning("failed to get user info")
				return
			}
			respVideoList[i].Author = userResponse.User
		}(i, v)

		// fill play url
		go func(i int, v *models.Video) {
			defer wg.Done()
			playUrl, localErr := file.GetLink(ctx, v.FileName)
			if localErr != nil {
				logger.WithFields(logrus.Fields{
					"video_id":  v.ID,
					"file_name": v.FileName,
					"err":       localErr,
				}).Warning("failed to fetch play url")
				return
			}
			respVideoList[i].PlayUrl = playUrl
		}(i, v)

		// fill cover url
		go func(i int, v *models.Video) {
			defer wg.Done()
			coverUrl, localErr := file.GetLink(ctx, v.CoverName)
			if localErr != nil {
				logger.WithFields(logrus.Fields{
					"video_id":   v.ID,
					"cover_name": v.CoverName,
					"err":        localErr,
				}).Warning("failed to fetch cover url")
				return
			}
			respVideoList[i].CoverUrl = coverUrl
		}(i, v)

		// fill favorite count
		go func(i int, v *models.Video) {
			defer wg.Done()
			favoriteCount, localErr := FavoriteClient.CountFavorite(ctx, &favorite.CountFavoriteRequest{
				VideoId: uint32(v.ID),
			})
			if localErr != nil {
				logger.WithFields(logrus.Fields{
					"video_id": v.ID,
					"err":      localErr,
				}).Warning("failed to fetch favorite count")
				return
			}
			respVideoList[i].FavoriteCount = favoriteCount.Count
		}(i, v)

		// fill comment count
		go func(i int, v *models.Video) {
			defer wg.Done()
			commentCount, localErr := CommentClient.ListComment(ctx, &comment.ListCommentRequest{
				VideoId: uint32(v.ID),
			})
			if localErr != nil {
				logger.WithFields(logrus.Fields{
					"video_id": v.ID,
					"err":      localErr,
				}).Warning("failed to fetch comment count")
				return
			}
			respVideoList[i].CommentCount = uint32(len(commentCount.CommentList))
		}(i, v)

		// fill is favorite
		go func(i int, v *models.Video) {
			defer wg.Done()
			isFavorite, localErr := FavoriteClient.IsFavorite(ctx, &favorite.IsFavoriteRequest{
				ActorId: actorId,
				VideoId: uint32(v.ID),
			})
			if localErr != nil {
				logger.WithFields(logrus.Fields{
					"video_id": v.ID,
					"err":      localErr,
				}).Warning("failed to fetch favorite status")
				return
			}
			respVideoList[i].IsFavorite = isFavorite.Result
		}(i, v)
	}
	wg.Wait()

	return
}

func query(ctx context.Context, logger *logrus.Entry, actorId uint32, videoIds []uint32) (resp []*feed.Video, err error) {
	var videos []*models.Video
	logger = logging.LogService("QueryVideos.query").WithContext(ctx)
	//Gorm的操作，以后不需要在单独开span，通过传ctx的方式完成 "WithContext(ctx)"，如果在函数需要这样写，但是这个的目的是为了获取子 Span 的 ctx
	err = database.Client.WithContext(ctx).Where("Id IN ?", videoIds).Find(&videos).Error
	if err != nil {
		return nil, err
	}
	return queryDetailed(ctx, logger, actorId, videos), nil
}
