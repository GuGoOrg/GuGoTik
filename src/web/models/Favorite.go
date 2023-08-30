package models

import (
	"GuGoTik/src/rpc/feed"
)

type ActionFavoriteReq struct {
	Token      string `form:"token" binding:"required"`
	ActorId    int    `form:"actor_id" binding:"required"`
	VideoId    int    `form:"video_id" binding:"required"`
	ActionType int    `form:"action_type" binding:"required"`
}

type ActionFavoriteRes struct {
	StatusCode int    `json:"status_code"`
	StatusMsg  string `json:"status_msg"`
}

type ListFavoriteReq struct {
	Token   string `form:"token"`
	ActorId int    `form:"actor_id"`
	UserId  int    `form:"user_id" binding:"required"`
}

type ListFavoriteRes struct {
	StatusCode int           `json:"status_code"`
	StatusMsg  string        `json:"status_msg"`
	VideoList  []*feed.Video `json:"video_list"`
}
