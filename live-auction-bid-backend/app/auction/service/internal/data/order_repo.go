package data

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"gorm.io/gorm"
	v1 "live-auction-bid/backend/api/auction/service/v1"
	"live-auction-bid/backend/app/auction/service/internal/biz/auction"
	"live-auction-bid/backend/app/auction/service/internal/pkg/apperr"
)

func (s *Store) CreateOrderForSettledLot(ctx context.Context, order auction.Order, lot *v1.Lot, expectedLotVersion int64, events []v1.AuctionEvent) error {
	if lot == nil {
		return errors.New("lot is required")
	}
	if expectedLotVersion <= 0 {
		return errors.New("lot expected version is required")
	}
	lotModel, err := lotToModel(lot)
	if err != nil {
		return err
	}
	orderModel, err := orderToModel(order)
	if err != nil {
		return err
	}
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&AuctionLotModel{}).
			Where("id = ? AND version = ?", lot.Id, expectedLotVersion).
			Select("*").
			Omit("created_at").
			Updates(lotModel)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return apperr.ErrLotVersionConflict
		}
		if err := tx.Create(orderModel).Error; err != nil {
			return err
		}
		return createEventModels(ctx, tx, events)
	}); err != nil {
		return err
	}
	return s.streamEvents(ctx, events)
}

func (s *Store) FindOrderByID(ctx context.Context, orderID string) (*auction.Order, error) {
	if orderID == "" {
		return nil, errors.New("order id is required")
	}
	var model AuctionOrderModel
	if err := s.db.WithContext(ctx).Where("id = ?", orderID).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperr.ErrNotFound
		}
		return nil, err
	}
	return modelToOrder(&model)
}

func (s *Store) FindOrderByLot(ctx context.Context, lotID string) (*auction.Order, bool, error) {
	if lotID == "" {
		return nil, false, errors.New("lot id is required")
	}
	var model AuctionOrderModel
	if err := s.db.WithContext(ctx).Where("lot_id = ?", lotID).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, false, nil
		}
		return nil, false, err
	}
	order, err := modelToOrder(&model)
	return order, err == nil, err
}

func (s *Store) ListOrdersByBuyer(ctx context.Context, buyerUserID string) ([]auction.Order, error) {
	if buyerUserID == "" {
		return nil, errors.New("buyer user id is required")
	}
	var models []AuctionOrderModel
	if err := s.db.WithContext(ctx).
		Where("buyer_user_id = ?", buyerUserID).
		Order("created_at_unix_ms DESC").
		Order("id ASC").
		Find(&models).Error; err != nil {
		return nil, err
	}
	orders := make([]auction.Order, 0, len(models))
	for i := range models {
		order, err := modelToOrder(&models[i])
		if err != nil {
			return nil, err
		}
		orders = append(orders, *order)
	}
	return orders, nil
}

func (s *Store) ListOrders(ctx context.Context, query auction.OrderQuery) (auction.OrderList, error) {
	query.Page, query.PageSize = auction.NormalizePagination(query.Page, query.PageSize)
	db := s.db.WithContext(ctx).Model(&AuctionOrderModel{})
	if query.BuyerUserID != "" {
		db = db.Where("buyer_user_id = ?", query.BuyerUserID)
	}
	if query.Status != "" {
		db = db.Where("status = ?", string(query.Status))
	}
	if query.PaymentStatus != "" {
		db = db.Where("payment_status = ?", string(query.PaymentStatus))
	}
	if query.LotID != "" {
		db = db.Where("lot_id = ?", query.LotID)
	}
	if buyer := strings.TrimSpace(query.Buyer); buyer != "" {
		like := "%" + buyer + "%"
		db = db.Where("buyer_user_id LIKE ? OR buyer_nickname LIKE ?", like, like)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return auction.OrderList{}, err
	}
	var models []AuctionOrderModel
	if err := db.
		Order("created_at_unix_ms DESC").
		Order("id ASC").
		Offset(auction.PageOffset(query.Page, query.PageSize)).
		Limit(query.PageSize).
		Find(&models).Error; err != nil {
		return auction.OrderList{}, err
	}
	orders := make([]auction.OrderSummary, 0, len(models))
	for i := range models {
		order, err := modelToOrder(&models[i])
		if err != nil {
			return auction.OrderList{}, err
		}
		orders = append(orders, order.Summary())
	}
	return auction.OrderList{Orders: orders, Total: total, Page: query.Page, PageSize: query.PageSize}, nil
}

func (s *Store) FindPaymentByIdempotencyKey(ctx context.Context, orderID, key string) (*auction.Payment, bool, error) {
	if orderID == "" {
		return nil, false, errors.New("order id is required")
	}
	if key == "" {
		return nil, false, errors.New("payment idempotency key is required")
	}
	var model AuctionPaymentModel
	if err := s.db.WithContext(ctx).
		Where("order_id = ? AND idempotency_key = ?", orderID, key).
		First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, false, nil
		}
		return nil, false, err
	}
	payment, err := modelToPayment(&model)
	return payment, err == nil, err
}

