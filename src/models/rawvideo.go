package models

import "GuGoTik/src/storage/database"

type RawVideo struct {
	ActorId   uint32
	VideoId   uint32
	Title     string
	FileName  string
	CoverName string
}

func init() {
	if err := database.Client.AutoMigrate(&RawVideo{}); err != nil {
		panic(err)
	}
}
