package auction

import (
	"sort"

	v1 "live-auction-bid/backend/api/auction/service/v1"
)

func BuildRanking(bids []v1.Bid) []*v1.RankingItem {
	bestByUser := make(map[string]*v1.RankingItem)
	for _, bid := range bids {
		current, ok := bestByUser[bid.UserId]
		if !ok || bid.GetAmount().GetAmount() > current.GetAmount().GetAmount() ||
			bid.GetAmount().GetAmount() == current.GetAmount().GetAmount() && bid.CreatedAtUnixMs < current.BidAtUnixMs {
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
