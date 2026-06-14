package data

import (
	"context"
	"errors"
	"time"

	"google.golang.org/protobuf/encoding/protojson"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	v1 "live-auction-bid/backend/api/auction/service/v1"
	"live-auction-bid/backend/app/auction/service/internal/biz/auction"
	"live-auction-bid/backend/app/auction/service/internal/pkg/apperr"
)

const bidIdempotencyCacheTTL = 24 * time.Hour

func (s *Store) CommitAcceptedBid(ctx context.Context, bid v1.Bid, lot *v1.Lot, expectedLotVersion int64, idempotencyKey string, order *auction.Order, events []v1.AuctionEvent) error {
	if bid.Id == "" {
		return errors.New("bid id is required")
	}
	if bid.LotId == "" {
		return errors.New("lot id is required")
	}
	if bid.UserId == "" {
		return errors.New("user id is required")
	}
	if bid.GetAmount() == nil || bid.GetAmount().GetCurrency() == "" {
		return errors.New("bid amount and currency are required")
	}
	if lot == nil {
		return errors.New("lot is required")
	}
	if expectedLotVersion <= 0 {
		return errors.New("lot expected version is required")
	}
	bidModel, err := bidToModel(bid, idempotencyKey, lot.GetMainAccountId())
	if err != nil {
		return err
	}
	lotModel, err := lotToModel(lot)
	if err != nil {
		return err
	}
	var orderModel *UserOrderModel
	var orderItemModel *UserOrderItemModel
	if order != nil {
		orderModel, orderItemModel, err = auctionOrderToUserModels(*order)
		if err != nil {
			return err
		}
	}
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockRoomStateForTerminalLot(ctx, tx, lot); err != nil {
			return err
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
		if err := tx.Create(bidModel).Error; err != nil {
			return err
		}
		if err := projectStatsForInsertedBid(ctx, tx, bid, lot); err != nil {
			return err
		}
		if orderModel != nil {
			if err := s.fillAuctionOrderShopName(ctx, tx, orderModel); err != nil {
				return err
			}
			if err := createUserOrderWithItemsIgnoringDuplicates(ctx, tx, orderModel, orderItemModel); err != nil {
				return err
			}
		}
		return createEventModels(ctx, tx, events)
	}); err != nil {
		return err
	}
	return s.streamEvents(ctx, events)
}

func (s *Store) ProjectRuntimeBid(ctx context.Context, bid v1.Bid, lot *v1.Lot, idempotencyKey string, order *auction.Order, events []v1.AuctionEvent) error {
	projection := auction.RuntimeProjectionEvent{
		RuntimeStreamID:    "",
		RoomID:             lot.GetRoomId(),
		LotID:              lot.GetId(),
		EventType:          "BID_ACCEPTED",
		IdempotencyKey:     idempotencyKey,
		Bid:                bid,
		Lot:                lot,
		PreviousLotVersion: lot.GetVersion() - 1,
		LotVersion:         lot.GetVersion(),
		OccurredAtUnixMs:   bid.GetCreatedAtUnixMs(),
	}
	if order != nil {
		projection.OrderID = order.ID
		projection.ShippingAddressID = order.ShippingAddressID
		if order.ShippingAddressSnapshot != nil {
			snapshot := *order.ShippingAddressSnapshot
			projection.ShippingAddressSnapshot = &snapshot
		}
	}
	_, err := s.ProjectRuntimeEvent(ctx, projection, order, events)
	return err
}

func (s *Store) ProjectRuntimeEvent(ctx context.Context, projection auction.RuntimeProjectionEvent, order *auction.Order, events []v1.AuctionEvent) (auction.RuntimeProjectionOutcome, error) {
	outcomes, err := s.projectRuntimeEventsBatch(ctx, []runtimeProjectionBatchItem{{
		projection: projection,
		events:     events,
		order:      order,
	}})
	if err != nil {
		if len(outcomes) > 0 {
			return outcomes[0], err
		}
		return auction.RuntimeProjectionOutcome{}, err
	}
	if len(outcomes) == 0 {
		return auction.RuntimeProjectionOutcome{}, nil
	}
	return outcomes[0], nil
}

func (s *Store) projectRuntimeEventsBatch(ctx context.Context, items []runtimeProjectionBatchItem) ([]auction.RuntimeProjectionOutcome, error) {
	outcomes := make([]auction.RuntimeProjectionOutcome, len(items))
	if len(items) == 0 {
		return outcomes, nil
	}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for i, item := range items {
			outcome, err := s.projectRuntimeEventInTx(ctx, tx, item.projection, item.order, item.events)
			outcomes[i] = outcome
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return outcomes, err
	}
	for i, outcome := range outcomes {
		if !outcome.Projected {
			continue
		}
		if err := s.streamEvents(ctx, items[i].events); err != nil {
			return outcomes, err
		}
	}
	return outcomes, nil
}

