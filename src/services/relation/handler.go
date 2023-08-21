package main

import (
	"GuGoTik/src/constant/config"
	"GuGoTik/src/constant/strings"
	"GuGoTik/src/extra/tracing"
	"GuGoTik/src/models"
	"GuGoTik/src/rpc/relation"
	"GuGoTik/src/rpc/user"
	"GuGoTik/src/storage/cached"
	"GuGoTik/src/storage/database"
	grpc2 "GuGoTik/src/utils/grpc"
	"GuGoTik/src/utils/logging"
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/trace"
	"strconv"
	"sync"
	"time"
)

var userClient user.UserServiceClient

type RelationServiceImpl struct {
	relation.RelationServiceServer
}

type CacheRelationList struct {
	rList []models.Relation
}

func (c *CacheRelationList) IsDirty() bool {
	return c.rList != nil
}

// GetID :   use userid as key for cache
func (c *CacheRelationList) GetID() uint32 {
	return 0
}

func init() {
	userRPCConn := grpc2.Connect(config.UserRpcServerName)
	userClient = user.NewUserServiceClient(userRPCConn)
}

func (r RelationServiceImpl) Follow(ctx context.Context, request *relation.RelationActionRequest) (resp *relation.RelationActionResponse, err error) {
	ctx, span := tracing.Tracer.Start(ctx, "FollowService")
	defer span.End()
	logger := logging.LogService("RelationService.Follow").WithContext(ctx)

	if request.UserId == request.ActorId {
		resp = &relation.RelationActionResponse{
			StatusCode: strings.UnableToRelateYourselfErrorCode,
			StatusMsg:  strings.UnableToRelateYourselfError,
		}
		return
	}

	userResponse, err := userClient.GetUserInfo(ctx, &user.UserRequest{
		UserId:  request.UserId,
		ActorId: request.ActorId,
	})

	if err != nil || userResponse.StatusCode != strings.ServiceOKCode {
		logger.WithFields(logrus.Fields{
			"err":     err,
			"ActorId": request.ActorId,
		}).Errorf("failed to get user info")
		logging.SetSpanError(span, err)

		resp = &relation.RelationActionResponse{
			StatusCode: strings.UnableToQueryUserErrorCode,
			StatusMsg:  strings.UnableToQueryUserError,
		}
		return
	}

	rRelation := models.Relation{
		ActorId: request.ActorId, // 关注者的 ID
		UserId:  request.UserId,  // 被关注者的 ID
	}

	tx := database.Client.WithContext(ctx).Begin() // 开始事务
	defer func() {
		if err != nil {
			tx.Rollback()
			return
		}
		tx.Commit()
	}()

	if err = tx.Create(&rRelation).Error; err != nil {
		resp = &relation.RelationActionResponse{
			StatusCode: strings.UnableToFollowErrorCode,
			StatusMsg:  strings.UnableToFollowError,
		}
		logging.SetSpanError(span, err)
		return
	}

	if err = updateFollowListCache(ctx, request.ActorId, rRelation, true, span, logger); err != nil {
		logger.WithFields(logrus.Fields{
			"err": err,
		}).Errorf("failed to update follow list cache")
		logging.SetSpanError(span, err)
		return
	}

	if err = updateFollowerListCache(ctx, request.UserId, rRelation, true, span, logger); err != nil {
		logger.WithFields(logrus.Fields{
			"err": err,
		}).Errorf("failed to update follower list cache")
		logging.SetSpanError(span, err)
		return
	}

	if err = updateFollowCountCache(ctx, request.ActorId, true, span, logger); err != nil {
		logger.WithFields(logrus.Fields{
			"err": err,
		}).Errorf("failed to update follow count cache")
		logging.SetSpanError(span, err)
		return
	}

	if err = updateFollowerCountCache(ctx, request.UserId, true, span, logger); err != nil {
		logger.WithFields(logrus.Fields{
			"err": err,
		}).Errorf("failed to update follower count cache")
		logging.SetSpanError(span, err)
		return
	}
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

	if request.UserId == request.ActorId {
		resp = &relation.RelationActionResponse{
			StatusCode: strings.UnableToRelateYourselfErrorCode,
			StatusMsg:  strings.UnableToRelateYourselfError,
		}
		return
	}

	userResponse, err := userClient.GetUserInfo(ctx, &user.UserRequest{
		UserId:  request.UserId,
		ActorId: request.ActorId,
	})

	if err != nil || userResponse.StatusCode != strings.ServiceOKCode {
		logger.WithFields(logrus.Fields{
			"err":     err,
			"ActorId": request.ActorId,
		}).Errorf("failed to get user info")
		logging.SetSpanError(span, err)

		resp = &relation.RelationActionResponse{
			StatusCode: strings.UnableToQueryUserErrorCode,
			StatusMsg:  strings.UnableToQueryUserError,
		}
		return
	}

	rRelation := models.Relation{
		ActorId: request.ActorId,
		UserId:  request.UserId,
	}

	// Check if relation exists before deleting
	existingRelation := models.Relation{}
	result := database.Client.WithContext(ctx).
		Where(&rRelation).
		First(&existingRelation)

	if result.Error != nil {
		resp = &relation.RelationActionResponse{
			StatusCode: strings.RelationNotFoundErrorCode,
			StatusMsg:  strings.RelationNotFoundError,
		}
		return
	}

	tx := database.Client.WithContext(ctx).Begin()
	defer func() {
		if err != nil {
			tx.Rollback()
			return
		}
		tx.Commit()
	}()

	if err = tx.Where(&rRelation).Delete(&rRelation).Error; err != nil {
		resp = &relation.RelationActionResponse{
			StatusCode: strings.UnableToUnFollowErrorCode,
			StatusMsg:  strings.UnableToUnFollowError,
		}
		logging.SetSpanError(span, err)
		return
	}

	if err = updateFollowListCache(ctx, request.ActorId, rRelation, false, span, logger); err != nil {
		logger.WithFields(logrus.Fields{
			"err": err,
		}).Errorf("failed to update follow list cache")
		logging.SetSpanError(span, err)
		return
	}

	if err = updateFollowerListCache(ctx, request.UserId, rRelation, false, span, logger); err != nil {
		logger.WithFields(logrus.Fields{
			"err": err,
		}).Errorf("failed to update follower list cache")
		logging.SetSpanError(span, err)
		return
	}

	if err = updateFollowCountCache(ctx, request.ActorId, false, span, logger); err != nil {
		logger.WithFields(logrus.Fields{
			"err": err,
		}).Errorf("failed to update follow count cache")
		logging.SetSpanError(span, err)
		return
	}

	if err = updateFollowerCountCache(ctx, request.UserId, false, span, logger); err != nil {
		logger.WithFields(logrus.Fields{
			"err": err,
		}).Errorf("failed to update follower count cache")
		logging.SetSpanError(span, err)
		return
	}

	resp = &relation.RelationActionResponse{
		StatusCode: strings.ServiceOKCode,
		StatusMsg:  strings.ServiceOK,
	}
	return
}

