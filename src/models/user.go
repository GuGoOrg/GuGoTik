package models

import (
	"GuGoTik/src/utils/logging"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"regexp"
)

type User struct {
	UserName        string `gorm:"not null;unique;size: 32;index"` // 用户名
	Password        string `gorm:"not null"`                       // 密码
	Role            int    `gorm:"default:1"`                      // 角色
	Avatar          string // 头像
	BackgroundImage string // 背景图片
	Signature       string // 个人简介
	gorm.Model
}

func (u *User) IsNameEmail() bool {
	pattern := `\w+([-+.]\w+)*@\w+([-.]\w+)*\.\w+([-.]\w+)*`
	reg := regexp.MustCompile(pattern)
	return reg.MatchString(u.UserName)
}

func (u *User) BeforeFind(tx *gorm.DB) (err error) {
	span, ctx := opentracing.StartSpanFromContext(tx.Statement.Context, "DB-Find")
	tx.Statement.Context = opentracing.ContextWithSpan(ctx, span)
	return nil
}

func (u *User) BeforeCreate(tx *gorm.DB) (err error) {
	span, ctx := opentracing.StartSpanFromContext(tx.Statement.Context, "DB-Create")
	tx.Statement.Context = opentracing.ContextWithSpan(ctx, span)
	return nil
}

func (u *User) AfterCreate(tx *gorm.DB) (err error) {
	span := opentracing.SpanFromContext(tx.Statement.Context)
	defer span.Finish()
	if tx.Error != nil {
		logging.SetSpanError(span, err)
		logging.GetSpanLogger(span, "DB.Create").WithFields(logrus.Fields{
			"err": err,
		}).Warnf("DB Create meet trouble")
	}
	return nil
}

func (u *User) AfterFindMust(tx *gorm.DB) (err error) {
	span := opentracing.SpanFromContext(tx.Statement.Context)
	defer span.Finish()
	if tx.Error != nil {
		logging.SetSpanError(span, err)
		logging.GetSpanLogger(span, "DB.Find").WithFields(logrus.Fields{
			"err": err,
		}).Warnf("DB Find meet trouble")
	}
	return nil
}
