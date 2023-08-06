package models

import (
	"GuGoTik/src/extra/tracing"
	"GuGoTik/src/storage/database"
	"GuGoTik/src/storage/redis"
	"GuGoTik/src/utils/logging"
	"context"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"regexp"
	"strconv"
)

type User struct {
	UserName        string `gorm:"not null;unique;size: 32;index" redis:"UserName"` // 用户名
	Password        string `gorm:"not null" redis:"Password"`                       // 密码
	Role            int    `gorm:"default:1" redis:"Role"`                          // 角色
	Avatar          string `redis:"Avatar"`                                         // 头像
	BackgroundImage string `redis:"BackGroundImage"`                                // 背景图片
	Signature       string `redis:"Signature"`                                      // 个人简介
	gorm.Model
}

// IsNameEmail 判断用户的名称是否为邮箱。
func (u *User) IsNameEmail() bool {
	pattern := `\w+([-+.]\w+)*@\w+([-.]\w+)*\.\w+([-.]\w+)*`
	reg := regexp.MustCompile(pattern)
	return reg.MatchString(u.UserName)
}

// FillFromRedis 利用实例的ID字段尝试从Redis中获取用户的完整信息，如果失败会查询数据库并写入缓存以方便下次查询。
func (u *User) FillFromRedis(ctx context.Context) error {
	ctx, span := tracing.Tracer.Start(ctx, "Redis-UserInfoGet")
	defer span.End()
	logger := logging.LogService("Redis.UserInfoGet").WithContext(ctx)
	if err := redis.Client.HGetAll(ctx, "UserInfo"+strconv.Itoa(int(u.ID))).Scan(u); err != nil {
		logger.WithFields(logrus.Fields{
			"err": err,
		}).Errorf("Redis error when find user info")
		logging.SetSpanError(span, err)
		return err
	}
	if u.UserName == "" {
		result := database.Client.WithContext(ctx).Find(u)
		if result.RowsAffected == 0 {
			return nil
		}
		if err := redis.Client.HSet(ctx, "UserInfo"+strconv.Itoa(int(u.ID)), u); err != nil {
			logger.WithFields(logrus.Fields{
				"err": err,
			}).Errorf("Redis error when find user info")
			logging.SetSpanError(span, err.Err())
			return err.Err()
		}
	}
	return nil
}

func init() {
	if err := database.Client.AutoMigrate(&User{}); err != nil {
		panic(err)
	}
}
