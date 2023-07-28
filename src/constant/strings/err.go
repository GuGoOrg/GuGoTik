package strings

// Bad Request
const (
	GateWayErrorCode       = 40001
	GateWayError           = "GuGoTik Gateway 暂时不能处理您的请求，请稍后重试！"
	GateWayParamsErrorCode = 40002
	GateWayParamsError     = "GuGoTik Gateway 无法响应您的请求，请重启 APP 或稍后再试!"
)

// Server Inner Error
const (
	AuthServiceInnerErrorCode = 50001
	AuthServiceInnerError     = "登录服务出现内部错误，请稍后重试！"
)

// Expected Error
const (
	AuthUserExistedCode = 10001
	AuthUserExisted     = "用户已存在，请更换用户名或尝试登录！"
)