func (r RelationServiceImpl) CountFollowList(ctx context.Context, request *relation.CountFollowListRequest) (resp *relation.CountFollowListResponse, err error) {
	ctx, span := tracing.Tracer.Start(ctx, "CountFollowListService")
	defer span.End()
	logger := logging.LogService("RelationService.CountFollowList").WithContext(ctx)

	cacheKey := fmt.Sprintf("follow_list_count_%d", request.UserId)
	cachedCountString, ok, err := cached.Get(ctx, cacheKey)

	if err != nil {
		logger.WithFields(logrus.Fields{
			"error": err,
		}).Errorf("Err when read Redis")
		logging.SetSpanError(span, err)
	}

	var cachedCount64 uint64
	if ok {
		cachedCount64, err = strconv.ParseUint(cachedCountString, 10, 32)
		if err != nil {
			logger.WithFields(logrus.Fields{
				"error": err,
			}).Errorf("fail to convert string to int when countFollow")
			logging.SetSpanError(span, err)
			resp = &relation.CountFollowListResponse{
				StatusCode: strings.StringToIntErrorCode,
				StatusMsg:  strings.StringToIntError,
			}
			return
		}
		cachedCount := uint32(cachedCount64)

		logger.WithFields(logrus.Fields{
			"userId": request.UserId,
		}).Infof("Cache hit for follow list count for user %d", request.UserId)
		resp = &relation.CountFollowListResponse{
			StatusCode: strings.ServiceOKCode,
			StatusMsg:  strings.ServiceOK,
			Count:      cachedCount,
		}
		return
	}

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
	countString := strconv.FormatUint(uint64(count), 10)
	cached.Write(ctx, cacheKey, countString, true)

	return
}

