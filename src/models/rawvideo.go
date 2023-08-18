package models

import "GuGoTik/src/storage/database"

type RawVideo struct {
	ActorId   uint32
	Title     string
	FilePath  string
	CoverPath string
}

func init() {
	if err := database.Client.AutoMigrate(&RawVideo{}); err != nil {
		panic(err)
	}
}
