package service

import (
	"context"
	"time"

	biz "live-auction-bid/backend/app/auction/service/internal/biz"
)

type Service struct {
	repo   biz.LotRepository
	domain *biz.DomainService
}

func NewService(repo biz.LotRepository, ds *biz.DomainService) *Service {
	return &Service{repo: repo, domain: ds}
}

type CreateLotCommand struct {
	RoomID       string       `json:"roomId"`
	Title        string       `json:"title"`
	Description  string       `json:"description"`
	ImageURL      string       `json:"imageUrl"`
	StartPrice   biz.Money `json:"startPrice"`
	MinIncrement biz.Money `json:"minIncrement"`
	DurationSec  int          `json:"durationSec"`
}

func (s *Service) CreateLot(ctx context.Context, cmd CreateLotCommand) (*biz.Lot, error) {
	if cmd.DurationSec == 0 {
		cmd.DurationSec = 15 * 60
	}
	lot := &biz.Lot{
		ID:           "lot_" + time.Now().Format("20060102150405"),
		RoomID:       cmd.RoomID,
		Title:        cmd.Title,
		Description:  cmd.Description,
		ImageURL:      cmd.ImageURL,
		StartPrice:   cmd.StartPrice,
		CurrentPrice: cmd.StartPrice,
		MinIncrement: cmd.MinIncrement,
		Status:       biz.LotLive,
		EndsAt:       time.Now().Add(time.Duration(cmd.DurationSec) * time.Second),
	}
	return lot, s.repo.Save(ctx, lot)
}

func (s *Service) ListLots(ctx context.Context) ([]*biz.Lot, error) { return s.repo.List(ctx) }
func (s *Service) LiveLot(ctx context.Context, roomID string) (*biz.Lot, error) { return s.repo.FindLiveByRoom(ctx, roomID) }
func (s *Service) PlaceBid(ctx context.Context, lotID, userID, nickname string, amount biz.Money) (*biz.Lot, error) {
	return s.domain.PlaceBid(ctx, lotID, userID, nickname, amount)
}
func (s *Service) Settle(ctx context.Context, lotID string) (*biz.Lot, error) { return s.domain.Settle(ctx, lotID) }
