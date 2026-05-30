package service

import (
	"context"
	"errors"

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
	auction  *auction.AuctionUsecase
	presence RoomPresenceProvider
}

type RoomPresenceProvider interface {
	RoomPresence(roomID string) *v1.RoomPresence
}

func NewAuctionService(auction *auction.AuctionUsecase, presence ...RoomPresenceProvider) *AuctionService {
	s := &AuctionService{auction: auction}
	if len(presence) > 0 {
		s.presence = presence[0]
	}
	return s
}

func lotResultViewerFromContext(ctx context.Context) auction.LotResultViewer {
	viewer := auction.LotResultViewer{}
	if claims, ok := auth.ClaimsFromContext(ctx); ok {
		viewer.UserID = claims.UserID
		viewer.MainAccountID = auth.EffectiveMainAccountID(claims)
		viewer.Role = claims.Role
	}
	return viewer
}

func requireBackofficeMainAccount(ctx context.Context) (*auth.Claims, string, error) {
	claims, err := auth.RequireRole(ctx, v1.UserRole_USER_ROLE_ANCHOR, v1.UserRole_USER_ROLE_OPERATOR, v1.UserRole_USER_ROLE_MAIN_ACCOUNT)
	if err != nil {
		return nil, "", err
	}
	mainAccountID := auth.EffectiveMainAccountID(claims)
	if mainAccountID == "" {
		return nil, "", errors.New("main account id is required")
	}
	return claims, mainAccountID, nil
}

func (s *AuctionService) CreateLot(ctx context.Context, req *v1.CreateLotRequest) (*v1.CreateLotReply, error) {
	claims, mainAccountID, err := requireBackofficeMainAccount(ctx)
	if err != nil {
		return &v1.CreateLotReply{Result: ErrorResult(ctx, err)}, nil
	}
	lot, err := s.auction.CreateLot(ctx, req, mainAccountID, claims.UserID)
	if err != nil {
		return &v1.CreateLotReply{Result: ErrorResult(ctx, err)}, nil
	}
	return &v1.CreateLotReply{Result: okResult(ctx), Lot: lot}, nil
}

func (s *AuctionService) CreateLotDraft(ctx context.Context, req *v1.CreateLotRequest) (*v1.CreateLotReply, error) {
	claims, mainAccountID, err := requireBackofficeMainAccount(ctx)
	if err != nil {
		return &v1.CreateLotReply{Result: ErrorResult(ctx, err)}, nil
	}
	lot, err := s.auction.CreateLotDraft(ctx, req, mainAccountID, claims.UserID)
	if err != nil {
		return &v1.CreateLotReply{Result: ErrorResult(ctx, err)}, nil
	}
	return &v1.CreateLotReply{Result: okResult(ctx), Lot: lot}, nil
}

func (s *AuctionService) PatchLotDraft(ctx context.Context, req *v1.PatchLotDraftRequest) (*v1.PatchLotDraftReply, error) {
	claims, mainAccountID, err := requireBackofficeMainAccount(ctx)
	if err != nil {
		return &v1.PatchLotDraftReply{Result: ErrorResult(ctx, err)}, nil
	}
	lot, err := s.auction.PatchLotDraft(ctx, req, mainAccountID, claims.UserID)
	if err != nil {
		return &v1.PatchLotDraftReply{Result: ErrorResult(ctx, err)}, nil
	}
	return &v1.PatchLotDraftReply{Result: okResult(ctx), Lot: lot}, nil
}

func (s *AuctionService) QueueLot(ctx context.Context, req *v1.QueueLotRequest) (*v1.QueueLotReply, error) {
	claims, mainAccountID, err := requireBackofficeMainAccount(ctx)
	if err != nil {
		return &v1.QueueLotReply{Result: ErrorResult(ctx, err)}, nil
	}
	lot, queuePosition, err := s.auction.QueueLot(ctx, req.GetLotId(), mainAccountID, claims.UserID)
	if err != nil {
		return &v1.QueueLotReply{Result: ErrorResult(ctx, err), Lot: lot}, nil
	}
	return &v1.QueueLotReply{Result: okResult(ctx), Lot: lot, QueuePosition: queuePosition}, nil
}

