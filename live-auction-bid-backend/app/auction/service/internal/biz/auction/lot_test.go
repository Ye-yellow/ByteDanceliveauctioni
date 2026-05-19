package auction

import (
	"testing"

	"live-auction-bid/backend/app/auction/service/internal/model"
	"live-auction-bid/backend/app/auction/service/internal/pkg/clock"
)

func TestLotStateMachine(t *testing.T) {
	lot := NewLotFromCommand("lot_1", model.CreateLotCommand{
		RoomID: "demo",
		Title:  "测试拍品",
		Rule: &model.BidRule{
			StartPrice:   model.CNY(10000),
			MinIncrement: model.CNY(1000),
		},
	})

	if err := AcceptBid(lot, model.Bid{Amount: model.CNY(11000)}, clock.NowMs()); err == nil {
		t.Fatal("DRAFT 状态不应该允许出价")
	}
	if err := StartLot(lot, 1000); err != nil {
		t.Fatalf("开拍失败：%v", err)
	}
	if lot.Status != model.LotStatusLive {
		t.Fatalf("期望状态 LIVE，实际 %s", lot.Status)
	}
	if err := AcceptBid(lot, model.Bid{UserId: "u1", Nickname: "用户1", Amount: model.CNY(10500)}, 2000); err == nil {
		t.Fatal("低于最低加价的出价应该被拒绝")
	}
	if err := AcceptBid(lot, model.Bid{UserId: "u1", Nickname: "用户1", Amount: model.CNY(11000)}, 2000); err != nil {
		t.Fatalf("合法出价失败：%v", err)
	}
	if lot.GetCurrentPrice().GetAmount() != 11000 || lot.LeadingUserId != "u1" {
		t.Fatalf("出价后领先状态错误：%+v", lot)
	}
	if err := SettleLot(lot, 3000); err != nil {
		t.Fatalf("落锤失败：%v", err)
	}
	if lot.Status != model.LotStatusSettled || lot.WinnerUserId != "u1" || lot.GetFinalPrice().GetAmount() != 11000 {
		t.Fatalf("成交状态错误：%+v", lot)
	}
}

func TestBuildRanking(t *testing.T) {
	bids := []model.Bid{
		{UserId: "u1", Nickname: "用户1", Amount: model.CNY(11000), CreatedAtUnixMs: 1000},
		{UserId: "u2", Nickname: "用户2", Amount: model.CNY(12000), CreatedAtUnixMs: 2000},
		{UserId: "u1", Nickname: "用户1", Amount: model.CNY(13000), CreatedAtUnixMs: 3000},
	}

	ranking := BuildRanking(bids)
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
