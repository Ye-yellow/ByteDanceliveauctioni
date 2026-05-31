package test

import (
	"context"
	"strings"
	"testing"

	v1 "live-auction-bid/backend/api/auction/service/v1"
	"live-auction-bid/backend/app/auction/service/internal/biz/auction"
	userbiz "live-auction-bid/backend/app/auction/service/internal/biz/user"
	"live-auction-bid/backend/app/auction/service/internal/pkg/auth"
	appsvc "live-auction-bid/backend/app/auction/service/internal/service"
)

func TestAuctionDraftAutosaveAndQueueFlow(t *testing.T) {
	store := newTestStore()
	uc := auction.NewAuctionUsecase(store, store, store, nil)
	svc := appsvc.NewAuctionService(uc)
	ctx := auth.WithClaims(context.Background(), claimsForRoleCode("operator-draft", "operator", "运营", userbiz.RoleOperator, testMainAccountID))

	empty, err := svc.CreateLotDraft(ctx, &v1.CreateLotRequest{})
	if err != nil || empty.GetResult().GetCode() != appsvc.ResultCodeOK {
		t.Fatalf("create fully empty draft failed: reply=%+v err=%v", empty, err)
	}

	create, err := svc.CreateLotDraft(ctx, &v1.CreateLotRequest{RoomId: "draft-room"})
	if err != nil || create.GetResult().GetCode() != appsvc.ResultCodeOK {
		t.Fatalf("create room draft failed: reply=%+v err=%v", create, err)
	}
	if create.GetLot().GetId() == "" || create.GetLot().GetStatus() != v1.LotStatus_LOT_STATUS_DRAFT || create.GetLot().GetQueueStatus() != v1.LotQueueStatus_LOT_QUEUE_STATUS_NONE {
		t.Fatalf("empty draft state mismatch: %+v", create.GetLot())
	}

	incompleteQueue, err := svc.QueueLot(ctx, &v1.QueueLotRequest{LotId: create.GetLot().GetId()})
	if err != nil {
		t.Fatalf("queue incomplete draft returned transport error: %v", err)
	}
	if incompleteQueue.GetResult().GetCode() == appsvc.ResultCodeOK || !strings.Contains(incompleteQueue.GetResult().GetMessage(), "lot title") {
		t.Fatalf("incomplete draft must not enter queue: %+v", incompleteQueue.GetResult())
	}

	patch, err := svc.PatchLotDraft(ctx, &v1.PatchLotDraftRequest{
		LotId:       create.GetLot().GetId(),
		Title:       "自动保存拍品",
		Description: "草稿自动保存",
		ImageUrl:    "https://example.com/draft.jpg",
		Rule: &v1.BidRule{
			StartPrice:             &v1.Money{Amount: 10000, Currency: "CNY"},
			MinIncrement:           &v1.Money{Amount: 1000, Currency: "CNY"},
			DurationSeconds:        300,
			AntiSnipeWindowSeconds: 15,
			AntiSnipeExtendSeconds: 15,
			MaxExtendCount:         3,
		},
	})
	if err != nil || patch.GetResult().GetCode() != appsvc.ResultCodeOK {
		t.Fatalf("patch draft failed: reply=%+v err=%v", patch, err)
	}
	if patch.GetLot().GetTitle() != "自动保存拍品" || patch.GetLot().GetQueueStatus() != v1.LotQueueStatus_LOT_QUEUE_STATUS_NONE {
		t.Fatalf("patched draft state mismatch: %+v", patch.GetLot())
	}

	queued, err := svc.QueueLot(ctx, &v1.QueueLotRequest{LotId: create.GetLot().GetId()})
	if err != nil || queued.GetResult().GetCode() != appsvc.ResultCodeOK {
		t.Fatalf("queue complete draft failed: reply=%+v err=%v", queued, err)
	}
	if queued.GetLot().GetStatus() != v1.LotStatus_LOT_STATUS_QUEUED || queued.GetLot().GetQueueStatus() != v1.LotQueueStatus_LOT_QUEUE_STATUS_QUEUED || queued.GetQueuePosition() != 1 || queued.GetLot().GetQueuePosition() != 1 {
		t.Fatalf("queued state mismatch: %+v queuePosition=%d", queued.GetLot(), queued.GetQueuePosition())
	}
	if queued.GetLot().GetStartedAtUnixMs() != 0 {
		t.Fatalf("queue endpoint must not start lot: %+v", queued.GetLot())
	}
}

func TestQueueLotRejectsInvalidRule(t *testing.T) {
	store := newTestStore()
	uc := auction.NewAuctionUsecase(store, store, store, nil)
	svc := appsvc.NewAuctionService(uc)
	ctx := auth.WithClaims(context.Background(), claimsForRoleCode("operator-invalid", "operator", "运营", userbiz.RoleOperator, testMainAccountID))

	create, err := svc.CreateLotDraft(ctx, &v1.CreateLotRequest{RoomId: "invalid-room", Title: "规则错误", ImageUrl: "https://example.com/lot.jpg"})
	if err != nil || create.GetResult().GetCode() != appsvc.ResultCodeOK {
		t.Fatalf("create partial draft failed: reply=%+v err=%v", create, err)
	}
	patch, err := svc.PatchLotDraft(ctx, &v1.PatchLotDraftRequest{
		LotId:    create.GetLot().GetId(),
		Title:    "规则错误",
		ImageUrl: "https://example.com/lot.jpg",
		Rule: &v1.BidRule{
			StartPrice:             &v1.Money{Amount: 10000, Currency: "CNY"},
			MinIncrement:           &v1.Money{Amount: 0, Currency: "CNY"},
			DurationSeconds:        30,
			AntiSnipeWindowSeconds: 15,
			AntiSnipeExtendSeconds: 9,
			MaxExtendCount:         3,
		},
	})
	if err != nil || patch.GetResult().GetCode() != appsvc.ResultCodeOK {
		t.Fatalf("patch invalid draft should be allowed before queue: reply=%+v err=%v", patch, err)
	}
	queued, err := svc.QueueLot(ctx, &v1.QueueLotRequest{LotId: create.GetLot().GetId()})
	if err != nil {
		t.Fatalf("queue invalid rule returned transport error: %v", err)
	}
	if queued.GetResult().GetCode() == appsvc.ResultCodeOK || !strings.Contains(queued.GetResult().GetMessage(), "min increment") {
		t.Fatalf("invalid rule must be rejected by queue endpoint: %+v", queued.GetResult())
	}
}