func (s *Store) projectRuntimeEventInTx(ctx context.Context, tx *gorm.DB, projection auction.RuntimeProjectionEvent, order *auction.Order, events []v1.AuctionEvent) (auction.RuntimeProjectionOutcome, error) {
	bid := projection.Bid
	lot := projection.Lot
	if bid.Id == "" {
		return auction.RuntimeProjectionOutcome{}, errors.New("bid id is required")
	}
	if bid.LotId == "" {
		return auction.RuntimeProjectionOutcome{}, errors.New("lot id is required")
	}
	if bid.UserId == "" {
		return auction.RuntimeProjectionOutcome{}, errors.New("user id is required")
	}
	if bid.GetAmount() == nil || bid.GetAmount().GetCurrency() == "" {
		return auction.RuntimeProjectionOutcome{}, errors.New("bid amount and currency are required")
	}
	if lot == nil {
		return auction.RuntimeProjectionOutcome{}, errors.New("lot is required")
	}
	if projection.PreviousLotVersion <= 0 || projection.LotVersion <= projection.PreviousLotVersion {
		return auction.RuntimeProjectionOutcome{}, errors.New("runtime projection lot version is required")
	}
	idempotencyKey := projection.IdempotencyKey
	bidModel, err := bidToModel(bid, idempotencyKey, lot.GetMainAccountId())
	if err != nil {
		return auction.RuntimeProjectionOutcome{}, err
	}
	lotModel, err := lotToModel(lot)
	if err != nil {
		return auction.RuntimeProjectionOutcome{}, err
	}
	var orderModel *UserOrderModel
	var orderItemModel *UserOrderItemModel
	if order != nil {
		orderModel, orderItemModel, err = auctionOrderToUserModels(*order)
		if err != nil {
			return auction.RuntimeProjectionOutcome{}, err
		}
		if err := s.fillAuctionOrderShopName(ctx, tx, orderModel); err != nil {
			return auction.RuntimeProjectionOutcome{}, err
		}
	}
	outcome := auction.RuntimeProjectionOutcome{}
	if err := lockRoomStateForTerminalLot(ctx, tx, lot); err != nil {
		return outcome, err
	}
	var current AuctionLotModel
	if err := tx.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", lot.Id).
		First(&current).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return outcome, apperr.ErrNotFound
		}
		return outcome, err
	}

	offset, err := lockProjectionOffset(ctx, tx, lot.Id, lot.RoomId, current.Version, projection.RuntimeStreamID, bid.CreatedAtUnixMs)
	if err != nil {
		return outcome, err
	}
	if projection.LotVersion <= offset.LastProjectedVersion {
		if err := projectRuntimeFactsWithoutLotUpdate(ctx, tx, bidModel, bid, lot, nil, nil, events); err != nil {
			return outcome, err
		}
		outcome.AlreadyProjected = true
		return outcome, nil
	}
	if offset.LastProjectedVersion != projection.PreviousLotVersion || current.Version != projection.PreviousLotVersion {
		if canFastForwardRuntimeProjection(current, offset, projection) {
			var projectionOrder *UserOrderModel
			var projectionOrderItem *UserOrderItemModel
			if v1.LotStatus(current.Status) == v1.LotStatus_LOT_STATUS_SETTLED && lot.GetStatus() == v1.LotStatus_LOT_STATUS_SETTLED {
				projectionOrder = orderModel
				projectionOrderItem = orderItemModel
			}
			if err := projectRuntimeFactsWithoutLotUpdate(ctx, tx, bidModel, bid, lot, projectionOrder, projectionOrderItem, events); err != nil {
				return outcome, err
			}
			if err := updateProjectionOffset(ctx, tx, lot.Id, lot.RoomId, projection.LotVersion, projection.RuntimeStreamID, bid.CreatedAtUnixMs); err != nil {
				return outcome, err
			}
			outcome.AlreadyProjected = true
			return outcome, nil
		}
		outcome.Gap = true
		return outcome, apperr.ErrRuntimeProjectionGap
	}
	lotUpdate := tx.Model(&AuctionLotModel{}).
		Where("id = ? AND version = ?", lot.Id, projection.PreviousLotVersion).
		Select("*").
		Omit("created_at").
		Updates(lotModel)
	if lotUpdate.Error != nil {
		return outcome, lotUpdate.Error
	}
	if lotUpdate.RowsAffected == 0 {
		exists, err := runtimeProjectionBidExists(ctx, tx, bid.Id, bid.LotId, bid.UserId, idempotencyKey)
		if err != nil {
			return outcome, err
		}
		if exists {
			outcome.AlreadyProjected = true
			return outcome, nil
		}
		outcome.Conflict = true
		return outcome, apperr.ErrRuntimeProjectionConflict
	}
	if err := releaseActiveLotIfTerminal(ctx, tx, lot); err != nil {
		return outcome, err
	}

	bidCreate := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(bidModel)
	if bidCreate.Error != nil {
		return outcome, bidCreate.Error
	}
	if bidCreate.RowsAffected > 0 {
		outcome.Projected = true
		if err := projectStatsForInsertedBid(ctx, tx, bid, lot); err != nil {
			return outcome, err
		}
	} else {
		outcome.AlreadyProjected = true
	}
	if orderModel != nil {
		if err := createUserOrderWithItemsIgnoringDuplicates(ctx, tx, orderModel, orderItemModel); err != nil {
			return outcome, err
		}
	}
	if err := createEventModelsIgnoringDuplicates(ctx, tx, events); err != nil {
		return outcome, err
	}
	return outcome, tx.WithContext(ctx).
		Model(&AuctionRuntimeProjectionOffsetModel{}).
		Where("lot_id = ?", lot.Id).
		Updates(map[string]any{
			"room_id":                lot.RoomId,
			"last_projected_version": projection.LotVersion,
			"last_stream_id":         projection.RuntimeStreamID,
			"updated_at_unix_ms":     bid.CreatedAtUnixMs,
		}).Error
}

