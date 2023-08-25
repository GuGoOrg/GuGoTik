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
	"gorm.io/gorm"
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

func (s FeedServiceImpl) New() {
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

	now := time.Now().Unix()
	latestTime := now
	if request.LatestTime != nil && *request.LatestTime != "" {
		// Check if request.LatestTime is a timestamp
		t, ok := isUnixTimestamp(*request.LatestTime)
		if ok {
			latestTime = t
		} else {
			logger.WithFields(logrus.Fields{
				"latestTime": request.LatestTime,
			}).Errorf("The latestTime is not a unix timestamp")
			logging.SetSpanError(span, errors.New("the latestTime is not a unit timestamp"))
		}
	}

	find, nextTime, err := findVideos(ctx, latestTime)
	nextTimeStamp := uint32(nextTime.Unix())
	if err != nil {
		logger.WithFields(logrus.Fields{
			"find": find,
		}).Warnf("func findVideos meet trouble.")
		logging.SetSpanError(span, err)

		resp = &feed.ListFeedResponse{
			StatusCode: strings.FeedServiceInnerErrorCode,
			StatusMsg:  strings.FeedServiceInnerError,
			NextTime:   &nextTimeStamp,
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
		NextTime:   &nextTimeStamp,
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

func (s FeedServiceImpl) QueryVideoExisted(ctx context.Context, req *feed.VideoExistRequest) (resp *feed.VideoExistResponse, err error) {
	ctx, span := tracing.Tracer.Start(ctx, "QueryVideoExistedService")
	defer span.End()
	logger := logging.LogService("FeedService.QueryVideoExisted").WithContext(ctx)
	var video models.Video
	result := database.Client.WithContext(ctx).Where("id = ?", req.VideoId).First(&video)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			logger.WithFields(logrus.Fields{
				"video_id": req.VideoId,
			}).Warnf("gorm.ErrRecordNotFound")
			logging.SetSpanError(span, err)
			resp = &feed.VideoExistResponse{
				StatusCode: strings.ServiceOKCode,
				StatusMsg:  strings.ServiceOK,
				Existed:    false,
			}
			return resp, nil
		} else {
			logger.WithFields(logrus.Fields{
				"video_id": req.VideoId,
			}).Warnf("Error occurred while querying database")
			logging.SetSpanError(span, err)
			resp = &feed.VideoExistResponse{
				StatusCode: strings.FeedServiceInnerErrorCode,
				StatusMsg:  strings.FeedServiceInnerError,
				Existed:    false,
			}
			return resp, result.Error
		}
	}
	resp = &feed.VideoExistResponse{
		StatusCode: strings.ServiceOKCode,
		StatusMsg:  strings.ServiceOK,
		Existed:    true,
	}
	return
}

func findVideos(ctx context.Context, latestTime int64) ([]*models.Video, time.Time, error) {
	logger := logging.LogService("ListVideos.findVideos").WithContext(ctx)

	nextTime := time.Unix(latestTime, 0)

	var videos []*models.Video
	result := database.Client.Where("created_at < ?", time.Unix(latestTime, 0)).
		Order("created_at DESC").
		Limit(VideoCount).
		Find(&videos)

	if result.Error != nil {
		logger.WithFields(logrus.Fields{
			"videos": videos,
		}).Warnf("database.Client.Where meet trouble")
		return nil, nextTime, result.Error
	}

	if len(videos) != 0 {
		nextTime = videos[len(videos)-1].CreatedAt
	}

	logger.WithFields(logrus.Fields{
		"latestTime":  time.Unix(latestTime, 0),
		"VideosCount": len(videos),
		"NextTime":    nextTime,
	}).Debugf("Find videos")
	return videos, nextTime, nil
}

func queryDetailed(ctx context.Context, logger *logrus.Entry, actorId uint32, videos []*models.Video) (respVideoList []*feed.Video) {
	ctx, span := tracing.Tracer.Start(ctx, "queryDetailed")
	defer span.End()
	logger = logging.LogService("ListVideos.queryDetailed").WithContext(ctx)
	respVideoList = make([]*feed.Video, len(videos))

	// Init respVideoList
	for i, v := range videos {
		respVideoList[i] = &feed.Video{
			Id:     v.ID,
			Title:  v.Title,
			Author: &user.User{Id: v.UserId},
		}
	}

	// Create userid -> user map to reduce duplicate user info query
	userMap := make(map[uint32]*user.User)
	for _, video := range videos {
		userMap[video.UserId] = &user.User{}
	}

	userWg := sync.WaitGroup{}
	userWg.Add(len(userMap))
	for userId := range userMap {
		go func(userId uint32) {
			defer userWg.Done()
			userResponse, localErr := UserClient.GetUserInfo(ctx, &user.UserRequest{
				UserId:  userId,
				ActorId: actorId,
			})
			if localErr != nil || userResponse.StatusCode != strings.ServiceOKCode {
				logger.WithFields(logrus.Fields{
					"UserId": userId,
					"cause":  localErr,
				}).Warning("failed to get user info")
				logging.SetSpanError(span, localErr)
			}
			userMap[userId] = userResponse.User
		}(userId)
	}

	wg := sync.WaitGroup{}
	for i, v := range videos {
		wg.Add(4)
		// fill play url
		go func(i int, v *models.Video) {
			defer wg.Done()
			playUrl, localErr := file.GetLink(ctx, v.FileName, v.UserId)
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
			coverUrl, localErr := file.GetLink(ctx, v.CoverName, v.UserId)
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

		// fill comment count
		go func(i int, v *models.Video) {
			defer wg.Done()
			commentCount, localErr := CommentClient.CountComment(ctx, &comment.CountCommentRequest{
				ActorId: actorId,
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
			respVideoList[i].CommentCount = commentCount.CommentCount
		}(i, v)

		// fill is favorite
		if actorId != 0 {
			wg.Add(1)
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
		} else {
			respVideoList[i].IsFavorite = false
		}
	}
	userWg.Wait()
	wg.Wait()

	for i, respVideo := range respVideoList {
		authorId := respVideo.Author.Id
		respVideoList[i].Author = userMap[authorId]
	}

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

func isUnixTimestamp(s string) (int64, bool) {
	timestamp, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, false
	}

	startTime := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	endTime := time.Now().AddDate(100, 0, 0)

	t := time.Unix(timestamp, 0)
	res := t.After(startTime) && t.Before(endTime)

	return timestamp, res
}
