package biz

import (
	"errors"
	"sort"
	"time"
)

type LotStatus string

const (
	LotDraft   LotStatus = "DRAFT"
	LotLive    LotStatus = "LIVE"
	LotSettled LotStatus = "SETTLED"
)

type Money int64 // cents/fen style minor unit

type Lot struct {
	ID             string    `json:"id"`
	RoomID         string    `json:"roomId"`
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

type Bid struct {
	ID        string    `json:"id"`
	LotID     string    `json:"lotId"`
	UserID    string    `json:"userId"`
	Nickname  string    `json:"nickname"`
	Amount    Money     `json:"amount"`
	CreatedAt time.Time `json:"createdAt"`
}

type RankingItem struct {
	Rank     int       `json:"rank"`
	UserID   string    `json:"userId"`
	Nickname string    `json:"nickname"`
	Amount   Money     `json:"amount"`
	At       time.Time `json:"at"`
}

var (
	ErrLotNotLive     = errors.New("lot is not live")
	ErrBidTooLow      = errors.New("bid amount is lower than required minimum")
	ErrLotAlreadyDone = errors.New("lot already settled")
)

func (l *Lot) RequiredNextBid() Money {
	base := l.CurrentPrice
	if base == 0 {
		base = l.StartPrice
	}
	return base + l.MinIncrement
}

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

func (l *Lot) Ranking(limit int) []RankingItem {
	bids := append([]Bid(nil), l.Bids...)
	sort.SliceStable(bids, func(i, j int) bool {
		if bids[i].Amount == bids[j].Amount {
			return bids[i].CreatedAt.Before(bids[j].CreatedAt)
		}
		return bids[i].Amount > bids[j].Amount
	})
	if limit <= 0 || limit > len(bids) {
		limit = len(bids)
	}
	out := make([]RankingItem, 0, limit)
	seen := map[string]bool{}
	for _, b := range bids {
		if seen[b.UserID] {
			continue
		}
		seen[b.UserID] = true
		out = append(out, RankingItem{Rank: len(out) + 1, UserID: b.UserID, Nickname: b.Nickname, Amount: b.Amount, At: b.CreatedAt})
		if len(out) == limit {
			break
		}
	}
	return out
}
