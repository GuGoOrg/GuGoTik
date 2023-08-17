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
	grpc2 "GuGoTik/src/utils/grpc"
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

const (
	VideoCount = 30
)

var UserClient user.UserServiceClient
var CommentClient comment.CommentServiceClient
var FavoriteClient favorite.FavoriteServiceClient

func init() {
	userRpcConn := grpc2.Connect(config.UserRpcServerName)
	UserClient = user.NewUserServiceClient(userRpcConn)
	commentRpcConn := grpc2.Connect(config.CommentRpcServerName)
	CommentClient = comment.NewCommentServiceClient(commentRpcConn)
	favoriteRpcConn := grpc2.Connect(config.FavoriteRpcServerName)
	FavoriteClient = favorite.NewFavoriteServiceClient(favoriteRpcConn)
}

func (s FeedServiceImpl) ListVideos(ctx context.Context, request *feed.ListFeedRequest) (resp *feed.ListFeedResponse, err error) {
	ctx, span := tracing.Tracer.Start(ctx, "ListVideosService")
	defer span.End()
	logger := logging.LogService("FeedService.ListVideos").WithContext(ctx)

	now := uint32(time.Now().UnixMilli())

	layout := "2006-01-02T15:04:05.999Z"
	t, err := time.Parse(layout, *request.LatestTime)
	latestTime := t.Unix()
	if err != nil {
		var numError *strconv.NumError
		if errors.As(err, &numError) {
			latestTime = int64(now)
			logger.WithFields(logrus.Fields{
				"latestTime": latestTime,
				"err":        err,
			}).Warnf("strconv.ParseInt meet trouble.")
			logging.SetSpanError(span, err)
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
		return resp, err
	}
	if len(find) == 0 {
		resp = &feed.ListFeedResponse{
			StatusCode: strings.ServiceOKCode,
			StatusMsg:  strings.ServiceOK,
			NextTime:   nil,
			VideoList:  nil,
		}
		return resp, err
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
		return resp, err
	}
	resp = &feed.ListFeedResponse{
		StatusCode: strings.ServiceOKCode,
		StatusMsg:  strings.ServiceOK,
		NextTime:   &nextTime,
		VideoList:  videos,
	}
	return resp, err
}

func (s FeedServiceImpl) QueryVideos(ctx context.Context, req *feed.QueryVideosRequest) (resp *feed.QueryVideosResponse, err error) {
	ctx, span := tracing.Tracer.Start(ctx, "QueryVideosService")
	defer span.End()
	logger := logging.LogService("FeedService.QueryVideos").WithContext(ctx)

	rst, err := query(ctx, logger, req.ActorId, req.VideoIds)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"rst": rst,
		}).Warnf("func query meet trouble.")
		logging.SetSpanError(span, err)
		resp = &feed.QueryVideosResponse{
			StatusCode: strings.FeedServiceInnerErrorCode,
			StatusMsg:  strings.FeedServiceInnerError,
			VideoList:  rst,
		}
		return resp, err
	}

	resp = &feed.QueryVideosResponse{
		StatusCode: strings.ServiceOKCode,
		StatusMsg:  strings.ServiceOK,
		VideoList:  rst,
	}
	return resp, err
}

func findVideos(ctx context.Context, latestTime int64) ([]*models.Video, error) {
	logger := logging.LogService("ListVideos.findVideos").WithContext(ctx)

	var videos []*models.Video
	result := database.Client.Where("created_at <= ?", time.Unix(latestTime, 0)).
		Order("created_at DESC").
		Limit(VideoCount).
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
	ctx, span := tracing.Tracer.Start(ctx, "queryDetailed")
	defer span.End()
	logger = logging.LogService("ListVideos.queryDetailed").WithContext(ctx)
	wg := sync.WaitGroup{}
	respVideoList = make([]*feed.Video, len(videos))
	for i, v := range videos {
		respVideoList[i] = &feed.Video{
			Id:     v.ID,
			Title:  v.Title,
			Author: &user.User{Id: v.ID},
		}
		wg.Add(6)
		// fill author
		go func(i int, v *models.Video) {
			defer wg.Done()
			userResponse, localErr := UserClient.GetUserInfo(ctx, &user.UserRequest{
				UserId:  v.ID,
				ActorId: actorId,
			})
			if localErr != nil || userResponse.StatusCode != strings.ServiceOKCode {
				logger.WithFields(logrus.Fields{
					"video_id": v.ID,
					"user_id":  v.ID,
					"cause":    localErr,
				}).Warning("failed to get user info")
				logging.SetSpanError(span, localErr)
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
				logging.SetSpanError(span, localErr)
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
				logging.SetSpanError(span, localErr)
				return
			}
			respVideoList[i].CoverUrl = coverUrl
		}(i, v)

		// fill favorite count
		go func(i int, v *models.Video) {
			defer wg.Done()
			favoriteCount, localErr := FavoriteClient.CountFavorite(ctx, &favorite.CountFavoriteRequest{
				VideoId: v.ID,
			})
			if localErr != nil {
				logger.WithFields(logrus.Fields{
					"video_id": v.ID,
					"err":      localErr,
				}).Warning("failed to fetch favorite count")
				logging.SetSpanError(span, localErr)
				return
			}
			respVideoList[i].FavoriteCount = favoriteCount.Count
		}(i, v)

		// mock favorite count
		//go func(i int, v *models.Video) {
		//	defer wg.Done()
		//	respVideoList[i].FavoriteCount = uint32(countFavorite())
		//}(i, v)

		// fill comment count
		go func(i int, v *models.Video) {
			defer wg.Done()
			commentCount, localErr := CommentClient.ListComment(ctx, &comment.ListCommentRequest{
				VideoId: v.ID,
			})
			if localErr != nil {
				logger.WithFields(logrus.Fields{
					"video_id": v.ID,
					"err":      localErr,
				}).Warning("failed to fetch comment count")
				logging.SetSpanError(span, localErr)
				return
			}
			respVideoList[i].CommentCount = uint32(len(commentCount.CommentList))
		}(i, v)

		// fill is favorite
		go func(i int, v *models.Video) {
			defer wg.Done()
			isFavorite, localErr := FavoriteClient.IsFavorite(ctx, &favorite.IsFavoriteRequest{
				ActorId: actorId,
				VideoId: v.ID,
			})
			if localErr != nil {
				logger.WithFields(logrus.Fields{
					"video_id": v.ID,
					"err":      localErr,
				}).Warning("failed to fetch favorite status")
				logging.SetSpanError(span, localErr)
				return
			}
			respVideoList[i].IsFavorite = isFavorite.Result
		}(i, v)

		// mock isFavorite
		//go func(i int, v *models.Video) {
		//	defer wg.Done()
		//	respVideoList[i].IsFavorite = isFavorite()
		//}(i, v)

	}
	wg.Wait()

	return
}

func query(ctx context.Context, logger *logrus.Entry, actorId uint32, videoIds []uint32) (resp []*feed.Video, err error) {
	var videos []*models.Video
	//Gorm的操作，以后不需要在单独开span，通过传ctx的方式完成 "WithContext(ctx)"，如果在函数需要这样写，但是这个的目的是为了获取子 Span 的 ctx
	err = database.Client.WithContext(ctx).Where("Id IN ?", videoIds).Find(&videos).Error
	if err != nil {
		return nil, err
	}
	return queryDetailed(ctx, logger, actorId, videos), nil
}

//func countFavorite() int {
//	return 0
//}
//func isFavorite() bool {
//	return true
//}