func (s *AuctionService) GetLot(ctx context.Context, req *v1.GetLotRequest) (*v1.GetLotReply, error) {
	lot, err := s.auction.GetLot(ctx, req.GetLotId())
	if err != nil {
		return &v1.GetLotReply{Result: ErrorResult(ctx, err)}, nil
	}
	return &v1.GetLotReply{Result: okResult(ctx), Lot: auction.LotForViewer(lot, lotResultViewerFromContext(ctx))}, nil
}

func (s *AuctionService) ListLots(ctx context.Context, req *v1.ListLotsRequest) (*v1.ListLotsReply, error) {
	lots, err := s.auction.ListLots(ctx, req.GetRoomId(), req.GetStatus())
	if err != nil {
		return &v1.ListLotsReply{Result: ErrorResult(ctx, err)}, nil
	}
	return &v1.ListLotsReply{Result: okResult(ctx), Lots: auction.LotsForViewer(lots, lotResultViewerFromContext(ctx))}, nil
}

func (s *AuctionService) ListAdminLots(ctx context.Context, query auction.LotQuery) (auction.LotList, error) {
	_, mainAccountID, err := requireBackofficeMainAccount(ctx)
	if err != nil {
		return auction.LotList{}, err
	}
	query.MainAccountID = mainAccountID
	return s.auction.ListLotsByQuery(ctx, query)
}

func (s *AuctionService) ListAdminRooms(ctx context.Context) ([]auction.Room, error) {
	claims, mainAccountID, err := requireBackofficeMainAccount(ctx)
	if err != nil {
		return nil, err
	}
	if _, err := s.auction.EnsureDefaultRoom(ctx, mainAccountID, claims.UserID); err != nil {
		return nil, err
	}
	return s.auction.ListRooms(ctx, auction.RoomQuery{MainAccountID: mainAccountID})
}

func (s *AuctionService) ListPublicRooms(ctx context.Context) ([]auction.Room, error) {
	return s.auction.ListRooms(ctx, auction.RoomQuery{PublicOnly: true, PublicVisibleOnly: true})
}

func (s *AuctionService) StartLot(ctx context.Context, req *v1.StartLotRequest) (*v1.StartLotReply, error) {
	_, mainAccountID, err := requireBackofficeMainAccount(ctx)
	if err != nil {
		return &v1.StartLotReply{Result: ErrorResult(ctx, err)}, nil
	}
	lot, err := s.auction.StartLot(ctx, req.GetLotId(), mainAccountID)
	if err != nil {
		return &v1.StartLotReply{Result: ErrorResult(ctx, err)}, nil
	}
	return &v1.StartLotReply{Result: okResult(ctx), Lot: lot}, nil
}

func (s *AuctionService) PlaceBid(ctx context.Context, req *v1.PlaceBidRequest) (*v1.PlaceBidReply, error) {
	claims, authErr := auth.RequireRole(ctx, v1.UserRole_USER_ROLE_BUYER)
	if authErr != nil {
		result := ErrorResult(ctx, authErr)
		return &v1.PlaceBidReply{Result: result, Accepted: false, RejectReason: result.GetMessage()}, nil
	}
	lot, bid, ranking, err := s.auction.PlaceBid(ctx, req, claims.UserID, claims.Nickname)
	if err != nil {
		result := ErrorResult(ctx, err)
		viewer := lotResultViewerFromContext(ctx)
		return &v1.PlaceBidReply{Result: result, Accepted: false, Lot: auction.LotForViewer(lot, viewer), Ranking: auction.RankingForViewer(ranking, viewer), RejectReason: result.GetMessage()}, nil
	}
	viewer := lotResultViewerFromContext(ctx)
	return &v1.PlaceBidReply{
		Result:   okResult(ctx),
		Accepted: true,
		Lot:      auction.LotForViewer(lot, viewer),
		Bid:      auction.BidForViewer(bid, viewer),
		Ranking:  auction.RankingForViewer(ranking, viewer),
	}, nil
}