func canFastForwardRuntimeProjection(current AuctionLotModel, offset AuctionRuntimeProjectionOffsetModel, projection auction.RuntimeProjectionEvent) bool {
	if projection.LotVersion <= 0 {
		return false
	}
	if current.Version < projection.LotVersion {
		return false
	}
	if offset.LastProjectedVersion >= projection.LotVersion {
		return true
	}
	if current.Version > projection.PreviousLotVersion {
		return true
	}
	return false
}

func projectRuntimeFactsWithoutLotUpdate(ctx context.Context, tx *gorm.DB, bidModel *AuctionBidModel, bid v1.Bid, lot *v1.Lot, orderModel *UserOrderModel, orderItemModel *UserOrderItemModel, events []v1.AuctionEvent) error {
	bidCreate := tx.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(bidModel)
	if bidCreate.Error != nil {
		return bidCreate.Error
	}
	if bidCreate.RowsAffected > 0 {
		if err := projectStatsForInsertedBid(ctx, tx, bid, lot); err != nil {
			return err
		}
	}
	if orderModel != nil && lot.GetStatus() == v1.LotStatus_LOT_STATUS_SETTLED {
		if err := createUserOrderWithItemsIgnoringDuplicates(ctx, tx, orderModel, orderItemModel); err != nil {
			return err
		}
	}
	return createEventModelsIgnoringDuplicates(ctx, tx, events)
}

func updateProjectionOffset(ctx context.Context, tx *gorm.DB, lotID, roomID string, version int64, streamID string, updatedAtUnixMs int64) error {
	if updatedAtUnixMs <= 0 {
		updatedAtUnixMs = time.Now().UnixMilli()
	}
	return tx.WithContext(ctx).
		Model(&AuctionRuntimeProjectionOffsetModel{}).
		Where("lot_id = ?", lotID).
		Updates(map[string]any{
			"room_id":                roomID,
			"last_projected_version": version,
			"last_stream_id":         streamID,
			"updated_at_unix_ms":     updatedAtUnixMs,
		}).Error
}

func lockProjectionOffset(ctx context.Context, tx *gorm.DB, lotID, roomID string, currentVersion int64, streamID string, nowMs int64) (AuctionRuntimeProjectionOffsetModel, error) {
	if nowMs <= 0 {
		nowMs = time.Now().UnixMilli()
	}
	offset := AuctionRuntimeProjectionOffsetModel{}
	err := tx.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("lot_id = ?", lotID).
		First(&offset).Error
	if err == nil {
		return offset, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return offset, err
	}
	offset = AuctionRuntimeProjectionOffsetModel{
		LotID:                lotID,
		RoomID:               roomID,
		LastProjectedVersion: currentVersion,
		LastStreamID:         streamID,
		UpdatedAtUnixMs:      nowMs,
	}
	if err := tx.WithContext(ctx).Create(&offset).Error; err != nil {
		return offset, err
	}
	return offset, nil
}

