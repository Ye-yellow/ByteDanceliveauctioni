package test

import (
	"testing"

	v1 "live-auction-bid/backend/api/auction/service/v1"
	"live-auction-bid/backend/app/auction/service/internal/biz/auction"
	"live-auction-bid/backend/app/auction/service/internal/pkg/clock"
)

func TestLotStateMachine(t *testing.T) {
	lot := auction.NewLotFromRequest("lot_1", &v1.CreateLotRequest{
		RoomId: "demo",
		Title:  "测试拍品",
		Rule: &v1.BidRule{
			StartPrice:   auction.CNY(10000),
			MinIncrement: auction.CNY(1000),
		},
	})

	if err := auction.AcceptBid(lot, v1.Bid{Amount: auction.CNY(11000)}, clock.NowMs()); err == nil {
		t.Fatal("DRAFT 状态不应该允许出价")
	}
	if err := auction.StartLot(lot, 1000); err != nil {
		t.Fatalf("开拍失败：%v", err)
	}
	if lot.Status != v1.LotStatus_LOT_STATUS_LIVE {
		t.Fatalf("期望状态 LIVE，实际 %s", lot.Status)
	}
	if err := auction.AcceptBid(lot, v1.Bid{UserId: "u1", Nickname: "用户1", Amount: auction.CNY(10500)}, 2000); err == nil {
		t.Fatal("低于最低加价的出价应该被拒绝")
	}
	if err := auction.AcceptBid(lot, v1.Bid{UserId: "u1", Nickname: "用户1", Amount: auction.CNY(11000)}, 2000); err != nil {
		t.Fatalf("合法出价失败：%v", err)
	}
	if lot.GetCurrentPrice().GetAmount() != 11000 || lot.LeadingUserId != "u1" {
		t.Fatalf("出价后领先状态错误：%+v", lot)
	}
	if err := auction.SettleLot(lot, 3000); err != nil {
		t.Fatalf("落锤失败：%v", err)
	}
	if lot.Status != v1.LotStatus_LOT_STATUS_SETTLED || lot.WinnerUserId != "u1" || lot.GetFinalPrice().GetAmount() != 11000 {
		t.Fatalf("成交状态错误：%+v", lot)
	}
}

func TestBuildRanking(t *testing.T) {
	bids := []v1.Bid{
		{UserId: "u1", Nickname: "用户1", Amount: auction.CNY(11000), CreatedAtUnixMs: 1000},
		{UserId: "u2", Nickname: "用户2", Amount: auction.CNY(12000), CreatedAtUnixMs: 2000},
		{UserId: "u1", Nickname: "用户1", Amount: auction.CNY(13000), CreatedAtUnixMs: 3000},
	}

	ranking := auction.BuildRanking(bids)
	if len(ranking) != 2 {
		t.Fatalf("期望 2 个用户，实际 %d", len(ranking))
	}
	if ranking[0].UserId != "u1" || ranking[0].GetAmount().GetAmount() != 13000 {
		t.Fatalf("排名第一错误：%+v", ranking[0])
	}
	if ranking[1].UserId != "u2" || ranking[1].GetAmount().GetAmount() != 12000 {
		t.Fatalf("排名第二错误：%+v", ranking[1])
	}
}