func (r RelationServiceImpl) CountFollowerList(ctx context.Context, request *relation.CountFollowerListRequest) (resp *relation.CountFollowerListResponse, err error) {
	ctx, span := tracing.Tracer.Start(ctx, "CountFollowerListService")
	defer span.End()
	logger := logging.LogService("RelationService.CountFollowerList").WithContext(ctx)

	cacheKey := fmt.Sprintf("follower_count_%d", request.UserId)

	cachedCountString, ok, err := cached.Get(ctx, cacheKey)

	if err != nil {
		logger.WithFields(logrus.Fields{
			"error": err,
		}).Errorf("Err when read Redis")
		logging.SetSpanError(span, err)
	}

	var cachedCount64 uint64
	if ok {
		cachedCount64, err = strconv.ParseUint(cachedCountString, 10, 32)
		if err != nil {
			logger.WithFields(logrus.Fields{
				"error": err,
			}).Errorf("fail to convert string to int when countFollower")
			logging.SetSpanError(span, err)
			resp = &relation.CountFollowerListResponse{
				StatusCode: strings.StringToIntErrorCode,
				StatusMsg:  strings.StringToIntError,
			}
			return
		}
		cachedCount := uint32(cachedCount64)

		logger.Infof("Cache hit for follower count for user %d", request.UserId)
		resp = &relation.CountFollowerListResponse{
			StatusCode: strings.ServiceOKCode,
			StatusMsg:  strings.ServiceOK,
			Count:      cachedCount,
		}
		return
	}

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
	countString := strconv.FormatUint(uint64(count), 10)
	cached.Write(ctx, cacheKey, countString, true)
	return
}

