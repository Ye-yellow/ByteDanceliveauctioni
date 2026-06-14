package data

import (
	"context"
	"errors"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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
	orderModel, itemModel, err := auctionOrderToUserModels(order)
	if err != nil {
		return err
	}
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockRoomStateForTerminalLot(ctx, tx, lot); err != nil {
			return err
		}
		var current AuctionLotModel
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", lot.Id).
			First(&current).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return apperr.ErrNotFound
			}
			return err
		}
		if current.Version != expectedLotVersion {
			return apperr.ErrLotVersionConflict
		}
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
		if err := releaseActiveLotIfTerminal(ctx, tx, lot); err != nil {
			return err
		}
		if err := s.fillAuctionOrderShopName(ctx, tx, orderModel); err != nil {
			return err
		}
		if err := tx.Create(orderModel).Error; err != nil {
			return err
		}
		if err := tx.Create(itemModel).Error; err != nil {
			return err
		}
		return createEventModels(ctx, tx, events)
	}); err != nil {
		return err
	}
	return s.streamEvents(ctx, events)
}

func (s *Store) fillAuctionOrderShopName(ctx context.Context, tx *gorm.DB, orderModel *UserOrderModel) error {
	if orderModel == nil || orderModel.Source != userOrderSourceAuction {
		return nil
	}
	name, err := s.auctionShopNameForMainAccount(ctx, tx, orderModel.MainAccountID)
	if err != nil {
		return err
	}
	orderModel.ShopName = name
	return nil
}

func (s *Store) auctionShopNameForMainAccount(ctx context.Context, tx *gorm.DB, mainAccountID string) (string, error) {
	mainAccountID = strings.TrimSpace(mainAccountID)
	if mainAccountID == "" {
		return "直播竞拍", nil
	}
	var user AuctionUserModel
	if err := tx.WithContext(ctx).
		Where("id = ?", mainAccountID).
		First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "直播竞拍", nil
		}
		return "", err
	}
	if name := strings.TrimSpace(user.Nickname); name != "" {
		return name, nil
	}
	if name := strings.TrimSpace(user.Username); name != "" {
		return name, nil
	}
	return "直播竞拍", nil
}

func (s *Store) FindOrderByID(ctx context.Context, orderID string) (*auction.Order, error) {
	if orderID == "" {
		return nil, errors.New("order id is required")
	}
	var model UserOrderModel
	if err := s.db.WithContext(ctx).
		Where("id = ? AND source = ?", orderID, userOrderSourceAuction).
		First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperr.ErrNotFound
		}
		return nil, err
	}
	items, err := s.findUserOrderItems(ctx, model.ID)
	if err != nil {
		return nil, err
	}
	return userModelToAuctionOrderWithItem(&model, items)
}

func (s *Store) FindOrderByLot(ctx context.Context, lotID string) (*auction.Order, bool, error) {
	if lotID == "" {
		return nil, false, errors.New("lot id is required")
	}
	var item UserOrderItemModel
	if err := s.db.WithContext(ctx).
		Where("source = ? AND lot_id = ?", userOrderSourceAuction, lotID).
		First(&item).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, false, nil
		}
		return nil, false, err
	}
	order, err := s.FindOrderByID(ctx, item.OrderID)
	return order, err == nil, err
}

func (s *Store) ListOrdersByBuyer(ctx context.Context, buyerUserID string) ([]auction.Order, error) {
	if buyerUserID == "" {
		return nil, errors.New("buyer user id is required")
	}
	var models []UserOrderModel
	if err := s.db.WithContext(ctx).
		Where("source = ? AND user_id = ?", userOrderSourceAuction, buyerUserID).
		Order("created_at_unix_ms DESC").
		Order("id ASC").
		Find(&models).Error; err != nil {
		return nil, err
	}
	return s.auctionOrdersFromUserModels(ctx, models)
}

func (s *Store) ListOrders(ctx context.Context, query auction.OrderQuery) (auction.OrderList, error) {
	query.Page, query.PageSize = auction.NormalizePagination(query.Page, query.PageSize)
	db := s.db.WithContext(ctx).Model(&UserOrderModel{}).Where("source = ?", userOrderSourceAuction)
	if query.MainAccountID != "" {
		db = db.Where("main_account_id = ?", query.MainAccountID)
	}
	if query.BuyerUserID != "" {
		db = db.Where("user_id = ?", query.BuyerUserID)
	}
	if query.Status != "" {
		db = db.Where("status = ?", string(auctionOrderStatusToUser(query.Status)))
	}
	if query.PaymentStatus != "" {
		db = db.Where("payment_status = ?", string(auctionPaymentStatusToUser(query.PaymentStatus)))
	}
	if query.LotID != "" {
		db = db.Where(
			"EXISTS (SELECT 1 FROM user_order_items WHERE user_order_items.order_id = user_orders.id AND user_order_items.source = ? AND user_order_items.lot_id = ?)",
			userOrderSourceAuction,
			query.LotID,
		)
	}
	if buyer := strings.TrimSpace(query.Buyer); buyer != "" {
		like := "%" + buyer + "%"
		db = db.Where("user_id LIKE ? OR nickname LIKE ?", like, like)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return auction.OrderList{}, err
	}
	var models []UserOrderModel
	if err := db.
		Order("created_at_unix_ms DESC").
		Order("id ASC").
		Offset(auction.PageOffset(query.Page, query.PageSize)).
		Limit(query.PageSize).
		Find(&models).Error; err != nil {
		return auction.OrderList{}, err
	}
	orders, err := s.auctionOrdersFromUserModels(ctx, models)
	if err != nil {
		return auction.OrderList{}, err
	}
	summaries := make([]auction.OrderSummary, 0, len(orders))
	for _, order := range orders {
		summaries = append(summaries, order.Summary())
	}
	return auction.OrderList{Orders: summaries, Total: total, Page: query.Page, PageSize: query.PageSize}, nil
}

