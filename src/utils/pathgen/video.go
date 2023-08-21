package pathgen

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"
)

// GenerateRawVideoName 生成初始视频链接，此链接仅用于内部使用，暴露给用户的视频地址
func GenerateRawVideoName(actorId uint32, title string) string {
	hash := sha256.Sum256([]byte("RAW" + strconv.FormatUint(uint64(actorId), 10) + title))
	return hex.EncodeToString(hash[:]) + ".mp4"
}

// GenerateFinalVideoName 最终暴露给用户的视频地址
func GenerateFinalVideoName(actorId uint32, title string) string {
	hash := sha256.Sum256([]byte(strconv.FormatUint(uint64(actorId), 10) + title))
	return hex.EncodeToString(hash[:]) + ".mp4"
}