func (s *Store) CommitPaymentSuccess(ctx context.Context, payment auction.Payment, order auction.Order, expectedOrderVersion int64, events []v1.AuctionEvent) error {
	if expectedOrderVersion <= 0 {
		return errors.New("order expected version is required")
	}
	paymentModel, err := paymentToModel(payment)
	if err != nil {
		return err
	}
	orderModel, err := orderToModel(order)
	if err != nil {
		return err
	}
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(paymentModel).Error; err != nil {
			return err
		}
		result := tx.Model(&AuctionOrderModel{}).
			Where("id = ? AND version = ?", order.ID, expectedOrderVersion).
			Select("*").
			Omit("created_at").
			Updates(orderModel)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return apperr.ErrLotVersionConflict
		}
		return createEventModels(ctx, tx, events)
	}); err != nil {
		return err
	}
	return s.streamEvents(ctx, events)
}

func orderToModel(order auction.Order) (*AuctionOrderModel, error) {
	payload, err := json.Marshal(order)
	if err != nil {
		return nil, err
	}
	return &AuctionOrderModel{
		ID:              order.ID,
		LotID:           order.LotID,
		RoomID:          order.RoomID,
		LotTitle:        order.LotTitle,
		LotImageURL:     order.LotImageURL,
		BuyerUserID:     order.BuyerUserID,
		BuyerNickname:   order.BuyerNickname,
		Status:          string(order.Status),
		PaymentStatus:   string(order.PaymentStatus),
		PaymentID:       order.PaymentID,
		Amount:          order.Amount,
		Currency:        order.Currency,
		CreatedAtUnixMs: order.CreatedAtUnixMs,
		UpdatedAtUnixMs: order.UpdatedAtUnixMs,
		ExpiresAtUnixMs: order.ExpiresAtUnixMs,
		PaidAtUnixMs:    order.PaidAtUnixMs,
		Version:         order.Version,
		Payload:         string(payload),
	}, nil
}

func modelToOrder(model *AuctionOrderModel) (*auction.Order, error) {
	var order auction.Order
	if err := json.Unmarshal([]byte(model.Payload), &order); err != nil {
		return nil, err
	}
	order.ID = model.ID
	order.LotID = model.LotID
	order.RoomID = model.RoomID
	order.LotTitle = model.LotTitle
	order.LotImageURL = model.LotImageURL
	order.BuyerUserID = model.BuyerUserID
	order.BuyerNickname = model.BuyerNickname
	order.Status = auction.OrderStatus(model.Status)
	order.PaymentStatus = auction.PaymentStatus(model.PaymentStatus)
	order.PaymentID = model.PaymentID
	order.Amount = model.Amount
	order.Currency = model.Currency
	order.CreatedAtUnixMs = model.CreatedAtUnixMs
	order.UpdatedAtUnixMs = model.UpdatedAtUnixMs
	order.ExpiresAtUnixMs = model.ExpiresAtUnixMs
	order.PaidAtUnixMs = model.PaidAtUnixMs
	order.Version = model.Version
	return &order, nil
}

func paymentToModel(payment auction.Payment) (*AuctionPaymentModel, error) {
	payload, err := json.Marshal(payment)
	if err != nil {
		return nil, err
	}
	var idem *string
	if payment.IdempotencyKey != "" {
		idem = &payment.IdempotencyKey
	}
	return &AuctionPaymentModel{
		ID:              payment.ID,
		OrderID:         payment.OrderID,
		LotID:           payment.LotID,
		BuyerUserID:     payment.BuyerUserID,
		Status:          string(payment.Status),
		Amount:          payment.Amount,
		Currency:        payment.Currency,
		IdempotencyKey:  idem,
		CreatedAtUnixMs: payment.CreatedAtUnixMs,
		UpdatedAtUnixMs: payment.UpdatedAtUnixMs,
		SucceededAtMs:   payment.SucceededAtMs,
		Payload:         string(payload),
	}, nil
}

func modelToPayment(model *AuctionPaymentModel) (*auction.Payment, error) {
	var payment auction.Payment
	if err := json.Unmarshal([]byte(model.Payload), &payment); err != nil {
		return nil, err
	}
	payment.ID = model.ID
	payment.OrderID = model.OrderID
	payment.LotID = model.LotID
	payment.BuyerUserID = model.BuyerUserID
	payment.Status = auction.PaymentStatus(model.Status)
	payment.Amount = model.Amount
	payment.Currency = model.Currency
	if model.IdempotencyKey != nil {
		payment.IdempotencyKey = *model.IdempotencyKey
	}
	payment.CreatedAtUnixMs = model.CreatedAtUnixMs
	payment.UpdatedAtUnixMs = model.UpdatedAtUnixMs
	payment.SucceededAtMs = model.SucceededAtMs
	return &payment, nil
}