func (r RelationServiceImpl) GetFriendList(ctx context.Context, request *relation.FriendListRequest) (resp *relation.FriendListResponse, err error) {
	ctx, span := tracing.Tracer.Start(ctx, "GetFriendListService")
	defer span.End()
	logger := logging.LogService("RelationService.GetFriendList").WithContext(ctx)

	//followList
	cacheKey := fmt.Sprintf("follow_list_%d", request.UserId)
	followList := CacheRelationList{}
	ok, err := cached.CacheAndRedisGet(ctx, cacheKey, &followList)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"error": err,
		}).Errorf("Err when read Redis")
		logging.SetSpanError(span, err)
	}

	if ok {
		logger.Infof("Cache hit for follow list for user %d", request.UserId)
	} else {
		followResult := database.Client.WithContext(ctx).
			Where("actor_id = ?", request.UserId).
			Find(&followList.rList)

		if followResult.Error != nil {
			logger.WithFields(logrus.Fields{
				"err": followResult.Error,
			}).Errorf("GetFriendListService failed with error")
			logging.SetSpanError(span, followResult.Error)

			resp = &relation.FriendListResponse{
				StatusCode: strings.UnableToGetFollowListErrorCode,
				StatusMsg:  strings.UnableToGetFollowListError,
			}
			return
		}
	}
	err = cached.ScanWriteCache(ctx, cacheKey, &followList, true)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"err": err,
			"key": cacheKey,
		}).Errorf("failed to write cache for follow list")
		logging.SetSpanError(span, err)
	}

	// 构建关注列表的用户 ID 映射
	followingMap := make(map[uint32]bool)
	for _, follow := range followList.rList {
		followingMap[follow.UserId] = true
	}

	//followerList
	cacheKey = fmt.Sprintf("follower_list_%d", request.UserId)
	followerList := CacheRelationList{}
	ok, err = cached.CacheAndRedisGet(ctx, cacheKey, &followerList)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"error": err,
		}).Errorf("Err when read Redis")
		logging.SetSpanError(span, err)
	}

	if ok {
		logger.WithFields(logrus.Fields{
			"userId": request.UserId,
		}).Infof("Cache hit for follower list for user %d", request.UserId)
	} else {
		followerResult := database.Client.WithContext(ctx).
			Where("user_id = ?", request.UserId).
			Find(&followerList.rList)

		if followerResult.Error != nil {
			logger.WithFields(logrus.Fields{
				"err": followerResult.Error,
			}).Errorf("GetFriendListService failed with error")
			logging.SetSpanError(span, followerResult.Error)

			resp = &relation.FriendListResponse{
				StatusCode: strings.UnableToGetFollowerListErrorCode,
				StatusMsg:  strings.UnableToGetFollowerListError,
			}
			return
		}
	}
	err = cached.ScanWriteCache(ctx, cacheKey, &followerList, true)

	if err != nil {
		logger.WithFields(logrus.Fields{
			"err": err,
			"key": cacheKey,
		}).Errorf("failed to write cache for follower list")
		logging.SetSpanError(span, err)
	}

	// 构建互相关注的用户列表（既关注了关注者又被关注者所关注的用户）
	mutualFriends := make([]*user.User, 0)

	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, follower := range followerList.rList {
		wg.Add(1)

		go func(follower models.Relation) {
			defer wg.Done()

			if followingMap[follower.ActorId] {
				userResponse, err := userClient.GetUserInfo(ctx, &user.UserRequest{
					UserId:  follower.ActorId,
					ActorId: request.ActorId,
				})
				if err != nil || userResponse.StatusCode != strings.ServiceOKCode {
					logger.WithFields(logrus.Fields{
						"err":      err,
						"follower": follower,
					}).Errorf("Unable to get information about users who follow each other")
					logging.SetSpanError(span, err)
					mu.Lock()
					mutualFriends = append(mutualFriends, userResponse.User)
					mu.Unlock()
				}
			}
		}(follower)
	}

	wg.Wait()

	resp = &relation.FriendListResponse{
		StatusCode: strings.ServiceOKCode,
		StatusMsg:  strings.ServiceOK,
		UserList:   mutualFriends,
	}
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

	resp = &relation.IsFollowResponse{
		Result: count > 0,
	}
	return
}

func (r RelationServiceImpl) GetFollowList(ctx context.Context, request *relation.FollowListRequest) (resp *relation.FollowListResponse, err error) {
	ctx, span := tracing.Tracer.Start(ctx, "GetFollowListService")
	defer span.End()
	logger := logging.LogService("RelationService.GetFollowList").WithContext(ctx)

	cacheKey := fmt.Sprintf("follow_list_%d", request.UserId)
	cachedFollowList := CacheRelationList{}

	// cache and redis
	ok, err := cached.CacheAndRedisGet(ctx, cacheKey, &cachedFollowList)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"error": err,
		}).Errorf("Err when read Redis")
		logging.SetSpanError(span, err)
	}

	var rFollowList []*user.User
	if ok {
		logger.Infof("Cache hit, retrieving follow list for user %d", request.UserId)

		rFollowList, err = r.fetchUserListInfo(ctx, cachedFollowList.rList, request.ActorId, logger, span)
		if err != nil {
			resp = &relation.FollowListResponse{
				StatusCode: strings.UnableToGetFollowListErrorCode,
				StatusMsg:  strings.UnableToGetFollowListError,
				UserList:   nil,
			}
			logger.WithFields(logrus.Fields{
				"err": err,
			}).Errorf("failed to convert relation to user")
			logging.SetSpanError(span, err)
			return
		}

		resp = &relation.FollowListResponse{
			StatusCode: strings.ServiceOKCode,
			StatusMsg:  strings.ServiceOK,
			UserList:   rFollowList,
		}
		return
	}

	//database
	var followList []models.Relation
	result := database.Client.WithContext(ctx).
		Where("actor_id = ?", request.UserId).
		Order("created_at desc").
		Find(&followList)

	if result.Error != nil {
		logger.WithFields(logrus.Fields{
			"err": result.Error,
		}).Errorf("Failed to retrieve follow list")
		logging.SetSpanError(span, err)

		resp = &relation.FollowListResponse{
			StatusCode: strings.UnableToGetFollowListErrorCode,
			StatusMsg:  strings.UnableToGetFollowListError,
		}
		return
	}
	cachedFollowList.rList = followList

	err = cached.ScanWriteCache(ctx, cacheKey, &cachedFollowList, true)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"err": err,
			"key": cacheKey,
		}).Errorf("failed to write cache for follow list")
		logging.SetSpanError(span, err)
	}

	rFollowList, err = r.fetchUserListInfo(ctx, followList, request.ActorId, logger, span)
	if err != nil {
		resp = &relation.FollowListResponse{
			StatusCode: strings.UnableToGetFollowListErrorCode,
			StatusMsg:  strings.UnableToGetFollowListError,
		}
		logger.WithFields(logrus.Fields{
			"err": err,
		}).Errorf("failed to convert relation to user")
		logging.SetSpanError(span, err)
		return
	}

	resp = &relation.FollowListResponse{
		StatusCode: strings.ServiceOKCode,
		StatusMsg:  strings.ServiceOK,
		UserList:   rFollowList,
	}

	return
}

