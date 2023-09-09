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
	VideoServiceInnerErrorCode       = 50002
	VideoServiceInnerError           = "视频发布服务出现内部错误，请稍后重试！"
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
	UnableToAddMessageErrorCode      = 50009
	UnableToAddMessageError          = "发送消息出错"
	UnableToQueryMessageErrorCode    = 50010
	UnableToQueryMessageError        = "查消息出错"
	PublishServiceInnerErrorCode     = 50011
	PublishServiceInnerError         = "发布服务出现内部错误，请稍后重试！"
	UnableToFollowErrorCode          = 50012
	UnableToFollowError              = "关注该用户失败"
	UnableToUnFollowErrorCode        = 50013
	UnableToUnFollowError            = "取消关注失败"
	UnableToGetFollowListErrorCode   = 50014
	UnableToGetFollowListError       = "无法查询到关注列表"
	UnableToGetFollowerListErrorCode = 50015
	UnableToGetFollowerListError     = "无法查询到粉丝列表"
	UnableToRelateYourselfErrorCode  = 50016
	UnableToRelateYourselfError      = "无法关注自己"
	RelationNotFoundErrorCode        = 50017
	RelationNotFoundError            = "未关注该用户"
	StringToIntErrorCode             = 50018
	StringToIntError                 = "字符串转数字失败"
	RelationServiceIntErrorCode      = 50019
	RelationServiceIntError          = "关系服务出现内部错误"
	FavoriteServiceErrorCode         = 50020
	FavoriteServiceError             = "点赞服务内部出错"
	UserServiceInnerErrorCode        = 50021
	UserServiceInnerError            = "登录服务出现内部错误，请稍后重试！"
	UnableToQueryVideoErrorCode      = 50022
	UnableToQueryVideoError          = "无法查询到该视频"
	AlreadyFollowingErrorCode        = 50023
	AlreadyFollowingError            = "无法关注已关注的人"
	UnableToGetFriendListErrorCode   = 50024
	UnableToGetFriendListError       = "无法查询到好友列表"
	RecommendServiceInnerErrorCode   = 50025
	RecommendServiceInnerError       = "推荐系统内部错误"
)

// Expected Error
const (
	AuthUserExistedCode           = 10001
	AuthUserExisted               = "用户已存在，请更换用户名或尝试登录！"
	UserNotExistedCode            = 10002
	UserNotExisted                = "用户不存在，请先注册或检查你的用户名是否正确！"
	AuthUserLoginFailedCode       = 10003
	AuthUserLoginFailed           = "用户信息错误，请检查账号密码是否正确"
	AuthUserNeededCode            = 10004
	AuthUserNeeded                = "用户无权限操作，请登陆后重试！"
	ActionCommentTypeInvalidCode  = 10005
	ActionCommentTypeInvalid      = "不合法的评论类型"
	ActionCommentLimitedCode      = 10006
	ActionCommentLimited          = "评论频繁，请稍后再试！"
	InvalidContentTypeCode        = 10007
	InvalidContentType            = "不合法的内容类型"
	FavoriteServiceDuplicateCode  = 10008
	FavoriteServiceDuplicateError = "不能重复点赞"
	FavoriteServiceCancelCode     = 10009
	FavoriteServiceCancelError    = "没有点赞,不能取消点赞"
	PublishVideoLimitedCode       = 10010
	PublishVideoLimited           = "视频发布频繁，请稍后再试！"
	ChatActionLimitedCode         = 10011
	ChatActionLimitedError        = "发送消息频繁，请稍后再试！"
	FollowLimitedCode             = 10012
	FollowLimited                 = "关注频繁，请稍后再试！"
	UserDoNotExistedCode          = 10013
	UserDoNotExisted              = "查询用户不存在！"
	OversizeVideoCode             = 10014
	OversizeVideo                 = "上传视频超过了200MB"
)