func (s *Store) FindPaymentByIdempotencyKey(ctx context.Context, orderID, key string) (*auction.Payment, bool, error) {
	if orderID == "" {
		return nil, false, errors.New("order id is required")
	}
	if key == "" {
		return nil, false, errors.New("payment idempotency key is required")
	}
	var model UserOrderPaymentModel
	if err := s.db.WithContext(ctx).
		Where("source = ? AND order_id = ? AND idempotency_key = ?", userOrderSourceAuction, orderID, key).
		First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, false, nil
		}
		return nil, false, err
	}
	payment, err := userModelToAuctionPayment(&model)
	return payment, err == nil, err
}

func (s *Store) CommitPaymentSuccess(ctx context.Context, payment auction.Payment, order auction.Order, expectedOrderVersion int64, events []v1.AuctionEvent) error {
	if expectedOrderVersion <= 0 {
		return errors.New("order expected version is required")
	}
	paymentModel, err := auctionPaymentToUserModel(payment)
	if err != nil {
		return err
	}
	orderModel, _, err := auctionOrderToUserModels(order)
	if err != nil {
		return err
	}
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(paymentModel).Error; err != nil {
			return err
		}
		updates := map[string]any{
			"status":                  orderModel.Status,
			"payment_status":          orderModel.PaymentStatus,
			"payment_id":              orderModel.PaymentID,
			"payment_idempotency_key": payment.IdempotencyKey,
			"paid_at_unix_ms":         orderModel.PaidAtUnixMs,
			"updated_at_unix_ms":      orderModel.UpdatedAtUnixMs,
			"version":                 orderModel.Version,
			"source_payload":          orderModel.SourcePayload,
		}
		result := tx.Model(&UserOrderModel{}).
			Where("id = ? AND source = ? AND version = ?", order.ID, userOrderSourceAuction, expectedOrderVersion).
			Updates(updates)
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

func (s *Store) findUserOrderItems(ctx context.Context, orderID string) ([]UserOrderItemModel, error) {
	var items []UserOrderItemModel
	if err := s.db.WithContext(ctx).
		Where("order_id = ?", orderID).
		Order("id ASC").
		Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (s *Store) auctionOrdersFromUserModels(ctx context.Context, models []UserOrderModel) ([]auction.Order, error) {
	if len(models) == 0 {
		return []auction.Order{}, nil
	}
	orderIDs := make([]string, 0, len(models))
	for _, model := range models {
		orderIDs = append(orderIDs, model.ID)
	}
	itemsByOrder, err := s.userOrderItemsByOrderID(ctx, orderIDs)
	if err != nil {
		return nil, err
	}
	orders := make([]auction.Order, 0, len(models))
	for i := range models {
		order, err := userModelToAuctionOrderWithItem(&models[i], itemsByOrder[models[i].ID])
		if err != nil {
			return nil, err
		}
		orders = append(orders, *order)
	}
	return orders, nil
}

func (s *Store) userOrderItemsByOrderID(ctx context.Context, orderIDs []string) (map[string][]UserOrderItemModel, error) {
	itemsByOrder := make(map[string][]UserOrderItemModel, len(orderIDs))
	if len(orderIDs) == 0 {
		return itemsByOrder, nil
	}
	var itemModels []UserOrderItemModel
	if err := s.db.WithContext(ctx).
		Where("order_id IN ?", orderIDs).
		Order("id ASC").
		Find(&itemModels).Error; err != nil {
		return nil, err
	}
	for _, item := range itemModels {
		itemsByOrder[item.OrderID] = append(itemsByOrder[item.OrderID], item)
	}
	return itemsByOrder, nil
}

func createUserOrderWithItemsIgnoringDuplicates(ctx context.Context, tx *gorm.DB, orderModel *UserOrderModel, itemModels ...*UserOrderItemModel) error {
	if orderModel == nil {
		return nil
	}
	if err := tx.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(orderModel).Error; err != nil {
		return err
	}
	for _, itemModel := range itemModels {
		if itemModel == nil {
			continue
		}
		if err := tx.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(itemModel).Error; err != nil {
			return err
		}
	}
	return nil
}
