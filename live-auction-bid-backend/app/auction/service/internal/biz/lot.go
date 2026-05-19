package biz

import (
	"errors"
	"time"
)

type LotStatus string

const (
	LotDraft   LotStatus = "DRAFT"
	LotLive    LotStatus = "LIVE"
	LotSettled LotStatus = "SETTLED"
)

type Lot struct {
	ID             string    `json:"id"`
	RoomID         string    `json:"roomId"`
	SessionID      string    `json:"sessionId,omitempty"`
	Title          string    `json:"title"`
	Description    string    `json:"description"`
	ImageURL        string    `json:"imageUrl"`
	StartPrice     Money     `json:"startPrice"`
	CurrentPrice   Money     `json:"currentPrice"`
	MinIncrement   Money     `json:"minIncrement"`
	Status         LotStatus `json:"status"`
	EndsAt         time.Time `json:"endsAt"`
	WinnerUserID   string    `json:"winnerUserId,omitempty"`
	Version        int64     `json:"version"`
	Bids           []Bid     `json:"bids"`
	AtmosphereText string    `json:"atmosphereText,omitempty"`
}

var (
	ErrLotNotLive     = errors.New("lot is not live")
	ErrLotAlreadyDone = errors.New("lot already settled")
)

func (l *Lot) RequiredNextBid() Money {
	base := l.CurrentPrice
	if base == 0 {
		base = l.StartPrice
	}
	return base + l.MinIncrement
}