func (r RelationServiceImpl) GetFollowerList(ctx context.Context, request *relation.FollowerListRequest) (resp *relation.FollowerListResponse, err error) {
	ctx, span := tracing.Tracer.Start(ctx, "GetFollowerListService")
	defer span.End()
	logger := logging.LogService("RelationService.GetFollowerList").WithContext(ctx)

	cacheKey := fmt.Sprintf("follower_list_%d", request.UserId)
	cachedFollowerList := CacheRelationList{}

	ok, err := cached.CacheAndRedisGet(ctx, cacheKey, &cachedFollowerList)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"error": err,
		}).Errorf("Err when read Redis")
		logging.SetSpanError(span, err)
	}

	var rFollowerList []*user.User
	if ok {
		logger.Infof("Cache hit, retrieving follower list for user %d", request.UserId)

		rFollowerList, err = r.fetchUserListInfo(ctx, cachedFollowerList.rList, request.ActorId, logger, span)
		if err != nil {
			resp = &relation.FollowerListResponse{
				StatusCode: strings.UnableToGetFollowerListErrorCode,
				StatusMsg:  strings.UnableToGetFollowerListError,
				UserList:   nil,
			}
			logger.WithFields(logrus.Fields{
				"err": err,
			}).Errorf("failed to convert relation to user")
			logging.SetSpanError(span, err)
			return
		}

		resp = &relation.FollowerListResponse{
			StatusCode: strings.ServiceOKCode,
			StatusMsg:  strings.ServiceOK,
			UserList:   rFollowerList,
		}
		return
	}

	var followerList []models.Relation
	result := database.Client.WithContext(ctx).
		Where("user_id = ?", request.UserId).
		Order("created_at desc").
		Find(&followerList)

	if result.Error != nil {
		logger.WithFields(logrus.Fields{
			"err": result.Error,
		}).Errorf("Failed to retrieve follower list")
		logging.SetSpanError(span, err)

		resp = &relation.FollowerListResponse{
			StatusCode: strings.UnableToGetFollowerListErrorCode,
			StatusMsg:  strings.UnableToGetFollowerListError,
		}
		return
	}

	cachedFollowerList.rList = followerList
	err = cached.ScanWriteCache(ctx, cacheKey, &cachedFollowerList, true)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"err": err,
			"key": cacheKey,
		}).Errorf("failed to write cache for follower list")
		logging.SetSpanError(span, err)
	}

	rFollowerList, err = r.fetchUserListInfo(ctx, followerList, request.ActorId, logger, span)
	if err != nil {
		resp = &relation.FollowerListResponse{
			StatusCode: strings.UnableToGetFollowerListErrorCode,
			StatusMsg:  strings.UnableToGetFollowerListError,
		}
		logger.WithFields(logrus.Fields{
			"err": err,
		}).Errorf("failed to convert relation to user")
		logging.SetSpanError(span, err)
		return
	}

	resp = &relation.FollowerListResponse{
		StatusCode: strings.ServiceOKCode,
		StatusMsg:  strings.ServiceOK,
		UserList:   rFollowerList,
	}

	return
}

