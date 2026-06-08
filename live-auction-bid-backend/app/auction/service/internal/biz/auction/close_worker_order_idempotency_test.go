package auction

import (
	"context"
	"errors"
	"testing"

	v1 "live-auction-bid/backend/api/auction/service/v1"
)

func TestCloseExpiredLotSkipsOrderCreateWhenOrderAlreadyExists(t *testing.T) {
	ctx := context.Background()
	nowMs := int64(1_700_000_000_000)
	lot := &v1.Lot{
		Id:              "lot_existing_order",
		MainAccountId:   "main_1",
		RoomId:          "room_1",
		Title:           "旧压测拍品",
		ImageUrl:        "https://example.com/stress-test.jpg",
		Status:          v1.LotStatus_LOT_STATUS_LIVE,
		Version:         7,
		EndsAtUnixMs:    nowMs - 1,
		LeadingUserId:   "buyer_1",
		LeadingNickname: "压测买家",
		CurrentPrice:    &v1.Money{Amount: 11800000, Currency: "CNY"},
		Rule: &v1.BidRule{
			StartPrice:      &v1.Money{Amount: 1000000, Currency: "CNY"},
			MinIncrement:    &v1.Money{Amount: 10000, Currency: "CNY"},
			DurationSeconds: 60,
		},
	}
	lots := &closeExpiredLotRepo{lot: lot}
	orders := &closeExpiredOrderRepo{
		existing: &Order{
			ID:              "order_existing",
			MainAccountID:   lot.MainAccountId,
			LotID:           lot.Id,
			RoomID:          lot.RoomId,
			LotTitle:        lot.Title,
			LotImageURL:     lot.ImageUrl,
			BuyerUserID:     lot.LeadingUserId,
			BuyerNickname:   lot.LeadingNickname,
			Status:          OrderStatusPendingPayment,
			PaymentStatus:   PaymentStatusInit,
			Amount:          lot.GetCurrentPrice().GetAmount(),
			Currency:        lot.GetCurrentPrice().GetCurrency(),
			CreatedAtUnixMs: nowMs - 1000,
			UpdatedAtUnixMs: nowMs - 1000,
			ExpiresAtUnixMs: nowMs + OrderPaymentWindowMs,
			Version:         1,
		},
	}
	uc := &AuctionUsecase{
		lots:   lots,
		bids:   &closeExpiredBidRepo{bids: []v1.Bid{acceptedBidForCloseWorkerTest(lot, nowMs-1000)}},
		orders: orders,
	}

	closed, settled, err := uc.closeExpiredLot(ctx, lot.Id, nowMs)
	if err != nil {
		t.Fatalf("closeExpiredLot returned error: %v", err)
	}
	if !closed || !settled {
		t.Fatalf("closeExpiredLot should close and settle existing-order lot, closed=%v settled=%v", closed, settled)
	}
	if orders.createCalls != 0 {
		t.Fatalf("existing order must be replayed, not inserted again; create calls=%d", orders.createCalls)
	}
	if lots.saveCalls != 1 {
		t.Fatalf("lot should be saved once to reconcile terminal state, got %d", lots.saveCalls)
	}
	if lots.expectedVersion != 7 {
		t.Fatalf("expected version mismatch: got %d", lots.expectedVersion)
	}
	if lots.savedLot.GetStatus() != v1.LotStatus_LOT_STATUS_SETTLED {
		t.Fatalf("lot status mismatch: got %s", lots.savedLot.GetStatus())
	}
	if lots.savedLot.GetWinnerUserId() != lot.GetLeadingUserId() {
		t.Fatalf("winner mismatch: got %q", lots.savedLot.GetWinnerUserId())
	}
	if lots.savedLot.GetFinalPrice().GetAmount() != lot.GetCurrentPrice().GetAmount() {
		t.Fatalf("final price mismatch: got %d", lots.savedLot.GetFinalPrice().GetAmount())
	}
	eventTypes := make([]v1.AuctionEventType, 0, len(lots.events))
	for _, event := range lots.events {
		eventTypes = append(eventTypes, event.GetType())
		if event.GetType() == v1.AuctionEventType_AUCTION_EVENT_TYPE_ORDER_CREATED {
			t.Fatalf("existing order replay should not emit a second order-created event")
		}
	}
	if len(eventTypes) != 2 ||
		eventTypes[0] != v1.AuctionEventType_AUCTION_EVENT_TYPE_AUCTION_CLOSED ||
		eventTypes[1] != v1.AuctionEventType_AUCTION_EVENT_TYPE_LOT_SETTLED {
		t.Fatalf("event types mismatch: got %+v", eventTypes)
	}
}

