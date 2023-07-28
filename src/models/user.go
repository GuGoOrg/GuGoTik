package models

import (
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

func (u *User) isEmail() bool {
	pattern := `\w+([-+.]\w+)*@\w+([-.]\w+)*\.\w+([-.]\w+)*`
	reg := regexp.MustCompile(pattern)
	return reg.MatchString(u.UserName)
}
