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
	AuthServiceInnerErrorCode        = 50001
	AuthServiceInnerError            = "登录服务出现内部错误，请稍后重试！"
	UserServiceInnerErrorCode        = 50002
	UserServiceInnerError            = "用户信息服务出现内部错误，请稍后重试！"
	UnableToQueryUserErrorCode       = 50003
	UnableToQueryUserError           = "无法查询到对应用户"
	UnableToQueryCommentErrorCode    = 50004
	UnableToQueryCommentError        = "无法查询到视频评论"
	UnableToCreateCommentErrorCode   = 50005
	UnableToCreateCommentError       = "无法创建评论"
	FeedServiceInnerErrorCode        = 50006
	FeedServiceInnerError            = "视频服务出现内部错误，请稍后重试！"
	ActorIDNotMatchErrorCode         = 50007
	ActorIDNotMatchError             = "用户不匹配"
	UnableToDeleteCommentErrorCode   = 50008
	UnableToDeleteCommentError       = "无法删除视频评论"
	RelationAlreadyExistsErrorCode   = 50009
	RelationAlreadyExistsError       = "无法关注该用户"
	UnableToUnFollowErrorCode        = 50010
	UnableToUnFollowError            = "取消关注失败"
	UnableToGetFollowListErrorCode   = 50011
	UnableToGetFollowListError       = "无法查询到关注列表"
	UnableToGetFollowerListErrorCode = 50012
	UnableToGetFollowerListError     = "无法查询到粉丝列表"
	UnableToRelateYourselfErrorCode  = 50013
	UnableToRelateYourselfError      = "无法关注自己"
	RelationNotFoundErrorCode        = 50014
	RelationNotFoundError            = "未关注该用户"
)

// Expected Error
const (
	AuthUserExistedCode          = 10001
	AuthUserExisted              = "用户已存在，请更换用户名或尝试登录！"
	UserNotExistedCode           = 10002
	UserNotExisted               = "用户不存在，请先注册或检查你的用户名是否正确！"
	AuthUserLoginFailedCode      = 10003
	AuthUserLoginFailed          = "用户信息错误，请检查账号密码是否正确"
	AuthUserNeededCode           = 10004
	AuthUserNeeded               = "用户无权限操作，请登陆后重试！"
	ActionCommentTypeInvalidCode = 10005
	ActionCommentTypeInvalid     = "不合法的评论类型"
)
