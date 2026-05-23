package auction

import (
	"fmt"

	v1 "live-auction-bid/backend/api/auction/service/v1"
	"live-auction-bid/backend/app/auction/service/internal/pkg/apperr"
)

type AuctionState string

const (
	AuctionStateDraft     AuctionState = "DRAFT"
	AuctionStateScheduled AuctionState = "SCHEDULED"
	AuctionStateLive      AuctionState = "LIVE"
	AuctionStateExtended  AuctionState = "EXTENDED"
	AuctionStateSold      AuctionState = "SOLD"
	AuctionStateCancelled AuctionState = "CANCELLED"
	AuctionStateFailed    AuctionState = "FAILED"
)

type OrderStatus string

const (
	OrderStatusCreated        OrderStatus = "CREATED"
	OrderStatusPendingPayment OrderStatus = "PENDING_PAYMENT"
	OrderStatusPaid           OrderStatus = "PAID"
	OrderStatusCancelled      OrderStatus = "CANCELLED"
	OrderStatusExpired        OrderStatus = "EXPIRED"
	OrderStatusRefunded       OrderStatus = "REFUNDED"
)

type PaymentStatus string

const (
	PaymentStatusInit       PaymentStatus = "INIT"
	PaymentStatusProcessing PaymentStatus = "PROCESSING"
	PaymentStatusSuccess    PaymentStatus = "SUCCESS"
	PaymentStatusFailed     PaymentStatus = "FAILED"
	PaymentStatusClosed     PaymentStatus = "CLOSED"
)

type Order struct {
	ID              string        `json:"id"`
	LotID           string        `json:"lotId"`
	RoomID          string        `json:"roomId"`
	LotTitle        string        `json:"lotTitle"`
	LotImageURL     string        `json:"lotImageUrl"`
	BuyerUserID     string        `json:"buyerUserId"`
	BuyerNickname   string        `json:"buyerNickname"`
	Status          OrderStatus   `json:"status"`
	PaymentStatus   PaymentStatus `json:"paymentStatus"`
	PaymentID       string        `json:"paymentId,omitempty"`
	Amount          int64         `json:"amount"`
	Currency        string        `json:"currency"`
	CreatedAtUnixMs int64         `json:"createdAtUnixMs"`
	UpdatedAtUnixMs int64         `json:"updatedAtUnixMs"`
	ExpiresAtUnixMs int64         `json:"expiresAtUnixMs"`
	PaidAtUnixMs    int64         `json:"paidAtUnixMs,omitempty"`
	Version         int64         `json:"version"`
}

type Payment struct {
	ID              string        `json:"id"`
	OrderID         string        `json:"orderId"`
	LotID           string        `json:"lotId"`
	BuyerUserID     string        `json:"buyerUserId"`
	Status          PaymentStatus `json:"status"`
	Amount          int64         `json:"amount"`
	Currency        string        `json:"currency"`
	IdempotencyKey  string        `json:"idempotencyKey,omitempty"`
	CreatedAtUnixMs int64         `json:"createdAtUnixMs"`
	UpdatedAtUnixMs int64         `json:"updatedAtUnixMs"`
	SucceededAtMs   int64         `json:"succeededAtUnixMs,omitempty"`
}

type OrderSummary struct {
	ID              string        `json:"id"`
	LotID           string        `json:"lotId"`
	RoomID          string        `json:"roomId"`
	LotTitle        string        `json:"lotTitle"`
	LotImageURL     string        `json:"lotImageUrl"`
	BuyerUserID     string        `json:"buyerUserId"`
	Status          OrderStatus   `json:"status"`
	PaymentStatus   PaymentStatus `json:"paymentStatus"`
	PaymentID       string        `json:"paymentId,omitempty"`
	Amount          int64         `json:"amount"`
	Currency        string        `json:"currency"`
	CreatedAtUnixMs int64         `json:"createdAtUnixMs"`
	UpdatedAtUnixMs int64         `json:"updatedAtUnixMs"`
	ExpiresAtUnixMs int64         `json:"expiresAtUnixMs"`
	PaidAtUnixMs    int64         `json:"paidAtUnixMs,omitempty"`
}

type PaymentSummary struct {
	ID              string        `json:"id"`
	OrderID         string        `json:"orderId"`
	Status          PaymentStatus `json:"status"`
	Amount          int64         `json:"amount"`
	Currency        string        `json:"currency"`
	CreatedAtUnixMs int64         `json:"createdAtUnixMs"`
	SucceededAtMs   int64         `json:"succeededAtUnixMs,omitempty"`
}

type OrderQuery struct {
	Page        int         `json:"page"`
	PageSize    int         `json:"pageSize"`
	Status      OrderStatus `json:"status,omitempty"`
	LotID       string      `json:"lotId,omitempty"`
	Buyer       string      `json:"buyer,omitempty"`
	BuyerUserID string      `json:"buyerUserId,omitempty"`
}

type OrderList struct {
	Orders   []OrderSummary `json:"orders"`
	Total    int64          `json:"total"`
	Page     int            `json:"page"`
	PageSize int            `json:"pageSize"`
}

