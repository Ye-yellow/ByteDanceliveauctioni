package auction

import "sort"

func BuildRanking(bids []Bid) []RankingItem {
	bestByUser := make(map[string]RankingItem)
	for _, bid := range bids {
		current, ok := bestByUser[bid.UserID]
		if !ok || isBetterBid(bid, current) {
			bestByUser[bid.UserID] = RankingItem{
				UserID:      bid.UserID,
				Nickname:    bid.Nickname,
				Amount:      bid.Amount,
				BidAtUnixMs: bid.CreatedAtUnixMs,
			}
		}
	}

	items := make([]RankingItem, 0, len(bestByUser))
	for _, item := range bestByUser {
		items = append(items, item)
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].Amount.Amount == items[j].Amount.Amount {
			return items[i].BidAtUnixMs < items[j].BidAtUnixMs
		}
		return items[i].Amount.Amount > items[j].Amount.Amount
	})

	for i := range items {
		items[i].Rank = int32(i + 1)
	}
	return items
}

func isBetterBid(bid Bid, current RankingItem) bool {
	if bid.Amount.Amount != current.Amount.Amount {
		return bid.Amount.Amount > current.Amount.Amount
	}
	return bid.CreatedAtUnixMs < current.BidAtUnixMs
}

func RecentBids(bids []Bid, limit int) []Bid {
	if len(bids) <= limit {
		return append([]Bid(nil), bids...)
	}
	return append([]Bid(nil), bids[len(bids)-limit:]...)
}

func ShouldAutoStartDuel(lot *Lot, ranking []RankingItem, bids []Bid, nowMs int64) bool {
	if lot.DuelState.Active || len(ranking) < 2 || len(bids) < 3 {
		return false
	}
	if lot.EndsAtUnixMs-nowMs > 60_000 {
		return false
	}
	return ranking[0].Amount.Amount-ranking[1].Amount.Amount <= lot.Rule.MinIncrement.Amount*3
}
