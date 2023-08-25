package models

import (
	"GuGoTik/src/rpc/feed"
)

type ListVideosReq struct {
	LatestTime string `form:"latest_time"`
	ActorId    int    `form:"actor_id"`
}

type ListVideosRes struct {
	StatusCode int           `json:"status_code"`
	StatusMsg  string        `json:"status_msg"`
	NextTime   *int64        `json:"next_time,omitempty"`
	VideoList  []*feed.Video `json:"video_list,omitempty"`
}