type LotQuery struct {
	Page     int          `json:"page"`
	PageSize int          `json:"pageSize"`
	Status   v1.LotStatus `json:"status,omitempty"`
	Keyword  string       `json:"keyword,omitempty"`
	RoomID   string       `json:"roomId,omitempty"`
}

type LotList struct {
	Lots     []*v1.Lot `json:"lots"`
	Total    int64     `json:"total"`
	Page     int       `json:"page"`
	PageSize int       `json:"pageSize"`
}

type BidRecord struct {
	ID              string       `json:"id"`
	LotID           string       `json:"lotId"`
	RoomID          string       `json:"roomId"`
	LotTitle        string       `json:"lotTitle"`
	LotImageURL     string       `json:"lotImageUrl"`
	UserID          string       `json:"userId"`
	Nickname        string       `json:"nickname"`
	Amount          int64        `json:"amount"`
	Currency        string       `json:"currency"`
	CreatedAtUnixMs int64        `json:"createdAtUnixMs"`
	LotStatus       v1.LotStatus `json:"lotStatus"`
	AuctionState    AuctionState `json:"auctionState"`
	Won             bool         `json:"won"`
}

type BidRecordQuery struct {
	Page     int    `json:"page"`
	PageSize int    `json:"pageSize"`
	LotID    string `json:"lotId,omitempty"`
}

type BidRecordList struct {
	Bids     []BidRecord `json:"bids"`
	Total    int64       `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"pageSize"`
}

type LotResult struct {
	Lot          *v1.Lot       `json:"lot"`
	AuctionState AuctionState  `json:"auctionState"`
	Order        *OrderSummary `json:"order,omitempty"`
}

type LotResultViewer struct {
	UserID string
	Role   v1.UserRole
}

func (v LotResultViewer) CanViewOrder(order *Order) bool {
	if order == nil {
		return false
	}
	switch v.Role {
	case v1.UserRole_USER_ROLE_ADMIN, v1.UserRole_USER_ROLE_ANCHOR, v1.UserRole_USER_ROLE_OPERATOR:
		return true
	case v1.UserRole_USER_ROLE_BUYER:
		return v.UserID != "" && v.UserID == order.BuyerUserID
	default:
		return false
	}
}

type MockPayRequest struct {
	IdempotencyKey string `json:"idempotencyKey"`
	Amount         int64  `json:"amount"`
	Currency       string `json:"currency"`
}

type PaymentResult struct {
	Order   OrderSummary   `json:"order"`
	Payment PaymentSummary `json:"payment"`
	Paid    bool           `json:"paid"`
}

func AuctionStateOf(lot *v1.Lot) AuctionState {
	if lot == nil {
		return AuctionStateFailed
	}
	switch lot.Status {
	case v1.LotStatus_LOT_STATUS_DRAFT, v1.LotStatus_LOT_STATUS_READY:
		return AuctionStateDraft
	case v1.LotStatus_LOT_STATUS_QUEUED:
		return AuctionStateScheduled
	case v1.LotStatus_LOT_STATUS_LIVE:
		if lot.GetDuelState().GetExtendCount() > 0 {
			return AuctionStateExtended
		}
		return AuctionStateLive
	case v1.LotStatus_LOT_STATUS_EXTENDED:
		return AuctionStateExtended
	case v1.LotStatus_LOT_STATUS_SETTLED:
		return AuctionStateSold
	case v1.LotStatus_LOT_STATUS_CANCELLED:
		return AuctionStateCancelled
	case v1.LotStatus_LOT_STATUS_FAILED:
		return AuctionStateFailed
	default:
		return AuctionStateFailed
	}
}

func IsAuctionOpenStatus(status v1.LotStatus) bool {
	return status == v1.LotStatus_LOT_STATUS_LIVE || status == v1.LotStatus_LOT_STATUS_EXTENDED
}

func NormalizePagination(page, pageSize int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}

func PageOffset(page, pageSize int) int {
	page, pageSize = NormalizePagination(page, pageSize)
	return (page - 1) * pageSize
}

func NewOrderFromSettledLot(id string, lot *v1.Lot, nowMs int64) (*Order, error) {
	if id == "" {
		return nil, fmt.Errorf("%w: order id is required", apperr.ErrInvalidArgument)
	}
	if lot == nil {
		return nil, fmt.Errorf("%w: lot is required", apperr.ErrInvalidArgument)
	}
	if lot.WinnerUserId == "" {
		return nil, fmt.Errorf("%w: settled lot winner is required", apperr.ErrInvalidArgument)
	}
	price := lot.GetFinalPrice()
	if price == nil || price.GetCurrency() == "" {
		return nil, fmt.Errorf("%w: settled lot final price is required", apperr.ErrInvalidArgument)
	}
	return &Order{
		ID:              id,
		LotID:           lot.Id,
		RoomID:          lot.RoomId,
		LotTitle:        lot.Title,
		LotImageURL:     lot.ImageUrl,
		BuyerUserID:     lot.WinnerUserId,
		BuyerNickname:   lot.WinnerNickname,
		Status:          OrderStatusPendingPayment,
		PaymentStatus:   PaymentStatusInit,
		Amount:          price.GetAmount(),
		Currency:        price.GetCurrency(),
		CreatedAtUnixMs: nowMs,
		UpdatedAtUnixMs: nowMs,
		ExpiresAtUnixMs: nowMs + 15*60*1000,
		Version:         1,
	}, nil
}

