package biz

import (
	"errors"
	"time"
)

type Bid struct {
	ID        string    `json:"id"`
	LotID     string    `json:"lotId"`
	UserID    string    `json:"userId"`
	Nickname  string    `json:"nickname"`
	Amount    Money     `json:"amount"`
	CreatedAt time.Time `json:"createdAt"`
}

var ErrBidTooLow = errors.New("bid amount is lower than required minimum")

func (l *Lot) PlaceBid(b Bid, antiSnipeExtend time.Duration) error {
	if l.Status != LotLive {
		return ErrLotNotLive
	}
	if time.Now().After(l.EndsAt) {
		return ErrLotAlreadyDone
	}
	if b.Amount < l.RequiredNextBid() {
		return ErrBidTooLow
	}
	l.Bids = append(l.Bids, b)
	l.CurrentPrice = b.Amount
	l.WinnerUserID = b.UserID
	l.Version++
	if time.Until(l.EndsAt) <= antiSnipeExtend {
		l.EndsAt = time.Now().Add(antiSnipeExtend)
	}
	return nil
}
