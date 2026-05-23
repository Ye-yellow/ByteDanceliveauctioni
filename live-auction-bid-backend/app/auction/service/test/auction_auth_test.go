package test

import (
	"context"
	"testing"

	v1 "live-auction-bid/backend/api/auction/service/v1"
	"live-auction-bid/backend/app/auction/service/internal/biz/auction"
	"live-auction-bid/backend/app/auction/service/internal/pkg/apperr"
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
		LotId:          create.GetLot().GetId(),
		Amount:         &v1.Money{Amount: 11000, Currency: "CNY"},
		IdempotencyKey: "auth-buyer-1",
	})
	if err != nil || bidReply.GetResult().GetCode() != appsvc.ResultCodeOK || !bidReply.GetAccepted() {
		t.Fatalf("buyer bid failed: reply=%+v err=%v", bidReply, err)
	}
	if bidReply.GetBid().GetUserId() != "buyer1" || bidReply.GetBid().GetNickname() != "买家一号" {
		t.Fatalf("bidder must come from token claims, got %+v", bidReply.GetBid())
	}
	missingIdemBid, err := svc.PlaceBid(buyerCtx, &v1.PlaceBidRequest{
		LotId:  create.GetLot().GetId(),
		Amount: &v1.Money{Amount: 12000, Currency: "CNY"},
	})
	if err != nil {
		t.Fatalf("missing idempotency key bid returned transport error: %v", err)
	}
	if missingIdemBid.GetResult().GetCode() != appsvc.ResultCodeInvalidArgument || missingIdemBid.GetAccepted() {
		t.Fatalf("missing idempotency key must return invalid argument, got %+v", missingIdemBid)
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
	if _, err := svc.ListMyOrders(operatorCtx); err == nil {
		t.Fatal("operator must not access buyer order list")
	}
	if _, err := svc.MockPayOrder(operatorCtx, "order1", auction.MockPayRequest{IdempotencyKey: "pay-auth", Amount: 1, Currency: "CNY"}); err == nil {
		t.Fatal("operator must not access buyer payment endpoint")
	}
	orders, err := svc.ListMyOrders(buyerCtx)
	if err != nil || len(orders) != 0 {
		t.Fatalf("buyer should access own empty order list, orders=%+v err=%v", orders, err)
	}
}

func TestGetLotResultOrderVisibility(t *testing.T) {
	store := newTestStore()
	uc := auction.NewAuctionUsecase(store, store, store, nil)
	svc := appsvc.NewAuctionService(uc)
	ctx := context.Background()

	anchorCtx := auth.WithClaims(ctx, &auth.Claims{UserID: "anchor1", Username: "anchor1", Nickname: "主播", Role: v1.UserRole_USER_ROLE_ANCHOR})
	create, err := svc.CreateLot(anchorCtx, validCreateLotRequest("result-visibility-room"))
	if err != nil || create.GetResult().GetCode() != appsvc.ResultCodeOK {
		t.Fatalf("create lot failed: reply=%+v err=%v", create, err)
	}
	lotID := create.GetLot().GetId()
	if start, err := svc.StartLot(anchorCtx, &v1.StartLotRequest{LotId: lotID}); err != nil || start.GetResult().GetCode() != appsvc.ResultCodeOK {
		t.Fatalf("start lot failed: reply=%+v err=%v", start, err)
	}
	winnerCtx := auth.WithClaims(ctx, &auth.Claims{UserID: "buyer1", Username: "buyer1", Nickname: "买家一号", Role: v1.UserRole_USER_ROLE_BUYER})
	if bid, err := svc.PlaceBid(winnerCtx, &v1.PlaceBidRequest{LotId: lotID, Amount: &v1.Money{Amount: 11000, Currency: "CNY"}, IdempotencyKey: "result-winner-1"}); err != nil || bid.GetResult().GetCode() != appsvc.ResultCodeOK || !bid.GetAccepted() {
		t.Fatalf("buyer bid failed: reply=%+v err=%v", bid, err)
	}
	if settle, err := svc.SettleLot(anchorCtx, &v1.SettleLotRequest{LotId: lotID}); err != nil || settle.GetResult().GetCode() != appsvc.ResultCodeOK {
		t.Fatalf("settle lot failed: reply=%+v err=%v", settle, err)
	}

	publicResult, err := svc.GetLotResult(ctx, lotID)
	if err != nil {
		t.Fatalf("public get lot result failed: %v", err)
	}
	if publicResult.Order != nil {
		t.Fatalf("anonymous viewer must not see full order: %+v", publicResult.Order)
	}
	otherBuyerCtx := auth.WithClaims(ctx, &auth.Claims{UserID: "buyer2", Username: "buyer2", Nickname: "买家二号", Role: v1.UserRole_USER_ROLE_BUYER})
	otherBuyerResult, err := svc.GetLotResult(otherBuyerCtx, lotID)
	if err != nil {
		t.Fatalf("other buyer get lot result failed: %v", err)
	}
	if otherBuyerResult.Order != nil {
		t.Fatalf("non-winning buyer must not see order id/payment id/buyer id: %+v", otherBuyerResult.Order)
	}
	winnerResult, err := svc.GetLotResult(winnerCtx, lotID)
	if err != nil {
		t.Fatalf("winner get lot result failed: %v", err)
	}
	if winnerResult.Order == nil || winnerResult.Order.ID == "" || winnerResult.Order.BuyerUserID != "buyer1" {
		t.Fatalf("winning buyer should see own full order: %+v", winnerResult.Order)
	}
	for _, claims := range []*auth.Claims{
		{UserID: "anchor1", Username: "anchor1", Nickname: "主播", Role: v1.UserRole_USER_ROLE_ANCHOR},
		{UserID: "op1", Username: "op1", Nickname: "运营", Role: v1.UserRole_USER_ROLE_OPERATOR},
		{UserID: "admin1", Username: "admin1", Nickname: "管理员", Role: v1.UserRole_USER_ROLE_ADMIN},
	} {
		result, err := svc.GetLotResult(auth.WithClaims(ctx, claims), lotID)
		if err != nil {
			t.Fatalf("%s get lot result failed: %v", claims.Role, err)
		}
		if result.Order == nil || result.Order.ID == "" || result.Order.BuyerUserID != "buyer1" {
			t.Fatalf("%s should see full order: %+v", claims.Role, result.Order)
		}
	}
}