func (o Order) Summary() OrderSummary {
	return OrderSummary{
		ID:              o.ID,
		LotID:           o.LotID,
		RoomID:          o.RoomID,
		LotTitle:        o.LotTitle,
		LotImageURL:     o.LotImageURL,
		BuyerUserID:     o.BuyerUserID,
		Status:          o.Status,
		PaymentStatus:   o.PaymentStatus,
		PaymentID:       o.PaymentID,
		Amount:          o.Amount,
		Currency:        o.Currency,
		CreatedAtUnixMs: o.CreatedAtUnixMs,
		UpdatedAtUnixMs: o.UpdatedAtUnixMs,
		ExpiresAtUnixMs: o.ExpiresAtUnixMs,
		PaidAtUnixMs:    o.PaidAtUnixMs,
	}
}

func NewPayment(id string, order Order, idempotencyKey string, amount int64, currency string, nowMs int64) (*Payment, error) {
	if id == "" {
		return nil, fmt.Errorf("%w: payment id is required", apperr.ErrInvalidArgument)
	}
	if idempotencyKey == "" {
		return nil, fmt.Errorf("%w: payment idempotency key is required", apperr.ErrInvalidArgument)
	}
	if order.ID == "" {
		return nil, fmt.Errorf("%w: order is required", apperr.ErrInvalidArgument)
	}
	if order.Status != OrderStatusPendingPayment {
		return nil, fmt.Errorf("%w: only pending payment order can be paid, current status: %s", apperr.ErrInvalidArgument, order.Status)
	}
	if amount != order.Amount || currency != order.Currency {
		return nil, fmt.Errorf("%w: payment amount does not match order", apperr.ErrInvalidArgument)
	}
	return &Payment{
		ID:              id,
		OrderID:         order.ID,
		LotID:           order.LotID,
		BuyerUserID:     order.BuyerUserID,
		Status:          PaymentStatusInit,
		Amount:          amount,
		Currency:        currency,
		IdempotencyKey:  idempotencyKey,
		CreatedAtUnixMs: nowMs,
		UpdatedAtUnixMs: nowMs,
	}, nil
}

func (p *Payment) MarkProcessing(nowMs int64) error {
	if p == nil {
		return fmt.Errorf("%w: payment is required", apperr.ErrInvalidArgument)
	}
	if p.Status != PaymentStatusInit {
		return fmt.Errorf("%w: only init payment can be processed, current status: %s", apperr.ErrInvalidArgument, p.Status)
	}
	p.Status = PaymentStatusProcessing
	p.UpdatedAtUnixMs = nowMs
	return nil
}

func (p *Payment) MarkSuccess(nowMs int64) error {
	if p == nil {
		return fmt.Errorf("%w: payment is required", apperr.ErrInvalidArgument)
	}
	if p.Status != PaymentStatusProcessing {
		return fmt.Errorf("%w: only processing payment can succeed, current status: %s", apperr.ErrInvalidArgument, p.Status)
	}
	p.Status = PaymentStatusSuccess
	p.UpdatedAtUnixMs = nowMs
	p.SucceededAtMs = nowMs
	return nil
}

func (p Payment) Summary() PaymentSummary {
	return PaymentSummary{
		ID:              p.ID,
		OrderID:         p.OrderID,
		Status:          p.Status,
		Amount:          p.Amount,
		Currency:        p.Currency,
		CreatedAtUnixMs: p.CreatedAtUnixMs,
		SucceededAtMs:   p.SucceededAtMs,
	}
}

func MarkOrderPaid(order *Order, payment Payment, nowMs int64) error {
	if order == nil {
		return fmt.Errorf("%w: order is required", apperr.ErrInvalidArgument)
	}
	if order.Status != OrderStatusPendingPayment {
		return fmt.Errorf("%w: only pending payment order can be paid, current status: %s", apperr.ErrInvalidArgument, order.Status)
	}
	if payment.Status != PaymentStatusSuccess {
		return fmt.Errorf("%w: successful payment is required", apperr.ErrInvalidArgument)
	}
	if payment.Amount != order.Amount || payment.Currency != order.Currency {
		return fmt.Errorf("%w: payment amount does not match order", apperr.ErrInvalidArgument)
	}
	order.Status = OrderStatusPaid
	order.PaymentStatus = PaymentStatusSuccess
	order.PaymentID = payment.ID
	order.PaidAtUnixMs = nowMs
	order.UpdatedAtUnixMs = nowMs
	order.Version++
	return nil
}
