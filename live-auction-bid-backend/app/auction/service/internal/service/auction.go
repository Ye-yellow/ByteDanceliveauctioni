package service

import (
	"context"

	"live-auction-bid/backend/app/auction/service/internal/biz/auction"
)

// AuctionService 是 Kratos service 层适配器。
//
// 正式接入 Kratos 生成代码后，这个结构体应该变成：
//
//	type AuctionService struct {
//	    v1.UnimplementedAuctionServiceServer
//	    auction *auction.AuctionUsecase
//	}
//
// 并实现 api/auction/service/v1/auction.proto 生成出来的 RPC 方法。
// 当前由于项目还没有提交 protoc/kratos 生成产物，先用领域命令作为入参，
// 但职责已经固定：service 只做协议适配，不写业务规则。
type AuctionService struct {
	auction *auction.AuctionUsecase
}

func NewAuctionService(auction *auction.AuctionUsecase) *AuctionService {
	return &AuctionService{auction: auction}
}

func (s *AuctionService) CreateLot(ctx context.Context, cmd auction.CreateLotCommand) (*auction.Lot, error) {
	return s.auction.CreateLot(ctx, cmd)
}

func (s *AuctionService) GetLot(ctx context.Context, lotID string) (*auction.Lot, error) {
	return s.auction.GetLot(ctx, lotID)
}

func (s *AuctionService) ListLots(ctx context.Context, roomID string, status auction.LotStatus) ([]*auction.Lot, error) {
	return s.auction.ListLots(ctx, roomID, status)
}

func (s *AuctionService) StartLot(ctx context.Context, lotID string) (*auction.Lot, error) {
	return s.auction.StartLot(ctx, lotID)
}

func (s *AuctionService) PlaceBid(ctx context.Context, cmd auction.PlaceBidCommand) (*auction.Lot, *auction.Bid, []auction.RankingItem, error) {
	return s.auction.PlaceBid(ctx, cmd)
}

func (s *AuctionService) RevealTrustCard(ctx context.Context, lotID, cardID, operatorID string) (*auction.Lot, *auction.TrustRevealCard, error) {
	return s.auction.RevealTrustCard(ctx, lotID, cardID, operatorID)
}

func (s *AuctionService) StartDuel(ctx context.Context, lotID, operatorID, userAID, userBID string) (*auction.Lot, *auction.DuelState, error) {
	return s.auction.StartDuel(ctx, lotID, operatorID, userAID, userBID)
}

func (s *AuctionService) SettleLot(ctx context.Context, lotID, operatorID string) (*auction.Lot, error) {
	return s.auction.SettleLot(ctx, lotID, operatorID)
}

func (s *AuctionService) Snapshot(ctx context.Context, roomID string) (*auction.RoomSnapshot, error) {
	return s.auction.Snapshot(ctx, roomID)
}
