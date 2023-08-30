package models

import (
	"GuGoTik/src/rpc/chat"
)

// SMessageReq 这个是发数据的数据结构
type SMessageReq struct {
	ActorId    int    `form:"actor_id" binding:"required"`
	ToUserId   int    `form:"to_user_id" binding:"required"`
	Content    string `form:"content" binding:"required"`
	ActionType int    `form:"action_type" binding:"required"` // send message
	//Create_time string //time maybe have some question
}

// SMessageRes 收的状态
// status_code 状态码 0- 成功  其他值 -失败
// status_msg  返回状态描述
type SMessageRes struct {
	StatusCode int    `json:"status_code"`
	StatusMsg  string `json:"status_msg"`
}

type ListMessageReq struct {
	ActorId    uint32 `form:"actor_id" binding:"required"`
	ToUserId   uint32 `form:"to_user_id" binding:"required"`
	PreMsgTime uint64 `form:"pre_msg_time"`
}

type ListMessageRes struct {
	StatusCode  int             `json:"status_code"`
	StatusMsg   string          `json:"status_msg"`
	MessageList []*chat.Message `json:"message_list"`
}
