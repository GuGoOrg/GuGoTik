package models

import (
	"GuGoTik/src/constant/config"
	"GuGoTik/src/storage/database"
	"GuGoTik/src/utils/logging"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"regexp"
)

type User struct {
	ID              uint32 `gorm:"not null;primarykey;autoIncrement"`               //用户 Id
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

func (u *User) IsDirty() bool {
	return u.UserName != ""
}

func (u *User) GetID() uint32 {
	return u.ID
}

func init() {
	if err := database.Client.AutoMigrate(&User{}); err != nil {
		panic(err)
	}

	// Create magic user: show video summary and keywords, and act as ChatGPT
	magicUser := User{
		UserName:        "ChatGPT",
		Password:        "chatgpt",
		Role:            2,
		Avatar:          "https://maples31-blog.oss-cn-beijing.aliyuncs.com/img/ChatGPT_logo.svg.png",
		BackgroundImage: "https://maples31-blog.oss-cn-beijing.aliyuncs.com/img/ChatGPT.jpg",
		Signature:       "GuGoTik 小助手",
	}
	result := database.Client.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_name"}},
		DoUpdates: clause.AssignmentColumns([]string{"password", "role", "avatar", "background_image", "signature"}),
	}).Create(&magicUser)

	if result.Error != nil {
		logging.Logger.Errorf("Cannot create magic user because of %s", result.Error)
	}

	config.EnvCfg.MagicUserId = magicUser.ID
	logging.Logger.WithFields(logrus.Fields{
		"MagicUserId": magicUser.ID,
	}).Infof("Successfully create the magic user")
}