func runtimeProjectionBidExists(ctx context.Context, tx *gorm.DB, bidID, lotID, userID, idempotencyKey string) (bool, error) {
	var count int64
	query := tx.WithContext(ctx).Model(&AuctionBidModel{}).Where("id = ?", bidID)
	if err := query.Count(&count).Error; err != nil {
		return false, err
	}
	if count > 0 {
		return true, nil
	}
	if lotID == "" || userID == "" || idempotencyKey == "" {
		return false, nil
	}
	if err := tx.WithContext(ctx).Model(&AuctionBidModel{}).
		Where("lot_id = ? AND user_id = ? AND idempotency_key = ?", lotID, userID, idempotencyKey).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func projectStatsForInsertedBid(ctx context.Context, tx *gorm.DB, bid v1.Bid, lot *v1.Lot) error {
	if lot == nil {
		return errors.New("lot is required")
	}
	participant := AuctionLotParticipantModel{
		LotID:            bid.LotId,
		UserID:           bid.UserId,
		MainAccountID:    lot.GetMainAccountId(),
		RoomID:           lot.RoomId,
		FirstBidID:       bid.Id,
		FirstBidAtUnixMs: bid.CreatedAtUnixMs,
	}
	participantCreate := tx.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&participant)
	if participantCreate.Error != nil {
		return participantCreate.Error
	}
	participantDelta := int64(0)
	if participantCreate.RowsAffected > 0 {
		participantDelta = 1
	}
	stats := AuctionLotStatsModel{
		LotID:           bid.LotId,
		MainAccountID:   lot.GetMainAccountId(),
		RoomID:          lot.RoomId,
		UpdatedAtUnixMs: bid.CreatedAtUnixMs,
	}
	if err := tx.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&stats).Error; err != nil {
		return err
	}
	return tx.WithContext(ctx).
		Model(&AuctionLotStatsModel{}).
		Where("lot_id = ?", bid.LotId).
		Updates(map[string]any{
			"room_id":             lot.RoomId,
			"main_account_id":     lot.GetMainAccountId(),
			"bid_count":           gorm.Expr("bid_count + ?", 1),
			"participant_count":   gorm.Expr("participant_count + ?", participantDelta),
			"last_bid_id":         bid.Id,
			"last_bid_at_unix_ms": bid.CreatedAtUnixMs,
			"projected_version":   gorm.Expr("GREATEST(projected_version, ?)", lot.Version),
			"updated_at_unix_ms":  bid.CreatedAtUnixMs,
		}).Error
}

func createEventModelsIgnoringDuplicates(ctx context.Context, tx *gorm.DB, events []v1.AuctionEvent) error {
	for _, event := range events {
		model, err := eventToModel(event)
		if err != nil {
			return err
		}
		if err := tx.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(model).Error; err != nil {
			return err
		}
	}
	return nil
}

func bidToModel(bid v1.Bid, idempotencyKey string, mainAccountID string) (*AuctionBidModel, error) {
	payload, err := protojson.Marshal(&bid)
	if err != nil {
		return nil, err
	}
	return &AuctionBidModel{
		ID:              bid.Id,
		MainAccountID:   mainAccountID,
		LotID:           bid.LotId,
		UserID:          bid.UserId,
		Nickname:        bid.Nickname,
		Amount:          bid.GetAmount().GetAmount(),
		Currency:        bid.GetAmount().GetCurrency(),
		IdempotencyKey:  idempotencyKey,
		CreatedAtUnixMs: bid.CreatedAtUnixMs,
		Payload:         string(payload),
	}, nil
}

func (s *Store) ListByLot(ctx context.Context, lotID string) ([]v1.Bid, error) {
	if lotID == "" {
		return nil, errors.New("lot id is required")
	}
	var models []AuctionBidModel
	if err := s.db.WithContext(ctx).
		Where("lot_id = ?", lotID).
		Order("created_at_unix_ms ASC").
		Order("id ASC").
		Find(&models).Error; err != nil {
		return nil, err
	}

	bids := make([]v1.Bid, 0, len(models))
	for i := range models {
		bid := v1.Bid{}
		if err := protojson.Unmarshal([]byte(models[i].Payload), &bid); err != nil {
			return nil, err
		}
		bids = append(bids, bid)
	}
	return bids, nil
}

