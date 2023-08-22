package models

type UserReq struct {
	UserId  uint32 `form:"user_id" binding:"required"`
	ActorId uint32 `form:"actor_id" binding:"required"`
}

type UserRes struct {
	StatusCode int32  `json:"status_code"` // 状态码，0-成功，其他值-失败
	StatusMsg  string `json:"status_msg"`  // 返回状态描述
}
