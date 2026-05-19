package service

import (
	"context"

	"live-auction-bid/backend/app/auction/service/internal/biz"
)

// Service 是接入层服务，负责把 HTTP/未来 proto 请求转给 biz 用例。
// 当前没有生成 Kratos proto service，因此先保留手写适配层。
type Service struct {
	auction *biz.AuctionUsecase
}

func New(auction *biz.AuctionUsecase) *Service {
	return &Service{auction: auction}
}

func (s *Service) CreateLot(ctx context.Context, cmd biz.CreateLotCommand) (*biz.Lot, error) {
	return s.auction.CreateLot(ctx, cmd)
}

func (s *Service) GetLot(ctx context.Context, lotID string) (*biz.Lot, error) {
	return s.auction.GetLot(ctx, lotID)
}

func (s *Service) ListLots(ctx context.Context, roomID string, status biz.LotStatus) ([]*biz.Lot, error) {
	return s.auction.ListLots(ctx, roomID, status)
}

func (s *Service) StartLot(ctx context.Context, lotID string) (*biz.Lot, error) {
	return s.auction.StartLot(ctx, lotID)
}

func (s *Service) PlaceBid(ctx context.Context, cmd biz.PlaceBidCommand) (*biz.Lot, *biz.Bid, []biz.RankingItem, error) {
	return s.auction.PlaceBid(ctx, cmd)
}

func (s *Service) RevealTrustCard(ctx context.Context, lotID, cardID, operatorID string) (*biz.Lot, *biz.TrustRevealCard, error) {
	return s.auction.RevealTrustCard(ctx, lotID, cardID, operatorID)
}

func (s *Service) StartDuel(ctx context.Context, lotID, operatorID, userAID, userBID string) (*biz.Lot, *biz.DuelState, error) {
	return s.auction.StartDuel(ctx, lotID, operatorID, userAID, userBID)
}

func (s *Service) SettleLot(ctx context.Context, lotID, operatorID string) (*biz.Lot, error) {
	return s.auction.SettleLot(ctx, lotID, operatorID)
}

func (s *Service) Snapshot(ctx context.Context, roomID string) (*biz.RoomSnapshot, error) {
	return s.auction.Snapshot(ctx, roomID)
}
