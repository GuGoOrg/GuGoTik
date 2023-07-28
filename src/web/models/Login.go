package models

type LoginReq struct {
	UserName string `uri:"username" binding:"required"`
	Password string `uri:"password" binding:"required"`
}

type LoginRes struct {
	StatusCode int    `json:"status_code"`
	StatusMsg  string `json:"status_msg"`
	UserId     uint32 `json:"user_id"`
	Token      string `json:"token"`
}

type RegisterReq struct {
	UserName string `uri:"username" binding:"required"`
	Password string `uri:"password" binding:"required"`
}

type RegisterRes struct {
	StatusCode int    `json:"status_code"`
	StatusMsg  string `json:"status_msg"`
	UserId     int    `json:"user_id"`
	Token      string `json:"token"`
}
