package cached

import (
	"GuGoTik/src/constant/config"
	"GuGoTik/src/extra/tracing"
	"GuGoTik/src/storage/database"
	"GuGoTik/src/storage/redis"
	"GuGoTik/src/utils/logging"
	"context"
	"github.com/patrickmn/go-cache"
	redis2 "github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"math/rand"
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
	IsDirty() bool
}

// ScanGet 采用二级缓存(Memory-Redis)的模式读取结构体类型，并且填充到传入的结构体中，结构体需要实现IDGetter且确保ID可用。
func ScanGet(ctx context.Context, key string, obj interface{}) (bool, error) {
	ctx, span := tracing.Tracer.Start(ctx, "Cached-GetFromScanCache")
	defer span.End()
	logging.SetSpanWithHostname(span)
	logger := logging.LogService("Cached.GetFromScanCache").WithContext(ctx)
	key = config.EnvCfg.RedisPrefix + key

	c := getOrCreateCache(key)
	wrappedObj := obj.(cachedItem)
	key = key + strconv.FormatUint(uint64(wrappedObj.GetID()), 10)
	if x, found := c.Get(key); found {
		dstVal := reflect.ValueOf(obj)
		dstVal.Elem().Set(x.(reflect.Value))
		return true, nil
	}

	//缓存没有命中，Fallback 到 Redis
	logger.WithFields(logrus.Fields{
		"key": key,
	}).Infof("Missed local memory cached")

	if err := redis.Client.HGetAll(ctx, key).Scan(obj); err != nil {
		if err != redis2.Nil {
			logger.WithFields(logrus.Fields{
				"err": err,
				"key": key,
			}).Errorf("Redis error when find struct")
			logging.SetSpanError(span, err)
			return false, err
		}
	}

	// 如果 Redis 命中，那么就存到 localCached 然后返回
	if wrappedObj.IsDirty() {
		logger.WithFields(logrus.Fields{
			"key": key,
		}).Infof("Redis hit the key")
		c.Set(key, reflect.ValueOf(obj).Elem(), cache.DefaultExpiration)
		return true, nil
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
		return false, result.Error
	}

	if result := redis.Client.HSet(ctx, key, obj); result.Err() != nil {
		logger.WithFields(logrus.Fields{
			"err": result.Err(),
			"key": key,
		}).Errorf("Redis error when set struct info")
		logging.SetSpanError(span, result.Err())
		return false, nil
	}

	c.Set(key, reflect.ValueOf(obj).Elem(), cache.DefaultExpiration)
	return true, nil
}

// ScanTagDelete 将缓存值标记为删除，下次从 cache 读取时会 FallBack 到数据库。
func ScanTagDelete(ctx context.Context, key string, obj interface{}) {
	ctx, span := tracing.Tracer.Start(ctx, "Cached-ScanTagDelete")
	defer span.End()
	logging.SetSpanWithHostname(span)
	key = config.EnvCfg.RedisPrefix + key

	redis.Client.HDel(ctx, key)

	c := getOrCreateCache(key)
	wrappedObj := obj.(cachedItem)
	key = key + strconv.FormatUint(uint64(wrappedObj.GetID()), 10)
	c.Delete(key)
}

// ScanWriteCache 写入缓存，如果 state 为 false 那么只会写入 localCached
func ScanWriteCache(ctx context.Context, key string, obj interface{}, state bool) (err error) {
	ctx, span := tracing.Tracer.Start(ctx, "Cached-ScanWriteCache")
	defer span.End()
	logging.SetSpanWithHostname(span)
	logger := logging.LogService("Cached.ScanWriteCache").WithContext(ctx)
	key = config.EnvCfg.RedisPrefix + key

	wrappedObj := obj.(cachedItem)
	key = key + strconv.FormatUint(uint64(wrappedObj.GetID()), 10)
	c := getOrCreateCache(key)
	c.Set(key, reflect.ValueOf(obj).Elem(), cache.DefaultExpiration)

	if state {
		if err = redis.Client.HGetAll(ctx, key).Scan(obj); err != nil {
			logger.WithFields(logrus.Fields{
				"err": err,
				"key": key,
			}).Errorf("Redis error when find struct info")
			logging.SetSpanError(span, err)
			return
		}
	}

	return
}

