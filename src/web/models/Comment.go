package models

import "GuGoTik/src/rpc/comment"

type ActionCommentReq struct {
	Token       string `form:"token" binding:"required"`
	ActorId     int    `form:"actor_id"`
	VideoId     int    `form:"video_id" binding:"-"`
	ActionType  int    `form:"action_type" binding:"required"`
	CommentText string `form:"comment_text"`
	CommentId   int    `form:"comment_id"`
}

type ActionCommentRes struct {
	StatusCode int             `json:"status_code"`
	StatusMsg  string          `json:"status_msg"`
	Comment    comment.Comment `json:"comment"`
}

type ListCommentReq struct {
	Token   string `form:"token"`
	ActorId int    `form:"actor_id"`
	VideoId int    `form:"video_id" binding:"-"`
}

type ListCommentRes struct {
	StatusCode  int                `json:"status_code"`
	StatusMsg   string             `json:"status_msg"`
	CommentList []*comment.Comment `json:"comment_list"`
}

type CountCommentReq struct {
	Token   string `form:"token"`
	ActorId int    `form:"actor_id"`
	VideoId int    `form:"video_id" binding:"-"`
}

type CountCommentRes struct {
	StatusCode   int    `json:"status_code"`
	StatusMsg    string `json:"status_msg"`
	CommentCount int    `json:"comment_count"`
}
