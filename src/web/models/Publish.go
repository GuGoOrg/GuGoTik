package models

import "GuGoTik/src/rpc/feed"

type ListPublishReq struct {
	ActorId uint32 `form:"actor_id" binding:"required"`
	UserId  uint32 `form:"user_id" binding:"required"`
}

type ListPublishRes struct {
	StatusCode int           `json:"status_code"`
	StatusMsg  string        `json:"status_msg"`
	VideoList  []*feed.Video `json:"vide_list"`
}