func TestP2QueryPermissionsAndBuyerIsolation(t *testing.T) {
	store := newTestStore()
	uc := auction.NewAuctionUsecase(store, store, store, nil)
	svc := appsvc.NewAuctionService(uc)
	ctx := context.Background()

	anchorCtx := auth.WithClaims(ctx, &auth.Claims{UserID: "anchor1", Username: "anchor1", Nickname: "主播", Role: v1.UserRole_USER_ROLE_ANCHOR})
	adminCtx := auth.WithClaims(ctx, &auth.Claims{UserID: "admin1", Username: "admin1", Nickname: "管理员", Role: v1.UserRole_USER_ROLE_ADMIN})
	operatorCtx := auth.WithClaims(ctx, &auth.Claims{UserID: "op1", Username: "op1", Nickname: "运营", Role: v1.UserRole_USER_ROLE_OPERATOR})
	buyer1Ctx := auth.WithClaims(ctx, &auth.Claims{UserID: "buyer1", Username: "buyer1", Nickname: "买家一号", Role: v1.UserRole_USER_ROLE_BUYER})
	buyer2Ctx := auth.WithClaims(ctx, &auth.Claims{UserID: "buyer2", Username: "buyer2", Nickname: "买家二号", Role: v1.UserRole_USER_ROLE_BUYER})

	for _, item := range []struct {
		roomID string
		buyer  context.Context
		amount int64
	}{
		{roomID: "p2-query-room-1", buyer: buyer1Ctx, amount: 11000},
		{roomID: "p2-query-room-2", buyer: buyer2Ctx, amount: 12000},
	} {
		create, err := svc.CreateLot(anchorCtx, validCreateLotRequest(item.roomID))
		if err != nil || create.GetResult().GetCode() != appsvc.ResultCodeOK {
			t.Fatalf("create lot failed: reply=%+v err=%v", create, err)
		}
		lotID := create.GetLot().GetId()
		if start, err := svc.StartLot(anchorCtx, &v1.StartLotRequest{LotId: lotID}); err != nil || start.GetResult().GetCode() != appsvc.ResultCodeOK {
			t.Fatalf("start lot failed: reply=%+v err=%v", start, err)
		}
		if bid, err := svc.PlaceBid(item.buyer, &v1.PlaceBidRequest{LotId: lotID, Amount: &v1.Money{Amount: item.amount, Currency: "CNY"}, IdempotencyKey: "p2-query-" + item.roomID}); err != nil || bid.GetResult().GetCode() != appsvc.ResultCodeOK || !bid.GetAccepted() {
			t.Fatalf("bid failed: reply=%+v err=%v", bid, err)
		}
		if settle, err := svc.SettleLot(anchorCtx, &v1.SettleLotRequest{LotId: lotID}); err != nil || settle.GetResult().GetCode() != appsvc.ResultCodeOK {
			t.Fatalf("settle lot failed: reply=%+v err=%v", settle, err)
		}
	}

	for _, allowedCtx := range []context.Context{anchorCtx, operatorCtx, adminCtx} {
		orders, err := svc.ListOrders(allowedCtx, auction.OrderQuery{Page: 1, PageSize: 10})
		if err != nil || orders.Total != 2 || len(orders.Orders) != 2 {
			t.Fatalf("admin role should list all orders: orders=%+v err=%v", orders, err)
		}
	}
	if _, err := svc.ListOrders(buyer1Ctx, auction.OrderQuery{}); !apperr.IsPermissionDenied(err) {
		t.Fatalf("buyer must not list admin orders, got %v", err)
	}

	buyer1Orders, err := svc.ListMyOrdersPage(buyer1Ctx, auction.OrderQuery{Page: 1, PageSize: 10})
	if err != nil || buyer1Orders.Total != 1 || len(buyer1Orders.Orders) != 1 || buyer1Orders.Orders[0].BuyerUserID != "buyer1" {
		t.Fatalf("buyer should only see own orders: orders=%+v err=%v", buyer1Orders, err)
	}
	buyer2Orders, err := svc.ListMyOrdersPage(buyer2Ctx, auction.OrderQuery{Page: 1, PageSize: 10})
	if err != nil || buyer2Orders.Total != 1 || len(buyer2Orders.Orders) != 1 || buyer2Orders.Orders[0].BuyerUserID != "buyer2" {
		t.Fatalf("buyer2 should only see own orders: orders=%+v err=%v", buyer2Orders, err)
	}
	buyer1Bids, err := svc.ListMyBids(buyer1Ctx, auction.BidRecordQuery{Page: 1, PageSize: 10})
	if err != nil || buyer1Bids.Total != 1 || len(buyer1Bids.Bids) != 1 || buyer1Bids.Bids[0].UserID != "buyer1" {
		t.Fatalf("buyer should only see own bid records: bids=%+v err=%v", buyer1Bids, err)
	}
}

func validCreateLotRequest(roomID string) *v1.CreateLotRequest {
	return &v1.CreateLotRequest{
		RoomId:   roomID,
		Title:    "鉴权测试拍品",
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
