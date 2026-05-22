package service

import (
	"context"

	v1 "live-auction-bid/backend/api/auction/service/v1"
	"live-auction-bid/backend/app/auction/service/internal/biz/auction"
	"live-auction-bid/backend/app/auction/service/internal/pkg/auth"
)

// AuctionService 是 Kratos service 层适配器。
//
// 它实现 api/auction/service/v1/auction.proto 生成的 HTTP/gRPC service 接口。
// 分层规则：这里只做 proto 入参组装、调用 usecase、包装 proto reply，不写竞拍业务规则。
type AuctionService struct {
	v1.UnimplementedAuctionServiceServer
	auction *auction.AuctionUsecase
}

func NewAuctionService(auction *auction.AuctionUsecase) *AuctionService {
	return &AuctionService{auction: auction}
}

func (s *AuctionService) CreateLot(ctx context.Context, req *v1.CreateLotRequest) (*v1.CreateLotReply, error) {
	claims, err := auth.RequireRole(ctx, v1.UserRole_USER_ROLE_ANCHOR, v1.UserRole_USER_ROLE_OPERATOR, v1.UserRole_USER_ROLE_ADMIN)
	if err != nil {
		return &v1.CreateLotReply{Result: ErrorResult(err)}, nil
	}
	lot, err := s.auction.CreateLot(ctx, req, claims.UserID)
	if err != nil {
		return &v1.CreateLotReply{Result: ErrorResult(err)}, nil
	}
	return &v1.CreateLotReply{Result: okResult(), Lot: lot}, nil
}

func (s *AuctionService) GetLot(ctx context.Context, req *v1.GetLotRequest) (*v1.GetLotReply, error) {
	lot, err := s.auction.GetLot(ctx, req.GetLotId())
	if err != nil {
		return &v1.GetLotReply{Result: ErrorResult(err)}, nil
	}
	return &v1.GetLotReply{Result: okResult(), Lot: lot}, nil
}

func (s *AuctionService) ListLots(ctx context.Context, req *v1.ListLotsRequest) (*v1.ListLotsReply, error) {
	lots, err := s.auction.ListLots(ctx, req.GetRoomId(), req.GetStatus())
	if err != nil {
		return &v1.ListLotsReply{Result: ErrorResult(err)}, nil
	}
	return &v1.ListLotsReply{Result: okResult(), Lots: lots}, nil
}

func (s *AuctionService) StartLot(ctx context.Context, req *v1.StartLotRequest) (*v1.StartLotReply, error) {
	if _, err := auth.RequireRole(ctx, v1.UserRole_USER_ROLE_ANCHOR, v1.UserRole_USER_ROLE_OPERATOR, v1.UserRole_USER_ROLE_ADMIN); err != nil {
		return &v1.StartLotReply{Result: ErrorResult(err)}, nil
	}
	lot, err := s.auction.StartLot(ctx, req.GetLotId())
	if err != nil {
		return &v1.StartLotReply{Result: ErrorResult(err)}, nil
	}
	return &v1.StartLotReply{Result: okResult(), Lot: lot}, nil
}

func (s *AuctionService) PlaceBid(ctx context.Context, req *v1.PlaceBidRequest) (*v1.PlaceBidReply, error) {
	claims, authErr := auth.RequireRole(ctx, v1.UserRole_USER_ROLE_BUYER)
	if authErr != nil {
		return &v1.PlaceBidReply{Result: ErrorResult(authErr), Accepted: false, RejectReason: authErr.Error()}, nil
	}
	lot, bid, ranking, err := s.auction.PlaceBid(ctx, req, claims.UserID, claims.Nickname)
	if err != nil {
		return &v1.PlaceBidReply{Result: ErrorResult(err), Accepted: false, Lot: lot, Ranking: ranking, RejectReason: err.Error()}, nil
	}
	return &v1.PlaceBidReply{
		Result:   okResult(),
		Accepted: true,
		Lot:      lot,
		Bid:      bid,
		Ranking:  ranking,
	}, nil
}