func (s *AuctionService) RevealTrustCard(ctx context.Context, req *v1.RevealTrustCardRequest) (*v1.RevealTrustCardReply, error) {
	_, mainAccountID, err := requireBackofficeMainAccount(ctx)
	if err != nil {
		return &v1.RevealTrustCardReply{Result: ErrorResult(ctx, err)}, nil
	}
	lot, card, err := s.auction.RevealTrustCard(ctx, req.GetLotId(), mainAccountID, req.GetCardId(), req.GetOperatorId())
	if err != nil {
		return &v1.RevealTrustCardReply{Result: ErrorResult(ctx, err), Lot: lot, TrustCard: card}, nil
	}
	return &v1.RevealTrustCardReply{Result: okResult(ctx), Lot: lot, TrustCard: card}, nil
}

func (s *AuctionService) StartDuel(ctx context.Context, req *v1.StartDuelRequest) (*v1.StartDuelReply, error) {
	_, mainAccountID, err := requireBackofficeMainAccount(ctx)
	if err != nil {
		return &v1.StartDuelReply{Result: ErrorResult(ctx, err)}, nil
	}
	lot, duel, err := s.auction.StartDuel(ctx, req.GetLotId(), mainAccountID, req.GetOperatorId(), req.GetUserAId(), req.GetUserBId())
	if err != nil {
		return &v1.StartDuelReply{Result: ErrorResult(ctx, err), Lot: lot, DuelState: duel}, nil
	}
	return &v1.StartDuelReply{Result: okResult(ctx), Lot: lot, DuelState: duel}, nil
}

func (s *AuctionService) SettleLot(ctx context.Context, req *v1.SettleLotRequest) (*v1.SettleLotReply, error) {
	_, mainAccountID, err := requireBackofficeMainAccount(ctx)
	if err != nil {
		return &v1.SettleLotReply{Result: ErrorResult(ctx, err)}, nil
	}
	lot, err := s.auction.SettleLot(ctx, req.GetLotId(), mainAccountID, req.GetOperatorId())
	if err != nil {
		return &v1.SettleLotReply{Result: ErrorResult(ctx, err), Lot: lot}, nil
	}
	return &v1.SettleLotReply{Result: okResult(ctx), Lot: lot}, nil
}

func (s *AuctionService) CancelLot(ctx context.Context, req *v1.CancelLotRequest) (*v1.CancelLotReply, error) {
	_, mainAccountID, err := requireBackofficeMainAccount(ctx)
	if err != nil {
		return &v1.CancelLotReply{Result: ErrorResult(ctx, err)}, nil
	}
	lot, err := s.auction.CancelLot(ctx, req.GetLotId(), mainAccountID, req.GetOperatorId(), req.GetReason())
	if err != nil {
		return &v1.CancelLotReply{Result: ErrorResult(ctx, err), Lot: lot}, nil
	}
	return &v1.CancelLotReply{Result: okResult(ctx), Lot: lot}, nil
}

func (s *AuctionService) GetRoomSnapshot(ctx context.Context, req *v1.GetRoomSnapshotRequest) (*v1.GetRoomSnapshotReply, error) {
	snapshot, err := s.auction.Snapshot(ctx, req.GetRoomId())
	if err != nil {
		return &v1.GetRoomSnapshotReply{Result: ErrorResult(ctx, err)}, nil
	}
	return &v1.GetRoomSnapshotReply{Result: okResult(ctx), Snapshot: auction.SnapshotForViewer(snapshot, lotResultViewerFromContext(ctx))}, nil
}

