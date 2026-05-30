package test

import (
	"context"
	"errors"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"
	v1 "live-auction-bid/backend/api/auction/service/v1"
	"live-auction-bid/backend/app/auction/service/internal/biz/auction"
	"live-auction-bid/backend/app/auction/service/internal/pkg/apperr"
	"live-auction-bid/backend/app/auction/service/internal/pkg/clock"
)

func TestLotStateMachine(t *testing.T) {
	lot, err := auction.NewLotFromRequest("lot_1", &v1.CreateLotRequest{
		RoomId:   "demo",
		Title:    "测试拍品",
		ImageUrl: "https://example.com/lot.jpg",
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
	if err := auction.AcceptBid(lot, v1.Bid{UserId: "u1", Nickname: "用户1", Amount: &v1.Money{Amount: 12000, Currency: "CNY"}}, 2100); err == nil || !strings.Contains(err.Error(), "leading bidder must wait") {
		t.Fatalf("领先者不应该能连续给自己加价，实际错误：%v", err)
	}
	if lot.GetCurrentPrice().GetAmount() != 11000 || lot.LeadingUserId != "u1" {
		t.Fatalf("被拒绝的连续出价不应该改变领先状态：%+v", lot)
	}
	if err := auction.AcceptBid(lot, v1.Bid{UserId: "u2", Nickname: "用户2", Amount: &v1.Money{Amount: 12000, Currency: "CNY"}}, 2200); err != nil {
		t.Fatalf("其他买家应该可以超过当前领先者：%v", err)
	}
	if err := auction.AcceptBid(lot, v1.Bid{UserId: "u1", Nickname: "用户1", Amount: &v1.Money{Amount: 13000, Currency: "CNY"}}, 2300); err != nil {
		t.Fatalf("被别人超过后应该可以继续加价：%v", err)
	}
	if err := auction.SettleLot(lot, 3000); err != nil {
		t.Fatalf("落锤失败：%v", err)
	}
	if lot.Status != v1.LotStatus_LOT_STATUS_SETTLED || lot.WinnerUserId != "u1" || lot.GetFinalPrice().GetAmount() != 13000 {
		t.Fatalf("成交状态错误：%+v", lot)
	}
}

func TestLotLifecycleClearsQueueState(t *testing.T) {
	queuedLot := func(id string) *v1.Lot {
		t.Helper()
		lot, err := auction.NewLotFromRequest(id, &v1.CreateLotRequest{
			RoomId:   "demo",
			Title:    "队列拍品",
			ImageUrl: "https://example.com/lot.jpg",
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
		if err := auction.QueueLot(lot, 3); err != nil {
			t.Fatalf("queue lot failed: %v", err)
		}
		return lot
	}

	started := queuedLot("lot_queue_start")
	if err := auction.StartLot(started, 1000); err != nil {
		t.Fatalf("start queued lot failed: %v", err)
	}
	if started.QueueStatus != v1.LotQueueStatus_LOT_QUEUE_STATUS_NONE || started.QueuePosition != 0 {
		t.Fatalf("started lot should leave queue: %+v", started)
	}

	settled := queuedLot("lot_queue_settle")
	if err := auction.StartLot(settled, 1000); err != nil {
		t.Fatalf("start queued lot failed: %v", err)
	}
	if err := auction.AcceptBid(settled, v1.Bid{UserId: "u1", Nickname: "用户1", Amount: &v1.Money{Amount: 11000, Currency: "CNY"}}, 2000); err != nil {
		t.Fatalf("accept bid failed: %v", err)
	}
	if err := auction.SettleLot(settled, 3000); err != nil {
		t.Fatalf("settle lot failed: %v", err)
	}
	if settled.QueueStatus != v1.LotQueueStatus_LOT_QUEUE_STATUS_NONE || settled.QueuePosition != 0 {
		t.Fatalf("settled lot should not remain in queue: %+v", settled)
	}
}

func TestCancelLotStateMachineAllowsPreStartAndRejectsLive(t *testing.T) {
	lot, err := auction.NewLotFromRequest("lot_cancel", &v1.CreateLotRequest{
		RoomId:   "demo",
		Title:    "取消测试拍品",
		ImageUrl: "https://example.com/lot.jpg",
		Rule: &v1.BidRule{
			StartPrice:             &v1.Money{Amount: 0, Currency: "CNY"},
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
	if err := auction.CancelLot(lot, "资料误填", 2000); err != nil {
		t.Fatalf("draft lot should be cancellable, got %v", err)
	}
	if lot.Status != v1.LotStatus_LOT_STATUS_CANCELLED || lot.CancelReason != "资料误填" || lot.CancelledAtUnixMs != 2000 {
		t.Fatalf("draft cancel state mismatch: %+v", lot)
	}

	liveLot, err := auction.NewLotFromRequest("lot_live_cancel", &v1.CreateLotRequest{
		RoomId:   "demo",
		Title:    "开拍后不可取消拍品",
		ImageUrl: "https://example.com/lot.jpg",
		Rule: &v1.BidRule{
			StartPrice:             &v1.Money{Amount: 0, Currency: "CNY"},
			MinIncrement:           &v1.Money{Amount: 1000, Currency: "CNY"},
			DurationSeconds:        300,
			AntiSnipeWindowSeconds: 15,
			AntiSnipeExtendSeconds: 15,
			MaxExtendCount:         3,
		},
	})
	if err != nil {
		t.Fatalf("create live lot failed: %v", err)
	}
	if err := auction.StartLot(liveLot, 1000); err != nil {
		t.Fatalf("start lot failed: %v", err)
	}
	if err := auction.CancelLot(liveLot, "", 2000); err == nil || !strings.Contains(err.Error(), "only draft, ready, or queued lot") {
		t.Fatalf("live lot should be rejected before reason validation, got %v", err)
	}
	if err := auction.CancelLot(liveLot, "主播网络异常", 3000); err == nil || !strings.Contains(err.Error(), "only draft, ready, or queued lot") {
		t.Fatalf("live lot should not be cancellable, got %v", err)
	}

	queuedLot, err := auction.NewLotFromRequest("lot_queue_cancel", &v1.CreateLotRequest{
		RoomId:   "demo",
		Title:    "队列取消拍品",
		ImageUrl: "https://example.com/lot.jpg",
		Rule: &v1.BidRule{
			StartPrice:             &v1.Money{Amount: 0, Currency: "CNY"},
			MinIncrement:           &v1.Money{Amount: 1000, Currency: "CNY"},
			DurationSeconds:        300,
			AntiSnipeWindowSeconds: 15,
			AntiSnipeExtendSeconds: 15,
			MaxExtendCount:         3,
		},
	})
	if err != nil {
		t.Fatalf("create queued lot failed: %v", err)
	}
	queuedLot.Status = v1.LotStatus_LOT_STATUS_QUEUED
	queuedLot.QueueStatus = v1.LotQueueStatus_LOT_QUEUE_STATUS_QUEUED
	queuedLot.QueuePosition = 1
	if err := auction.CancelLot(queuedLot, "", 2000); err == nil || !strings.Contains(err.Error(), "cancel reason") {
		t.Fatalf("empty cancel reason should be rejected, got %v", err)
	}
	if err := auction.CancelLot(queuedLot, "误加入队列", 3000); err != nil {
		t.Fatalf("cancel queued lot failed: %v", err)
	}
	if queuedLot.Status != v1.LotStatus_LOT_STATUS_CANCELLED || queuedLot.QueueStatus != v1.LotQueueStatus_LOT_QUEUE_STATUS_NONE || queuedLot.QueuePosition != 0 || queuedLot.CancelReason != "误加入队列" || queuedLot.CancelledAtUnixMs != 3000 {
		t.Fatalf("queued cancel state mismatch: %+v", queuedLot)
	}
	if err := auction.AcceptBid(queuedLot, v1.Bid{UserId: "u1", Nickname: "用户1", Amount: &v1.Money{Amount: 1000, Currency: "CNY"}}, 4000); err == nil || !strings.Contains(err.Error(), "lot is not live") {
		t.Fatalf("cancelled lot should reject bids, got %v", err)
	}
}

func TestEventForViewerRedactsPublicSettlementAndBuyerIdentity(t *testing.T) {
	event := v1.AuctionEvent{
		Type:   v1.AuctionEventType_AUCTION_EVENT_TYPE_ORDER_CREATED,
		Reason: "order_id=order-secret",
		Lot: &v1.Lot{
			Id:              "lot_privacy",
			Status:          v1.LotStatus_LOT_STATUS_SETTLED,
			CurrentPrice:    &v1.Money{Amount: 12000, Currency: "CNY"},
			LeadingUserId:   "buyer1",
			LeadingNickname: "买家一号",
			WinnerUserId:    "buyer1",
			WinnerNickname:  "买家一号",
			FinalPrice:      &v1.Money{Amount: 12000, Currency: "CNY"},
		},
		Bid: &v1.Bid{
			UserId:   "buyer1",
			Nickname: "买家一号",
			Amount:   &v1.Money{Amount: 12000, Currency: "CNY"},
		},
		Ranking: []*v1.RankingItem{{
			Rank:     1,
			UserId:   "buyer1",
			Nickname: "买家一号",
			Amount:   &v1.Money{Amount: 12000, Currency: "CNY"},
		}},
	}

	publicEvent := auction.EventForViewer(event, auction.LotResultViewer{})
	if publicEvent.Reason != "" {
		t.Fatalf("public order event reason should not leak order data: %q", publicEvent.Reason)
	}
	if publicEvent.GetLot().GetFinalPrice().GetAmount() != 12000 || publicEvent.GetLot().GetWinnerUserId() != "" || publicEvent.GetLot().GetWinnerNickname() != "买***" || publicEvent.GetLot().GetLeadingNickname() != "" {
		t.Fatalf("public settlement lot should keep final price and masked winner nickname but hide buyer id: %+v", publicEvent.GetLot())
	}
	if publicEvent.GetBid().GetUserId() != "" || publicEvent.GetBid().GetNickname() != "买***" {
		t.Fatalf("public bid should mask buyer identity: %+v", publicEvent.GetBid())
	}
	if publicEvent.GetRanking()[0].GetUserId() != "" || publicEvent.GetRanking()[0].GetNickname() != "买***" {
		t.Fatalf("public ranking should mask buyer identity: %+v", publicEvent.GetRanking())
	}

	winnerEvent := auction.EventForViewer(event, auction.LotResultViewer{UserID: "buyer1", Role: v1.UserRole_USER_ROLE_BUYER})
	if winnerEvent.GetLot().GetWinnerUserId() != "buyer1" || winnerEvent.GetBid().GetUserId() != "buyer1" || winnerEvent.GetRanking()[0].GetUserId() != "buyer1" {
		t.Fatalf("winning buyer should see own identity: lot=%+v bid=%+v ranking=%+v", winnerEvent.GetLot(), winnerEvent.GetBid(), winnerEvent.GetRanking())
	}
}

func TestAntiSnipeExtensionKeepsDuelStateInSync(t *testing.T) {
	lot, err := auction.NewLotFromRequest("lot_extend", &v1.CreateLotRequest{
		RoomId:   "demo",
		Title:    "延时测试拍品",
		ImageUrl: "https://example.com/lot.jpg",
		Rule: &v1.BidRule{
			StartPrice:             &v1.Money{Amount: 10000, Currency: "CNY"},
			MinIncrement:           &v1.Money{Amount: 1000, Currency: "CNY"},
			DurationSeconds:        60,
			AntiSnipeWindowSeconds: 10,
			AntiSnipeExtendSeconds: 15,
			MaxExtendCount:         2,
		},
	})
	if err != nil {
		t.Fatalf("create lot failed: %v", err)
	}
	if err := auction.StartLot(lot, 1000); err != nil {
		t.Fatalf("start lot failed: %v", err)
	}
	originalEndsAt := lot.EndsAtUnixMs
	if err := auction.AcceptBid(lot, v1.Bid{UserId: "u1", Nickname: "用户1", Amount: &v1.Money{Amount: 11000, Currency: "CNY"}}, originalEndsAt-5_000); err != nil {
		t.Fatalf("bid in anti-snipe window failed: %v", err)
	}
	if lot.EndsAtUnixMs != originalEndsAt+15_000 {
		t.Fatalf("expected lot ends_at to extend from %d to %d, got %d", originalEndsAt, originalEndsAt+15_000, lot.EndsAtUnixMs)
	}
	if lot.GetDuelState().GetExtendCount() != 1 || lot.GetDuelState().GetLotId() != lot.Id || lot.GetDuelState().GetEndsAtUnixMs() != lot.EndsAtUnixMs || lot.GetDuelState().GetMaxExtendCount() != 2 {
		t.Fatalf("duel state should mirror anti-snipe extension counters: %+v", lot.GetDuelState())
	}

	secondEndsAt := lot.EndsAtUnixMs
	if err := auction.AcceptBid(lot, v1.Bid{UserId: "u2", Nickname: "用户2", Amount: &v1.Money{Amount: 12000, Currency: "CNY"}}, secondEndsAt-5_000); err != nil {
		t.Fatalf("second extension bid failed: %v", err)
	}
	thirdEndsAt := lot.EndsAtUnixMs
	if err := auction.AcceptBid(lot, v1.Bid{UserId: "u3", Nickname: "用户3", Amount: &v1.Money{Amount: 13000, Currency: "CNY"}}, thirdEndsAt-5_000); err != nil {
		t.Fatalf("third bid at max extension boundary failed: %v", err)
	}
	if lot.EndsAtUnixMs != thirdEndsAt || lot.GetDuelState().GetExtendCount() != 2 {
		t.Fatalf("extension should stop at max count, lot=%+v duel=%+v", lot, lot.GetDuelState())
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
		RoomId:   "demo",
		Title:    "测试拍品",
		ImageUrl: "https://example.com/lot.jpg",
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

func TestCreateLotValidatesRequiredImageAndCapPrice(t *testing.T) {
	base := &v1.CreateLotRequest{
		RoomId:   "demo",
		Title:    "封顶价测试拍品",
		ImageUrl: "https://example.com/lot.jpg",
		Rule: &v1.BidRule{
			StartPrice:             &v1.Money{Amount: 10000, Currency: "CNY"},
			MinIncrement:           &v1.Money{Amount: 1000, Currency: "CNY"},
			DurationSeconds:        300,
			AntiSnipeWindowSeconds: 15,
			AntiSnipeExtendSeconds: 15,
			MaxExtendCount:         3,
			CapPrice:               &v1.Money{Amount: 50000, Currency: "CNY"},
		},
	}
	lot, err := auction.NewLotFromRequest("lot_cap", base)
	if err != nil {
		t.Fatalf("valid cap price should pass: %v", err)
	}
	if lot.GetRule().GetCapPrice().GetAmount() != 50000 {
		t.Fatalf("cap price should be kept on lot: %+v", lot.GetRule().GetCapPrice())
	}

	missingImage := proto.Clone(base).(*v1.CreateLotRequest)
	missingImage.ImageUrl = ""
	if _, err := auction.NewLotFromRequest("lot_no_image", missingImage); err == nil || !strings.Contains(err.Error(), "image url") {
		t.Fatalf("missing image should be rejected, got %v", err)
	}

	badCap := proto.Clone(base).(*v1.CreateLotRequest)
	badCap.Rule.CapPrice = &v1.Money{Amount: 9000, Currency: "CNY"}
	if _, err := auction.NewLotFromRequest("lot_bad_cap", badCap); err == nil || !strings.Contains(err.Error(), "greater than start price") {
		t.Fatalf("cap <= start should be rejected, got %v", err)
	}
}

func TestCreateLotKeepsAddLotDetailFields(t *testing.T) {
	lot, err := auction.NewLotFromRequest("lot_detail", &v1.CreateLotRequest{
		RoomId:           "demo",
		Title:            "添加拍品详情测试",
		Description:      "带图库和保障卡的拍品",
		ImageUrl:         "https://tos.example.com/main.jpg",
		GalleryImageUrls: []string{" https://tos.example.com/gallery-a.jpg ", "https://tos.example.com/gallery-b.jpg"},
		Category:         "珠宝首饰",
		Tags:             []string{" 翡翠 ", "收藏级"},
		EstimatePrice:    &v1.Money{Amount: 280000, Currency: "CNY"},
		Stock:            3,
		AfterSaleNotes:   "支持复检",
		Rule: &v1.BidRule{
			StartPrice:             &v1.Money{Amount: 10000, Currency: "CNY"},
			MinIncrement:           &v1.Money{Amount: 1000, Currency: "CNY"},
			DurationSeconds:        300,
			AntiSnipeWindowSeconds: 15,
			AntiSnipeExtendSeconds: 15,
			MaxExtendCount:         3,
		},
		TrustCards: []*v1.TrustRevealCard{{
			Type:     v1.TrustCardType_TRUST_CARD_TYPE_CERTIFICATE,
			Title:    "证书",
			Content:  "NGTC 可查",
			ImageUrl: "https://tos.example.com/cert.jpg",
		}},
	})
	if err != nil {
		t.Fatalf("create lot with add-lot detail fields failed: %v", err)
	}
	if lot.GetGalleryImageUrls()[0] != "https://tos.example.com/gallery-a.jpg" || lot.GetCategory() != "珠宝首饰" || lot.GetTags()[0] != "翡翠" {
		t.Fatalf("detail fields should be normalized and kept: %+v", lot)
	}
	if lot.GetEstimatePrice().GetAmount() != 280000 || lot.GetStock() != 3 || lot.GetAfterSaleNotes() != "支持复检" {
		t.Fatalf("price/stock/after-sale fields should be kept: %+v", lot)
	}
	if lot.GetTrustCards()[0].GetImageUrl() != "https://tos.example.com/cert.jpg" || lot.GetTrustCards()[0].GetLotId() != lot.GetId() {
		t.Fatalf("trust card image and identity should be kept: %+v", lot.GetTrustCards()[0])
	}
}

func TestCreateLotRejectsTemporaryPreviewImageURLs(t *testing.T) {
	base := &v1.CreateLotRequest{
		RoomId:   "demo",
		Title:    "临时图片地址测试",
		ImageUrl: "https://tos.example.com/main.jpg",
		Rule: &v1.BidRule{
			StartPrice:             &v1.Money{Amount: 10000, Currency: "CNY"},
			MinIncrement:           &v1.Money{Amount: 1000, Currency: "CNY"},
			DurationSeconds:        300,
			AntiSnipeWindowSeconds: 15,
			AntiSnipeExtendSeconds: 15,
			MaxExtendCount:         3,
		},
	}

	cases := []struct {
		name    string
		mutate  func(*v1.CreateLotRequest)
		wantErr string
	}{
		{
			name: "main blob",
			mutate: func(req *v1.CreateLotRequest) {
				req.ImageUrl = "blob:http://localhost/preview"
			},
			wantErr: "imageUrl",
		},
		{
			name: "gallery data url",
			mutate: func(req *v1.CreateLotRequest) {
				req.GalleryImageUrls = []string{"data:image/png;base64,abc"}
			},
			wantErr: "galleryImageUrls",
		},
		{
			name: "trust card blob",
			mutate: func(req *v1.CreateLotRequest) {
				req.TrustCards = []*v1.TrustRevealCard{{
					Type:     v1.TrustCardType_TRUST_CARD_TYPE_CERTIFICATE,
					Title:    "证书",
					Content:  "NGTC 可查",
					ImageUrl: "blob:http://localhost/cert",
				}}
			},
			wantErr: "trustCards.imageUrl",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := proto.Clone(base).(*v1.CreateLotRequest)
			tc.mutate(req)
			if _, err := auction.NewLotFromRequest("lot_bad_preview", req); err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("expected %s error, got %v", tc.wantErr, err)
			}
		})
	}
}

func TestListLotsByQueryViewsRespectPageBoundaries(t *testing.T) {
	store := newTestStore()
	uc := auction.NewAuctionUsecase(store, store, store, nil)
	ctx := context.Background()

	createLotWithStatus := func(id string, status v1.LotStatus) {
		t.Helper()
		lot, err := auction.NewLotFromRequest(id, &v1.CreateLotRequest{
			RoomId:   "room_views",
			Title:    "视图边界 " + id,
			ImageUrl: "https://example.com/" + id + ".jpg",
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
			t.Fatalf("create lot %s failed: %v", id, err)
		}
		lot.Status = status
		if status == v1.LotStatus_LOT_STATUS_QUEUED {
			lot.QueueStatus = v1.LotQueueStatus_LOT_QUEUE_STATUS_QUEUED
			lot.QueuePosition = 1
		}
		if status == v1.LotStatus_LOT_STATUS_CANCELLED {
			lot.CancelReason = "误加入队列"
			lot.CancelledAtUnixMs = 1000
		}
		if err := store.Create(ctx, lot, "owner", nil); err != nil {
			t.Fatalf("store lot %s failed: %v", id, err)
		}
	}

	createLotWithStatus("lot_draft", v1.LotStatus_LOT_STATUS_DRAFT)
	createLotWithStatus("lot_ready", v1.LotStatus_LOT_STATUS_READY)
	createLotWithStatus("lot_queued", v1.LotStatus_LOT_STATUS_QUEUED)
	createLotWithStatus("lot_live", v1.LotStatus_LOT_STATUS_LIVE)
	createLotWithStatus("lot_settled", v1.LotStatus_LOT_STATUS_SETTLED)
	createLotWithStatus("lot_cancelled", v1.LotStatus_LOT_STATUS_CANCELLED)
	createLotWithStatus("lot_failed", v1.LotStatus_LOT_STATUS_FAILED)

	library, err := uc.ListLotsByQuery(ctx, auction.LotQuery{MainAccountID: testMainAccountID, RoomID: "room_views", View: "library", Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("list library failed: %v", err)
	}
	if library.Total != 2 {
		t.Fatalf("library should only include draft/ready, got total=%d lots=%v", library.Total, testLotIDs(library.Lots))
	}

	current, err := uc.ListLotsByQuery(ctx, auction.LotQuery{MainAccountID: testMainAccountID, RoomID: "room_views", View: "current", Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("list current failed: %v", err)
	}
	if current.Total != 4 {
		t.Fatalf("current should exclude terminal records, got total=%d lots=%v", current.Total, testLotIDs(current.Lots))
	}

	history, err := uc.ListLotsByQuery(ctx, auction.LotQuery{MainAccountID: testMainAccountID, RoomID: "room_views", View: "history", Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("list history failed: %v", err)
	}
	if history.Total != 3 {
		t.Fatalf("history should include settled/cancelled/failed, got total=%d lots=%v", history.Total, testLotIDs(history.Lots))
	}

	cancelledInLibrary, err := uc.ListLotsByQuery(ctx, auction.LotQuery{MainAccountID: testMainAccountID, RoomID: "room_views", View: "library", Status: v1.LotStatus_LOT_STATUS_CANCELLED, Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("list cancelled in library failed: %v", err)
	}
	if cancelledInLibrary.Total != 0 {
		t.Fatalf("cancelled lots must not leak into library view, got total=%d", cancelledInLibrary.Total)
	}

	if _, err := uc.ListLotsByQuery(ctx, auction.LotQuery{MainAccountID: testMainAccountID, RoomID: "room_views", View: "unknown"}); err == nil || !strings.Contains(err.Error(), "unsupported lot list view") {
		t.Fatalf("invalid view should failfast, got %v", err)
	}
}

func TestPublicLotReadsRequireActiveRoom(t *testing.T) {
	store := newTestStore()
	store.strictRooms = true
	uc := auction.NewAuctionUsecase(store, store, store, nil)
	ctx := context.Background()
	lot, err := auction.NewLotFromRequest("lot_orphan_room", &v1.CreateLotRequest{
		RoomId:   "orphan-room",
		Title:    "孤儿房间拍品",
		ImageUrl: "https://example.com/orphan.jpg",
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
	if err := store.Create(ctx, lot, "owner", nil); err != nil {
		t.Fatalf("store lot failed: %v", err)
	}

	if _, err := uc.ListLots(ctx, "orphan-room", 0); !apperr.IsNotFound(err) {
		t.Fatalf("public list should reject orphan room, got %v", err)
	}
	if _, err := uc.Snapshot(ctx, "orphan-room"); !apperr.IsNotFound(err) {
		t.Fatalf("snapshot should reject orphan room, got %v", err)
	}
	if _, err := uc.GetLot(ctx, "lot_orphan_room"); !apperr.IsNotFound(err) {
		t.Fatalf("get lot should reject orphan room, got %v", err)
	}
}

func TestPublicRoomsRequireVisibleAuctionContent(t *testing.T) {
	store := newTestStore()
	uc := auction.NewAuctionUsecase(store, store, store, nil)
	ctx := context.Background()

	cases := []struct {
		id         string
		status     auction.RoomStatus
		lotStatus  []v1.LotStatus
		wantPublic bool
	}{
		{id: "empty", status: auction.RoomStatusActive},
		{id: "draft", status: auction.RoomStatusActive, lotStatus: []v1.LotStatus{v1.LotStatus_LOT_STATUS_DRAFT}},
		{id: "ready", status: auction.RoomStatusActive, lotStatus: []v1.LotStatus{v1.LotStatus_LOT_STATUS_READY}},
		{id: "terminal", status: auction.RoomStatusActive, lotStatus: []v1.LotStatus{v1.LotStatus_LOT_STATUS_SETTLED, v1.LotStatus_LOT_STATUS_CANCELLED, v1.LotStatus_LOT_STATUS_FAILED}},
		{id: "queued", status: auction.RoomStatusActive, lotStatus: []v1.LotStatus{v1.LotStatus_LOT_STATUS_QUEUED}, wantPublic: true},
		{id: "live", status: auction.RoomStatusActive, lotStatus: []v1.LotStatus{v1.LotStatus_LOT_STATUS_LIVE}, wantPublic: true},
		{id: "extended", status: auction.RoomStatusActive, lotStatus: []v1.LotStatus{v1.LotStatus_LOT_STATUS_EXTENDED}, wantPublic: true},
		{id: "disabled-queued", status: auction.RoomStatusDisabled, lotStatus: []v1.LotStatus{v1.LotStatus_LOT_STATUS_QUEUED}},
	}

	wantPublicIDs := make([]string, 0)
	for _, tc := range cases {
		roomID := "room_" + tc.id
		store.rooms[roomID] = auction.Room{
			ID:              roomID,
			MainAccountID:   testMainAccountID,
			Name:            "直播间 " + tc.id,
			Platform:        "douyin",
			Status:          tc.status,
			CreatedByUserID: "owner",
			CreatedAtUnixMs: int64(len(store.rooms) + 1),
			UpdatedAtUnixMs: int64(len(store.rooms) + 1),
		}
		for index, status := range tc.lotStatus {
			store.lots[roomID+"_lot_"+strconv.Itoa(index)] = &v1.Lot{
				Id:            roomID + "_lot_" + strconv.Itoa(index),
				RoomId:        roomID,
				MainAccountId: testMainAccountID,
				Title:         "可见性拍品",
				Status:        status,
			}
		}
		if tc.wantPublic {
			wantPublicIDs = append(wantPublicIDs, roomID)
		}
	}
	sort.Strings(wantPublicIDs)

	publicRooms, err := uc.ListRooms(ctx, auction.RoomQuery{PublicOnly: true, PublicVisibleOnly: true})
	if err != nil {
		t.Fatalf("list public rooms failed: %v", err)
	}
	if got := testRoomIDs(publicRooms); strings.Join(got, ",") != strings.Join(wantPublicIDs, ",") {
		t.Fatalf("public rooms mismatch got=%v want=%v", got, wantPublicIDs)
	}

	adminRooms, err := uc.ListRooms(ctx, auction.RoomQuery{MainAccountID: testMainAccountID})
	if err != nil {
		t.Fatalf("list admin rooms failed: %v", err)
	}
	if len(adminRooms) != len(cases) {
		t.Fatalf("admin room list should remain unfiltered, got=%d want=%d ids=%v", len(adminRooms), len(cases), testRoomIDs(adminRooms))
	}
}

func TestBuildRealtimeRankingHonorsEnvLimit(t *testing.T) {
	t.Setenv("AUCTION_REALTIME_RANKING_LIMIT", "2")
	ranking := auction.BuildRealtimeRanking([]v1.Bid{
		{UserId: "u1", Nickname: "用户1", Amount: &v1.Money{Amount: 11000, Currency: "CNY"}, CreatedAtUnixMs: 1},
		{UserId: "u2", Nickname: "用户2", Amount: &v1.Money{Amount: 13000, Currency: "CNY"}, CreatedAtUnixMs: 2},
		{UserId: "u3", Nickname: "用户3", Amount: &v1.Money{Amount: 12000, Currency: "CNY"}, CreatedAtUnixMs: 3},
	})
	if len(ranking) != 2 {
		t.Fatalf("ranking should be capped to 2, got %d", len(ranking))
	}
	if ranking[0].UserId != "u2" || ranking[1].UserId != "u3" {
		t.Fatalf("ranking should keep sorted top bidders, got %+v", ranking)
	}
}

func TestConcurrentBidSmokeMaintainsLeaderRankingLimitIdempotencyAndCapOrder(t *testing.T) {
	t.Setenv("AUCTION_REALTIME_RANKING_LIMIT", "5")
	store := newTestStore()
	pub := &testPublisher{}
	uc := auction.NewAuctionUsecase(store, store, store, pub)
	ctx := context.Background()

	const (
		concurrency  = 100
		startPrice   = int64(10000)
		minIncrement = int64(1000)
	)
	capPrice := startPrice + concurrency*minIncrement
	room, err := uc.EnsureDefaultRoom(ctx, testMainAccountID, "test-owner")
	if err != nil {
		t.Fatalf("ensure default room failed: %v", err)
	}
	lot, err := uc.CreateLot(ctx, &v1.CreateLotRequest{
		RoomId:   room.ID,
		Title:    "并发封顶拍品",
		ImageUrl: "https://example.com/lot.jpg",
		Rule: &v1.BidRule{
			StartPrice:             &v1.Money{Amount: startPrice, Currency: "CNY"},
			MinIncrement:           &v1.Money{Amount: minIncrement, Currency: "CNY"},
			CapPrice:               &v1.Money{Amount: capPrice, Currency: "CNY"},
			DurationSeconds:        300,
			AntiSnipeWindowSeconds: 15,
			AntiSnipeExtendSeconds: 15,
			MaxExtendCount:         3,
		},
	}, testMainAccountID, "test-owner")
	if err != nil {
		t.Fatalf("create lot failed: %v", err)
	}
	if _, err := uc.StartLot(ctx, lot.Id, testMainAccountID); err != nil {
		t.Fatalf("start lot failed: %v", err)
	}
	publicRooms, err := uc.ListRooms(ctx, auction.RoomQuery{PublicOnly: true, PublicVisibleOnly: true})
	if err != nil || len(publicRooms) != 1 || publicRooms[0].ID != lot.RoomId {
		t.Fatalf("started lot should make room public: rooms=%+v err=%v", publicRooms, err)
	}

	type bidAttempt struct {
		index    int
		userID   string
		key      string
		amount   int64
		bidID    string
		accepted bool
		err      error
		latency  time.Duration
	}
	start := make(chan struct{})
	results := make(chan bidAttempt, concurrency)
	var wg sync.WaitGroup
	for i := 1; i <= concurrency; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			attempt := bidAttempt{
				index:  i,
				userID: "buyer-" + strconv.Itoa(i),
				key:    "concurrent-bid-" + strconv.Itoa(i),
				amount: startPrice + int64(i)*minIncrement,
			}
			started := time.Now()
			_, bid, _, err := uc.PlaceBid(ctx, &v1.PlaceBidRequest{
				LotId:          lot.Id,
				Amount:         &v1.Money{Amount: attempt.amount, Currency: "CNY"},
				IdempotencyKey: attempt.key,
			}, attempt.userID, "买家"+strconv.Itoa(i))
			attempt.latency = time.Since(started)
			attempt.err = err
			if err == nil && bid != nil {
				attempt.accepted = true
				attempt.bidID = bid.Id
			}
			results <- attempt
		}()
	}
	close(start)
	wg.Wait()
	close(results)

	var attempts []bidAttempt
	var highest bidAttempt
	accepted := 0
	rejected := 0
	for attempt := range results {
		attempts = append(attempts, attempt)
		if attempt.accepted {
			accepted++
			if !highest.accepted || attempt.amount > highest.amount {
				highest = attempt
			}
			continue
		}
		rejected++
		if attempt.err == nil {
			t.Fatalf("nil bid with nil error for attempt %+v", attempt)
		}
	}
	if accepted == 0 {
		t.Fatalf("expected at least one accepted bid, attempts=%+v", attempts)
	}
	if !highest.accepted || highest.amount != capPrice || highest.userID != "buyer-100" {
		t.Fatalf("highest accepted bid mismatch: highest=%+v cap=%d", highest, capPrice)
	}

	finalLot, err := store.FindByID(ctx, lot.Id)
	if err != nil {
		t.Fatalf("find final lot failed: %v", err)
	}
	if finalLot.Status != v1.LotStatus_LOT_STATUS_SETTLED || finalLot.WinnerUserId != highest.userID || finalLot.GetFinalPrice().GetAmount() != capPrice {
		t.Fatalf("cap bid should settle with highest accepted bidder: lot=%+v highest=%+v", finalLot, highest)
	}
	bids, err := store.ListByLot(ctx, lot.Id)
	if err != nil {
		t.Fatalf("list bids failed: %v", err)
	}
	if len(bids) != accepted {
		t.Fatalf("accepted count must match persisted bids: accepted=%d persisted=%d", accepted, len(bids))
	}
	ranking := auction.BuildRealtimeRanking(bids)
	if len(ranking) == 0 || len(ranking) > 5 {
		t.Fatalf("realtime ranking should be non-empty and capped to 5, got %d: %+v", len(ranking), ranking)
	}
	for i := 1; i < len(ranking); i++ {
		if ranking[i-1].GetAmount().GetAmount() < ranking[i].GetAmount().GetAmount() {
			t.Fatalf("ranking must be sorted descending: %+v", ranking)
		}
	}
	if ranking[0].UserId != highest.userID || ranking[0].GetAmount().GetAmount() != highest.amount {
		t.Fatalf("ranking leader mismatch: ranking=%+v highest=%+v", ranking[0], highest)
	}

	beforeReplayCount := len(bids)
	_, replayed, replayRanking, err := uc.PlaceBid(ctx, &v1.PlaceBidRequest{
		LotId:          lot.Id,
		Amount:         &v1.Money{Amount: highest.amount, Currency: "CNY"},
		IdempotencyKey: highest.key,
	}, highest.userID, "买家"+strconv.Itoa(highest.index))
	if err != nil || replayed == nil || replayed.Id != highest.bidID {
		t.Fatalf("idempotent replay should return original bid: bid=%+v highest=%+v ranking=%+v err=%v", replayed, highest, replayRanking, err)
	}
	afterReplayBids, err := store.ListByLot(ctx, lot.Id)
	if err != nil {
		t.Fatalf("list bids after replay failed: %v", err)
	}
	if len(afterReplayBids) != beforeReplayCount || len(replayRanking) == 0 || len(replayRanking) > 5 {
		t.Fatalf("idempotent replay must not append bid and must keep ranking cap: before=%d after=%d ranking=%d", beforeReplayCount, len(afterReplayBids), len(replayRanking))
	}

	orders, err := uc.ListOrders(ctx, auction.OrderQuery{Page: 1, PageSize: 10})
	if err != nil || orders.Total != 1 || len(orders.Orders) != 1 {
		t.Fatalf("cap settlement should create exactly one order: orders=%+v err=%v", orders, err)
	}
	if orders.Orders[0].BuyerUserID != highest.userID || orders.Orders[0].Amount != capPrice {
		t.Fatalf("created order should belong to highest accepted bidder: order=%+v highest=%+v", orders.Orders[0], highest)
	}

	latencies := make([]time.Duration, 0, len(attempts))
	for _, attempt := range attempts {
		latencies = append(latencies, attempt.latency)
	}
	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })
	t.Logf("concurrent bid smoke: total=%d accepted=%d rejected=%d p50=%s p95=%s p99=%s final_price=%d leader=%s ranking_len=%d",
		len(attempts),
		accepted,
		rejected,
		latencies[len(latencies)*50/100],
		latencies[len(latencies)*95/100],
		latencies[len(latencies)*99/100],
		finalLot.GetFinalPrice().GetAmount(),
		finalLot.WinnerUserId,
		len(ranking),
	)
}

func testLotIDs(lots []*v1.Lot) []string {
	ids := make([]string, 0, len(lots))
	for _, lot := range lots {
		ids = append(ids, lot.GetId())
	}
	return ids
}

func testRoomIDs(rooms []auction.Room) []string {
	ids := make([]string, 0, len(rooms))
	for _, room := range rooms {
		ids = append(ids, room.ID)
	}
	sort.Strings(ids)
	return ids
}

func TestPlaceBidRejectsMissingUserInsteadOfDefaulting(t *testing.T) {
	store := newTestStore()
	uc := auction.NewAuctionUsecase(store, store, store, nil)
	ctx := context.Background()

	lot, err := uc.CreateLot(ctx, &v1.CreateLotRequest{
		RoomId:   "demo",
		Title:    "测试拍品",
		ImageUrl: "https://example.com/lot.jpg",
		Rule: &v1.BidRule{
			StartPrice:             &v1.Money{Amount: 10000, Currency: "CNY"},
			MinIncrement:           &v1.Money{Amount: 1000, Currency: "CNY"},
			DurationSeconds:        300,
			AntiSnipeWindowSeconds: 15,
			AntiSnipeExtendSeconds: 15,
			MaxExtendCount:         3,
		},
	}, testMainAccountID, "test-owner")
	if err != nil {
		t.Fatalf("create lot failed: %v", err)
	}
	if _, err := uc.StartLot(ctx, lot.Id, testMainAccountID); err != nil {
		t.Fatalf("start lot failed: %v", err)
	}

	_, _, _, err = uc.PlaceBid(ctx, &v1.PlaceBidRequest{
		LotId:  lot.Id,
		Amount: &v1.Money{Amount: 11000, Currency: "CNY"},
	}, "", "用户1")
	if err == nil || !strings.Contains(err.Error(), "user id is required") {
		t.Fatalf("expected user id error, got %v", err)
	}
}

func TestPlaceBidStatsOnlyCountAcceptedBidsAndUniqueParticipants(t *testing.T) {
	store := newTestStore()
	uc := auction.NewAuctionUsecase(store, store, store, nil)
	ctx := context.Background()

	lot, err := uc.CreateLot(ctx, &v1.CreateLotRequest{
		RoomId:   "room_stats",
		Title:    "统计测试拍品",
		ImageUrl: "https://example.com/lot.jpg",
		Rule: &v1.BidRule{
			StartPrice:             &v1.Money{Amount: 10000, Currency: "CNY"},
			MinIncrement:           &v1.Money{Amount: 1000, Currency: "CNY"},
			DurationSeconds:        300,
			AntiSnipeWindowSeconds: 15,
			AntiSnipeExtendSeconds: 15,
			MaxExtendCount:         3,
		},
	}, testMainAccountID, "test-owner")
	if err != nil {
		t.Fatalf("create lot failed: %v", err)
	}
	if _, err := uc.StartLot(ctx, lot.Id, testMainAccountID); err != nil {
		t.Fatalf("start lot failed: %v", err)
	}
	if _, bid, _, err := uc.PlaceBid(ctx, &v1.PlaceBidRequest{LotId: lot.Id, Amount: &v1.Money{Amount: 11000, Currency: "CNY"}, IdempotencyKey: "stats-1"}, "u1", "用户1"); err != nil || bid == nil {
		t.Fatalf("first bid failed: bid=%+v err=%v", bid, err)
	}
	if _, _, _, err := uc.PlaceBid(ctx, &v1.PlaceBidRequest{LotId: lot.Id, Amount: &v1.Money{Amount: 12000, Currency: "CNY"}, IdempotencyKey: "stats-leading-repeat"}, "u1", "用户1"); err == nil || !strings.Contains(err.Error(), "leading bidder must wait") {
		t.Fatalf("leading bidder repeat should be rejected, got %v", err)
	}
	snapshot, err := uc.Snapshot(ctx, "room_stats")
	if err != nil {
		t.Fatalf("snapshot after rejected repeat failed: %v", err)
	}
	if snapshot.GetCurrentLot().GetStats().GetBidCount() != 1 || snapshot.GetCurrentLot().GetStats().GetParticipantCount() != 1 {
		t.Fatalf("rejected leading repeat must not change stats: %+v", snapshot.GetCurrentLot().GetStats())
	}
	if _, bid, _, err := uc.PlaceBid(ctx, &v1.PlaceBidRequest{LotId: lot.Id, Amount: &v1.Money{Amount: 12000, Currency: "CNY"}, IdempotencyKey: "stats-2"}, "u2", "用户2"); err != nil || bid == nil {
		t.Fatalf("second user bid failed: bid=%+v err=%v", bid, err)
	}
	if _, bid, _, err := uc.PlaceBid(ctx, &v1.PlaceBidRequest{LotId: lot.Id, Amount: &v1.Money{Amount: 13000, Currency: "CNY"}, IdempotencyKey: "stats-3"}, "u1", "用户1"); err != nil || bid == nil {
		t.Fatalf("outbid user should be allowed to bid again: bid=%+v err=%v", bid, err)
	}
	snapshot, err = uc.Snapshot(ctx, "room_stats")
	if err != nil {
		t.Fatalf("snapshot after accepted bids failed: %v", err)
	}
	if snapshot.GetCurrentLot().GetStats().GetBidCount() != 3 || snapshot.GetCurrentLot().GetStats().GetParticipantCount() != 2 {
		t.Fatalf("stats should count accepted bids and unique participants: %+v", snapshot.GetCurrentLot().GetStats())
	}
}

func TestLotSaveRejectsStaleExpectedVersion(t *testing.T) {
	store := newTestStore()
	ctx := context.Background()
	lot, err := auction.NewLotFromRequest("lot_conflict", &v1.CreateLotRequest{
		RoomId:   "demo",
		Title:    "版本冲突测试",
		ImageUrl: "https://example.com/lot.jpg",
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
	if err := store.Create(ctx, lot, "", nil); err != nil {
		t.Fatalf("persist lot failed: %v", err)
	}

	fresh, err := store.FindByID(ctx, lot.Id)
	if err != nil {
		t.Fatalf("find lot failed: %v", err)
	}
	expectedVersion := fresh.Version
	if err := auction.StartLot(fresh, 1000); err != nil {
		t.Fatalf("start lot failed: %v", err)
	}
	if err := store.Save(ctx, fresh, expectedVersion, nil); err != nil {
		t.Fatalf("first save failed: %v", err)
	}

	stale, err := store.FindByID(ctx, lot.Id)
	if err != nil {
		t.Fatalf("find lot failed: %v", err)
	}
	if err := auction.AcceptBid(stale, v1.Bid{UserId: "u1", Nickname: "用户1", Amount: &v1.Money{Amount: 11000, Currency: "CNY"}}, 2000); err != nil {
		t.Fatalf("accept bid failed: %v", err)
	}
	if err := store.Save(ctx, stale, expectedVersion, nil); err == nil || !strings.Contains(err.Error(), "lot version conflict") {
		t.Fatalf("expected stale version conflict, got %v", err)
	}
}

func TestCommitAcceptedBidRejectsStaleLotWithoutAppendingBid(t *testing.T) {
	store := newTestStore()
	ctx := context.Background()
	lot, err := auction.NewLotFromRequest("lot_bid_conflict", &v1.CreateLotRequest{
		RoomId:   "demo",
		Title:    "出价事务冲突测试",
		ImageUrl: "https://example.com/lot.jpg",
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
	if err := store.Create(ctx, lot, "", nil); err != nil {
		t.Fatalf("persist lot failed: %v", err)
	}
	fresh, err := store.FindByID(ctx, lot.Id)
	if err != nil {
		t.Fatalf("find lot failed: %v", err)
	}
	if err := auction.StartLot(fresh, 1000); err != nil {
		t.Fatalf("start lot failed: %v", err)
	}
	if err := store.Save(ctx, fresh, fresh.Version-1, nil); err != nil {
		t.Fatalf("save started lot failed: %v", err)
	}

	stale := proto.Clone(fresh).(*v1.Lot)
	bid := v1.Bid{Id: "bid_conflict", LotId: stale.Id, UserId: "u1", Nickname: "用户1", Amount: &v1.Money{Amount: 11000, Currency: "CNY"}, CreatedAtUnixMs: 2000}
	if err := auction.AcceptBid(stale, bid, 2000); err != nil {
		t.Fatalf("accept bid failed: %v", err)
	}
	if err := store.CommitAcceptedBid(ctx, bid, stale, fresh.Version-1, "idem-conflict", nil, nil); err == nil || !strings.Contains(err.Error(), "lot version conflict") {
		t.Fatalf("expected lot version conflict, got %v", err)
	}
	bids, err := store.ListByLot(ctx, stale.Id)
	if err != nil {
		t.Fatalf("list bids failed: %v", err)
	}
	if len(bids) != 0 {
		t.Fatalf("stale commit must not append bid, got %+v", bids)
	}
}

type testStore struct {
	mu                         sync.RWMutex
	lots                       map[string]*v1.Lot
	rooms                      map[string]auction.Room
	strictRooms                bool
	bidsByLot                  map[string][]v1.Bid
	idemByScope                map[string]v1.Bid
	ordersByID                 map[string]auction.Order
	orderIDByLot               map[string]string
	paymentsByOrder            map[string]map[string]auction.Payment
	events                     []v1.AuctionEvent
	failNextBidCommit          error
	beforeBidCommitFailure     func(s *testStore, bid v1.Bid, lot *v1.Lot, idempotencyKey string, order *auction.Order, events []v1.AuctionEvent)
	failNextPaymentCommit      error
	beforePaymentCommitFailure func(s *testStore, payment auction.Payment, order auction.Order, events []v1.AuctionEvent)
}

const testMainAccountID = "main-test"

func newTestStore() *testStore {
	return &testStore{
		lots:            make(map[string]*v1.Lot),
		rooms:           make(map[string]auction.Room),
		bidsByLot:       make(map[string][]v1.Bid),
		idemByScope:     make(map[string]v1.Bid),
		ordersByID:      make(map[string]auction.Order),
		orderIDByLot:    make(map[string]string),
		paymentsByOrder: make(map[string]map[string]auction.Payment),
	}
}

func (s *testStore) EnsureDefaultRoom(ctx context.Context, mainAccountID, createdByUserID string, nowMs int64) (*auction.Room, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, room := range s.rooms {
		if room.MainAccountID == mainAccountID && room.Status == auction.RoomStatusActive {
			return cloneTestRoom(room), nil
		}
	}
	room := auction.Room{
		ID:              "room_default_" + mainAccountID,
		MainAccountID:   mainAccountID,
		Name:            "默认直播间",
		Platform:        "douyin",
		Status:          auction.RoomStatusActive,
		CreatedByUserID: createdByUserID,
		CreatedAtUnixMs: nowMs,
		UpdatedAtUnixMs: nowMs,
	}
	s.rooms[room.ID] = room
	return cloneTestRoom(room), nil
}

func (s *testStore) ListRooms(ctx context.Context, query auction.RoomQuery) ([]auction.Room, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rooms := make([]auction.Room, 0, len(s.rooms))
	for _, room := range s.rooms {
		if query.MainAccountID != "" && room.MainAccountID != query.MainAccountID {
			continue
		}
		if query.PublicOnly && room.Status != auction.RoomStatusActive {
			continue
		}
		if query.PublicVisibleOnly && !s.roomHasPublicVisibleLotLocked(room) {
			continue
		}
		rooms = append(rooms, room)
	}
	sort.Slice(rooms, func(i, j int) bool { return rooms[i].ID < rooms[j].ID })
	return rooms, nil
}

func (s *testStore) roomHasPublicVisibleLotLocked(room auction.Room) bool {
	for _, lot := range s.lots {
		if lot.RoomId == room.ID && lot.MainAccountId == room.MainAccountID && auction.IsPublicVisibleLotStatus(lot.Status) {
			return true
		}
	}
	return false
}

func (s *testStore) FindRoomByID(ctx context.Context, roomID string) (*auction.Room, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if room, ok := s.rooms[roomID]; ok {
		return cloneTestRoom(room), true, nil
	}
	if s.strictRooms {
		return nil, false, nil
	}
	return &auction.Room{ID: roomID, MainAccountID: testMainAccountID, Name: "测试直播间", Platform: "douyin", Status: auction.RoomStatusActive}, true, nil
}

func cloneTestRoom(room auction.Room) *auction.Room {
	next := room
	return &next
}

func (s *testStore) Create(ctx context.Context, lot *v1.Lot, ownerUserID string, events []v1.AuctionEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if lot.MainAccountId == "" {
		lot.MainAccountId = testMainAccountID
	}
	s.lots[lot.Id] = proto.Clone(lot).(*v1.Lot)
	s.events = append(s.events, events...)
	return nil
}

func (s *testStore) Save(ctx context.Context, lot *v1.Lot, expectedVersion int64, events []v1.AuctionEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	current, ok := s.lots[lot.Id]
	if !ok {
		return errors.New("lot not found")
	}
	if expectedVersion <= 0 {
		return errors.New("lot expected version is required")
	}
	if current.Version != expectedVersion {
		return apperr.ErrLotVersionConflict
	}
	s.lots[lot.Id] = proto.Clone(lot).(*v1.Lot)
	s.events = append(s.events, events...)
	return nil
}

func (s *testStore) AttachAssets(ctx context.Context, ownerUserID string, lot *v1.Lot) error {
	return nil
}

func (s *testStore) FindByID(ctx context.Context, lotID string) (*v1.Lot, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	lot, ok := s.lots[lotID]
	if !ok {
		return nil, errors.New("lot not found")
	}
	return proto.Clone(lot).(*v1.Lot), nil
}

func (s *testStore) List(ctx context.Context, roomID string, status v1.LotStatus) ([]*v1.Lot, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	lots := make([]*v1.Lot, 0, len(s.lots))
	for _, lot := range s.lots {
		if roomID != "" && lot.RoomId != roomID {
			continue
		}
		if status != 0 && lot.Status != status && !(status == v1.LotStatus_LOT_STATUS_LIVE && lot.Status == v1.LotStatus_LOT_STATUS_EXTENDED) {
			continue
		}
		lots = append(lots, proto.Clone(lot).(*v1.Lot))
	}
	sort.Slice(lots, func(i, j int) bool { return lots[i].Id < lots[j].Id })
	return lots, nil
}

func (s *testStore) ListLots(ctx context.Context, query auction.LotQuery) (auction.LotList, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	query.Page, query.PageSize = auction.NormalizePagination(query.Page, query.PageSize)
	lots := make([]*v1.Lot, 0, len(s.lots))
	keyword := strings.ToLower(strings.TrimSpace(query.Keyword))
	for _, lot := range s.lots {
		if query.MainAccountID != "" && lot.MainAccountId != query.MainAccountID {
			continue
		}
		if query.RoomID != "" && lot.RoomId != query.RoomID {
			continue
		}
		if query.Status != v1.LotStatus_LOT_STATUS_UNSPECIFIED && lot.Status != query.Status {
			continue
		}
		if strings.EqualFold(query.View, "current") && !isCurrentLotStatusForTest(lot.Status) {
			continue
		}
		if strings.EqualFold(query.View, "history") && !isHistoryLotStatusForTest(lot.Status) {
			continue
		}
		if strings.EqualFold(query.View, "library") && !isLibraryLotStatusForTest(lot.Status) {
			continue
		}
		if keyword != "" && !strings.Contains(strings.ToLower(lot.Id+" "+lot.Title+" "+lot.Description+" "+lot.CancelReason), keyword) {
			continue
		}
		lots = append(lots, proto.Clone(lot).(*v1.Lot))
	}
	sort.Slice(lots, func(i, j int) bool { return lots[i].Id < lots[j].Id })
	total := int64(len(lots))
	start := auction.PageOffset(query.Page, query.PageSize)
	if start >= len(lots) {
		return auction.LotList{Lots: []*v1.Lot{}, Total: total, Page: query.Page, PageSize: query.PageSize}, nil
	}
	end := start + query.PageSize
	if end > len(lots) {
		end = len(lots)
	}
	return auction.LotList{Lots: lots[start:end], Total: total, Page: query.Page, PageSize: query.PageSize}, nil
}

func isCurrentLotStatusForTest(status v1.LotStatus) bool {
	switch status {
	case v1.LotStatus_LOT_STATUS_DRAFT, v1.LotStatus_LOT_STATUS_READY, v1.LotStatus_LOT_STATUS_QUEUED, v1.LotStatus_LOT_STATUS_LIVE, v1.LotStatus_LOT_STATUS_EXTENDED:
		return true
	default:
		return false
	}
}

func isHistoryLotStatusForTest(status v1.LotStatus) bool {
	switch status {
	case v1.LotStatus_LOT_STATUS_SETTLED, v1.LotStatus_LOT_STATUS_CANCELLED, v1.LotStatus_LOT_STATUS_FAILED:
		return true
	default:
		return false
	}
}

func isLibraryLotStatusForTest(status v1.LotStatus) bool {
	return status == v1.LotStatus_LOT_STATUS_DRAFT || status == v1.LotStatus_LOT_STATUS_READY
}

func (s *testStore) ListExpiredOpen(ctx context.Context, nowMs int64, limit int) ([]*v1.Lot, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	lots := make([]*v1.Lot, 0, len(s.lots))
	for _, lot := range s.lots {
		if !auction.IsAuctionOpenStatus(lot.Status) || lot.EndsAtUnixMs == 0 || lot.EndsAtUnixMs > nowMs {
			continue
		}
		lots = append(lots, proto.Clone(lot).(*v1.Lot))
	}
	sort.Slice(lots, func(i, j int) bool {
		if lots[i].EndsAtUnixMs == lots[j].EndsAtUnixMs {
			return lots[i].Id < lots[j].Id
		}
		return lots[i].EndsAtUnixMs < lots[j].EndsAtUnixMs
	})
	if limit > 0 && len(lots) > limit {
		lots = lots[:limit]
	}
	return lots, nil
}

func (s *testStore) CommitAcceptedBid(ctx context.Context, bid v1.Bid, lot *v1.Lot, expectedLotVersion int64, idempotencyKey string, order *auction.Order, events []v1.AuctionEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.failNextBidCommit != nil {
		if s.beforeBidCommitFailure != nil {
			s.beforeBidCommitFailure(s, bid, lot, idempotencyKey, order, events)
		}
		err := s.failNextBidCommit
		s.failNextBidCommit = nil
		s.beforeBidCommitFailure = nil
		return err
	}
	current, ok := s.lots[lot.Id]
	if !ok {
		return errors.New("lot not found")
	}
	if expectedLotVersion <= 0 {
		return errors.New("lot expected version is required")
	}
	if current.Version != expectedLotVersion {
		return apperr.ErrLotVersionConflict
	}
	committedBids := append(s.bidsByLot[bid.LotId], bid)
	committedLot := proto.Clone(lot).(*v1.Lot)
	committedLot.Stats = testLotStats(committedBids)
	s.lots[lot.Id] = committedLot
	s.bidsByLot[bid.LotId] = committedBids
	if order != nil {
		if _, exists := s.orderIDByLot[order.LotID]; exists {
			return errors.New("order already exists")
		}
		s.ordersByID[order.ID] = *order
		s.orderIDByLot[order.LotID] = order.ID
	}
	if idempotencyKey != "" {
		s.idemByScope[testBidIdempotencyScope(bid.LotId, bid.UserId, idempotencyKey)] = bid
	}
	s.events = append(s.events, events...)
	return nil
}

func (s *testStore) CreateOrderForSettledLot(ctx context.Context, order auction.Order, lot *v1.Lot, expectedLotVersion int64, events []v1.AuctionEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	current, ok := s.lots[lot.Id]
	if !ok {
		return errors.New("lot not found")
	}
	if expectedLotVersion <= 0 {
		return errors.New("lot expected version is required")
	}
	if current.Version != expectedLotVersion {
		return apperr.ErrLotVersionConflict
	}
	if _, exists := s.orderIDByLot[order.LotID]; exists {
		return errors.New("order already exists")
	}
	s.lots[lot.Id] = proto.Clone(lot).(*v1.Lot)
	s.ordersByID[order.ID] = order
	s.orderIDByLot[order.LotID] = order.ID
	s.events = append(s.events, events...)
	return nil
}

func (s *testStore) FindOrderByID(ctx context.Context, orderID string) (*auction.Order, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	order, ok := s.ordersByID[orderID]
	if !ok {
		return nil, apperr.ErrNotFound
	}
	return &order, nil
}

func (s *testStore) FindOrderByLot(ctx context.Context, lotID string) (*auction.Order, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	orderID, ok := s.orderIDByLot[lotID]
	if !ok {
		return nil, false, nil
	}
	order := s.ordersByID[orderID]
	return &order, true, nil
}

func (s *testStore) ListOrdersByBuyer(ctx context.Context, buyerUserID string) ([]auction.Order, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	orders := make([]auction.Order, 0, len(s.ordersByID))
	for _, order := range s.ordersByID {
		if order.BuyerUserID == buyerUserID {
			orders = append(orders, order)
		}
	}
	sort.Slice(orders, func(i, j int) bool { return orders[i].CreatedAtUnixMs > orders[j].CreatedAtUnixMs })
	return orders, nil
}

func (s *testStore) ListOrders(ctx context.Context, query auction.OrderQuery) (auction.OrderList, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	query.Page, query.PageSize = auction.NormalizePagination(query.Page, query.PageSize)
	orders := make([]auction.OrderSummary, 0, len(s.ordersByID))
	buyer := strings.ToLower(strings.TrimSpace(query.Buyer))
	for _, order := range s.ordersByID {
		if query.MainAccountID != "" && order.MainAccountID != query.MainAccountID {
			continue
		}
		if query.BuyerUserID != "" && order.BuyerUserID != query.BuyerUserID {
			continue
		}
		if query.Status != "" && order.Status != query.Status {
			continue
		}
		if query.LotID != "" && order.LotID != query.LotID {
			continue
		}
		if buyer != "" && !strings.Contains(strings.ToLower(order.BuyerUserID+" "+order.BuyerNickname), buyer) {
			continue
		}
		orders = append(orders, order.Summary())
	}
	sort.Slice(orders, func(i, j int) bool { return orders[i].CreatedAtUnixMs > orders[j].CreatedAtUnixMs })
	total := int64(len(orders))
	start := auction.PageOffset(query.Page, query.PageSize)
	if start >= len(orders) {
		return auction.OrderList{Orders: []auction.OrderSummary{}, Total: total, Page: query.Page, PageSize: query.PageSize}, nil
	}
	end := start + query.PageSize
	if end > len(orders) {
		end = len(orders)
	}
	return auction.OrderList{Orders: orders[start:end], Total: total, Page: query.Page, PageSize: query.PageSize}, nil
}

func (s *testStore) FindPaymentByIdempotencyKey(ctx context.Context, orderID, key string) (*auction.Payment, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.paymentsByOrder[orderID] == nil {
		return nil, false, nil
	}
	payment, ok := s.paymentsByOrder[orderID][key]
	return &payment, ok, nil
}

func (s *testStore) CommitPaymentSuccess(ctx context.Context, payment auction.Payment, order auction.Order, expectedOrderVersion int64, events []v1.AuctionEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.failNextPaymentCommit != nil {
		if s.beforePaymentCommitFailure != nil {
			s.beforePaymentCommitFailure(s, payment, order, events)
		}
		err := s.failNextPaymentCommit
		s.failNextPaymentCommit = nil
		s.beforePaymentCommitFailure = nil
		return err
	}
	current, ok := s.ordersByID[order.ID]
	if !ok {
		return apperr.ErrNotFound
	}
	if current.Version != expectedOrderVersion {
		return apperr.ErrLotVersionConflict
	}
	if s.paymentsByOrder[order.ID] == nil {
		s.paymentsByOrder[order.ID] = make(map[string]auction.Payment)
	}
	if _, exists := s.paymentsByOrder[order.ID][payment.IdempotencyKey]; exists {
		return errors.New("payment already exists")
	}
	s.paymentsByOrder[order.ID][payment.IdempotencyKey] = payment
	s.ordersByID[order.ID] = order
	s.events = append(s.events, events...)
	return nil
}

func (s *testStore) PersistEvents(ctx context.Context, events []v1.AuctionEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, events...)
	return nil
}

func (s *testStore) ListRoomEvents(ctx context.Context, query auction.RoomEventQuery) (auction.RoomEventList, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if query.RoomID == "" {
		return auction.RoomEventList{}, errors.New("room id is required")
	}
	_, pageSize := auction.NormalizePagination(1, query.PageSize)
	offset := 0
	if strings.TrimSpace(query.PageToken) != "" {
		nextOffset, err := strconv.Atoi(query.PageToken)
		if err != nil || nextOffset < 0 {
			return auction.RoomEventList{}, errors.New("invalid page token")
		}
		offset = nextOffset
	}
	events := make([]v1.AuctionEvent, 0, len(s.events))
	for _, event := range s.events {
		if query.MainAccountID != "" && event.MainAccountId != query.MainAccountID {
			continue
		}
		if event.RoomId == query.RoomID {
			events = append(events, event)
		}
	}
	sort.SliceStable(events, func(i, j int) bool {
		if events[i].OccurredAtUnixMs == events[j].OccurredAtUnixMs {
			return events[i].Id > events[j].Id
		}
		return events[i].OccurredAtUnixMs > events[j].OccurredAtUnixMs
	})
	if offset > len(events) {
		offset = len(events)
	}
	end := offset + pageSize
	nextPageToken := ""
	if end < len(events) {
		nextPageToken = strconv.Itoa(end)
	} else {
		end = len(events)
	}
	result := make([]*v1.AuctionEvent, 0, end-offset)
	for _, event := range events[offset:end] {
		result = append(result, proto.Clone(&event).(*v1.AuctionEvent))
	}
	return auction.RoomEventList{Events: result, NextPageToken: nextPageToken}, nil
}

func (s *testStore) eventTypes() []v1.AuctionEventType {
	s.mu.RLock()
	defer s.mu.RUnlock()
	types := make([]v1.AuctionEventType, 0, len(s.events))
	for _, event := range s.events {
		types = append(types, event.Type)
	}
	return types
}

func (s *testStore) ListByLot(ctx context.Context, lotID string) ([]v1.Bid, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]v1.Bid(nil), s.bidsByLot[lotID]...), nil
}

func (s *testStore) ListBidRecordsByBuyer(ctx context.Context, buyerUserID string, query auction.BidRecordQuery) (auction.BidRecordList, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	query.Page, query.PageSize = auction.NormalizePagination(query.Page, query.PageSize)
	records := make([]auction.BidRecord, 0)
	for lotID, bids := range s.bidsByLot {
		if query.LotID != "" && lotID != query.LotID {
			continue
		}
		lot := s.lots[lotID]
		for _, bid := range bids {
			if bid.UserId != buyerUserID {
				continue
			}
			record := auction.BidRecord{
				ID:              bid.Id,
				LotID:           bid.LotId,
				UserID:          bid.UserId,
				Nickname:        bid.Nickname,
				Amount:          bid.GetAmount().GetAmount(),
				Currency:        bid.GetAmount().GetCurrency(),
				CreatedAtUnixMs: bid.CreatedAtUnixMs,
			}
			if lot != nil {
				record.RoomID = lot.RoomId
				record.LotTitle = lot.Title
				record.LotImageURL = lot.ImageUrl
				record.LotStatus = lot.Status.String()
				record.AuctionState = auction.AuctionStateOf(lot)
				record.Won = lot.WinnerUserId == buyerUserID
			}
			records = append(records, record)
		}
	}
	sort.Slice(records, func(i, j int) bool { return records[i].CreatedAtUnixMs > records[j].CreatedAtUnixMs })
	total := int64(len(records))
	start := auction.PageOffset(query.Page, query.PageSize)
	if start >= len(records) {
		return auction.BidRecordList{Bids: []auction.BidRecord{}, Total: total, Page: query.Page, PageSize: query.PageSize}, nil
	}
	end := start + query.PageSize
	if end > len(records) {
		end = len(records)
	}
	return auction.BidRecordList{Bids: records[start:end], Total: total, Page: query.Page, PageSize: query.PageSize}, nil
}

func (s *testStore) FindByIdempotencyKey(ctx context.Context, lotID, userID, key string) (v1.Bid, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	bid, ok := s.idemByScope[testBidIdempotencyScope(lotID, userID, key)]
	return bid, ok, nil
}

func (s *testStore) CacheIdempotencyKey(ctx context.Context, lotID, userID, key string, bid v1.Bid) {
	s.mu.Lock()
	defer s.mu.Unlock()
	scope := testBidIdempotencyScope(lotID, userID, key)
	if _, exists := s.idemByScope[scope]; exists {
		return
	}
	s.idemByScope[scope] = bid
}

func testBidIdempotencyScope(lotID, userID, key string) string {
	return lotID + "\x00" + userID + "\x00" + key
}

func testLotStats(bids []v1.Bid) *v1.LotStats {
	participants := make(map[string]bool)
	for _, bid := range bids {
		if bid.UserId != "" {
			participants[bid.UserId] = true
		}
	}
	return &v1.LotStats{ParticipantCount: int64(len(participants)), BidCount: int64(len(bids))}
}

func TestAuctionUsecaseCoreClosurePublishesEventsAndSnapshot(t *testing.T) {
	store := newTestStore()
	pub := &testPublisher{}
	uc := auction.NewAuctionUsecase(store, store, store, pub)
	ctx := context.Background()

	lot, err := uc.CreateLot(ctx, &v1.CreateLotRequest{
		RoomId:      "room_core",
		Title:       "核心闭环拍品",
		ImageUrl:    "https://example.com/lot.jpg",
		Description: "核心业务闭环测试",
		Rule: &v1.BidRule{
			StartPrice:             &v1.Money{Amount: 10000, Currency: "CNY"},
			MinIncrement:           &v1.Money{Amount: 1000, Currency: "CNY"},
			DurationSeconds:        300,
			AntiSnipeWindowSeconds: 15,
			AntiSnipeExtendSeconds: 15,
			MaxExtendCount:         3,
		},
		TrustCards: []*v1.TrustRevealCard{{Title: "证书", Content: "可复检"}},
	}, testMainAccountID, "test-owner")
	if err != nil {
		t.Fatalf("create lot failed: %v", err)
	}
	if len(lot.TrustCards) != 1 || lot.TrustCards[0].Id == "" || lot.TrustCards[0].LotId != lot.Id {
		t.Fatalf("trust card should be normalized on create: %+v", lot.TrustCards)
	}
	if _, err := uc.StartLot(ctx, lot.Id, testMainAccountID); err != nil {
		t.Fatalf("start lot failed: %v", err)
	}
	if _, _, _, err := uc.PlaceBid(ctx, &v1.PlaceBidRequest{
		LotId: lot.Id, Amount: &v1.Money{Amount: 11000, Currency: "CNY"},
	}, "u1", "用户1"); err == nil || !apperr.IsInvalidArgument(err) || !strings.Contains(err.Error(), "bid idempotency key is required") {
		t.Fatalf("missing bid idempotency key should be rejected as invalid argument, got %v", err)
	}
	_, firstBid, ranking, err := uc.PlaceBid(ctx, &v1.PlaceBidRequest{
		LotId: lot.Id, Amount: &v1.Money{Amount: 11000, Currency: "CNY"}, IdempotencyKey: "idem-1",
	}, "u1", "用户1")
	if err != nil || firstBid == nil || len(ranking) != 1 {
		t.Fatalf("first bid failed: bid=%+v ranking=%+v err=%v", firstBid, ranking, err)
	}
	_, replayBid, ranking, err := uc.PlaceBid(ctx, &v1.PlaceBidRequest{
		LotId: lot.Id, Amount: &v1.Money{Amount: 11000, Currency: "CNY"}, IdempotencyKey: "idem-1",
	}, "u1", "用户1")
	if err != nil || replayBid == nil || replayBid.Id != firstBid.Id || len(ranking) != 1 {
		t.Fatalf("idempotent bid replay failed: first=%+v replay=%+v ranking=%+v err=%v", firstBid, replayBid, ranking, err)
	}
	_, otherUserBid, ranking, err := uc.PlaceBid(ctx, &v1.PlaceBidRequest{
		LotId: lot.Id, Amount: &v1.Money{Amount: 12000, Currency: "CNY"}, IdempotencyKey: "idem-1",
	}, "u2", "用户2")
	if err != nil || otherUserBid == nil || otherUserBid.Id == firstBid.Id || otherUserBid.UserId != "u2" || len(ranking) != 2 {
		t.Fatalf("same idempotency key from another user must not replay first user bid: first=%+v other=%+v ranking=%+v err=%v", firstBid, otherUserBid, ranking, err)
	}
	if _, _, _, err := uc.PlaceBid(ctx, &v1.PlaceBidRequest{
		LotId: lot.Id, Amount: &v1.Money{Amount: 13000, Currency: "CNY"}, IdempotencyKey: "idem-2",
	}, "u1", "用户1"); err != nil {
		t.Fatalf("second bid failed: %v", err)
	}
	if _, _, _, err := uc.PlaceBid(ctx, &v1.PlaceBidRequest{
		LotId: lot.Id, Amount: &v1.Money{Amount: 14000, Currency: "CNY"}, IdempotencyKey: "idem-3",
	}, "u2", "用户2"); err != nil {
		t.Fatalf("third bid failed: %v", err)
	}
	if _, card, err := uc.RevealTrustCard(ctx, lot.Id, testMainAccountID, lot.TrustCards[0].Id, "op"); err != nil || card == nil || !card.Revealed {
		t.Fatalf("reveal trust card failed: card=%+v err=%v", card, err)
	}
	if lotAfterDuel, duel, err := uc.StartDuel(ctx, lot.Id, testMainAccountID, "op", "u2", "u1"); err != nil || duel == nil || !duel.Active || duel.UserAId != "u2" || duel.UserBId != "u1" || lotAfterDuel.PlaybookStage != v1.PlaybookStage_PLAYBOOK_STAGE_DUEL_MODE {
		t.Fatalf("start duel failed: lot=%+v duel=%+v err=%v", lotAfterDuel, duel, err)
	}
	settled, err := uc.SettleLot(ctx, lot.Id, testMainAccountID, "op")
	if err != nil {
		t.Fatalf("settle failed: %v", err)
	}
	if settled.Status != v1.LotStatus_LOT_STATUS_SETTLED || settled.WinnerUserId != "u2" || settled.GetFinalPrice().GetAmount() != 14000 || settled.GetDuelState().GetActive() {
		t.Fatalf("settled lot state mismatch: %+v", settled)
	}
	snapshot, err := uc.Snapshot(ctx, "room_core")
	if err != nil {
		t.Fatalf("snapshot failed: %v", err)
	}
	if snapshot.CurrentLot != nil || len(snapshot.Ranking) != 0 || len(snapshot.RecentBids) != 0 {
		t.Fatalf("snapshot mismatch: %+v", snapshot)
	}

	pub.assertContains(t,
		v1.AuctionEventType_AUCTION_EVENT_TYPE_LOT_CREATED,
		v1.AuctionEventType_AUCTION_EVENT_TYPE_LOT_STARTED,
		v1.AuctionEventType_AUCTION_EVENT_TYPE_BID_ACCEPTED,
		v1.AuctionEventType_AUCTION_EVENT_TYPE_BID_OUTBID,
		v1.AuctionEventType_AUCTION_EVENT_TYPE_RANKING_UPDATED,
		v1.AuctionEventType_AUCTION_EVENT_TYPE_TRUST_REVEALED,
		v1.AuctionEventType_AUCTION_EVENT_TYPE_DUEL_STARTED,
		v1.AuctionEventType_AUCTION_EVENT_TYPE_LOT_SETTLED,
	)
	assertEventTypesContain(t, store.eventTypes(),
		v1.AuctionEventType_AUCTION_EVENT_TYPE_LOT_CREATED,
		v1.AuctionEventType_AUCTION_EVENT_TYPE_LOT_STARTED,
		v1.AuctionEventType_AUCTION_EVENT_TYPE_BID_ACCEPTED,
		v1.AuctionEventType_AUCTION_EVENT_TYPE_BID_OUTBID,
		v1.AuctionEventType_AUCTION_EVENT_TYPE_RANKING_UPDATED,
		v1.AuctionEventType_AUCTION_EVENT_TYPE_TRUST_REVEALED,
		v1.AuctionEventType_AUCTION_EVENT_TYPE_DUEL_STARTED,
		v1.AuctionEventType_AUCTION_EVENT_TYPE_LOT_SETTLED,
	)
}

func TestPlaceBidReplaysIdempotentBidAfterConcurrentCommitConflict(t *testing.T) {
	store := newTestStore()
	uc := auction.NewAuctionUsecase(store, store, store, nil)
	ctx := context.Background()

	lot, err := uc.CreateLot(ctx, &v1.CreateLotRequest{
		RoomId:   "room_bid_race",
		Title:    "并发幂等出价拍品",
		ImageUrl: "https://example.com/lot.jpg",
		Rule: &v1.BidRule{
			StartPrice:             &v1.Money{Amount: 10000, Currency: "CNY"},
			MinIncrement:           &v1.Money{Amount: 1000, Currency: "CNY"},
			DurationSeconds:        300,
			AntiSnipeWindowSeconds: 15,
			AntiSnipeExtendSeconds: 15,
			MaxExtendCount:         3,
		},
	}, testMainAccountID, "test-owner")
	if err != nil {
		t.Fatalf("create lot failed: %v", err)
	}
	if _, err := uc.StartLot(ctx, lot.Id, testMainAccountID); err != nil {
		t.Fatalf("start lot failed: %v", err)
	}

	store.failNextBidCommit = apperr.ErrLotVersionConflict
	store.beforeBidCommitFailure = func(s *testStore, bid v1.Bid, lot *v1.Lot, idempotencyKey string, order *auction.Order, events []v1.AuctionEvent) {
		s.lots[lot.Id] = proto.Clone(lot).(*v1.Lot)
		s.bidsByLot[bid.LotId] = append(s.bidsByLot[bid.LotId], bid)
		s.idemByScope[testBidIdempotencyScope(bid.LotId, bid.UserId, idempotencyKey)] = bid
		if order != nil {
			s.ordersByID[order.ID] = *order
			s.orderIDByLot[order.LotID] = order.ID
		}
		s.events = append(s.events, events...)
	}

	updated, bid, ranking, err := uc.PlaceBid(ctx, &v1.PlaceBidRequest{
		LotId: lot.Id, Amount: &v1.Money{Amount: 11000, Currency: "CNY"}, IdempotencyKey: "bid-race-1",
	}, "buyer1", "买家一号")
	if err != nil || bid == nil || updated == nil || len(ranking) != 1 {
		t.Fatalf("idempotent race replay should succeed: lot=%+v bid=%+v ranking=%+v err=%v", updated, bid, ranking, err)
	}
	bids, err := store.ListByLot(ctx, lot.Id)
	if err != nil {
		t.Fatalf("list bids failed: %v", err)
	}
	if len(bids) != 1 || bids[0].Id != bid.Id {
		t.Fatalf("concurrent replay must not append duplicate bids: stored=%+v returned=%+v", bids, bid)
	}

	_, replayed, replayRanking, err := uc.PlaceBid(ctx, &v1.PlaceBidRequest{
		LotId: lot.Id, Amount: &v1.Money{Amount: 11000, Currency: "CNY"}, IdempotencyKey: "bid-race-1",
	}, "buyer1", "买家一号")
	if err != nil || replayed == nil || replayed.Id != bid.Id || len(replayRanking) != 1 {
		t.Fatalf("second replay should return same bid: first=%+v replay=%+v ranking=%+v err=%v", bid, replayed, replayRanking, err)
	}
}

func TestAuctionUsecaseCancelLotPersistsAndPublishesEvent(t *testing.T) {
	store := newTestStore()
	pub := &testPublisher{}
	uc := auction.NewAuctionUsecase(store, store, store, pub)
	ctx := context.Background()

	lot, err := uc.CreateLot(ctx, &v1.CreateLotRequest{
		RoomId:   "room_cancel",
		Title:    "取消拍品",
		ImageUrl: "https://example.com/lot.jpg",
		Rule: &v1.BidRule{
			StartPrice:             &v1.Money{Amount: 0, Currency: "CNY"},
			MinIncrement:           &v1.Money{Amount: 1000, Currency: "CNY"},
			DurationSeconds:        300,
			AntiSnipeWindowSeconds: 15,
			AntiSnipeExtendSeconds: 15,
			MaxExtendCount:         3,
		},
	}, testMainAccountID, "test-owner")
	if err != nil {
		t.Fatalf("create lot failed: %v", err)
	}
	queued, _, err := uc.QueueLot(ctx, lot.Id, testMainAccountID, "test-owner")
	if err != nil {
		t.Fatalf("queue lot failed: %v", err)
	}
	cancelled, err := uc.CancelLot(ctx, queued.Id, testMainAccountID, "op", "未开拍误操作")
	if err != nil {
		t.Fatalf("cancel lot failed: %v", err)
	}
	if cancelled.Status != v1.LotStatus_LOT_STATUS_CANCELLED || cancelled.QueueStatus != v1.LotQueueStatus_LOT_QUEUE_STATUS_NONE || cancelled.QueuePosition != 0 || cancelled.CancelReason != "未开拍误操作" || cancelled.CancelledAtUnixMs == 0 {
		t.Fatalf("cancelled lot mismatch: %+v", cancelled)
	}
	fresh, err := store.FindByID(ctx, lot.Id)
	if err != nil {
		t.Fatalf("find cancelled lot failed: %v", err)
	}
	if fresh.Status != v1.LotStatus_LOT_STATUS_CANCELLED || fresh.CancelReason != "未开拍误操作" {
		t.Fatalf("persisted cancelled lot mismatch: %+v", fresh)
	}
	pub.assertContains(t, v1.AuctionEventType_AUCTION_EVENT_TYPE_LOT_CANCELLED)
	assertEventTypesContain(t, store.eventTypes(), v1.AuctionEventType_AUCTION_EVENT_TYPE_LOT_CANCELLED)
}

func TestPlaceBidAntiSnipeExtensionPersistsLotUpdatedEvent(t *testing.T) {
	store := newTestStore()
	pub := &testPublisher{}
	uc := auction.NewAuctionUsecase(store, store, store, pub)
	ctx := context.Background()

	lot, err := uc.CreateLot(ctx, &v1.CreateLotRequest{
		RoomId:   "room_extend",
		Title:    "延时事件拍品",
		ImageUrl: "https://example.com/lot.jpg",
		Rule: &v1.BidRule{
			StartPrice:             &v1.Money{Amount: 10000, Currency: "CNY"},
			MinIncrement:           &v1.Money{Amount: 1000, Currency: "CNY"},
			DurationSeconds:        60,
			AntiSnipeWindowSeconds: 70,
			AntiSnipeExtendSeconds: 15,
			MaxExtendCount:         3,
		},
	}, testMainAccountID, "test-owner")
	if err != nil {
		t.Fatalf("create lot failed: %v", err)
	}
	started, err := uc.StartLot(ctx, lot.Id, testMainAccountID)
	if err != nil {
		t.Fatalf("start lot failed: %v", err)
	}
	originalEndsAt := started.EndsAtUnixMs
	updated, bid, ranking, err := uc.PlaceBid(ctx, &v1.PlaceBidRequest{LotId: lot.Id, Amount: &v1.Money{Amount: 11000, Currency: "CNY"}, IdempotencyKey: "bid-extend-1"}, "u1", "用户1")
	if err != nil || bid == nil || len(ranking) != 1 {
		t.Fatalf("extension bid failed: updated=%+v bid=%+v ranking=%+v err=%v", updated, bid, ranking, err)
	}
	if updated.EndsAtUnixMs <= originalEndsAt || updated.GetDuelState().GetExtendCount() != 1 || updated.GetDuelState().GetLotId() != updated.Id || updated.GetDuelState().GetEndsAtUnixMs() != updated.EndsAtUnixMs {
		t.Fatalf("accepted bid should extend live lot and sync duel state: before=%d after=%+v", originalEndsAt, updated)
	}
	pub.assertContains(t, v1.AuctionEventType_AUCTION_EVENT_TYPE_LOT_UPDATED, v1.AuctionEventType_AUCTION_EVENT_TYPE_AUCTION_EXTENDED)
	assertEventTypesContain(t, store.eventTypes(), v1.AuctionEventType_AUCTION_EVENT_TYPE_LOT_UPDATED, v1.AuctionEventType_AUCTION_EVENT_TYPE_AUCTION_EXTENDED)
}

func TestCapPriceCreatesOrderAndMockPaymentIsIdempotent(t *testing.T) {
	store := newTestStore()
	pub := &testPublisher{}
	uc := auction.NewAuctionUsecase(store, store, store, pub)
	ctx := context.Background()

	lot, err := uc.CreateLot(ctx, &v1.CreateLotRequest{
		RoomId:   "room_payment",
		Title:    "封顶成交拍品",
		ImageUrl: "https://example.com/lot.jpg",
		Rule: &v1.BidRule{
			StartPrice:             &v1.Money{Amount: 10000, Currency: "CNY"},
			MinIncrement:           &v1.Money{Amount: 1000, Currency: "CNY"},
			CapPrice:               &v1.Money{Amount: 11000, Currency: "CNY"},
			DurationSeconds:        300,
			AntiSnipeWindowSeconds: 15,
			AntiSnipeExtendSeconds: 15,
			MaxExtendCount:         3,
		},
	}, testMainAccountID, "test-owner")
	if err != nil {
		t.Fatalf("create lot failed: %v", err)
	}
	if _, err := uc.StartLot(ctx, lot.Id, testMainAccountID); err != nil {
		t.Fatalf("start lot failed: %v", err)
	}
	settled, bid, _, err := uc.PlaceBid(ctx, &v1.PlaceBidRequest{
		LotId: lot.Id, Amount: &v1.Money{Amount: 11000, Currency: "CNY"}, IdempotencyKey: "bid-cap-1",
	}, "buyer1", "买家一号")
	if err != nil || bid == nil {
		t.Fatalf("cap bid failed: lot=%+v bid=%+v err=%v", settled, bid, err)
	}
	if settled.Status != v1.LotStatus_LOT_STATUS_SETTLED || settled.WinnerUserId != "buyer1" || settled.GetFinalPrice().GetAmount() != 11000 {
		t.Fatalf("cap bid should settle lot, got %+v", settled)
	}
	order, found, err := store.FindOrderByLot(ctx, lot.Id)
	if err != nil || !found {
		t.Fatalf("settled lot should create order: found=%v err=%v", found, err)
	}
	if order.Status != auction.OrderStatusPendingPayment || order.PaymentStatus != auction.PaymentStatusInit || order.Amount != 11000 || order.BuyerUserID != "buyer1" {
		t.Fatalf("created order mismatch: %+v", order)
	}
	if order.ExpiresAtUnixMs-order.CreatedAtUnixMs != auction.OrderPaymentWindowMs {
		t.Fatalf("created order payment window mismatch: %+v", order)
	}
	if _, err := uc.MockPayOrder(ctx, "buyer1", order.ID, auction.MockPayRequest{IdempotencyKey: "pay-bad", Amount: 10000, Currency: "CNY"}); err == nil || !strings.Contains(err.Error(), "amount") {
		t.Fatalf("payment with wrong amount should fail, got %v", err)
	}
	paid, err := uc.MockPayOrder(ctx, "buyer1", order.ID, auction.MockPayRequest{IdempotencyKey: "pay-1", Amount: 11000, Currency: "CNY"})
	if err != nil || !paid.Paid || paid.Order.Status != auction.OrderStatusPaid || paid.Payment.Status != auction.PaymentStatusSuccess {
		t.Fatalf("payment should succeed: result=%+v err=%v", paid, err)
	}
	replayed, err := uc.MockPayOrder(ctx, "buyer1", order.ID, auction.MockPayRequest{IdempotencyKey: "pay-1", Amount: 11000, Currency: "CNY"})
	if err != nil || !replayed.Paid || replayed.Payment.ID != paid.Payment.ID {
		t.Fatalf("payment replay should return same payment: first=%+v replay=%+v err=%v", paid, replayed, err)
	}
	pub.mu.Lock()
	published := append([]v1.AuctionEvent(nil), pub.events...)
	pub.mu.Unlock()
	seenOrderCreated := false
	seenPaymentSuccess := false
	for _, event := range published {
		if event.Type == v1.AuctionEventType_AUCTION_EVENT_TYPE_ORDER_CREATED {
			seenOrderCreated = true
			if event.Reason != "order_created" || strings.Contains(event.Reason, "order_id=") || strings.Contains(event.Reason, order.ID) {
				t.Fatalf("ORDER_CREATED broadcast reason must not leak order id: reason=%q order=%s", event.Reason, order.ID)
			}
		}
		if event.Type == v1.AuctionEventType_AUCTION_EVENT_TYPE_PAYMENT_SUCCESS {
			seenPaymentSuccess = true
			if event.Reason != "payment_success" || strings.Contains(event.Reason, "payment_id=") || strings.Contains(event.Reason, paid.Payment.ID) || strings.Contains(event.Reason, order.ID) {
				t.Fatalf("PAYMENT_SUCCESS broadcast reason must not leak order/payment id: reason=%q order=%s payment=%s", event.Reason, order.ID, paid.Payment.ID)
			}
		}
	}
	if !seenOrderCreated || !seenPaymentSuccess {
		t.Fatalf("expected ORDER_CREATED and PAYMENT_SUCCESS broadcasts, got %+v", published)
	}
	orders, err := uc.ListOrdersByBuyer(ctx, "buyer1")
	if err != nil || len(orders) != 1 || orders[0].Status != auction.OrderStatusPaid {
		t.Fatalf("buyer orders mismatch: orders=%+v err=%v", orders, err)
	}
	result, err := uc.GetLotResult(ctx, lot.Id, auction.LotResultViewer{UserID: "buyer1", Role: v1.UserRole_USER_ROLE_BUYER})
	if err != nil || result.Order == nil || result.Order.ID != order.ID || result.AuctionState != auction.AuctionStateSettled {
		t.Fatalf("lot result mismatch: result=%+v err=%v", result, err)
	}
	pub.assertContains(t,
		v1.AuctionEventType_AUCTION_EVENT_TYPE_ORDER_CREATED,
		v1.AuctionEventType_AUCTION_EVENT_TYPE_AUCTION_CLOSED,
		v1.AuctionEventType_AUCTION_EVENT_TYPE_PAYMENT_SUCCESS,
	)
	assertEventTypesContain(t, store.eventTypes(),
		v1.AuctionEventType_AUCTION_EVENT_TYPE_ORDER_CREATED,
		v1.AuctionEventType_AUCTION_EVENT_TYPE_AUCTION_CLOSED,
		v1.AuctionEventType_AUCTION_EVENT_TYPE_PAYMENT_SUCCESS,
	)
}

func TestMockPayOrderRejectsExpiredPendingOrder(t *testing.T) {
	store := newTestStore()
	uc := auction.NewAuctionUsecase(store, store, store, nil)
	ctx := context.Background()

	lot, err := uc.CreateLot(ctx, validCreateLotRequest("room_expired_pay"), testMainAccountID, "test-owner")
	if err != nil {
		t.Fatalf("create lot failed: %v", err)
	}
	if _, err := uc.StartLot(ctx, lot.Id, testMainAccountID); err != nil {
		t.Fatalf("start lot failed: %v", err)
	}
	if _, bid, _, err := uc.PlaceBid(ctx, &v1.PlaceBidRequest{
		LotId: lot.Id, Amount: &v1.Money{Amount: 11000, Currency: "CNY"}, IdempotencyKey: "expired-pay-bid-1",
	}, "buyer1", "买家一号"); err != nil || bid == nil {
		t.Fatalf("place bid failed: bid=%+v err=%v", bid, err)
	}
	if _, err := uc.SettleLot(ctx, lot.Id, testMainAccountID, "op"); err != nil {
		t.Fatalf("settle lot failed: %v", err)
	}

	order, found, err := store.FindOrderByLot(ctx, lot.Id)
	if err != nil || !found {
		t.Fatalf("settled lot should create order: found=%v err=%v", found, err)
	}
	store.mu.Lock()
	expiredOrder := store.ordersByID[order.ID]
	expiredOrder.ExpiresAtUnixMs = clock.NowMs() - 1
	store.ordersByID[order.ID] = expiredOrder
	store.mu.Unlock()

	if _, err := uc.MockPayOrder(ctx, "buyer1", order.ID, auction.MockPayRequest{IdempotencyKey: "expired-pay-1", Amount: 11000, Currency: "CNY"}); !apperr.IsInvalidArgument(err) || !strings.Contains(err.Error(), "expired") {
		t.Fatalf("expired order payment should fail with invalid argument, got %v", err)
	}
	orders, err := uc.ListOrdersByBuyer(ctx, "buyer1")
	if err != nil || len(orders) != 1 {
		t.Fatalf("list expired buyer orders failed: orders=%+v err=%v", orders, err)
	}
	if orders[0].Status != auction.OrderStatusExpired || orders[0].PaymentStatus != auction.PaymentStatusClosed {
		t.Fatalf("expired order summary mismatch: %+v", orders[0])
	}
}

func TestMockPayOrderReplaysPaymentAfterConcurrentCommitConflict(t *testing.T) {
	store := newTestStore()
	uc := auction.NewAuctionUsecase(store, store, store, nil)
	ctx := context.Background()

	lot, err := uc.CreateLot(ctx, &v1.CreateLotRequest{
		RoomId:   "room_payment_race",
		Title:    "并发幂等支付拍品",
		ImageUrl: "https://example.com/lot.jpg",
		Rule: &v1.BidRule{
			StartPrice:             &v1.Money{Amount: 10000, Currency: "CNY"},
			MinIncrement:           &v1.Money{Amount: 1000, Currency: "CNY"},
			CapPrice:               &v1.Money{Amount: 11000, Currency: "CNY"},
			DurationSeconds:        300,
			AntiSnipeWindowSeconds: 15,
			AntiSnipeExtendSeconds: 15,
			MaxExtendCount:         3,
		},
	}, testMainAccountID, "test-owner")
	if err != nil {
		t.Fatalf("create lot failed: %v", err)
	}
	if _, err := uc.StartLot(ctx, lot.Id, testMainAccountID); err != nil {
		t.Fatalf("start lot failed: %v", err)
	}
	if _, bid, _, err := uc.PlaceBid(ctx, &v1.PlaceBidRequest{
		LotId: lot.Id, Amount: &v1.Money{Amount: 11000, Currency: "CNY"}, IdempotencyKey: "bid-payment-race",
	}, "buyer1", "买家一号"); err != nil || bid == nil {
		t.Fatalf("cap bid failed: bid=%+v err=%v", bid, err)
	}
	order, found, err := store.FindOrderByLot(ctx, lot.Id)
	if err != nil || !found {
		t.Fatalf("settled lot should create order: found=%v err=%v", found, err)
	}

	store.failNextPaymentCommit = apperr.ErrLotVersionConflict
	store.beforePaymentCommitFailure = func(s *testStore, payment auction.Payment, order auction.Order, events []v1.AuctionEvent) {
		if s.paymentsByOrder[order.ID] == nil {
			s.paymentsByOrder[order.ID] = make(map[string]auction.Payment)
		}
		s.paymentsByOrder[order.ID][payment.IdempotencyKey] = payment
		s.ordersByID[order.ID] = order
		s.events = append(s.events, events...)
	}

	paid, err := uc.MockPayOrder(ctx, "buyer1", order.ID, auction.MockPayRequest{IdempotencyKey: "pay-race-1", Amount: 11000, Currency: "CNY"})
	if err != nil || paid == nil || !paid.Paid || paid.Order.Status != auction.OrderStatusPaid || paid.Payment.Status != auction.PaymentStatusSuccess {
		t.Fatalf("idempotent payment race replay should succeed: result=%+v err=%v", paid, err)
	}
	replayed, err := uc.MockPayOrder(ctx, "buyer1", order.ID, auction.MockPayRequest{IdempotencyKey: "pay-race-1", Amount: 11000, Currency: "CNY"})
	if err != nil || replayed == nil || replayed.Payment.ID != paid.Payment.ID || !replayed.Paid {
		t.Fatalf("second payment replay should return same payment: first=%+v replay=%+v err=%v", paid, replayed, err)
	}
}

func TestCloseExpiredLotsSettlesLeadingBidCreatesOrderAndIsIdempotent(t *testing.T) {
	store := newTestStore()
	pub := &testPublisher{}
	uc := auction.NewAuctionUsecase(store, store, store, pub)
	ctx := context.Background()

	lot, err := uc.CreateLot(ctx, &v1.CreateLotRequest{
		RoomId:   "room_auto_close",
		Title:    "倒计时成交拍品",
		ImageUrl: "https://example.com/lot.jpg",
		Rule: &v1.BidRule{
			StartPrice:             &v1.Money{Amount: 10000, Currency: "CNY"},
			MinIncrement:           &v1.Money{Amount: 1000, Currency: "CNY"},
			DurationSeconds:        300,
			AntiSnipeWindowSeconds: 15,
			AntiSnipeExtendSeconds: 15,
			MaxExtendCount:         3,
		},
	}, testMainAccountID, "test-owner")
	if err != nil {
		t.Fatalf("create lot failed: %v", err)
	}
	if _, err := uc.StartLot(ctx, lot.Id, testMainAccountID); err != nil {
		t.Fatalf("start lot failed: %v", err)
	}
	if _, bid, _, err := uc.PlaceBid(ctx, &v1.PlaceBidRequest{LotId: lot.Id, Amount: &v1.Money{Amount: 11000, Currency: "CNY"}, IdempotencyKey: "bid-auto-close-1"}, "buyer1", "买家一号"); err != nil || bid == nil {
		t.Fatalf("bid before expiry failed: bid=%+v err=%v", bid, err)
	}
	forceLotEndsAt(t, store, ctx, lot.Id, 1000)

	summary, err := uc.CloseExpiredLots(ctx, 2000, 10)
	if err != nil {
		t.Fatalf("close expired lots failed: %v", err)
	}
	if summary.Closed != 1 || summary.Settled != 1 || summary.Failed != 0 {
		t.Fatalf("expired leading lot should settle once, summary=%+v", summary)
	}
	closedLot, err := store.FindByID(ctx, lot.Id)
	if err != nil {
		t.Fatalf("find closed lot failed: %v", err)
	}
	if closedLot.Status != v1.LotStatus_LOT_STATUS_SETTLED || closedLot.WinnerUserId != "buyer1" || closedLot.GetFinalPrice().GetAmount() != 11000 {
		t.Fatalf("expired leading lot should be sold: %+v", closedLot)
	}
	order, found, err := store.FindOrderByLot(ctx, lot.Id)
	if err != nil || !found {
		t.Fatalf("expired leading lot should create order: found=%v err=%v", found, err)
	}
	if order.BuyerUserID != "buyer1" || order.Amount != 11000 || order.Status != auction.OrderStatusPendingPayment {
		t.Fatalf("auto-created order mismatch: %+v", order)
	}
	replayed, err := uc.CloseExpiredLots(ctx, 3000, 10)
	if err != nil {
		t.Fatalf("second close scan failed: %v", err)
	}
	if replayed.Closed != 0 || len(store.orderIDByLot) != 1 {
		t.Fatalf("repeat scan must not create duplicate orders: summary=%+v orders=%+v", replayed, store.orderIDByLot)
	}
	pub.assertContains(t, v1.AuctionEventType_AUCTION_EVENT_TYPE_AUCTION_CLOSED, v1.AuctionEventType_AUCTION_EVENT_TYPE_LOT_SETTLED, v1.AuctionEventType_AUCTION_EVENT_TYPE_ORDER_CREATED)
	assertEventTypesContain(t, store.eventTypes(), v1.AuctionEventType_AUCTION_EVENT_TYPE_AUCTION_CLOSED, v1.AuctionEventType_AUCTION_EVENT_TYPE_LOT_SETTLED, v1.AuctionEventType_AUCTION_EVENT_TYPE_ORDER_CREATED)
}

func TestCloseExpiredLotsWithoutBidMarksFailedWithoutOrder(t *testing.T) {
	store := newTestStore()
	pub := &testPublisher{}
	uc := auction.NewAuctionUsecase(store, store, store, pub)
	ctx := context.Background()

	lot, err := uc.CreateLot(ctx, &v1.CreateLotRequest{
		RoomId:   "room_auto_fail",
		Title:    "倒计时流拍拍品",
		ImageUrl: "https://example.com/lot.jpg",
		Rule: &v1.BidRule{
			StartPrice:             &v1.Money{Amount: 10000, Currency: "CNY"},
			MinIncrement:           &v1.Money{Amount: 1000, Currency: "CNY"},
			DurationSeconds:        300,
			AntiSnipeWindowSeconds: 15,
			AntiSnipeExtendSeconds: 15,
			MaxExtendCount:         3,
		},
	}, testMainAccountID, "test-owner")
	if err != nil {
		t.Fatalf("create lot failed: %v", err)
	}
	if _, err := uc.StartLot(ctx, lot.Id, testMainAccountID); err != nil {
		t.Fatalf("start lot failed: %v", err)
	}
	forceLotEndsAt(t, store, ctx, lot.Id, 1000)

	summary, err := uc.CloseExpiredLots(ctx, 2000, 10)
	if err != nil {
		t.Fatalf("close expired lots failed: %v", err)
	}
	if summary.Closed != 1 || summary.Settled != 0 || summary.Failed != 1 {
		t.Fatalf("expired no-bid lot should fail once, summary=%+v", summary)
	}
	closedLot, err := store.FindByID(ctx, lot.Id)
	if err != nil {
		t.Fatalf("find closed lot failed: %v", err)
	}
	if closedLot.Status != v1.LotStatus_LOT_STATUS_FAILED || closedLot.CancelReason == "" {
		t.Fatalf("expired no-bid lot should be failed with reason: %+v", closedLot)
	}
	if _, found, err := store.FindOrderByLot(ctx, lot.Id); err != nil || found {
		t.Fatalf("expired no-bid lot must not create order: found=%v err=%v", found, err)
	}
	pub.assertContains(t, v1.AuctionEventType_AUCTION_EVENT_TYPE_AUCTION_CLOSED)
	assertEventTypesContain(t, store.eventTypes(), v1.AuctionEventType_AUCTION_EVENT_TYPE_AUCTION_CLOSED)
}

func forceLotEndsAt(t *testing.T, store *testStore, ctx context.Context, lotID string, endsAtUnixMs int64) {
	t.Helper()
	lot, err := store.FindByID(ctx, lotID)
	if err != nil {
		t.Fatalf("find lot for expiry failed: %v", err)
	}
	expectedVersion := lot.Version
	lot.EndsAtUnixMs = endsAtUnixMs
	if err := store.Save(ctx, lot, expectedVersion, nil); err != nil {
		t.Fatalf("force lot expiry failed: %v", err)
	}
}

type testPublisher struct {
	mu     sync.Mutex
	events []v1.AuctionEvent
}

func assertEventTypesContain(t *testing.T, got []v1.AuctionEventType, want ...v1.AuctionEventType) {
	t.Helper()
	seen := make(map[v1.AuctionEventType]bool, len(got))
	for _, typ := range got {
		seen[typ] = true
	}
	for _, typ := range want {
		if !seen[typ] {
			t.Fatalf("missing persisted event type %s in %+v", typ, got)
		}
	}
}

func (p *testPublisher) Publish(ctx context.Context, event v1.AuctionEvent) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.events = append(p.events, event)
	return nil
}

func (p *testPublisher) assertContains(t *testing.T, types ...v1.AuctionEventType) {
	t.Helper()
	p.mu.Lock()
	defer p.mu.Unlock()
	seen := make(map[v1.AuctionEventType]bool)
	for _, event := range p.events {
		seen[event.Type] = true
	}
	for _, typ := range types {
		if !seen[typ] {
			t.Fatalf("missing event type %s in %+v", typ, p.events)
		}
	}
}

func TestStartDuelWithOnlyUserBDoesNotDuplicateUsers(t *testing.T) {
	lot, err := auction.NewLotFromRequest("lot_duel", &v1.CreateLotRequest{
		RoomId:   "demo",
		Title:    "Duel 指定测试",
		ImageUrl: "https://example.com/lot.jpg",
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
	if err := auction.StartLot(lot, 1000); err != nil {
		t.Fatalf("start lot failed: %v", err)
	}
	ranking := []*v1.RankingItem{
		{Rank: 1, UserId: "u1", Nickname: "用户1", Amount: &v1.Money{Amount: 13000, Currency: "CNY"}, BidAtUnixMs: 3000},
		{Rank: 2, UserId: "u2", Nickname: "用户2", Amount: &v1.Money{Amount: 12000, Currency: "CNY"}, BidAtUnixMs: 2000},
	}
	if err := auction.StartDuel(lot, ranking, 4000, "", "u1"); err != nil {
		t.Fatalf("start duel failed: %v", err)
	}
	if lot.GetDuelState().GetUserAId() != "u2" || lot.GetDuelState().GetUserBId() != "u1" {
		t.Fatalf("expected service to fill distinct user A around requested B, got %+v", lot.GetDuelState())
	}
}
