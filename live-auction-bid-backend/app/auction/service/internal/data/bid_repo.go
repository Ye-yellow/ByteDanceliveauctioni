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
	bidModel, err := bidToModel(bid, idempotencyKey)
	if err != nil {
		return err
	}
	lotModel, err := lotToModel(lot)
	if err != nil {
		return err
	}
	var orderModel *AuctionOrderModel
	if order != nil {
		orderModel, err = orderToModel(*order)
		if err != nil {
			return err
		}
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
		if err := tx.Create(bidModel).Error; err != nil {
			return err
		}
		if err := projectStatsForInsertedBid(ctx, tx, bid, lot); err != nil {
			return err
		}
		if orderModel != nil {
			if err := tx.Create(orderModel).Error; err != nil {
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
	bidModel, err := bidToModel(bid, idempotencyKey)
	if err != nil {
		return err
	}
	lotModel, err := lotToModel(lot)
	if err != nil {
		return err
	}
	var orderModel *AuctionOrderModel
	if order != nil {
		orderModel, err = orderToModel(*order)
		if err != nil {
			return err
		}
	}
	projected := false
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		lotUpdate := tx.Model(&AuctionLotModel{}).
			Where("id = ? AND version < ?", lot.Id, lot.Version).
			Select("*").
			Omit("created_at").
			Updates(lotModel)
		if lotUpdate.Error != nil {
			return lotUpdate.Error
		}
		if lotUpdate.RowsAffected == 0 {
			var existing int64
			if err := tx.Model(&AuctionLotModel{}).Where("id = ?", lot.Id).Count(&existing).Error; err != nil {
				return err
			}
			if existing == 0 {
				return apperr.ErrNotFound
			}
		}

		bidCreate := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(bidModel)
		if bidCreate.Error != nil {
			return bidCreate.Error
		}
		if bidCreate.RowsAffected > 0 {
			projected = true
			if err := projectStatsForInsertedBid(ctx, tx, bid, lot); err != nil {
				return err
			}
		} else {
			return nil
		}
		if orderModel != nil {
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(orderModel).Error; err != nil {
				return err
			}
		}
		return createEventModelsIgnoringDuplicates(ctx, tx, events)
	}); err != nil {
		return err
	}
	if !projected {
		return nil
	}
	return s.streamEvents(ctx, events)
}

func projectStatsForInsertedBid(ctx context.Context, tx *gorm.DB, bid v1.Bid, lot *v1.Lot) error {
	if lot == nil {
		return errors.New("lot is required")
	}
	participant := AuctionLotParticipantModel{
		LotID:            bid.LotId,
		UserID:           bid.UserId,
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

func bidToModel(bid v1.Bid, idempotencyKey string) (*AuctionBidModel, error) {
	payload, err := protojson.Marshal(&bid)
	if err != nil {
		return nil, err
	}
	return &AuctionBidModel{
		ID:              bid.Id,
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
