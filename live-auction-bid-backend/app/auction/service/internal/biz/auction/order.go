package auction

import (
	"fmt"
	"strings"

	v1 "live-auction-bid/backend/api/auction/service/v1"
	userbiz "live-auction-bid/backend/app/auction/service/internal/biz/user"
	"live-auction-bid/backend/app/auction/service/internal/pkg/apperr"
	"live-auction-bid/backend/app/auction/service/internal/pkg/clock"
)

type AuctionState string

const (
	AuctionStateDraft     AuctionState = "DRAFT"
	AuctionStateQueued    AuctionState = "QUEUED"
	AuctionStateLive      AuctionState = "LIVE"
	AuctionStateExtended  AuctionState = "EXTENDED"
	AuctionStateSettled   AuctionState = "SETTLED"
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

const OrderPaymentWindowMs int64 = 30 * 60 * 1000

type RoomStatus string

const (
	RoomStatusActive   RoomStatus = "ACTIVE"
	RoomStatusDisabled RoomStatus = "DISABLED"
)

type Room struct {
	ID              string     `json:"id"`
	MainAccountID   string     `json:"mainAccountId"`
	Name            string     `json:"name"`
	Platform        string     `json:"platform"`
	PlatformRoomID  string     `json:"platformRoomId,omitempty"`
	Status          RoomStatus `json:"status"`
	CreatedByUserID string     `json:"createdByUserId,omitempty"`
	CreatedAtUnixMs int64      `json:"createdAtUnixMs"`
	UpdatedAtUnixMs int64      `json:"updatedAtUnixMs"`
}

type RoomState struct {
	RoomID            string `json:"roomId"`
	MainAccountID     string `json:"mainAccountId"`
	ActiveLotID       string `json:"activeLotId,omitempty"`
	ActiveLotVersion  int64  `json:"activeLotVersion,omitempty"`
	NextQueuePosition int32  `json:"nextQueuePosition"`
	UpdatedAtUnixMs   int64  `json:"updatedAtUnixMs"`
}

type RoomQuery struct {
	MainAccountID     string `json:"mainAccountId,omitempty"`
	PublicOnly        bool   `json:"publicOnly,omitempty"`
	PublicVisibleOnly bool   `json:"publicVisibleOnly,omitempty"`
}

func IsPublicVisibleLotStatus(status v1.LotStatus) bool {
	switch status {
	case v1.LotStatus_LOT_STATUS_QUEUED,
		v1.LotStatus_LOT_STATUS_LIVE,
		v1.LotStatus_LOT_STATUS_EXTENDED:
		return true
	default:
		return false
	}
}

func PublicVisibleLotStatuses() []v1.LotStatus {
	return []v1.LotStatus{
		v1.LotStatus_LOT_STATUS_QUEUED,
		v1.LotStatus_LOT_STATUS_LIVE,
		v1.LotStatus_LOT_STATUS_EXTENDED,
	}
}

type Order struct {
	ID              string        `json:"id"`
	MainAccountID   string        `json:"mainAccountId"`
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
	MainAccountID   string        `json:"mainAccountId"`
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
	MainAccountID   string        `json:"mainAccountId"`
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
	Page          int           `json:"page"`
	PageSize      int           `json:"pageSize"`
	MainAccountID string        `json:"mainAccountId,omitempty"`
	Status        OrderStatus   `json:"status,omitempty"`
	PaymentStatus PaymentStatus `json:"paymentStatus,omitempty"`
	LotID         string        `json:"lotId,omitempty"`
	Buyer         string        `json:"buyer,omitempty"`
	BuyerUserID   string        `json:"buyerUserId,omitempty"`
}

type OrderList struct {
	Orders   []OrderSummary `json:"orders"`
	Total    int64          `json:"total"`
	Page     int            `json:"page"`
	PageSize int            `json:"pageSize"`
}

type LotQuery struct {
	Page          int          `json:"page"`
	PageSize      int          `json:"pageSize"`
	MainAccountID string       `json:"mainAccountId,omitempty"`
	Status        v1.LotStatus `json:"status,omitempty"`
	View          string       `json:"view,omitempty"`
	Keyword       string       `json:"keyword,omitempty"`
	RoomID        string       `json:"roomId,omitempty"`
}

type LotList struct {
	Lots     []*v1.Lot `json:"lots"`
	Total    int64     `json:"total"`
	Page     int       `json:"page"`
	PageSize int       `json:"pageSize"`
}

type RoomEventQuery struct {
	RoomID        string `json:"roomId"`
	MainAccountID string `json:"mainAccountId,omitempty"`
	PageSize      int    `json:"pageSize"`
	PageToken     string `json:"pageToken,omitempty"`
}

type RoomEventList struct {
	Events        []*v1.AuctionEvent `json:"events"`
	NextPageToken string             `json:"nextPageToken,omitempty"`
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
	LotStatus       string       `json:"lotStatus"`
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
	UserID          string
	MainAccountID   string
	RoleCodes       []string
	PermissionCodes []string
}

func (v LotResultViewer) CanViewOrder(order *Order) bool {
	if order == nil {
		return false
	}
	if v.hasAnyPermission(userbiz.PermissionOrderManage, userbiz.PermissionLotViewAdmin, userbiz.PermissionAuctionControl) {
		return v.MainAccountID != "" && v.MainAccountID == order.MainAccountID
	}
	return v.hasPermission(userbiz.PermissionOrderViewOwn) && v.UserID != "" && v.UserID == order.BuyerUserID
}

func (v LotResultViewer) hasPermission(permissionCode string) bool {
	permissionCode = strings.TrimSpace(permissionCode)
	for _, got := range v.PermissionCodes {
		if strings.TrimSpace(got) == permissionCode {
			return true
		}
	}
	return false
}

func (v LotResultViewer) hasAnyPermission(permissionCodes ...string) bool {
	for _, permissionCode := range permissionCodes {
		if v.hasPermission(permissionCode) {
			return true
		}
	}
	return false
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
		return AuctionStateQueued
	case v1.LotStatus_LOT_STATUS_LIVE:
		if lot.GetDuelState().GetExtendCount() > 0 {
			return AuctionStateExtended
		}
		return AuctionStateLive
	case v1.LotStatus_LOT_STATUS_EXTENDED:
		return AuctionStateExtended
	case v1.LotStatus_LOT_STATUS_SETTLED:
		return AuctionStateSettled
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
		MainAccountID:   lot.GetMainAccountId(),
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
		ExpiresAtUnixMs: nowMs + OrderPaymentWindowMs,
		Version:         1,
	}, nil
}

func (o Order) effectiveStatus(nowMs int64) (OrderStatus, PaymentStatus) {
	if o.Status == OrderStatusPendingPayment && o.ExpiresAtUnixMs > 0 && o.ExpiresAtUnixMs <= nowMs {
		return OrderStatusExpired, PaymentStatusClosed
	}
	return o.Status, o.PaymentStatus
}

func (o Order) Summary() OrderSummary {
	status, paymentStatus := o.effectiveStatus(clock.NowMs())
	return OrderSummary{
		ID:              o.ID,
		MainAccountID:   o.MainAccountID,
		LotID:           o.LotID,
		RoomID:          o.RoomID,
		LotTitle:        o.LotTitle,
		LotImageURL:     o.LotImageURL,
		BuyerUserID:     o.BuyerUserID,
		Status:          status,
		PaymentStatus:   paymentStatus,
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
		MainAccountID:   order.MainAccountID,
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
