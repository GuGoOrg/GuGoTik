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
	redis2 "GuGoTik/src/storage/redis"
	"GuGoTik/src/utils/audit"
	grpc2 "GuGoTik/src/utils/grpc"
	"GuGoTik/src/utils/logging"
	"GuGoTik/src/utils/rabbitmq"
	"context"
	"errors"
	"fmt"
	"github.com/go-redis/redis_rate/v10"
	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/gorm"
	"strconv"
	"sync"
	"time"
)

var userClient user.UserServiceClient

var actionRelationLimitKeyPrefix = config.EnvCfg.RedisPrefix + "relation_freq_limit"

const actionRelationMaxQPS = 3

type RelationServiceImpl struct {
	relation.RelationServiceServer
}

func actionRelationLimitKey(userId uint32) string {
	return fmt.Sprintf("%s-%d", actionRelationLimitKeyPrefix, userId)
}

func exitOnError(err error) {
	if err != nil {
		panic(err)
	}
}

func CloseMQConn() {
	if err := conn.Close(); err != nil {
		panic(err)
	}

	if err := channel.Close(); err != nil {
		panic(err)
	}
}

func (r RelationServiceImpl) New() {
	userRPCConn := grpc2.Connect(config.UserRpcServerName)
	userClient = user.NewUserServiceClient(userRPCConn)

	var err error

	conn, err = amqp.Dial(rabbitmq.BuildMQConnAddr())
	exitOnError(err)

	channel, err = conn.Channel()
	exitOnError(err)
}

