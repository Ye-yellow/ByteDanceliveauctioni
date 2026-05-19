package biz

import (
	"sort"
	"time"
)

type RankingItem struct {
	Rank     int       `json:"rank"`
	UserID   string    `json:"userId"`
	Nickname string    `json:"nickname"`
	Amount   Money     `json:"amount"`
	At       time.Time `json:"at"`
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
