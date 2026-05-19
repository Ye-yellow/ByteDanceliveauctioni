package realtime

type Presence struct {
	RoomID      string `json:"roomId"`
	OnlineCount int    `json:"onlineCount"`
}
