package models

import (
	"GuGoTik/src/storage/database"
	"gorm.io/gorm"
)

type Action struct {
	Type         uint   // 用户操作的行为类型，如：1表示点赞相关
	Name         string // 用户操作的动作名称，如：FavoriteNameActionLog 表示点赞相关操作
	SubName      string // 用户操作动作的子名称，如：FavoriteUpActionLog 表示给视频增加赞操作
	ServiceName  string // 服务来源，添加服务的名称，如 FavoriteService
	Attached     string // 附带信息，当 Name - SubName 无法说明时，添加一个额外的信息
	ActorId      uint32 // 操作者 Id
	VideoId      uint32 // 附属的视频 Id，没有填写为0
	AffectUserId uint32 // 操作的用户 Id，如：被关注的用户 Id
	AffectAction uint   // 操作的类型，如：1. 自增/自减某个数据，2. 直接修改某个数据
	AffectedData string // 操作的数值是什么，如果是自增，填 1，如果是修改为某个数据，那么填这个数据的值
	EventId      string // 如果这个操作是一个大操作的子类型，那么需要具有相同的 UUID
	TraceId      string // 这个操作的 TraceId
	SpanId       string // 这个操作的 SpanId
	gorm.Model          //数据库模型
}

func init() {
	if err := database.Client.AutoMigrate(&Action{}); err != nil {
		panic(err)
	}
}
