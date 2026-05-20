package data

import (
	"context"
	"errors"

	"google.golang.org/protobuf/encoding/protojson"
	"gorm.io/gorm"
	v1 "live-auction-bid/backend/api/auction/service/v1"
	"live-auction-bid/backend/app/auction/service/internal/pkg/apperr"
)

func (s *Store) Create(ctx context.Context, lot *v1.Lot, events []v1.AuctionEvent) error {
	if lot == nil {
		return errors.New("lot is required")
	}
	model, err := lotToModel(lot)
	if err != nil {
		return err
	}
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(model).Error; err != nil {
			return err
		}
		return createEventModels(ctx, tx, events)
	}); err != nil {
		return err
	}
	return s.streamEvents(ctx, events)
}

func (s *Store) Save(ctx context.Context, lot *v1.Lot, expectedVersion int64, events []v1.AuctionEvent) error {
	if lot == nil {
		return errors.New("lot is required")
	}
	if expectedVersion <= 0 {
		return errors.New("lot expected version is required")
	}
	model, err := lotToModel(lot)
	if err != nil {
		return err
	}
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.
			Model(&AuctionLotModel{}).
			Where("id = ? AND version = ?", lot.Id, expectedVersion).
			Select("*").
			Omit("created_at").
			Updates(model)
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

func (s *Store) FindByID(ctx context.Context, lotID string) (*v1.Lot, error) {
	if lotID == "" {
		return nil, errors.New("lot id is required")
	}
	var model AuctionLotModel
	if err := s.db.WithContext(ctx).Where("id = ?", lotID).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("lot not found")
		}
		return nil, err
	}
	return modelToLot(&model)
}

func (s *Store) List(ctx context.Context, roomID string, status v1.LotStatus) ([]*v1.Lot, error) {
	if roomID == "" {
		return nil, errors.New("room id is required")
	}
	query := s.db.WithContext(ctx).Where("room_id = ?", roomID)
	if status != 0 {
		query = query.Where("status = ?", int32(status))
	}
	var models []AuctionLotModel
	if err := query.Order("updated_at DESC").Order("id ASC").Find(&models).Error; err != nil {
		return nil, err
	}
	lots := make([]*v1.Lot, 0, len(models))
	for i := range models {
		lot, err := modelToLot(&models[i])
		if err != nil {
			return nil, err
		}
		lots = append(lots, lot)
	}
	return lots, nil
}

func lotToModel(lot *v1.Lot) (*AuctionLotModel, error) {
	payload, err := protojson.Marshal(lot)
	if err != nil {
		return nil, err
	}
	return &AuctionLotModel{
		ID:                     lot.Id,
		RoomID:                 lot.RoomId,
		Title:                  lot.Title,
		Description:            lot.Description,
		ImageURL:               lot.ImageUrl,
		Status:                 int32(lot.Status),
		StartPriceAmount:       lot.GetRule().GetStartPrice().GetAmount(),
		StartPriceCurrency:     lot.GetRule().GetStartPrice().GetCurrency(),
		MinIncrementAmount:     lot.GetRule().GetMinIncrement().GetAmount(),
		MinIncrementCurrency:   lot.GetRule().GetMinIncrement().GetCurrency(),
		DurationSeconds:        lot.GetRule().GetDurationSeconds(),
		AntiSnipeWindowSeconds: lot.GetRule().GetAntiSnipeWindowSeconds(),
		AntiSnipeExtendSeconds: lot.GetRule().GetAntiSnipeExtendSeconds(),
		MaxExtendCount:         lot.GetRule().GetMaxExtendCount(),
		CurrentPriceAmount:     lot.GetCurrentPrice().GetAmount(),
		CurrentPriceCurrency:   lot.GetCurrentPrice().GetCurrency(),
		LeadingUserID:          lot.LeadingUserId,
		LeadingNickname:        lot.LeadingNickname,
		StartedAtUnixMs:        lot.StartedAtUnixMs,
		EndsAtUnixMs:           lot.EndsAtUnixMs,
		SettledAtUnixMs:        lot.SettledAtUnixMs,
		CancelReason:           lot.CancelReason,
		CancelledAtUnixMs:      lot.CancelledAtUnixMs,
		WinnerUserID:           lot.WinnerUserId,
		WinnerNickname:         lot.WinnerNickname,
		FinalPriceAmount:       lot.GetFinalPrice().GetAmount(),
		FinalPriceCurrency:     lot.GetFinalPrice().GetCurrency(),
		Version:                lot.Version,
		PlaybookStage:          int32(lot.PlaybookStage),
		Payload:                string(payload),
	}, nil
}

func modelToLot(model *AuctionLotModel) (*v1.Lot, error) {
	lot := &v1.Lot{}
	if err := protojson.Unmarshal([]byte(model.Payload), lot); err != nil {
		return nil, err
	}
	return lot, nil
}
