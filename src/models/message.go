package models

import (
	"GuGoTik/src/storage/database"

	"gorm.io/gorm"
)

type Message struct {
	ID             uint32 `gorm:"not null;primarykey;autoIncrement"`
	ToUserId       uint32 `gorm:"not null" `
	FromUserId     uint32 `gorm:"not null"`
	ConversationId string `gorm:"not null" index:"conversationid"`
	Content        string `gorm:"not null"`

	// Create_time  time.Time `gorm:"not null"`
	//Updatetime deleteTime
	gorm.Model
}

func init() {
	if err := database.Client.AutoMigrate(&Message{}); err != nil {
		panic(err)
	}
}
