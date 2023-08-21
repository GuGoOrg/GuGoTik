package models

import "GuGoTik/src/storage/database"

type RawVideo struct {
	ActorId   uint32 `gorm:"not null;primarykey;"`
	VideoId   uint32 `json:"video_id" column:"video_id" gorm:"not null;"` // 视频 ID
	Title     string `json:"title" gorm:"not null;"`
	FileName  string `json:"play_name" gorm:"not null;"`
	CoverName string `json:"cover_name" gorm:"not null;"`
}

func init() {
	if err := database.Client.AutoMigrate(&RawVideo{}); err != nil {
		panic(err)
	}
}
