package data

import (
	"context"
	"errors"
	"strings"

	"google.golang.org/protobuf/encoding/protojson"
	"gorm.io/gorm"
	v1 "live-auction-bid/backend/api/auction/service/v1"
	"live-auction-bid/backend/app/auction/service/internal/biz/auction"
	"live-auction-bid/backend/app/auction/service/internal/pkg/apperr"
)

func (s *Store) Create(ctx context.Context, lot *v1.Lot, ownerUserID string, events []v1.AuctionEvent) error {
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
		if err := attachAssetFilesByURL(tx, ownerUserID, lot.Id, lotAssetURLs(lot)); err != nil {
			return err
		}
		return createEventModels(ctx, tx, events)
	}); err != nil {
		return err
	}
	return s.streamEvents(ctx, events)
}

func lotAssetURLs(lot *v1.Lot) []string {
	if lot == nil {
		return nil
	}
	urls := []string{lot.ImageUrl}
	for _, card := range lot.TrustCards {
		if card != nil && card.ImageUrl != "" {
			urls = append(urls, card.ImageUrl)
		}
	}
	return urls
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

func (s *Store) AttachAssets(ctx context.Context, ownerUserID string, lot *v1.Lot) error {
	if lot == nil || ownerUserID == "" {
		return nil
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return attachAssetFilesByURL(tx, ownerUserID, lot.Id, lotAssetURLs(lot))
	})
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
		if status == v1.LotStatus_LOT_STATUS_LIVE {
			query = query.Where("status IN ?", []int32{int32(v1.LotStatus_LOT_STATUS_LIVE), int32(v1.LotStatus_LOT_STATUS_EXTENDED)})
		} else {
			query = query.Where("status = ?", int32(status))
		}
	}
	var models []AuctionLotModel
	if status == v1.LotStatus_LOT_STATUS_QUEUED {
		query = query.Order("queue_position ASC")
	} else {
		query = query.Order("updated_at DESC")
	}
	if err := query.Order("id ASC").Find(&models).Error; err != nil {
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

func (s *Store) ListLots(ctx context.Context, query auction.LotQuery) (auction.LotList, error) {
	query.Page, query.PageSize = auction.NormalizePagination(query.Page, query.PageSize)
	db := s.db.WithContext(ctx).Model(&AuctionLotModel{})
	if query.RoomID != "" {
		db = db.Where("room_id = ?", query.RoomID)
	}
	if query.Status != v1.LotStatus_LOT_STATUS_UNSPECIFIED {
		db = db.Where("status = ?", int32(query.Status))
	}
	if keyword := strings.TrimSpace(query.Keyword); keyword != "" {
		like := "%" + keyword + "%"
		db = db.Where("id LIKE ? OR title LIKE ? OR description LIKE ?", like, like, like)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return auction.LotList{}, err
	}
	var models []AuctionLotModel
	if err := db.
		Order("updated_at DESC").
		Order("id ASC").
		Offset(auction.PageOffset(query.Page, query.PageSize)).
		Limit(query.PageSize).
		Find(&models).Error; err != nil {
		return auction.LotList{}, err
	}
	lots := make([]*v1.Lot, 0, len(models))
	for i := range models {
		lot, err := modelToLot(&models[i])
		if err != nil {
			return auction.LotList{}, err
		}
		lots = append(lots, lot)
	}
	return auction.LotList{Lots: lots, Total: total, Page: query.Page, PageSize: query.PageSize}, nil
}

func (s *Store) ListExpiredOpen(ctx context.Context, nowMs int64, limit int) ([]*v1.Lot, error) {
	if nowMs <= 0 {
		return nil, errors.New("now ms is required")
	}
	if limit <= 0 {
		limit = 100
	}
	var models []AuctionLotModel
	if err := s.db.WithContext(ctx).
		Where("status IN ? AND ends_at_unix_ms > 0 AND ends_at_unix_ms <= ?", []int32{int32(v1.LotStatus_LOT_STATUS_LIVE), int32(v1.LotStatus_LOT_STATUS_EXTENDED)}, nowMs).
		Order("ends_at_unix_ms ASC").
		Order("id ASC").
		Limit(limit).
		Find(&models).Error; err != nil {
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
	var capAmount *int64
	var capCurrency string
	rule := lot.GetRule()
	if rule == nil {
		rule = &v1.BidRule{}
		lot.Rule = rule
	}
	startPrice := rule.GetStartPrice()
	if startPrice == nil {
		startPrice = &v1.Money{}
	}
	minIncrement := rule.GetMinIncrement()
	if minIncrement == nil {
		minIncrement = &v1.Money{}
	}
	currentPrice := lot.GetCurrentPrice()
	if currentPrice == nil {
		currentPrice = &v1.Money{}
	}
	finalPrice := lot.GetFinalPrice()
	if finalPrice == nil {
		finalPrice = &v1.Money{}
	}
	if rule.GetCapPrice() != nil {
		amount := rule.GetCapPrice().GetAmount()
		capAmount = &amount
		capCurrency = rule.GetCapPrice().GetCurrency()
	}
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
		QueueStatus:            int32(normalizeQueueStatus(lot.GetQueueStatus())),
		QueuePosition:          lot.GetQueuePosition(),
		StartPriceAmount:       startPrice.GetAmount(),
		StartPriceCurrency:     startPrice.GetCurrency(),
		MinIncrementAmount:     minIncrement.GetAmount(),
		MinIncrementCurrency:   minIncrement.GetCurrency(),
		CapPriceAmount:         capAmount,
		CapPriceCurrency:       capCurrency,
		DurationSeconds:        rule.GetDurationSeconds(),
		AntiSnipeWindowSeconds: rule.GetAntiSnipeWindowSeconds(),
		AntiSnipeExtendSeconds: rule.GetAntiSnipeExtendSeconds(),
		MaxExtendCount:         rule.GetMaxExtendCount(),
		CurrentPriceAmount:     currentPrice.GetAmount(),
		CurrentPriceCurrency:   currentPrice.GetCurrency(),
		LeadingUserID:          lot.LeadingUserId,
		LeadingNickname:        lot.LeadingNickname,
		StartedAtUnixMs:        lot.StartedAtUnixMs,
		EndsAtUnixMs:           lot.EndsAtUnixMs,
		SettledAtUnixMs:        lot.SettledAtUnixMs,
		CancelReason:           lot.CancelReason,
		CancelledAtUnixMs:      lot.CancelledAtUnixMs,
		WinnerUserID:           lot.WinnerUserId,
		WinnerNickname:         lot.WinnerNickname,
		FinalPriceAmount:       finalPrice.GetAmount(),
		FinalPriceCurrency:     finalPrice.GetCurrency(),
		Version:                lot.Version,
		PlaybookStage:          int32(lot.PlaybookStage),
		Payload:                string(payload),
	}, nil
}

func normalizeQueueStatus(status v1.LotQueueStatus) v1.LotQueueStatus {
	if status == v1.LotQueueStatus_LOT_QUEUE_STATUS_UNSPECIFIED {
		return v1.LotQueueStatus_LOT_QUEUE_STATUS_NONE
	}
	return status
}

func modelToLot(model *AuctionLotModel) (*v1.Lot, error) {
	lot := &v1.Lot{}
	if err := protojson.Unmarshal([]byte(model.Payload), lot); err != nil {
		return nil, err
	}
	if lot.Rule == nil {
		lot.Rule = &v1.BidRule{}
	}
	lot.QueueStatus = normalizeQueueStatus(v1.LotQueueStatus(model.QueueStatus))
	lot.QueuePosition = model.QueuePosition
	if model.CapPriceAmount != nil {
		lot.Rule.CapPrice = &v1.Money{Amount: *model.CapPriceAmount, Currency: model.CapPriceCurrency}
	} else {
		lot.Rule.CapPrice = nil
	}
	return lot, nil
}
