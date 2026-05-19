package biz

import "time"

type AuctionEventType string

const (
	AuctionEventLotUpdated  AuctionEventType = "lot.updated"
	AuctionEventBidAccepted AuctionEventType = "bid.accepted"
	AuctionEventLotSettled  AuctionEventType = "lot.settled"
)

type AuctionEvent struct {
	ID        string           `json:"id"`
	Type      AuctionEventType `json:"type"`
	RoomID    string           `json:"roomId"`
	LotID     string           `json:"lotId"`
	Version   int64            `json:"version"`
	CreatedAt time.Time        `json:"createdAt"`
}
