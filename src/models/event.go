package models

type RecommendEvent struct {
	Type     int    // 1. 已读 2. 喜欢 3. 插入新数据
	Source   string // 来源
	Slice    string // 附带信息
	ActorId  uint32
	VideoId  []uint32 // 代表视频 Id，可以批量操作，但是仅对于某一个唯一的用户
	Tag      []string // 插入时使用
	Category []string // 插入时使用
	Title    string
}
