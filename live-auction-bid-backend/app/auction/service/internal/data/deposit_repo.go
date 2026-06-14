package data

import (
	"context"
	"encoding/json"
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"live-auction-bid/backend/app/auction/service/internal/biz/auction"
	"live-auction-bid/backend/app/auction/service/internal/pkg/apperr"
)

func (s *Store) FindDepositHoldByLotBuyer(ctx context.Context, lotID, buyerUserID string) (*auction.DepositHold, bool, error) {
	var model AuctionDepositHoldModel
	if err := s.db.WithContext(ctx).
		Where("lot_id = ? AND buyer_user_id = ?", lotID, buyerUserID).
		First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, false, nil
		}
		return nil, false, err
	}
	hold, err := modelToDepositHold(&model)
	return hold, err == nil, err
}

func (s *Store) FindDepositHoldByIdempotencyKey(ctx context.Context, lotID, buyerUserID, key string) (*auction.DepositHold, bool, error) {
	var model AuctionDepositHoldModel
	if err := s.db.WithContext(ctx).
		Where("lot_id = ? AND buyer_user_id = ? AND idempotency_key = ?", lotID, buyerUserID, key).
		First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, false, nil
		}
		return nil, false, err
	}
	hold, err := modelToDepositHold(&model)
	return hold, err == nil, err
}

func (s *Store) CommitDepositHold(ctx context.Context, hold auction.DepositHold) (*auction.DepositHold, error) {
	model, err := depositHoldToModel(hold)
	if err != nil {
		return nil, err
	}
	var committed *auction.DepositHold
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing AuctionDepositHoldModel
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("lot_id = ? AND buyer_user_id = ?", hold.LotID, hold.BuyerUserID).
			First(&existing).Error
		if err == nil {
			next, convertErr := modelToDepositHold(&existing)
			if convertErr != nil {
				return convertErr
			}
			if next.IdempotencyKey == hold.IdempotencyKey || next.Status == auction.DepositStatusHeld {
				committed = next
				return nil
			}
			return fmtInvalidDepositState(next.Status)
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if err := tx.Create(model).Error; err != nil {
			return err
		}
		next, convertErr := modelToDepositHold(model)
		if convertErr != nil {
			return convertErr
		}
		committed = next
		return nil
	}); err != nil {
		return nil, err
	}
	return committed, nil
}

func fmtInvalidDepositState(status auction.DepositStatus) error {
	return errors.Join(apperr.ErrInvalidArgument, errors.New("deposit already exists with status "+string(status)))
}

func depositHoldToModel(hold auction.DepositHold) (*AuctionDepositHoldModel, error) {
	addressPayload, err := json.Marshal(hold.AddressSnapshot)
	if err != nil {
		return nil, err
	}
	payload, err := json.Marshal(hold)
	if err != nil {
		return nil, err
	}
	return &AuctionDepositHoldModel{
		ID:               hold.ID,
		MainAccountID:    hold.MainAccountID,
		RoomID:           hold.RoomID,
		LotID:            hold.LotID,
		BuyerUserID:      hold.BuyerUserID,
		BuyerNickname:    hold.BuyerNickname,
		Status:           string(hold.Status),
		Amount:           hold.Amount,
		Currency:         hold.Currency,
		PaymentProvider:  hold.PaymentProvider,
		PaymentID:        hold.PaymentID,
		IdempotencyKey:   hold.IdempotencyKey,
		AddressID:        hold.AddressID,
		AddressSnapshot:  string(addressPayload),
		CreatedAtUnixMs:  hold.CreatedAtUnixMs,
		UpdatedAtUnixMs:  hold.UpdatedAtUnixMs,
		HeldAtUnixMs:     hold.HeldAtUnixMs,
		ReleasedAtUnixMs: hold.ReleasedAtUnixMs,
		Payload:          string(payload),
	}, nil
}

func modelToDepositHold(model *AuctionDepositHoldModel) (*auction.DepositHold, error) {
	var hold auction.DepositHold
	if err := json.Unmarshal([]byte(model.Payload), &hold); err != nil {
		return nil, err
	}
	hold.ID = model.ID
	hold.MainAccountID = model.MainAccountID
	hold.RoomID = model.RoomID
	hold.LotID = model.LotID
	hold.BuyerUserID = model.BuyerUserID
	hold.BuyerNickname = model.BuyerNickname
	hold.Status = auction.DepositStatus(model.Status)
	hold.Amount = model.Amount
	hold.Currency = model.Currency
	hold.PaymentProvider = model.PaymentProvider
	hold.PaymentID = model.PaymentID
	hold.IdempotencyKey = model.IdempotencyKey
	hold.AddressID = model.AddressID
	hold.CreatedAtUnixMs = model.CreatedAtUnixMs
	hold.UpdatedAtUnixMs = model.UpdatedAtUnixMs
	hold.HeldAtUnixMs = model.HeldAtUnixMs
	hold.ReleasedAtUnixMs = model.ReleasedAtUnixMs
	if model.AddressSnapshot != "" {
		_ = json.Unmarshal([]byte(model.AddressSnapshot), &hold.AddressSnapshot)
	}
	return &hold, nil
}
