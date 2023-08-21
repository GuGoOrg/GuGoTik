package pathgen

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"
)

// GenerateRawVideoName 生成初始视频名称，此链接仅用于内部使用，暴露给用户的视频名称
func GenerateRawVideoName(actorId uint32, title string, videoId uint32) string {
	hash := sha256.Sum256([]byte("RAW" + strconv.FormatUint(uint64(actorId), 10) + title + strconv.FormatUint(uint64(videoId), 10)))
	return hex.EncodeToString(hash[:]) + ".mp4"
}

// GenerateFinalVideoName 最终暴露给用户的视频名称
func GenerateFinalVideoName(actorId uint32, title string, videoId uint32) string {
	hash := sha256.Sum256([]byte(strconv.FormatUint(uint64(actorId), 10) + title + strconv.FormatUint(uint64(videoId), 10)))
	return hex.EncodeToString(hash[:]) + ".mp4"
}

// GenerateCoverName 生成视频封面名称
func GenerateCoverName(actorId uint32, title string, videoId uint32) string {
	hash := sha256.Sum256([]byte(strconv.FormatUint(uint64(actorId), 10) + title + strconv.FormatUint(uint64(videoId), 10)))
	return hex.EncodeToString(hash[:]) + ".jpg"
}

// GenerateAudioName 生成音频链接，此链接仅用于内部使用，不暴露给用户
func GenerateAudioName(videoFileName string) string {
	hash := sha256.Sum256([]byte("AUDIO_" + videoFileName))
	return hex.EncodeToString(hash[:]) + ".mp3"
}
