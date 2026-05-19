package auction

import (
	"sort"

	v1 "live-auction-bid/backend/api/auction/service/v1"
)

func BuildRanking(bids []v1.Bid) []*v1.RankingItem {
	bestByUser := make(map[string]*v1.RankingItem)
	for _, bid := range bids {
		current, ok := bestByUser[bid.UserId]
		if !ok || isBetterBid(bid, current) {
			bestByUser[bid.UserId] = &v1.RankingItem{
				UserId:      bid.UserId,
				Nickname:    bid.Nickname,
				Amount:      cloneMoney(bid.GetAmount()),
				BidAtUnixMs: bid.CreatedAtUnixMs,
			}
		}
	}

	items := make([]*v1.RankingItem, 0, len(bestByUser))
	for _, item := range bestByUser {
		items = append(items, item)
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].GetAmount().GetAmount() == items[j].GetAmount().GetAmount() {
			return items[i].BidAtUnixMs < items[j].BidAtUnixMs
		}
		return items[i].GetAmount().GetAmount() > items[j].GetAmount().GetAmount()
	})

	for i := range items {
		items[i].Rank = int32(i + 1)
	}
	return items
}

func isBetterBid(bid v1.Bid, current *v1.RankingItem) bool {
	if bid.GetAmount().GetAmount() != current.GetAmount().GetAmount() {
		return bid.GetAmount().GetAmount() > current.GetAmount().GetAmount()
	}
	return bid.CreatedAtUnixMs < current.BidAtUnixMs
}

func RecentBids(bids []v1.Bid, limit int) []*v1.Bid {
	start := 0
	if len(bids) > limit {
		start = len(bids) - limit
	}
	out := make([]*v1.Bid, 0, len(bids)-start)
	for i := start; i < len(bids); i++ {
		bid := bids[i]
		out = append(out, &bid)
	}
	return out
}

func ShouldAutoStartDuel(lot *v1.Lot, ranking []*v1.RankingItem, bids []v1.Bid, nowMs int64) bool {
	if lot.GetDuelState().GetActive() || len(ranking) < 2 || len(bids) < 3 {
		return false
	}
	if lot.EndsAtUnixMs-nowMs > 60_000 {
		return false
	}
	return ranking[0].GetAmount().GetAmount()-ranking[1].GetAmount().GetAmount() <= lot.GetRule().GetMinIncrement().GetAmount()*3
}