func (s *AuctionService) GetRoomPresence(ctx context.Context, req *v1.GetRoomPresenceRequest) (*v1.GetRoomPresenceReply, error) {
	_, mainAccountID, err := requireBackofficeMainAccount(ctx)
	if err != nil {
		return &v1.GetRoomPresenceReply{Result: ErrorResult(ctx, err)}, nil
	}
	if err := s.auction.ValidateRoomInMainAccount(ctx, req.GetRoomId(), mainAccountID); err != nil {
		return &v1.GetRoomPresenceReply{Result: ErrorResult(ctx, err)}, nil
	}
	if s.presence == nil {
		return &v1.GetRoomPresenceReply{Result: ErrorResult(ctx, errors.New("room presence provider is required"))}, nil
	}
	presence := s.presence.RoomPresence(req.GetRoomId())
	if presence.GetRoomId() == "" {
		return &v1.GetRoomPresenceReply{Result: ErrorResult(ctx, errors.New("room id is required"))}, nil
	}
	return &v1.GetRoomPresenceReply{Result: okResult(ctx), Presence: presence}, nil
}

func (s *AuctionService) ListRoomEvents(ctx context.Context, req *v1.ListRoomEventsRequest) (*v1.ListRoomEventsReply, error) {
	_, mainAccountID, err := requireBackofficeMainAccount(ctx)
	if err != nil {
		return &v1.ListRoomEventsReply{Result: ErrorResult(ctx, err)}, nil
	}
	list, err := s.auction.ListRoomEvents(ctx, auction.RoomEventQuery{
		RoomID:        req.GetRoomId(),
		MainAccountID: mainAccountID,
		PageSize:      int(req.GetPageSize()),
		PageToken:     req.GetPageToken(),
	})
	if err != nil {
		return &v1.ListRoomEventsReply{Result: ErrorResult(ctx, err)}, nil
	}
	return &v1.ListRoomEventsReply{Result: okResult(ctx), Events: list.Events, NextPageToken: list.NextPageToken}, nil
}

func (s *AuctionService) GetLotResult(ctx context.Context, lotID string) (*auction.LotResult, error) {
	return s.auction.GetLotResult(ctx, lotID, lotResultViewerFromContext(ctx))
}

func (s *AuctionService) ListMyOrders(ctx context.Context, queries ...auction.OrderQuery) ([]auction.OrderSummary, error) {
	claims, err := auth.RequireRole(ctx, v1.UserRole_USER_ROLE_BUYER)
	if err != nil {
		return nil, err
	}
	if len(queries) == 0 {
		return s.auction.ListOrdersByBuyer(ctx, claims.UserID)
	}
	list, err := s.auction.ListOrdersByBuyerQuery(ctx, claims.UserID, queries[0])
	if err != nil {
		return nil, err
	}
	return list.Orders, nil
}

func (s *AuctionService) ListMyOrdersPage(ctx context.Context, query auction.OrderQuery) (auction.OrderList, error) {
	claims, err := auth.RequireRole(ctx, v1.UserRole_USER_ROLE_BUYER)
	if err != nil {
		return auction.OrderList{}, err
	}
	return s.auction.ListOrdersByBuyerQuery(ctx, claims.UserID, query)
}

func (s *AuctionService) ListOrders(ctx context.Context, query auction.OrderQuery) (auction.OrderList, error) {
	_, mainAccountID, err := requireBackofficeMainAccount(ctx)
	if err != nil {
		return auction.OrderList{}, err
	}
	query.MainAccountID = mainAccountID
	return s.auction.ListOrders(ctx, query)
}

func (s *AuctionService) ListMyBids(ctx context.Context, query auction.BidRecordQuery) (auction.BidRecordList, error) {
	claims, err := auth.RequireRole(ctx, v1.UserRole_USER_ROLE_BUYER)
	if err != nil {
		return auction.BidRecordList{}, err
	}
	return s.auction.ListBidRecordsByBuyer(ctx, claims.UserID, query)
}

func (s *AuctionService) MockPayOrder(ctx context.Context, orderID string, req auction.MockPayRequest) (*auction.PaymentResult, error) {
	claims, err := auth.RequireRole(ctx, v1.UserRole_USER_ROLE_BUYER)
	if err != nil {
		return nil, err
	}
	return s.auction.MockPayOrder(ctx, claims.UserID, orderID, req)
}
