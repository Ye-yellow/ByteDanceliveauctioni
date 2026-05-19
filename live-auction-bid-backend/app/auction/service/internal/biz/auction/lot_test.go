package auction

import "testing"

func TestLotStateMachine(t *testing.T) {
	lot := NewLotFromCommand("lot_1", CreateLotCommand{
		RoomID: "demo",
		Title:  "测试拍品",
		Rule: BidRule{
			StartPrice:   CNY(10000),
			MinIncrement: CNY(1000),
		},
	})

	if err := lot.AcceptBid(Bid{Amount: CNY(11000)}, NowMs()); err == nil {
		t.Fatal("DRAFT 状态不应该允许出价")
	}
	if err := lot.Start(1000); err != nil {
		t.Fatalf("开拍失败：%v", err)
	}
	if lot.Status != LotStatusLive {
		t.Fatalf("期望状态 LIVE，实际 %s", lot.Status)
	}
	if err := lot.AcceptBid(Bid{UserID: "u1", Nickname: "用户1", Amount: CNY(10500)}, 2000); err == nil {
		t.Fatal("低于最低加价的出价应该被拒绝")
	}
	if err := lot.AcceptBid(Bid{UserID: "u1", Nickname: "用户1", Amount: CNY(11000)}, 2000); err != nil {
		t.Fatalf("合法出价失败：%v", err)
	}
	if lot.CurrentPrice.Amount != 11000 || lot.LeadingUserID != "u1" {
		t.Fatalf("出价后领先状态错误：%+v", lot)
	}
	if err := lot.Settle(3000); err != nil {
		t.Fatalf("落锤失败：%v", err)
	}
	if lot.Status != LotStatusSettled || lot.WinnerUserID != "u1" || lot.FinalPrice.Amount != 11000 {
		t.Fatalf("成交状态错误：%+v", lot)
	}
}

func TestBuildRanking(t *testing.T) {
	bids := []Bid{
		{UserID: "u1", Nickname: "用户1", Amount: CNY(11000), CreatedAtUnixMs: 1000},
		{UserID: "u2", Nickname: "用户2", Amount: CNY(12000), CreatedAtUnixMs: 2000},
		{UserID: "u1", Nickname: "用户1", Amount: CNY(13000), CreatedAtUnixMs: 3000},
	}

	ranking := BuildRanking(bids)
	if len(ranking) != 2 {
		t.Fatalf("期望 2 个用户，实际 %d", len(ranking))
	}
	if ranking[0].UserID != "u1" || ranking[0].Amount.Amount != 13000 {
		t.Fatalf("排名第一错误：%+v", ranking[0])
	}
	if ranking[1].UserID != "u2" || ranking[1].Amount.Amount != 12000 {
		t.Fatalf("排名第二错误：%+v", ranking[1])
	}
}
