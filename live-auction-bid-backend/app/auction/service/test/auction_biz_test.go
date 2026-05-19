package test

import (
	"context"
	"strings"
	"testing"

	v1 "live-auction-bid/backend/api/auction/service/v1"
	"live-auction-bid/backend/app/auction/service/internal/biz/auction"
	"live-auction-bid/backend/app/auction/service/internal/data"
	"live-auction-bid/backend/app/auction/service/internal/pkg/clock"
)

func TestLotStateMachine(t *testing.T) {
	lot, err := auction.NewLotFromRequest("lot_1", &v1.CreateLotRequest{
		RoomId: "demo",
		Title:  "测试拍品",
		Rule: &v1.BidRule{
			StartPrice:             &v1.Money{Amount: 10000, Currency: "CNY"},
			MinIncrement:           &v1.Money{Amount: 1000, Currency: "CNY"},
			DurationSeconds:        300,
			AntiSnipeWindowSeconds: 15,
			AntiSnipeExtendSeconds: 15,
			MaxExtendCount:         3,
		},
	})
	if err != nil {
		t.Fatalf("创建拍品失败：%v", err)
	}

	if err := auction.AcceptBid(lot, v1.Bid{Amount: &v1.Money{Amount: 11000, Currency: "CNY"}}, clock.NowMs()); err == nil {
		t.Fatal("DRAFT 状态不应该允许出价")
	}
	if err := auction.StartLot(lot, 1000); err != nil {
		t.Fatalf("开拍失败：%v", err)
	}
	if lot.Status != v1.LotStatus_LOT_STATUS_LIVE {
		t.Fatalf("期望状态 LIVE，实际 %s", lot.Status)
	}
	if err := auction.AcceptBid(lot, v1.Bid{UserId: "u1", Nickname: "用户1", Amount: &v1.Money{Amount: 10500, Currency: "CNY"}}, 2000); err == nil {
		t.Fatal("低于最低加价的出价应该被拒绝")
	}
	if err := auction.AcceptBid(lot, v1.Bid{UserId: "u1", Nickname: "用户1", Amount: &v1.Money{Amount: 11000, Currency: "CNY"}}, 2000); err != nil {
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
		{UserId: "u1", Nickname: "用户1", Amount: &v1.Money{Amount: 11000, Currency: "CNY"}, CreatedAtUnixMs: 1000},
		{UserId: "u2", Nickname: "用户2", Amount: &v1.Money{Amount: 12000, Currency: "CNY"}, CreatedAtUnixMs: 2000},
		{UserId: "u1", Nickname: "用户1", Amount: &v1.Money{Amount: 13000, Currency: "CNY"}, CreatedAtUnixMs: 3000},
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

func TestCreateLotRejectsMismatchedCurrency(t *testing.T) {
	_, err := auction.NewLotFromRequest("lot_1", &v1.CreateLotRequest{
		RoomId: "demo",
		Title:  "测试拍品",
		Rule: &v1.BidRule{
			StartPrice:             &v1.Money{Currency: "CNY", Amount: 10000},
			MinIncrement:           &v1.Money{Currency: "USD", Amount: 1000},
			DurationSeconds:        300,
			AntiSnipeWindowSeconds: 15,
			AntiSnipeExtendSeconds: 15,
			MaxExtendCount:         3,
		},
	})
	if err == nil || !strings.Contains(err.Error(), "currency must match") {
		t.Fatalf("expected currency mismatch error, got %v", err)
	}
}

func TestPlaceBidRejectsMissingUserInsteadOfDefaulting(t *testing.T) {
	store := data.NewMemoryStore()
	uc := auction.NewAuctionUsecase(store, store, nil)
	ctx := context.Background()

	lot, err := uc.CreateLot(ctx, &v1.CreateLotRequest{
		RoomId: "demo",
		Title:  "测试拍品",
		Rule: &v1.BidRule{
			StartPrice:             &v1.Money{Amount: 10000, Currency: "CNY"},
			MinIncrement:           &v1.Money{Amount: 1000, Currency: "CNY"},
			DurationSeconds:        300,
			AntiSnipeWindowSeconds: 15,
			AntiSnipeExtendSeconds: 15,
			MaxExtendCount:         3,
		},
	})
	if err != nil {
		t.Fatalf("create lot failed: %v", err)
	}
	if _, err := uc.StartLot(ctx, lot.Id); err != nil {
		t.Fatalf("start lot failed: %v", err)
	}

	_, _, _, err = uc.PlaceBid(ctx, &v1.PlaceBidRequest{
		LotId:    lot.Id,
		Nickname: "用户1",
		Amount:   &v1.Money{Amount: 11000, Currency: "CNY"},
	})
	if err == nil || !strings.Contains(err.Error(), "user id is required") {
		t.Fatalf("expected user id error, got %v", err)
	}
}
