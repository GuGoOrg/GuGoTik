package models

import (
	"GuGoTik/src/storage/database"
	"gorm.io/gorm"
)

// Video 视频表
type Video struct {
	ID            uint32 `gorm:"not null;primaryKey;"`
	UserId        uint32 `json:"user_id" gorm:"not null;"`
	Title         string `json:"title" gorm:"not null;"`
	FileName      string `json:"play_name" gorm:"not null;"`
	CoverName     string `json:"cover_name" gorm:"not null;"`
	AudioFileName string
	Transcript    string
	Summary       string
	Keywords      string // e.g., "keywords1 | keywords2 | keywords3"
	gorm.Model
}

func init() {
	if err := database.Client.AutoMigrate(&Video{}); err != nil {
		panic(err)
	}
}
