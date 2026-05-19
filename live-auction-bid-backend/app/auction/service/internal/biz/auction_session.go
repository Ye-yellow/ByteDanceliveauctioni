package biz

import "time"

type AuctionSessionStatus string

const (
	AuctionSessionPreparing AuctionSessionStatus = "PREPARING"
	AuctionSessionLive      AuctionSessionStatus = "LIVE"
	AuctionSessionEnded     AuctionSessionStatus = "ENDED"
)

type AuctionSession struct {
	ID        string               `json:"id"`
	RoomID    string               `json:"roomId"`
	Title     string               `json:"title"`
	Status    AuctionSessionStatus `json:"status"`
	StartedAt *time.Time           `json:"startedAt,omitempty"`
	EndedAt   *time.Time           `json:"endedAt,omitempty"`
}
