package models

import "GuGoTik/src/rpc/user"

//
//type RelationActionReq struct {
//	Token   string `form:"token" binding:"required"`
//	ActorId int    `form:"actor_id"`
//	UserId  int    `form:"user_id"`
//}

type RelationActionReq struct {
	Token      string `form:"token" binding:"required"`
	ActorId    int    `form:"actor_id"`
	UserId     int    `form:"to_user_id"`
	ActionType int    `form:"action_type" binding:"required"`
}

type RelationActionRes struct {
	StatusCode int    `json:"status_code"`
	StatusMsg  string `json:"status_msg"`
}

type FollowListReq struct {
	Token   string `form:"token"`
	ActorId int    `form:"actor_id"`
	UserId  int    `form:"user_id"`
}

type FollowListRes struct {
	StatusCode int          `json:"status_code"`
	StatusMsg  string       `json:"status_msg"`
	UserList   []*user.User `json:"user_list"`
}

type CountFollowListReq struct {
	Token  string `form:"token" binding:"required"`
	UserId int    `form:"user_id"`
}

type CountFollowListRes struct {
	StatusCode int    `json:"status_code"`
	StatusMsg  string `json:"status_msg"`
	Count      int    `json:"count"`
}

type FollowerListReq struct {
	Token   string `form:"token"`
	ActorId int    `form:"actor_id"`
	UserId  int    `form:"user_id"`
}

type FollowerListRes struct {
	StatusCode int          `json:"status_code"`
	StatusMsg  string       `json:"status_msg"`
	UserList   []*user.User `json:"user_list"`
}

type CountFollowerListReq struct {
	Token  string `form:"token"`
	UserId int    `form:"user_id"`
}

type CountFollowerListRes struct {
	StatusCode int    `json:"status_code"`
	StatusMsg  string `json:"status_msg"`
	Count      int    `json:"count"`
}

type FriendListReq struct {
	Token   string `form:"token"`
	ActorId int    `form:"actor_id"`
	UserId  int    `form:"user_id"`
}

type FriendListRes struct {
	StatusCode int          `json:"status_code"`
	StatusMsg  string       `json:"status_msg"`
	UserList   []*user.User `json:"user_list"`
}

type IsFollowReq struct {
	Token   string `form:"token" binding:"required"`
	ActorId int    `form:"actor_id"`
	UserId  int    `form:"user_id"`
}

type IsFollowRes struct {
	Result bool `json:"result"`
}
