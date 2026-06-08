package auction

import (
	"testing"

	v1 "live-auction-bid/backend/api/auction/service/v1"
)

func TestBuildRuntimeBidProjectionArtifactsUsesStableEventAndOrderIDs(t *testing.T) {
	lot := &v1.Lot{
		Id:              "lot_projection",
		RoomId:          "room_projection",
		MainAccountId:   "main_projection",
		Title:           "投影测试",
		ImageUrl:        "https://example.com/lot.jpg",
		Status:          v1.LotStatus_LOT_STATUS_SETTLED,
		CurrentPrice:    &v1.Money{Amount: 12000, Currency: "CNY"},
		LeadingUserId:   "buyer_1",
		LeadingNickname: "买家一",
		WinnerUserId:    "buyer_1",
		WinnerNickname:  "买家一",
		FinalPrice:      &v1.Money{Amount: 12000, Currency: "CNY"},
		Rule: &v1.BidRule{
			StartPrice:             &v1.Money{Amount: 10000, Currency: "CNY"},
			MinIncrement:           &v1.Money{Amount: 1000, Currency: "CNY"},
			DurationSeconds:        300,
			AntiSnipeWindowSeconds: 15,
			AntiSnipeExtendSeconds: 15,
			MaxExtendCount:         3,
		},
		Version: 2,
	}
	bid := v1.Bid{
		Id:              "bid_projection",
		LotId:           lot.Id,
		UserId:          "buyer_1",
		Nickname:        "买家一",
		Amount:          &v1.Money{Amount: 12000, Currency: "CNY"},
		CreatedAtUnixMs: 1000,
	}
	projection := RuntimeProjectionEvent{
		RuntimeEventID:     "rte_projection_1",
		RoomID:             lot.RoomId,
		LotID:              lot.Id,
		Bid:                bid,
		Lot:                lot,
		Ranking:            []*v1.RankingItem{{Rank: 1, UserId: "buyer_1", Nickname: "买家一", Amount: bid.Amount, BidAtUnixMs: bid.CreatedAtUnixMs}},
		PreviousLotVersion: 1,
		LotVersion:         2,
		OccurredAtUnixMs:   1000,
		OrderID:            "order_projection_1",
	}

	events, order, err := BuildRuntimeBidProjectionArtifacts(projection)
	if err != nil {
		t.Fatalf("build artifacts failed: %v", err)
	}
	if order == nil || order.ID != "order_projection_1" || order.LotID != lot.Id {
		t.Fatalf("order should be created from runtime event order id: %+v", order)
	}
	if len(events) < 5 {
		t.Fatalf("settled projection should create bid, ranking, settled, closed and order events: %+v", events)
	}
	again, _, err := BuildRuntimeBidProjectionArtifacts(projection)
	if err != nil {
		t.Fatalf("build artifacts second time failed: %v", err)
	}
	for i := range events {
		if events[i].Id != again[i].Id {
			t.Fatalf("runtime projection event IDs must be deterministic: %s != %s", events[i].Id, again[i].Id)
		}
	}
}
