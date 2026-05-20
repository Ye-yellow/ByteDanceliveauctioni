package data

import (
	"context"
	"errors"

	"google.golang.org/protobuf/encoding/protojson"
	"gorm.io/gorm"
	v1 "live-auction-bid/backend/api/auction/service/v1"
	"live-auction-bid/backend/app/auction/service/internal/pkg/apperr"
)

func (s *Store) CommitAcceptedBid(ctx context.Context, bid v1.Bid, lot *v1.Lot, expectedLotVersion int64, idempotencyKey string, events []v1.AuctionEvent) error {
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
		return createEventModels(ctx, tx, events)
	}); err != nil {
		return err
	}
	return s.streamEvents(ctx, events)
}

func bidToModel(bid v1.Bid, idempotencyKey string) (*AuctionBidModel, error) {
	payload, err := protojson.Marshal(&bid)
	if err != nil {
		return nil, err
	}
	var idem *string
	if idempotencyKey != "" {
		idem = &idempotencyKey
	}
	return &AuctionBidModel{
		ID:              bid.Id,
		LotID:           bid.LotId,
		UserID:          bid.UserId,
		Nickname:        bid.Nickname,
		Amount:          bid.GetAmount().GetAmount(),
		Currency:        bid.GetAmount().GetCurrency(),
		IdempotencyKey:  idem,
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

func (s *Store) FindByIdempotencyKey(ctx context.Context, lotID, key string) (v1.Bid, bool, error) {
	if lotID == "" {
		return v1.Bid{}, false, errors.New("lot id is required")
	}
	if key == "" {
		return v1.Bid{}, false, errors.New("idempotency key is required")
	}
	payload, err := s.redis.Get(ctx, idempotencyKey(lotID, key)).Bytes()
	if err != nil {
		return s.findByIdempotencyKeyInDB(ctx, lotID, key)
	}
	bid := v1.Bid{}
	if err := protojson.Unmarshal(payload, &bid); err != nil {
		return s.findByIdempotencyKeyInDB(ctx, lotID, key)
	}
	return bid, true, nil
}

func (s *Store) findByIdempotencyKeyInDB(ctx context.Context, lotID, key string) (v1.Bid, bool, error) {
	var model AuctionBidModel
	if err := s.db.WithContext(ctx).
		Where("lot_id = ? AND idempotency_key = ?", lotID, key).
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

func (s *Store) CacheIdempotencyKey(ctx context.Context, lotID, key string, bid v1.Bid) {
	if lotID == "" || key == "" {
		return
	}
	payload, err := protojson.Marshal(&bid)
	if err != nil {
		return
	}
	_ = s.redis.SetNX(ctx, idempotencyKey(lotID, key), payload, 0).Err()
}

func idempotencyKey(lotID, key string) string {
	return "auction:idem:" + lotID + ":" + key
}