func (r RelationServiceImpl) Follow(ctx context.Context, request *relation.RelationActionRequest) (resp *relation.RelationActionResponse, err error) {
	ctx, span := tracing.Tracer.Start(ctx, "FollowService")
	defer span.End()
	logging.SetSpanWithHostname(span)
	logger := logging.LogService("RelationService.Follow").WithContext(ctx)

	//限流
	limiter := redis_rate.NewLimiter(redis2.Client)
	limiterKey := actionRelationLimitKey(request.ActorId)
	limiterRes, err := limiter.Allow(ctx, limiterKey, redis_rate.PerSecond(actionRelationMaxQPS))
	if err != nil {
		logger.WithFields(logrus.Fields{
			"err":     err,
			"ActorId": request.ActorId,
		}).Errorf("ActionRelation limiter error")
		logging.SetSpanError(span, err)

		resp = &relation.RelationActionResponse{
			StatusCode: strings.UnableToFollowErrorCode,
			StatusMsg:  strings.UnableToFollowError,
		}
		return
	}
	if limiterRes.Allowed == 0 {
		logger.WithFields(logrus.Fields{
			"err":     err,
			"ActorId": request.ActorId,
		}).Infof("Follow query too frequently by user %d", request.ActorId)

		resp = &relation.RelationActionResponse{
			StatusCode: strings.FollowLimitedCode,
			StatusMsg:  strings.FollowLimited,
		}
		return
	}

	//actor exists
	userExist, err := userClient.GetUserExistInformation(ctx, &user.UserExistRequest{UserId: request.ActorId})

	if err != nil || userExist.StatusCode != strings.ServiceOKCode {
		logger.WithFields(logrus.Fields{
			"err":     err,
			"ActorId": request.ActorId,
		}).Errorf("User service error")
		logging.SetSpanError(span, err)

		resp = &relation.RelationActionResponse{
			StatusCode: strings.UnableToQueryUserErrorCode,
			StatusMsg:  strings.UnableToQueryUserError,
		}
		return
	}

	if !userExist.Existed {
		resp = &relation.RelationActionResponse{
			StatusCode: strings.UserDoNotExistedCode,
			StatusMsg:  strings.UserDoNotExisted,
		}
		return
	}

	if request.UserId == request.ActorId {
		resp = &relation.RelationActionResponse{
			StatusCode: strings.UnableToRelateYourselfErrorCode,
			StatusMsg:  strings.UnableToRelateYourselfError,
		}
		return
	}

	userExist, err = userClient.GetUserExistInformation(ctx, &user.UserExistRequest{UserId: request.UserId})

	if err != nil || userExist.StatusCode != strings.ServiceOKCode {
		logger.WithFields(logrus.Fields{
			"err":     err,
			"ActorId": request.ActorId,
		}).Errorf("User service error")
		logging.SetSpanError(span, err)

		resp = &relation.RelationActionResponse{
			StatusCode: strings.UnableToQueryUserErrorCode,
			StatusMsg:  strings.UnableToQueryUserError,
		}
		return
	}

	if !userExist.Existed {
		resp = &relation.RelationActionResponse{
			StatusCode: strings.UserDoNotExistedCode,
			StatusMsg:  strings.UserDoNotExisted,
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

	// 检查是否已经存在相同的记录
	var count int64
	if err = tx.Model(&models.Relation{}).Where("actor_id = ? AND user_id = ?", rRelation.ActorId, rRelation.UserId).Count(&count).Error; err != nil {
		resp = &relation.RelationActionResponse{
			StatusCode: strings.UnableToFollowErrorCode,
			StatusMsg:  strings.UnableToFollowError,
		}
		logging.SetSpanError(span, err)
		return
	}
	if count > 0 {
		resp = &relation.RelationActionResponse{
			StatusCode: strings.AlreadyFollowingErrorCode,
			StatusMsg:  strings.AlreadyFollowingError,
		}
		return
	}

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
	cached.TagDelete(ctx, fmt.Sprintf("IsFollowedCache-%d-%d", request.UserId, request.ActorId))
	resp = &relation.RelationActionResponse{
		StatusCode: strings.ServiceOKCode,
		StatusMsg:  strings.ServiceOK,
	}

	// Publish event to event_exchange and audit_exchange
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		action := &models.Action{
			Type:         strings.FollowIdActionLog,
			Name:         strings.FollowNameActionLog,
			SubName:      strings.FollowUpActionSubLog,
			ServiceName:  strings.FollowServiceName,
			ActorId:      request.ActorId,
			VideoId:      0,
			AffectUserId: request.UserId,
			AffectAction: 1,
			AffectedData: "1",
			EventId:      uuid.New().String(),
			TraceId:      trace.SpanContextFromContext(ctx).TraceID().String(),
			SpanId:       trace.SpanContextFromContext(ctx).SpanID().String(),
		}
		audit.PublishAuditEvent(ctx, action, channel)
	}()
	wg.Wait()

	return
}

func (r RelationServiceImpl) Unfollow(ctx context.Context, request *relation.RelationActionRequest) (resp *relation.RelationActionResponse, err error) {
	ctx, span := tracing.Tracer.Start(ctx, "UnfollowService")
	defer span.End()
	logging.SetSpanWithHostname(span)
	logger := logging.LogService("RelationService.Unfollow").WithContext(ctx)

	//actor exists
	userExist, err := userClient.GetUserExistInformation(ctx, &user.UserExistRequest{UserId: request.ActorId})

	if err != nil || userExist.StatusCode != strings.ServiceOKCode {
		logger.WithFields(logrus.Fields{
			"err":     err,
			"ActorId": request.ActorId,
		}).Errorf("User service error")
		logging.SetSpanError(span, err)

		resp = &relation.RelationActionResponse{
			StatusCode: strings.UnableToQueryUserErrorCode,
			StatusMsg:  strings.UnableToQueryUserError,
		}
		return
	}
	if !userExist.Existed {
		resp = &relation.RelationActionResponse{
			StatusCode: strings.UserDoNotExistedCode,
			StatusMsg:  strings.UserDoNotExisted,
		}
		return
	}

	if request.UserId == request.ActorId {
		resp = &relation.RelationActionResponse{
			StatusCode: strings.UnableToRelateYourselfErrorCode,
			StatusMsg:  strings.UnableToRelateYourselfError,
		}
		return
	}

	userExist, err = userClient.GetUserExistInformation(ctx, &user.UserExistRequest{UserId: request.UserId})

	if err != nil || userExist.StatusCode != strings.ServiceOKCode {
		logger.WithFields(logrus.Fields{
			"err":     err,
			"ActorId": request.ActorId,
		}).Errorf("User service error")
		logging.SetSpanError(span, err)

		resp = &relation.RelationActionResponse{
			StatusCode: strings.UnableToQueryUserErrorCode,
			StatusMsg:  strings.UnableToQueryUserError,
		}
		return
	}

	if !userExist.Existed {
		resp = &relation.RelationActionResponse{
			StatusCode: strings.UserDoNotExistedCode,
			StatusMsg:  strings.UserDoNotExisted,
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

	if err = tx.Unscoped().Where(&rRelation).Delete(&rRelation).Error; err != nil {
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
	cached.TagDelete(ctx, fmt.Sprintf("IsFollowedCache-%d-%d", request.UserId, request.ActorId))
	resp = &relation.RelationActionResponse{
		StatusCode: strings.ServiceOKCode,
		StatusMsg:  strings.ServiceOK,
	}

	// Publish event to event_exchange and audit_exchange
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		action := &models.Action{
			Type:         strings.FollowIdActionLog,
			Name:         strings.FollowNameActionLog,
			SubName:      strings.FollowDownActionSubLog,
			ServiceName:  strings.FollowServiceName,
			ActorId:      request.ActorId,
			VideoId:      0,
			AffectUserId: request.UserId,
			AffectAction: 1,
			AffectedData: "-1",
			EventId:      uuid.New().String(),
			TraceId:      trace.SpanContextFromContext(ctx).TraceID().String(),
			SpanId:       trace.SpanContextFromContext(ctx).SpanID().String(),
		}
		audit.PublishAuditEvent(ctx, action, channel)
	}()
	wg.Wait()

	return
}

func (r RelationServiceImpl) CountFollowList(ctx context.Context, request *relation.CountFollowListRequest) (resp *relation.CountFollowListResponse, err error) {
	ctx, span := tracing.Tracer.Start(ctx, "CountFollowListService")
	defer span.End()
	logging.SetSpanWithHostname(span)
	logger := logging.LogService("RelationService.CountFollowList").WithContext(ctx)
	//actor exists
	userExist, err := userClient.GetUserExistInformation(ctx, &user.UserExistRequest{UserId: request.UserId})

	if err != nil || userExist.StatusCode != strings.ServiceOKCode {
		logger.WithFields(logrus.Fields{
			"err":    err,
			"UserId": request.UserId,
		}).Errorf("not find the user:%v", request.UserId)
		logging.SetSpanError(span, err)

		resp = &relation.CountFollowListResponse{
			StatusCode: strings.UnableToQueryUserErrorCode,
			StatusMsg:  strings.UnableToQueryUserError,
		}
		return
	}

	if !userExist.Existed {
		resp = &relation.CountFollowListResponse{
			StatusCode: strings.UserDoNotExistedCode,
			StatusMsg:  strings.UserDoNotExisted,
		}
		return
	}

	cacheKey := fmt.Sprintf("follow_count_%d", request.UserId)
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
	logging.SetSpanWithHostname(span)
	logger := logging.LogService("RelationService.CountFollowerList").WithContext(ctx)

	userExist, err := userClient.GetUserExistInformation(ctx, &user.UserExistRequest{UserId: request.UserId})

	if err != nil || userExist.StatusCode != strings.ServiceOKCode {
		logger.WithFields(logrus.Fields{
			"err":    err,
			"UserId": request.UserId,
		}).Errorf("not find the user:%v", request.UserId)
		logging.SetSpanError(span, err)

		resp = &relation.CountFollowerListResponse{
			StatusCode: strings.UnableToQueryUserErrorCode,
			StatusMsg:  strings.UnableToQueryUserError,
		}
		return
	}

	if !userExist.Existed {
		resp = &relation.CountFollowerListResponse{
			StatusCode: strings.UserDoNotExistedCode,
			StatusMsg:  strings.UserDoNotExisted,
		}
		return
	}

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
	logging.SetSpanWithHostname(span)
	logger := logging.LogService("RelationService.GetFriendList").WithContext(ctx)

	ok, err := isUserExist(ctx, request.ActorId, request.UserId, span, logger)
	if err != nil || !ok {
		resp = &relation.FriendListResponse{
			StatusCode: strings.UnableToQueryUserErrorCode,
			StatusMsg:  strings.UnableToQueryUserError,
		}
		return
	}

	//followList
	cacheKey := config.EnvCfg.RedisPrefix + fmt.Sprintf("follow_list_%d", request.UserId)
	followIdList, err := redis2.Client.SMembers(ctx, cacheKey).Result()
	var followRelationList []models.Relation
	// 构建关注列表的用户 ID 映射
	followingMap := make(map[uint32]bool)
	//判断是否需要读db
	db := false

	if err != nil {
		db = true
	} else {
		for _, id := range followIdList {
			idInt, err := strconv.Atoi(id)
			//redis存在不合法的id，删除redis中的整个set并重新读数据库，写redis缓存
			if err != nil {
				logger.WithFields(logrus.Fields{
					"id":  id,
					"err": err,
				}).Errorf("Redis exists illegal id %s", id)
				logging.SetSpanError(span, err)
				_, err := redis2.Client.Del(ctx, cacheKey).Result()
				if err != nil {
					logger.WithFields(logrus.Fields{
						"id":  id,
						"err": err,
					}).Errorf("Redis exists illegal id %s and delete redis failed", id)
					logging.SetSpanError(span, err)
				}
				break
			}
			followingMap[uint32(idInt)] = true
		}
	}

	if db {
		logger.WithFields(logrus.Fields{
			"error": err,
		}).Errorf("Err when read Redis or no data in Redis")
		logging.SetSpanError(span, err)

		followResult := database.Client.WithContext(ctx).
			Where("actor_id = ?", request.UserId).
			Find(&followRelationList)
		if followResult.Error != nil {
			logger.WithFields(logrus.Fields{
				"err": followResult.Error,
			}).Errorf("GetFriendListService failed with dataBaseError")
			logging.SetSpanError(span, followResult.Error)

			resp = &relation.FriendListResponse{
				StatusCode: strings.UnableToGetFollowListErrorCode,
				StatusMsg:  strings.UnableToGetFollowListError,
			}
			return
		}
		for _, rel := range followRelationList {
			followingMap[rel.UserId] = true
		}
		for _, rel := range followRelationList {
			redis2.Client.SAdd(ctx, cacheKey, rel.UserId)
		}
	}

	//followerList
	cacheKey = config.EnvCfg.RedisPrefix + fmt.Sprintf("follower_list_%d", request.UserId)
	followerIdList, err := redis2.Client.SMembers(ctx, cacheKey).Result()
	var followerRelationList []models.Relation
	followerIdListInt := make([]uint32, len(followerIdList))
	db = false

	if err != nil {
		db = true
	} else {
		for index, id := range followerIdList {
			idInt, err := strconv.Atoi(id)
			//redis存在不合法的id，删除redis中的整个set并重新读数据库，写redis缓存
			if err != nil {
				logger.WithFields(logrus.Fields{
					"id":  id,
					"err": err,
				}).Errorf("Redis exists illegal id %s", id)
				logging.SetSpanError(span, err)
				_, err := redis2.Client.Del(ctx, cacheKey).Result()
				if err != nil {
					logger.WithFields(logrus.Fields{
						"id":  id,
						"err": err,
					}).Errorf("Redis exists illegal id %s and delete redis failed", id)
					logging.SetSpanError(span, err)
				}
				break
			}
			followerIdListInt[index] = uint32(idInt)
		}
	}

	if db {
		logger.WithFields(logrus.Fields{
			"error": err,
		}).Errorf("Err when read Redis or no data in Redis")
		logging.SetSpanError(span, err)

		followerResult := database.Client.WithContext(ctx).
			Where("user_id = ?", request.UserId).
			Find(&followerRelationList)
		if followerResult.Error != nil {
			logger.WithFields(logrus.Fields{
				"err": followerResult.Error,
			}).Errorf("GetFriendListService failed with dataBaseError")
			logging.SetSpanError(span, followerResult.Error)

			resp = &relation.FriendListResponse{
				StatusCode: strings.UnableToGetFollowerListErrorCode,
				StatusMsg:  strings.UnableToGetFollowerListError,
			}
			return
		}
		for index, rel := range followerRelationList {
			followerIdListInt[index] = rel.ActorId
		}
		for _, rel := range followerRelationList {
			redis2.Client.SAdd(ctx, cacheKey, rel.ActorId)
		}
	}

	// 构建互相关注的用户列表（既关注了关注者又被关注者所关注的用户）
	mutualFriends := make([]*user.User, 0)

	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, id := range followerIdListInt {
		wg.Add(1)

		go func(id uint32) {
			defer wg.Done()

			if followingMap[id] {

				userResponse, err := userClient.GetUserInfo(ctx, &user.UserRequest{
					UserId:  id,
					ActorId: request.ActorId,
				})

				if err != nil || userResponse.StatusCode != strings.ServiceOKCode {
					logger.WithFields(logrus.Fields{
						"err":        err,
						"followerId": id,
					}).Errorf("Unable to get information about users who follow each other")
					logging.SetSpanError(span, err)
					resp = &relation.FriendListResponse{
						StatusCode: strings.UnableToGetFriendListErrorCode,
						StatusMsg:  strings.UnableToGetFriendListError,
						UserList:   nil,
					}
				} else {
					mu.Lock()
					mutualFriends = append(mutualFriends, userResponse.User)
					mu.Unlock()
				}

			}
		}(id)
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
	logging.SetSpanWithHostname(span)
	logger := logging.LogService("RelationService.isFollow").WithContext(ctx)

	res, err := cached.GetWithFunc(ctx, fmt.Sprintf("IsFollowedCache-%d-%d", request.UserId, request.ActorId), func(ctx context.Context, key string) (string, error) {
		var count int64
		row := database.Client.WithContext(ctx).
			Model(&models.Relation{}).
			Where("user_id = ? AND actor_id = ?", request.UserId, request.ActorId).
			Count(&count)
		if row.Error != nil && !errors.Is(row.Error, gorm.ErrRecordNotFound) {
			return "false", row.Error
		}
		return strconv.FormatInt(count, 10), nil
	})

	if err != nil {
		logger.WithFields(logrus.Fields{
			"err":     err,
			"ActorId": request.ActorId,
			"UserId":  request.UserId,
		}).Errorf("IsFollowService failed")
		logging.SetSpanError(span, err)

		resp = &relation.IsFollowResponse{
			StatusCode: strings.RelationServiceIntErrorCode,
			StatusMsg:  strings.RelationServiceIntError,
			Result:     false,
		}
		return
	}

	resp = &relation.IsFollowResponse{
		StatusCode: strings.ServiceOKCode,
		StatusMsg:  strings.ServiceOK,
		Result:     res != "0",
	}
	return
}

func (r RelationServiceImpl) GetFollowList(ctx context.Context, request *relation.FollowListRequest) (resp *relation.FollowListResponse, err error) {
	ctx, span := tracing.Tracer.Start(ctx, "GetFollowListService")
	defer span.End()
	logging.SetSpanWithHostname(span)
	logger := logging.LogService("RelationService.GetFollowList").WithContext(ctx)

	ok, err := isUserExist(ctx, request.ActorId, request.UserId, span, logger)
	if err != nil || !ok {
		resp = &relation.FollowListResponse{
			StatusCode: strings.UnableToQueryUserErrorCode,
			StatusMsg:  strings.UnableToQueryUserError,
		}
		return
	}

	cacheKey := config.EnvCfg.RedisPrefix + fmt.Sprintf("follow_list_%d", request.UserId)
	followIdList, err := redis2.Client.SMembers(ctx, cacheKey).Result()
	followIdListInt := make([]uint32, 0, len(followIdList))
	var followList []models.Relation

	if err != nil {
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

		for index, rel := range followList {
			redis2.Client.SAdd(ctx, cacheKey, rel.UserId)
			followIdListInt[index] = rel.UserId
		}
	} else {
		followIdListInt, err = string2Int(followIdList, logger, span)
		if err != nil {
			logger.WithFields(logrus.Fields{
				"err": err,
			}).Errorf("failed to convert string to int")
			logging.SetSpanError(span, err)
			resp = &relation.FollowListResponse{
				StatusCode: strings.UnableToGetFollowListErrorCode,
				StatusMsg:  strings.UnableToGetFollowListError,
			}
			return
		}
	}

	rFollowList, err := r.idList2UserList(ctx, followIdListInt, request.ActorId, logger, span)
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
	logging.SetSpanWithHostname(span)
	logger := logging.LogService("RelationService.GetFollowerList").WithContext(ctx)

	ok, err := isUserExist(ctx, request.ActorId, request.UserId, span, logger)
	if err != nil || !ok {
		resp = &relation.FollowerListResponse{
			StatusCode: strings.UnableToQueryUserErrorCode,
			StatusMsg:  strings.UnableToQueryUserError,
		}
		return
	}

	cacheKey := config.EnvCfg.RedisPrefix + fmt.Sprintf("follower_list_%d", request.UserId)
	followerIdList, err := redis2.Client.SMembers(ctx, cacheKey).Result()
	followerIdListInt := make([]uint32, 0, len(followerIdList))
	var followerList []models.Relation

	if err != nil {
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

		for index, rel := range followerList {
			redis2.Client.SAdd(ctx, cacheKey, rel.UserId)
			followerIdListInt[index] = rel.UserId
		}
	} else {
		followerIdListInt, err = string2Int(followerIdList, logger, span)
		if err != nil {
			logger.WithFields(logrus.Fields{
				"err": err,
			}).Errorf("failed to convert string to int")
			logging.SetSpanError(span, err)
			resp = &relation.FollowerListResponse{
				StatusCode: strings.UnableToGetFollowerListErrorCode,
				StatusMsg:  strings.UnableToGetFollowerListError,
			}
			return
		}
	}

	rFollowerList, err := r.idList2UserList(ctx, followerIdListInt, request.ActorId, logger, span)
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

func (r RelationServiceImpl) idList2UserList(ctx context.Context, idList []uint32, actorID uint32, logger *logrus.Entry, span trace.Span) ([]*user.User, error) {

	var wg sync.WaitGroup
	var mu sync.Mutex
	var wgErrors []error
	var err error

	maxRetries := 3
	retryInterval := 1

	rUserList := make([]*user.User, 0, len(idList))

	for _, id := range idList {
		wg.Add(1)
		go func(id uint32) {
			defer wg.Done()

			retryCount := 0
			for retryCount < maxRetries {
				userResponse, err := userClient.GetUserInfo(ctx, &user.UserRequest{
					UserId:  id,
					ActorId: actorID,
				})

				if err != nil || userResponse.StatusCode != strings.ServiceOKCode {
					logger.WithFields(logrus.Fields{
						"err":    err,
						"userId": id,
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
		}(id)
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

func string2Int(s []string, logger *logrus.Entry, span trace.Span) (i []uint32, err error) {

	i = make([]uint32, len(s))

	for index, v := range s {
		var idInt int
		idInt, err = strconv.Atoi(v)
		if err != nil {
			logger.WithFields(logrus.Fields{
				"err": err,
			}).Errorf("failed to convert string to int")
			logging.SetSpanError(span, err)
			return
		}
		i[index] = uint32(idInt)
	}
	return
}

// followOp = true  ->  follow
// followOp = false ->  unfollow
func updateFollowListCache(ctx context.Context, actorID uint32, relation models.Relation, followOp bool, span trace.Span, logger *logrus.Entry) (err error) {

	cacheKey := config.EnvCfg.RedisPrefix + fmt.Sprintf("follow_list_%d", actorID)

	if followOp {
		_, err = redis2.Client.SAdd(ctx, cacheKey, relation.UserId).Result()
	} else {
		_, err = redis2.Client.SRem(ctx, cacheKey, relation.UserId).Result()
	}
	if err != nil {
		logger.WithFields(logrus.Fields{
			"err": err,
		}).Errorf("update FollowList redis failed")
		logging.SetSpanError(span, err)
	}
	return
}

func updateFollowerListCache(ctx context.Context, userID uint32, relation models.Relation, followOp bool, span trace.Span, logger *logrus.Entry) (err error) {
	cacheKey := config.EnvCfg.RedisPrefix + fmt.Sprintf("follower_list_%d", userID)

	if followOp {
		_, err = redis2.Client.SAdd(ctx, cacheKey, relation.ActorId).Result()

	} else {
		_, err = redis2.Client.SRem(ctx, cacheKey, relation.ActorId).Result()
	}
	if err != nil {
		logger.WithFields(logrus.Fields{
			"err": err,
		}).Errorf("update FollowerList redis failed")
		logging.SetSpanError(span, err)
	}
	return
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

		if !followOp {
			// unfollow
			if dbCount > 0 {
				dbCount = dbCount - 1
			} else {
				dbCount = 0
			}
		} else {
			// follow
			dbCount = dbCount + 1
		}

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
		if !followOp {
			// unfollow
			if dbCount > 0 {
				dbCount = dbCount - 1
			} else {
				dbCount = 0
			}
		} else {
			// follow
			dbCount = dbCount + 1
		}

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

func isUserExist(ctx context.Context, actorID uint32, userID uint32, span trace.Span, logger *logrus.Entry) (ok bool, err error) {

	userExist, err := userClient.GetUserExistInformation(ctx, &user.UserExistRequest{UserId: actorID})

	if err != nil || userExist.StatusCode != strings.ServiceOKCode {
		logger.WithFields(logrus.Fields{
			"err":    err,
			"UserId": actorID,
		}).Errorf("not find the user:%v", actorID)
		logging.SetSpanError(span, err)
		ok = false
		return
	}

	if !userExist.Existed {
		ok = false
		return
	}
	userExist, err = userClient.GetUserExistInformation(ctx, &user.UserExistRequest{UserId: userID})

	if err != nil || userExist.StatusCode != strings.ServiceOKCode {
		logger.WithFields(logrus.Fields{
			"err":    err,
			"UserId": userID,
		}).Errorf("not find the user:%v", userID)
		logging.SetSpanError(span, err)
		ok = false
		return
	}

	if !userExist.Existed {
		ok = false
		return
	}

	ok = true
	return
}
