package models

import (
	"GuGoTik/src/rpc/chat"
)

// 这个是发数据的数据结构
type SMessageReq struct {
	Token       string `form:"token" binding:"required"`
	ActorId     int    `form:"actor_id"`
	User_id     int    `form:"user_id"`
	Content     string `form:"content"`
	Action_type int    `form:"action_type"` // send message
	//Create_time string //time maybe have some question
}

// 收的状态
// statuc code 状态码 0- 成功  其他值 -失败
// status_msg  返回状态描述
type SMessageRes struct {
	Status_code int    `json:"status_code"`
	Status_msg  string `json:"status_msg"`
}

type ListMessageReq struct {
	Token   string `form:"token" binding:"required"`
	ActorId uint32 `form:"actor_id"`
	UserId  uint32 `from:"user_id"`

	PreMsgTime uint32 `from:"preMsgTime"`
}

type ListMessageRes struct {
	StatusCode  int             `json:"status_code"`
	StatusMsg   string          `json:"status_msg"`
	MessageList []*chat.Message `json:"message_list"`
}