func (s *Store) ListBidRecordsByBuyer(ctx context.Context, buyerUserID string, query auction.BidRecordQuery) (auction.BidRecordList, error) {
	if buyerUserID == "" {
		return auction.BidRecordList{}, errors.New("buyer user id is required")
	}
	query.Page, query.PageSize = auction.NormalizePagination(query.Page, query.PageSize)
	db := s.db.WithContext(ctx).Model(&AuctionBidModel{}).Where("user_id = ?", buyerUserID)
	if query.LotID != "" {
		db = db.Where("lot_id = ?", query.LotID)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return auction.BidRecordList{}, err
	}
	var models []AuctionBidModel
	if err := db.
		Order("created_at_unix_ms DESC").
		Order("id ASC").
		Offset(auction.PageOffset(query.Page, query.PageSize)).
		Limit(query.PageSize).
		Find(&models).Error; err != nil {
		return auction.BidRecordList{}, err
	}
	lotIDs := make([]string, 0, len(models))
	seenLot := make(map[string]bool, len(models))
	for _, model := range models {
		if model.LotID != "" && !seenLot[model.LotID] {
			lotIDs = append(lotIDs, model.LotID)
			seenLot[model.LotID] = true
		}
	}
	lotsByID := make(map[string]*v1.Lot, len(lotIDs))
	if len(lotIDs) > 0 {
		var lotModels []AuctionLotModel
		if err := s.db.WithContext(ctx).Where("id IN ?", lotIDs).Find(&lotModels).Error; err != nil {
			return auction.BidRecordList{}, err
		}
		for i := range lotModels {
			lot, err := modelToLot(&lotModels[i])
			if err != nil {
				return auction.BidRecordList{}, err
			}
			lotsByID[lot.Id] = lot
		}
	}
	records := make([]auction.BidRecord, 0, len(models))
	for _, model := range models {
		lot := lotsByID[model.LotID]
		record := auction.BidRecord{
			ID:              model.ID,
			LotID:           model.LotID,
			UserID:          model.UserID,
			Nickname:        model.Nickname,
			Amount:          model.Amount,
			Currency:        model.Currency,
			CreatedAtUnixMs: model.CreatedAtUnixMs,
		}
		if lot != nil {
			record.RoomID = lot.RoomId
			record.LotTitle = lot.Title
			record.LotImageURL = lot.ImageUrl
			record.LotStatus = lot.Status.String()
			record.AuctionState = auction.AuctionStateOf(lot)
			record.Won = lot.WinnerUserId != "" && lot.WinnerUserId == buyerUserID
		}
		records = append(records, record)
	}
	return auction.BidRecordList{Bids: records, Total: total, Page: query.Page, PageSize: query.PageSize}, nil
}

func (s *Store) FindByIdempotencyKey(ctx context.Context, lotID, userID, key string) (v1.Bid, bool, error) {
	if lotID == "" {
		return v1.Bid{}, false, errors.New("lot id is required")
	}
	if userID == "" {
		return v1.Bid{}, false, errors.New("user id is required")
	}
	if key == "" {
		return v1.Bid{}, false, errors.New("idempotency key is required")
	}
	payload, err := s.redis.Get(ctx, idempotencyKey(lotID, userID, key)).Bytes()
	if err != nil {
		return s.findByIdempotencyKeyInDB(ctx, lotID, userID, key)
	}
	bid := v1.Bid{}
	if err := protojson.Unmarshal(payload, &bid); err != nil {
		return s.findByIdempotencyKeyInDB(ctx, lotID, userID, key)
	}
	return bid, true, nil
}

func (s *Store) findByIdempotencyKeyInDB(ctx context.Context, lotID, userID, key string) (v1.Bid, bool, error) {
	var model AuctionBidModel
	if err := s.db.WithContext(ctx).
		Where("lot_id = ? AND user_id = ? AND idempotency_key = ?", lotID, userID, key).
		First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return v1.Bid{}, false, nil
		}
		return v1.Bid{}, false, err
	}
	bid := v1.Bid{}
	if err := protojson.Unmarshal([]byte(model.Payload), &bid); err != nil {
		return v1.Bid{}, false, err
	}
	return bid, true, nil
}

func (s *Store) CacheIdempotencyKey(ctx context.Context, lotID, userID, key string, bid v1.Bid) {
	if lotID == "" || userID == "" || key == "" {
		return
	}
	payload, err := protojson.Marshal(&bid)
	if err != nil {
		return
	}
	_ = s.redis.SetNX(ctx, idempotencyKey(lotID, userID, key), payload, bidIdempotencyCacheTTL).Err()
}

func idempotencyKey(lotID, userID, key string) string {
	return "auction:idem:" + lotID + ":" + userID + ":" + key
}
