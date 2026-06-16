package user

import (
	"hash/crc32"
	"strings"
)

var defaultAvatarURLs = []string{
	"https://liveauction.tos-cn-beijing.volces.com/douyin-h5/images/avatar-71158770-d8597.jpeg",
	"https://liveauction.tos-cn-beijing.volces.com/douyin-h5/images/avatar-lsy0508-160edjy.jpeg",
	"https://liveauction.tos-cn-beijing.volces.com/douyin-h5/images/avatar-ll991221-1bmdvg4.jpeg",
	"https://liveauction.tos-cn-beijing.volces.com/douyin-h5/images/avatar-sunmeng333-qheb8m.jpeg",
	"https://liveauction.tos-cn-beijing.volces.com/douyin-h5/images/avatar-jingyiziran-176539n.jpeg",
	"https://liveauction.tos-cn-beijing.volces.com/douyin-h5/images/avatar-8357999-1bd1vnm.jpeg",
}

func AvatarURLForUserID(userID string) string {
	key := strings.TrimSpace(userID)
	if key == "" {
		key = "buyer"
	}
	return defaultAvatarURLs[int(crc32.ChecksumIEEE([]byte(key))%uint32(len(defaultAvatarURLs)))]
}