func acceptedBidForCloseWorkerTest(lot *v1.Lot, createdAtMs int64) v1.Bid {
	return v1.Bid{
		Id:              "bid_1",
		LotId:           lot.Id,
		UserId:          lot.LeadingUserId,
		Nickname:        lot.LeadingNickname,
		Amount:          lot.GetCurrentPrice(),
		CreatedAtUnixMs: createdAtMs,
	}
}

type closeExpiredLotRepo struct {
	lot             *v1.Lot
	savedLot        *v1.Lot
	expectedVersion int64
	events          []v1.AuctionEvent
	saveCalls       int
}

func (r *closeExpiredLotRepo) Create(context.Context, *v1.Lot, string, []v1.AuctionEvent) error {
	return errors.New("not implemented")
}

func (r *closeExpiredLotRepo) Save(_ context.Context, lot *v1.Lot, expectedVersion int64, events []v1.AuctionEvent) error {
	r.saveCalls++
	r.savedLot = lot
	r.expectedVersion = expectedVersion
	r.events = events
	return nil
}

func (r *closeExpiredLotRepo) QueueLotAsNext(context.Context, string, string, string, int64) (*v1.Lot, int32, []v1.AuctionEvent, error) {
	return nil, 0, nil, errors.New("not implemented")
}

func (r *closeExpiredLotRepo) StartLotAsOnlyActive(context.Context, *v1.Lot, int64, []v1.AuctionEvent) error {
	return errors.New("not implemented")
}

func (r *closeExpiredLotRepo) AttachAssets(context.Context, string, *v1.Lot) error {
	return nil
}

func (r *closeExpiredLotRepo) FindByID(_ context.Context, lotID string) (*v1.Lot, error) {
	if r.lot == nil || r.lot.Id != lotID {
		return nil, errors.New("lot not found")
	}
	return r.lot, nil
}

func (r *closeExpiredLotRepo) FindCoreByID(ctx context.Context, lotID string) (*v1.Lot, error) {
	return r.FindByID(ctx, lotID)
}

func (r *closeExpiredLotRepo) List(context.Context, string, v1.LotStatus) ([]*v1.Lot, error) {
	return nil, errors.New("not implemented")
}

func (r *closeExpiredLotRepo) ListLots(context.Context, LotQuery) (LotList, error) {
	return LotList{}, errors.New("not implemented")
}

func (r *closeExpiredLotRepo) FindOrCreateRoomState(context.Context, string, string, int64) (*RoomState, error) {
	return nil, errors.New("not implemented")
}

func (r *closeExpiredLotRepo) RepairRoomActiveLot(context.Context, string, string, int64) error {
	return nil
}

type closeExpiredBidRepo struct {
	bids []v1.Bid
}

func (r *closeExpiredBidRepo) CommitAcceptedBid(context.Context, v1.Bid, *v1.Lot, int64, string, *Order, []v1.AuctionEvent) error {
	return errors.New("not implemented")
}

func (r *closeExpiredBidRepo) ListByLot(context.Context, string) ([]v1.Bid, error) {
	return r.bids, nil
}

func (r *closeExpiredBidRepo) ListBidRecordsByBuyer(context.Context, string, BidRecordQuery) (BidRecordList, error) {
	return BidRecordList{}, errors.New("not implemented")
}

func (r *closeExpiredBidRepo) FindByIdempotencyKey(context.Context, string, string, string) (v1.Bid, bool, error) {
	return v1.Bid{}, false, nil
}

func (r *closeExpiredBidRepo) CacheIdempotencyKey(context.Context, string, string, string, v1.Bid) {}

type closeExpiredOrderRepo struct {
	existing    *Order
	createCalls int
}

func (r *closeExpiredOrderRepo) CreateOrderForSettledLot(context.Context, Order, *v1.Lot, int64, []v1.AuctionEvent) error {
	r.createCalls++
	return nil
}

func (r *closeExpiredOrderRepo) FindOrderByID(context.Context, string) (*Order, error) {
	return nil, errors.New("not implemented")
}

func (r *closeExpiredOrderRepo) FindOrderByLot(context.Context, string) (*Order, bool, error) {
	if r.existing == nil {
		return nil, false, nil
	}
	return r.existing, true, nil
}

func (r *closeExpiredOrderRepo) ListOrdersByBuyer(context.Context, string) ([]Order, error) {
	return nil, errors.New("not implemented")
}

func (r *closeExpiredOrderRepo) ListOrders(context.Context, OrderQuery) (OrderList, error) {
	return OrderList{}, errors.New("not implemented")
}