func (r RelationServiceImpl) fetchUserListInfo(ctx context.Context, userList []models.Relation, actorID uint32, logger *logrus.Entry, span trace.Span) ([]*user.User, error) {
	var wg sync.WaitGroup
	var mu sync.Mutex
	var wgErrors []error
	var err error

	maxRetries := 3
	retryInterval := 1

	rUserList := make([]*user.User, 0, len(userList))

	for _, r := range userList {
		wg.Add(1)
		go func(relation models.Relation) {
			defer wg.Done()

			retryCount := 0
			for retryCount < maxRetries {
				userResponse, err := userClient.GetUserInfo(ctx, &user.UserRequest{
					UserId:  relation.UserId,
					ActorId: actorID,
				})

				if err != nil || userResponse.StatusCode != strings.ServiceOKCode {
					logger.WithFields(logrus.Fields{
						"err":      err,
						"relation": relation,
					}).Errorf("Unable to get user information")
					retryCount++
					time.Sleep(time.Duration(retryInterval) * time.Second)
					continue
				} else {
					mu.Lock()
					rUserList = append(rUserList, userResponse.User)
					mu.Unlock()
					break
				}
			}

			if retryCount >= maxRetries {
				logging.SetSpanError(span, err)
			}
		}(r)
	}

	wg.Wait()

	if len(wgErrors) > 0 {
		logger.WithFields(logrus.Fields{
			"errorNum": wgErrors,
		}).Errorf("%d user information fails to be queried", len(wgErrors))
		return nil, fmt.Errorf("%d user information fails to be queried", len(wgErrors))
	}

	return rUserList, nil
}

// followOp = true  ->  follow
// followOp = false ->  unfollow
func updateFollowListCache(ctx context.Context, actorID uint32, relation models.Relation, followOp bool, span trace.Span, logger *logrus.Entry) error {

	cacheKey := fmt.Sprintf("follow_list_%d", actorID)
	cachedRelationList := CacheRelationList{}

	ok, err := cached.CacheAndRedisGet(ctx, cacheKey, &cachedRelationList)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"error": err,
		}).Errorf("Redis error when find struct")
		logging.SetSpanError(span, err)
		return err
	}

	if !ok {
		result := database.Client.WithContext(ctx).
			Where("actor_id = ?", actorID).
			Find(&cachedRelationList.rList)
		if result.Error != nil {
			logger.WithFields(logrus.Fields{
				"err": result.Error,
			}).Errorf("GetFollowList from database failed: %v", result.Error)
			logging.SetSpanError(span, result.Error)
			return result.Error
		}
	}

	if followOp {
		cachedRelationList.rList = append(cachedRelationList.rList, relation)
	} else {
		for i, r := range cachedRelationList.rList {
			if r.UserId == relation.UserId {
				cachedRelationList.rList = append(cachedRelationList.rList[:i], cachedRelationList.rList[i+1:]...)
				break
			}
		}
	}

	err = cached.ScanWriteCache(ctx, cacheKey, &cachedRelationList, true)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"err": err,
		}).Errorf("ScanWriteCache failed")
		logging.SetSpanError(span, err)
		return err
	}

	return nil
}

