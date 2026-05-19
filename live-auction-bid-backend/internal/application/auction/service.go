package auctionapp

import (
	"context"
	"time"

	domain "live-auction-bid/backend/internal/domain/auction"
)

type Service struct {
	repo   domain.LotRepository
	domain *domain.DomainService
}

func NewService(repo domain.LotRepository, ds *domain.DomainService) *Service {
	return &Service{repo: repo, domain: ds}
}

type CreateLotCommand struct {
	RoomID       string       `json:"roomId"`
	Title        string       `json:"title"`
	Description  string       `json:"description"`
	ImageURL      string       `json:"imageUrl"`
	StartPrice   domain.Money `json:"startPrice"`
	MinIncrement domain.Money `json:"minIncrement"`
	DurationSec  int          `json:"durationSec"`
}

func (s *Service) CreateLot(ctx context.Context, cmd CreateLotCommand) (*domain.Lot, error) {
	if cmd.DurationSec == 0 {
		cmd.DurationSec = 15 * 60
	}
	lot := &domain.Lot{
		ID:           "lot_" + time.Now().Format("20060102150405"),
		RoomID:       cmd.RoomID,
		Title:        cmd.Title,
		Description:  cmd.Description,
		ImageURL:      cmd.ImageURL,
		StartPrice:   cmd.StartPrice,
		CurrentPrice: cmd.StartPrice,
		MinIncrement: cmd.MinIncrement,
		Status:       domain.LotLive,
		EndsAt:       time.Now().Add(time.Duration(cmd.DurationSec) * time.Second),
	}
	return lot, s.repo.Save(ctx, lot)
}

func (s *Service) ListLots(ctx context.Context) ([]*domain.Lot, error) { return s.repo.List(ctx) }
func (s *Service) LiveLot(ctx context.Context, roomID string) (*domain.Lot, error) { return s.repo.FindLiveByRoom(ctx, roomID) }
func (s *Service) PlaceBid(ctx context.Context, lotID, userID, nickname string, amount domain.Money) (*domain.Lot, error) {
	return s.domain.PlaceBid(ctx, lotID, userID, nickname, amount)
}
func (s *Service) Settle(ctx context.Context, lotID string) (*domain.Lot, error) { return s.domain.Settle(ctx, lotID) }
