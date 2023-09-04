package strings

// Exchange name
const (
	VideoExchange   = "video_exchange"
	EventExchange   = "event"
	MessageExchange = "message_exchange"
	AuditExchange   = "audit_exchange"
)

// Queue name
const (
	VideoPicker   = "video_picker"
	VideoSummary  = "video_summary"
	MessageCommon = "message_common"
	MessageGPT    = "message_gpt"
	MessageES     = "message_es"
	AuditPicker   = "audit_picker"
)

// Routing key
const (
	FavoriteActionEvent = "video.favorite.action"
	VideoGetEvent       = "video.get.action"
	VideoCommentEvent   = "video.comment.action"
	VideoPublishEvent   = "video.publish.action"

	MessageActionEvent    = "message.common"
	MessageGptActionEvent = "message.gpt"
	AuditPublishEvent     = "audit"
)

// Action Type
const (
	FavoriteIdActionLog = 1 // 用户点赞相关操作
	FollowIdActionLog   = 2 // 用户关注相关操作
)

// Action Name
const (
	FavoriteNameActionLog    = "favorite.action" // 用户点赞操作名称
	FavoriteUpActionSubLog   = "up"
	FavoriteDownActionSubLog = "down"

	FollowNameActionLog    = "follow.action" // 用户关注操作名称
	FollowUpActionSubLog   = "up"
	FollowDownActionSubLog = "down"
)

// Action Service Name
const (
	FavoriteServiceName = "FavoriteService"
	FollowServiceName   = "FollowService"
)
