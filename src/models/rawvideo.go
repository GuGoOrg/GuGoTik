package models

import (
	"GuGoTik/src/storage/database"
	"gorm.io/gorm"
)

type RawVideo struct {
	ActorId   uint32
	VideoId   uint32 `gorm:"not null;primaryKey"`
	Title     string
	FileName  string
	CoverName string
	gorm.Model
}

func init() {
	if err := database.Client.AutoMigrate(&RawVideo{}); err != nil {
		panic(err)
	}
}