func (s *AuctionService) RevealTrustCard(ctx context.Context, req *v1.RevealTrustCardRequest) (*v1.RevealTrustCardReply, error) {
	if _, err := auth.RequireRole(ctx, v1.UserRole_USER_ROLE_ANCHOR, v1.UserRole_USER_ROLE_OPERATOR, v1.UserRole_USER_ROLE_ADMIN); err != nil {
		return &v1.RevealTrustCardReply{Result: ErrorResult(err)}, nil
	}
	lot, card, err := s.auction.RevealTrustCard(ctx, req.GetLotId(), req.GetCardId(), req.GetOperatorId())
	if err != nil {
		return &v1.RevealTrustCardReply{Result: ErrorResult(err), Lot: lot, TrustCard: card}, nil
	}
	return &v1.RevealTrustCardReply{Result: okResult(), Lot: lot, TrustCard: card}, nil
}

func (s *AuctionService) StartDuel(ctx context.Context, req *v1.StartDuelRequest) (*v1.StartDuelReply, error) {
	if _, err := auth.RequireRole(ctx, v1.UserRole_USER_ROLE_ANCHOR, v1.UserRole_USER_ROLE_OPERATOR, v1.UserRole_USER_ROLE_ADMIN); err != nil {
		return &v1.StartDuelReply{Result: ErrorResult(err)}, nil
	}
	lot, duel, err := s.auction.StartDuel(ctx, req.GetLotId(), req.GetOperatorId(), req.GetUserAId(), req.GetUserBId())
	if err != nil {
		return &v1.StartDuelReply{Result: ErrorResult(err), Lot: lot, DuelState: duel}, nil
	}
	return &v1.StartDuelReply{Result: okResult(), Lot: lot, DuelState: duel}, nil
}

func (s *AuctionService) SettleLot(ctx context.Context, req *v1.SettleLotRequest) (*v1.SettleLotReply, error) {
	if _, err := auth.RequireRole(ctx, v1.UserRole_USER_ROLE_ANCHOR, v1.UserRole_USER_ROLE_OPERATOR, v1.UserRole_USER_ROLE_ADMIN); err != nil {
		return &v1.SettleLotReply{Result: ErrorResult(err)}, nil
	}
	lot, err := s.auction.SettleLot(ctx, req.GetLotId(), req.GetOperatorId())
	if err != nil {
		return &v1.SettleLotReply{Result: ErrorResult(err), Lot: lot}, nil
	}
	return &v1.SettleLotReply{Result: okResult(), Lot: lot}, nil
}

func (s *AuctionService) CancelLot(ctx context.Context, req *v1.CancelLotRequest) (*v1.CancelLotReply, error) {
	if _, err := auth.RequireRole(ctx, v1.UserRole_USER_ROLE_ANCHOR, v1.UserRole_USER_ROLE_OPERATOR, v1.UserRole_USER_ROLE_ADMIN); err != nil {
		return &v1.CancelLotReply{Result: ErrorResult(err)}, nil
	}
	lot, err := s.auction.CancelLot(ctx, req.GetLotId(), req.GetOperatorId(), req.GetReason())
	if err != nil {
		return &v1.CancelLotReply{Result: ErrorResult(err), Lot: lot}, nil
	}
	return &v1.CancelLotReply{Result: okResult(), Lot: lot}, nil
}

func (s *AuctionService) GetRoomSnapshot(ctx context.Context, req *v1.GetRoomSnapshotRequest) (*v1.GetRoomSnapshotReply, error) {
	snapshot, err := s.auction.Snapshot(ctx, req.GetRoomId())
	if err != nil {
		return &v1.GetRoomSnapshotReply{Result: ErrorResult(err)}, nil
	}
	return &v1.GetRoomSnapshotReply{Result: okResult(), Snapshot: snapshot}, nil
}
