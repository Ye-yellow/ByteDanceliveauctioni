package auction

import (
	"sort"

	"live-auction-bid/backend/app/auction/service/internal/model"
)

func BuildRanking(bids []model.Bid) []*model.RankingItem {
	bestByUser := make(map[string]*model.RankingItem)
	for _, bid := range bids {
		current, ok := bestByUser[bid.UserId]
		if !ok || isBetterBid(bid, current) {
			bestByUser[bid.UserId] = &model.RankingItem{
				UserId:      bid.UserId,
				Nickname:    bid.Nickname,
				Amount:      cloneMoney(bid.GetAmount()),
				BidAtUnixMs: bid.CreatedAtUnixMs,
			}
		}
	}

	items := make([]*model.RankingItem, 0, len(bestByUser))
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

func isBetterBid(bid model.Bid, current *model.RankingItem) bool {
	if bid.GetAmount().GetAmount() != current.GetAmount().GetAmount() {
		return bid.GetAmount().GetAmount() > current.GetAmount().GetAmount()
	}
	return bid.CreatedAtUnixMs < current.BidAtUnixMs
}

func RecentBids(bids []model.Bid, limit int) []*model.Bid {
	start := 0
	if len(bids) > limit {
		start = len(bids) - limit
	}
	out := make([]*model.Bid, 0, len(bids)-start)
	for i := start; i < len(bids); i++ {
		bid := bids[i]
		out = append(out, &bid)
	}
	return out
}

func ShouldAutoStartDuel(lot *model.Lot, ranking []*model.RankingItem, bids []model.Bid, nowMs int64) bool {
	if lot.GetDuelState().GetActive() || len(ranking) < 2 || len(bids) < 3 {
		return false
	}
	if lot.EndsAtUnixMs-nowMs > 60_000 {
		return false
	}
	return ranking[0].GetAmount().GetAmount()-ranking[1].GetAmount().GetAmount() <= lot.GetRule().GetMinIncrement().GetAmount()*3
}
