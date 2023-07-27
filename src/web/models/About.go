package models

type AboutReq struct {
	Echo string `json:"echo" uri:"echo" form:"echo"`
}

type AboutRes struct {
	TimeStamp int64  `json:"time_stamp"`
	Echo      string `json:"echo,omitempty"`
}