// Get 读取字符串缓存, 其中找到了返回 True，没找到返回 False，异常也返回 False
func Get(ctx context.Context, key string) (string, bool, error) {
	ctx, span := tracing.Tracer.Start(ctx, "Cached-GetFromStringCache")
	defer span.End()
	logging.SetSpanWithHostname(span)
	logger := logging.LogService("Cached.GetFromStringCache").WithContext(ctx)
	key = config.EnvCfg.RedisPrefix + key

	c := getOrCreateCache("strings")
	if x, found := c.Get(key); found {
		return x.(string), true, nil
	}

	//缓存没有命中，Fallback 到 Redis
	logger.WithFields(logrus.Fields{
		"key": key,
	}).Infof("Missed local memory cached")

	var result *redis2.StringCmd
	if result = redis.Client.Get(ctx, key); result.Err() != nil && result.Err() != redis2.Nil {
		logger.WithFields(logrus.Fields{
			"err":    result.Err(),
			"string": key,
		}).Errorf("Redis error when find string")
		logging.SetSpanError(span, result.Err())
		return "", false, nil
	}

	value, err := result.Result()

	switch {
	case err == redis2.Nil:
		return "", false, nil
	case err != nil:
		logger.WithFields(logrus.Fields{
			"err": err,
		}).Errorf("Err when write Redis")
		logging.SetSpanError(span, err)
		return "", false, err
	default:
		c.Set(key, value, cache.DefaultExpiration)
		return value, true, nil
	}
}

// GetWithFunc 从缓存中获取字符串，如果不存在调用 Func 函数获取
func GetWithFunc(ctx context.Context, key string, f func(ctx context.Context, key string) (string, error)) (string, error) {
	ctx, span := tracing.Tracer.Start(ctx, "Cached-GetFromStringCacheWithFunc")
	defer span.End()
	logging.SetSpanWithHostname(span)
	value, ok, err := Get(ctx, key)

	if err != nil {
		return "", err
	}

	if ok {
		return value, nil
	}

	// 如果不存在，那么就获取它
	value, err = f(ctx, key)

	if err != nil {
		return "", err
	}

	Write(ctx, key, value, true)
	return value, nil
}

// Write 写入字符串缓存，如果 state 为 false 则只写入 Local Memory
func Write(ctx context.Context, key string, value string, state bool) {
	ctx, span := tracing.Tracer.Start(ctx, "Cached-SetStringCache")
	defer span.End()
	logging.SetSpanWithHostname(span)
	key = config.EnvCfg.RedisPrefix + key

	c := getOrCreateCache("strings")
	c.Set(key, value, cache.DefaultExpiration)

	if state {
		redis.Client.Set(ctx, key, value, 120*time.Hour+time.Duration(rand.Intn(redisRandomScope))*time.Second)
	}
}

// TagDelete 删除字符串缓存
func TagDelete(ctx context.Context, key string) {
	ctx, span := tracing.Tracer.Start(ctx, "Cached-DeleteStringCache")
	defer span.End()
	logging.SetSpanWithHostname(span)
	key = config.EnvCfg.RedisPrefix + key

	c := getOrCreateCache("strings")
	c.Delete(key)

	redis.Client.Del(ctx, key)
}

func getOrCreateCache(name string) *cache.Cache {
	cc, ok := cacheMaps[name]
	if !ok {
		m.Lock()
		defer m.Unlock()
		cc, ok := cacheMaps[name]
		if !ok {
			cc = cache.New(5*time.Minute, 10*time.Minute)
			cacheMaps[name] = cc
			return cc
		}
		return cc
	}
	return cc
}

// CacheAndRedisGet 从内存缓存和 Redis 缓存中读取数据
func CacheAndRedisGet(ctx context.Context, key string, obj interface{}) (bool, error) {
	ctx, span := tracing.Tracer.Start(ctx, "CacheAndRedisGet")
	defer span.End()
	logging.SetSpanWithHostname(span)
	logger := logging.LogService("CacheAndRedisGet").WithContext(ctx)
	key = config.EnvCfg.RedisPrefix + key

	c := getOrCreateCache(key)
	wrappedObj := obj.(cachedItem)
	key = key + strconv.FormatUint(uint64(wrappedObj.GetID()), 10)
	if x, found := c.Get(key); found {
		dstVal := reflect.ValueOf(obj)
		dstVal.Elem().Set(x.(reflect.Value))
		return true, nil
	}

	// 缓存没有命中，Fallback 到 Redis
	logger.WithFields(logrus.Fields{
		"key": key,
	}).Infof("Missed local memory cached")

	if err := redis.Client.HGetAll(ctx, key).Scan(obj); err != nil {
		logger.WithFields(logrus.Fields{
			"err": err,
			"key": key,
		}).Errorf("Redis error when find struct")
		logging.SetSpanError(span, err)
		return false, err
	}

	// 如果 Redis 命中，那么就存到 localCached 然后返回
	if wrappedObj.IsDirty() {
		logger.WithFields(logrus.Fields{
			"key": key,
		}).Infof("Redis hit the key")
		c.Set(key, reflect.ValueOf(obj).Elem(), cache.DefaultExpiration)
		return true, nil
	}

	logger.WithFields(logrus.Fields{
		"key": key,
	}).Warnf("Missed Redis Cached")

	return false, nil
}

func ActionRedisSync(time time.Duration, f func(client redis2.UniversalClient) error) {
	go func() {
		daemon := NewTick(time, f)
		daemon.Start()
	}()
}
