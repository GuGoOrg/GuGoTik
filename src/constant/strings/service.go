package strings

const (
	// Exchange name
	VideoExchange   = "video_exchange"
	EventExchange   = "event"
	MessageExchange = "message_exchange"

	// Queue name
	VideoPicker   = "video_picker"
	VideoSummary  = "video_summary"
	MessageCommon = "message_common"
	MessageGPT    = "message_gpt"
	MessageES     = "message_es"

	// Routing key
	FavoriteActionEvent = "video.favorite.action"
	VideoGetEvent       = "video.get.action"
	VideoCommentEvent   = "video.comment.action"
	VideoPublishEvent   = "video.publish.action"

	MessageActionEvent    = "message.common"
	MessageGptActionEvent = "message.gpt"
)