func updateFollowerListCache(ctx context.Context, userID uint32, relation models.Relation, followOp bool, span trace.Span, logger *logrus.Entry) error {
	cacheKey := fmt.Sprintf("follower_list_%d", userID)
	cachedRelationList := CacheRelationList{}

	ok, err := cached.CacheAndRedisGet(ctx, cacheKey, &cachedRelationList)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"error": err,
		}).Errorf("Redis error when find struct")
		logging.SetSpanError(span, err)
		return err
	}

	if !ok {
		result := database.Client.WithContext(ctx).
			Where("user_id = ?", userID).
			Find(&cachedRelationList.rList)
		if result.Error != nil {
			logger.WithFields(logrus.Fields{
				"err": result.Error,
			}).Errorf("GetFollowerList from database failed: %v", result.Error)
			logging.SetSpanError(span, result.Error)
			return result.Error
		}
	}

	if followOp {
		cachedRelationList.rList = append(cachedRelationList.rList, relation)
	} else {
		for i, r := range cachedRelationList.rList {
			if r.ActorId == relation.ActorId {
				cachedRelationList.rList = append(cachedRelationList.rList[:i], cachedRelationList.rList[i+1:]...)
				break
			}
		}
	}

	err = cached.ScanWriteCache(ctx, cacheKey, &cachedRelationList, true)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"err": err,
		}).Errorf("ScanWriteCache failed")
		logging.SetSpanError(span, err)
		return err
	}
	return nil
}

func updateFollowCountCache(ctx context.Context, actorID uint32, followOp bool, span trace.Span, logger *logrus.Entry) error {
	cacheKey := fmt.Sprintf("follow_count_%d", actorID)
	var count uint32

	cachedCountString, ok, err := cached.Get(ctx, cacheKey)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"error": err,
		}).Errorf("Err when read Redis")
		logging.SetSpanError(span, err)
	}

	if ok {
		cachedCount64, err := strconv.ParseUint(cachedCountString, 10, 32)
		if err != nil {
			logger.WithFields(logrus.Fields{
				"error": err,
			}).Errorf("fail to convert string to int when updateFollowCountCache")
			logging.SetSpanError(span, err)
			return err
		}
		cachedCount := uint32(cachedCount64)
		if !followOp {
			// unfollow
			if cachedCount > 0 {
				count = cachedCount - 1
			} else {
				count = 0
			}
		} else {
			// follow
			count = cachedCount + 1
		}
	} else {
		// not hit in cache
		var dbCount int64
		result := database.Client.WithContext(ctx).
			Model(&models.Relation{}).
			Where("actor_id = ?", actorID).
			Count(&dbCount)

		if result.Error != nil {
			logger.WithFields(logrus.Fields{
				"error": result.Error,
			}).Errorf("fail to get data from database when updatecache")
			logging.SetSpanError(span, result.Error)
			return result.Error
		}

		count = uint32(dbCount)
	}

	countString := strconv.FormatUint(uint64(count), 10)
	cached.Write(ctx, cacheKey, countString, true)

	return nil
}

func updateFollowerCountCache(ctx context.Context, userID uint32, followOp bool, span trace.Span, logger *logrus.Entry) error {
	cacheKey := fmt.Sprintf("follower_count_%d", userID)
	var count uint32

	cachedCountString, ok, err := cached.Get(ctx, cacheKey)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"error": err,
		}).Errorf("Err when read Redis")
		logging.SetSpanError(span, err)
	}

	if ok {
		cachedCount64, err := strconv.ParseUint(cachedCountString, 10, 32)
		if err != nil {
			logger.WithFields(logrus.Fields{
				"error": err,
			}).Errorf("fail to convert string to int when updateFollowerCountCache")
			logging.SetSpanError(span, err)
			return err
		}
		cachedCount := uint32(cachedCount64)
		if !followOp {
			// unfollow
			if cachedCount > 0 {
				count = cachedCount - 1
			} else {
				count = 0
			}
		} else {
			// follow
			count = cachedCount + 1
		}
	} else {
		// not hit in cache
		var dbCount int64
		result := database.Client.WithContext(ctx).
			Model(&models.Relation{}).
			Where("user_id = ?", userID).
			Count(&dbCount)

		if result.Error != nil {
			logger.WithFields(logrus.Fields{
				"error": result.Error,
			}).Errorf("fail to get data from database when updatecache")
			logging.SetSpanError(span, result.Error)
			return result.Error
		}

		count = uint32(dbCount)
	}
	countString := strconv.FormatUint(uint64(count), 10)
	cached.Write(ctx, cacheKey, countString, true)
	return nil
}
