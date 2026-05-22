package test

import (
	"context"
	"testing"

	v1 "live-auction-bid/backend/api/auction/service/v1"
	"live-auction-bid/backend/app/auction/service/internal/biz/auction"
	"live-auction-bid/backend/app/auction/service/internal/pkg/auth"
	appsvc "live-auction-bid/backend/app/auction/service/internal/service"
)

func TestAuctionServiceRequiresAuthForOperationsAndUsesTokenBidder(t *testing.T) {
	store := newTestStore()
	uc := auction.NewAuctionUsecase(store, store, store, nil)
	svc := appsvc.NewAuctionService(uc)
	ctx := context.Background()

	unauthCreate, err := svc.CreateLot(ctx, validCreateLotRequest("auth-room"))
	if err != nil {
		t.Fatalf("create lot returned transport error: %v", err)
	}
	if unauthCreate.GetResult().GetCode() != appsvc.ResultCodeUnauthenticated {
		t.Fatalf("expected unauthenticated create lot, got %+v", unauthCreate.GetResult())
	}

	anchorCtx := auth.WithClaims(ctx, &auth.Claims{UserID: "anchor1", Username: "anchor1", Nickname: "主播", Role: v1.UserRole_USER_ROLE_ANCHOR})
	create, err := svc.CreateLot(anchorCtx, validCreateLotRequest("auth-room"))
	if err != nil || create.GetResult().GetCode() != appsvc.ResultCodeOK {
		t.Fatalf("authorized create lot failed: reply=%+v err=%v", create, err)
	}
	if start, err := svc.StartLot(anchorCtx, &v1.StartLotRequest{LotId: create.GetLot().GetId()}); err != nil || start.GetResult().GetCode() != appsvc.ResultCodeOK {
		t.Fatalf("authorized start lot failed: reply=%+v err=%v", start, err)
	}

	buyerCtx := auth.WithClaims(ctx, &auth.Claims{UserID: "buyer1", Username: "buyer1", Nickname: "买家一号", Role: v1.UserRole_USER_ROLE_BUYER})
	bidReply, err := svc.PlaceBid(buyerCtx, &v1.PlaceBidRequest{
		LotId:  create.GetLot().GetId(),
		Amount: &v1.Money{Amount: 11000, Currency: "CNY"},
	})
	if err != nil || bidReply.GetResult().GetCode() != appsvc.ResultCodeOK || !bidReply.GetAccepted() {
		t.Fatalf("buyer bid failed: reply=%+v err=%v", bidReply, err)
	}
	if bidReply.GetBid().GetUserId() != "buyer1" || bidReply.GetBid().GetNickname() != "买家一号" {
		t.Fatalf("bidder must come from token claims, got %+v", bidReply.GetBid())
	}

	operatorCtx := auth.WithClaims(ctx, &auth.Claims{UserID: "op1", Username: "op1", Nickname: "运营", Role: v1.UserRole_USER_ROLE_OPERATOR})
	opBid, err := svc.PlaceBid(operatorCtx, &v1.PlaceBidRequest{
		LotId:  create.GetLot().GetId(),
		Amount: &v1.Money{Amount: 12000, Currency: "CNY"},
	})
	if err != nil {
		t.Fatalf("operator bid returned transport error: %v", err)
	}
	if opBid.GetResult().GetCode() != appsvc.ResultCodePermissionDenied {
		t.Fatalf("expected operator bid permission denied, got %+v", opBid.GetResult())
	}
}

func validCreateLotRequest(roomID string) *v1.CreateLotRequest {
	return &v1.CreateLotRequest{
		RoomId: roomID,
		Title:  "鉴权测试拍品",
		ImageUrl: "https://example.com/lot.jpg",
		Rule: &v1.BidRule{
			StartPrice:             &v1.Money{Amount: 10000, Currency: "CNY"},
			MinIncrement:           &v1.Money{Amount: 1000, Currency: "CNY"},
			DurationSeconds:        300,
			AntiSnipeWindowSeconds: 15,
			AntiSnipeExtendSeconds: 15,
			MaxExtendCount:         3,
		},
	}
}
