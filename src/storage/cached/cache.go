package cached

import (
	"GuGoTik/src/extra/tracing"
	"GuGoTik/src/storage/database"
	"GuGoTik/src/storage/redis"
	"GuGoTik/src/utils/logging"
	"context"
	"github.com/patrickmn/go-cache"
	"github.com/sirupsen/logrus"
	"reflect"
	"strconv"
	"sync"
	"time"
)

// 表示 Redis 随机缓存的时间范围
const redisRandomScope = 1

var cacheMaps = make(map[string]*cache.Cache)

var m = new(sync.Mutex)

type cachedItem interface {
	GetID() uint32
	IsCompleted() bool
}

// ScanGet 采用二级缓存(Memory-Redis)的模式读取结构体类型，并且填充到传入的结构体中，结构体需要实现IDGetter且确保ID可用。
func ScanGet(ctx context.Context, key string, obj interface{}) {
	ctx, span := tracing.Tracer.Start(ctx, "Cached-GetFromCache")
	defer span.End()
	logger := logging.LogService("Cached.GetFromCache").WithContext(ctx)

	c := getOrCreateCache(key)
	wrappedObj := obj.(cachedItem)
	key = key + strconv.FormatUint(uint64(wrappedObj.GetID()), 10)
	if x, found := c.Get(key); found {
		dstVal := reflect.ValueOf(obj)
		dstVal.Elem().Set(x.(reflect.Value))
		return
	}

	//缓存没有命中，Fallback 到 Redis
	logger.WithFields(logrus.Fields{
		"key": key,
	}).Infof("Missed local memory cached")

	if err := redis.Client.HGetAll(ctx, key).Scan(obj); err != nil {
		logger.WithFields(logrus.Fields{
			"err": err,
		}).Errorf("Redis error when find user info")
		logging.SetSpanError(span, err)
	}

	// 如果 Redis 命中，那么就存到 localCached 然后返回
	if wrappedObj.IsCompleted() {
		logger.WithFields(logrus.Fields{
			"key": key,
		}).Infof("Redis hit the key")
		c.Set(key, reflect.ValueOf(obj).Elem(), cache.DefaultExpiration)
		return
	}

	//缓存没有命中，Fallback 到 DB
	logger.WithFields(logrus.Fields{
		"key": key,
	}).Warnf("Missed Redis Cached")

	result := database.Client.WithContext(ctx).Find(obj)
	if result.RowsAffected == 0 {
		logger.WithFields(logrus.Fields{
			"key": key,
		}).Warnf("Missed DB obj, seems wrong key")
		return
	}

	if err := redis.Client.HSet(ctx, key, obj); err != nil {
		logger.WithFields(logrus.Fields{
			"err": err,
		}).Errorf("Redis error when set user info")
		logging.SetSpanError(span, err.Err())
	}

	c.Set(key, reflect.ValueOf(obj).Elem(), cache.DefaultExpiration)
	return
}

// TagDelete 将缓存值标记为删除，下次从 cache 读取时会 FallBack 到数据库。
func TagDelete(ctx context.Context, key string, obj interface{}) {
	ctx, span := tracing.Tracer.Start(ctx, "Cached-TagDelete")
	defer span.End()
	redis.Client.HDel(ctx, key)

	c := getOrCreateCache(key)
	wrappedObj := obj.(cachedItem)
	key = key + strconv.FormatUint(uint64(wrappedObj.GetID()), 10)
	c.Delete(key)
}

// WriteCache 写入缓存，如果 state 为 false 那么只会写入 localCached
func WriteCache(ctx context.Context, key string, obj interface{}, state bool) (err error) {
	ctx, span := tracing.Tracer.Start(ctx, "Cached-WriteCache")
	defer span.End()
	logger := logging.LogService("Cached.WriteCache").WithContext(ctx)

	wrappedObj := obj.(cachedItem)
	key = key + strconv.FormatUint(uint64(wrappedObj.GetID()), 10)
	c := getOrCreateCache(key)
	c.Set(key, reflect.ValueOf(obj).Elem(), cache.DefaultExpiration)

	if state {
		if err = redis.Client.HGetAll(ctx, key).Scan(obj); err != nil {
			logger.WithFields(logrus.Fields{
				"err": err,
			}).Errorf("Redis error when find user info")
			logging.SetSpanError(span, err)
		}
	}

	return
}

func getOrCreateCache(name string) *cache.Cache {
	cc, ok := cacheMaps[name]
	if !ok {
		m.Lock()
		defer m.Unlock()
		cc = cache.New(5*time.Minute, 10*time.Minute)
		cacheMaps[name] = cc
		return cc
	}
	return cc
}
