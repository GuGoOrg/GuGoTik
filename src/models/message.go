package models

import (
	"GuGoTik/src/storage/database"
	"time"

	"gorm.io/gorm"
)

type Message struct {
	ID             uint32 `gorm:"not null;primarykey;autoIncrement"`
	ToUserId       uint32 `gorm:"not null"`
	FromUserId     uint32 `gorm:"not null"`
	ConversationId string `gorm:"not null" index:"conversationid"`
	Content        string `gorm:"not null"`

	// Create_time  time.Time `gorm:"not null"`
	//Updatetime deleteTime
	gorm.Model
}

// es 使用
type EsMessage struct {
	ToUserId       uint32    `json:"toUserid"`
	FromUserId     uint32    `json:"fromUserId"`
	ConversationId string    `json:"conversationId"`
	Content        string    `json:"content"`
	CreateTime     time.Time `json:"createTime"`
}

func init() {
	if err := database.Client.AutoMigrate(&Message{}); err != nil {
		panic(err)
	}
}
