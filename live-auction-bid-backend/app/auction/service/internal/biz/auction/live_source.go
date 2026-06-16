package auction

import (
	"hash/crc32"
	"strings"
)

var defaultLiveSourceURLs = []string{
	"https://liveauction.tos-cn-beijing.volces.com/douyin-h5/videos/6931271799195831566.mp4",
	"https://liveauction.tos-cn-beijing.volces.com/douyin-h5/videos/7326744032997166387.mp4",
	"https://liveauction.tos-cn-beijing.volces.com/douyin-h5/videos/7161000281575148800.mp4",
	"https://liveauction.tos-cn-beijing.volces.com/douyin-h5/videos/6882368275695586568.mp4",
	"https://liveauction.tos-cn-beijing.volces.com/douyin-h5/videos/6993228049399549198.mp4",
	"https://liveauction.tos-cn-beijing.volces.com/douyin-h5/videos/7260749400622894336.mp4",
	"https://liveauction.tos-cn-beijing.volces.com/douyin-h5/videos/7280132304427666722.mp4",
	"https://liveauction.tos-cn-beijing.volces.com/douyin-h5/videos/7110263965858549003.mp4",
}

func LiveSourceURLForRoomID(roomID string) string {
	key := strings.TrimSpace(roomID)
	if key == "" {
		key = "room"
	}
	return defaultLiveSourceURLs[int(crc32.ChecksumIEEE([]byte(key))%uint32(len(defaultLiveSourceURLs)))]
}
