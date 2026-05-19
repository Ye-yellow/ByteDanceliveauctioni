package service

import (
	"context"

	v1 "live-auction-bid/backend/api/auction/service/v1"
	"live-auction-bid/backend/app/auction/service/internal/biz/auction"
	"live-auction-bid/backend/app/auction/service/internal/model"
)

// AuctionService 是 Kratos service 层适配器。
//
// 它实现 api/auction/service/v1/auction.proto 生成的 HTTP/gRPC service 接口。
// 分层规则：这里只做 proto DTO <-> biz domain 的转换，不写竞拍业务规则。
type AuctionService struct {
	v1.UnimplementedAuctionServiceServer
	auction *auction.AuctionUsecase
}

func NewAuctionService(auction *auction.AuctionUsecase) *AuctionService {
	return &AuctionService{auction: auction}
}

func (s *AuctionService) CreateLot(ctx context.Context, req *v1.CreateLotRequest) (*v1.CreateLotReply, error) {
	lot, err := s.auction.CreateLot(ctx, toCreateLotCommand(req))
	if err != nil {
		return nil, err
	}
	return &v1.CreateLotReply{Result: okResult(), Lot: toProtoLot(lot)}, nil
}

func (s *AuctionService) GetLot(ctx context.Context, req *v1.GetLotRequest) (*v1.GetLotReply, error) {
	lot, err := s.auction.GetLot(ctx, req.GetLotId())
	if err != nil {
		return nil, err
	}
	return &v1.GetLotReply{Result: okResult(), Lot: toProtoLot(lot)}, nil
}

func (s *AuctionService) ListLots(ctx context.Context, req *v1.ListLotsRequest) (*v1.ListLotsReply, error) {
	lots, err := s.auction.ListLots(ctx, req.GetRoomId(), toDomainLotStatus(req.GetStatus()))
	if err != nil {
		return nil, err
	}
	return &v1.ListLotsReply{Result: okResult(), Lots: toProtoLots(lots)}, nil
}

func (s *AuctionService) StartLot(ctx context.Context, req *v1.StartLotRequest) (*v1.StartLotReply, error) {
	lot, err := s.auction.StartLot(ctx, req.GetLotId())
	if err != nil {
		return nil, err
	}
	return &v1.StartLotReply{Result: okResult(), Lot: toProtoLot(lot)}, nil
}

func (s *AuctionService) PlaceBid(ctx context.Context, req *v1.PlaceBidRequest) (*v1.PlaceBidReply, error) {
	lot, bid, ranking, err := s.auction.PlaceBid(ctx, model.PlaceBidCommand{
		LotID:              req.GetLotId(),
		UserID:             req.GetUserId(),
		Nickname:           req.GetNickname(),
		Amount:             toDomainMoney(req.GetAmount()),
		ClientKnownVersion: req.GetClientKnownVersion(),
		IdempotencyKey:     req.GetIdempotencyKey(),
	})
	if err != nil {
		return nil, err
	}
	return &v1.PlaceBidReply{
		Result:   okResult(),
		Accepted: true,
		Lot:      toProtoLot(lot),
		Bid:      toProtoBidPtr(bid),
		Ranking:  toProtoRanking(ranking),
	}, nil
}

func (s *AuctionService) RevealTrustCard(ctx context.Context, req *v1.RevealTrustCardRequest) (*v1.RevealTrustCardReply, error) {
	lot, card, err := s.auction.RevealTrustCard(ctx, req.GetLotId(), req.GetCardId(), req.GetOperatorId())
	if err != nil {
		return nil, err
	}
	return &v1.RevealTrustCardReply{Result: okResult(), Lot: toProtoLot(lot), TrustCard: toProtoTrustCardPtr(card)}, nil
}

func (s *AuctionService) StartDuel(ctx context.Context, req *v1.StartDuelRequest) (*v1.StartDuelReply, error) {
	lot, duel, err := s.auction.StartDuel(ctx, req.GetLotId(), req.GetOperatorId(), req.GetUserAId(), req.GetUserBId())
	if err != nil {
		return nil, err
	}
	return &v1.StartDuelReply{Result: okResult(), Lot: toProtoLot(lot), DuelState: toProtoDuelStatePtr(duel)}, nil
}

func (s *AuctionService) SettleLot(ctx context.Context, req *v1.SettleLotRequest) (*v1.SettleLotReply, error) {
	lot, err := s.auction.SettleLot(ctx, req.GetLotId(), req.GetOperatorId())
	if err != nil {
		return nil, err
	}
	return &v1.SettleLotReply{Result: okResult(), Lot: toProtoLot(lot)}, nil
}

func (s *AuctionService) GetRoomSnapshot(ctx context.Context, req *v1.GetRoomSnapshotRequest) (*v1.GetRoomSnapshotReply, error) {
	snapshot, err := s.auction.Snapshot(ctx, req.GetRoomId())
	if err != nil {
		return nil, err
	}
	return &v1.GetRoomSnapshotReply{Result: okResult(), Snapshot: toProtoSnapshot(snapshot)}, nil
}
func okResult() *v1.ReplyResult {
	return &v1.ReplyResult{Code: 0, Message: "ok"}
}

func toCreateLotCommand(req *v1.CreateLotRequest) model.CreateLotCommand {
	return model.CreateLotCommand{
		RoomID:      req.GetRoomId(),
		Title:       req.GetTitle(),
		Description: req.GetDescription(),
		ImageURL:    req.GetImageUrl(),
		Rule:        toDomainBidRule(req.GetRule()),
		TrustCards:  toDomainTrustCards(req.GetTrustCards()),
	}
}

func toDomainBidRule(rule *v1.BidRule) model.BidRule {
	if rule == nil {
		return model.BidRule{}
	}
	return model.BidRule{
		StartPrice:             toDomainMoney(rule.GetStartPrice()),
		MinIncrement:           toDomainMoney(rule.GetMinIncrement()),
		DurationSeconds:        rule.GetDurationSeconds(),
		AntiSnipeWindowSeconds: rule.GetAntiSnipeWindowSeconds(),
		AntiSnipeExtendSeconds: rule.GetAntiSnipeExtendSeconds(),
		MaxExtendCount:         rule.GetMaxExtendCount(),
	}
}

func toProtoBidRule(rule model.BidRule) *v1.BidRule {
	return &v1.BidRule{
		StartPrice:             toProtoMoney(rule.StartPrice),
		MinIncrement:           toProtoMoney(rule.MinIncrement),
		DurationSeconds:        rule.DurationSeconds,
		AntiSnipeWindowSeconds: rule.AntiSnipeWindowSeconds,
		AntiSnipeExtendSeconds: rule.AntiSnipeExtendSeconds,
		MaxExtendCount:         rule.MaxExtendCount,
	}
}

func toDomainMoney(m *v1.Money) model.Money {
	if m == nil {
		return model.Money{}
	}
	return model.Money{Amount: m.GetAmount(), Currency: m.GetCurrency()}
}

func toProtoMoney(m model.Money) *v1.Money {
	return &v1.Money{Amount: m.Amount, Currency: m.Currency}
}

func toDomainTrustCards(cards []*v1.TrustRevealCard) []model.TrustRevealCard {
	out := make([]model.TrustRevealCard, 0, len(cards))
	for _, card := range cards {
		out = append(out, model.TrustRevealCard{
			ID:               card.GetId(),
			LotID:            card.GetLotId(),
			Type:             model.TrustCardType(card.GetType().String()),
			Title:            card.GetTitle(),
			Content:          card.GetContent(),
			ImageURL:         card.GetImageUrl(),
			Revealed:         card.GetRevealed(),
			RevealedAtUnixMs: card.GetRevealedAtUnixMs(),
		})
	}
	return out
}

func toProtoTrustCards(cards []model.TrustRevealCard) []*v1.TrustRevealCard {
	out := make([]*v1.TrustRevealCard, 0, len(cards))
	for i := range cards {
		out = append(out, toProtoTrustCard(cards[i]))
	}
	return out
}

func toProtoTrustCardPtr(card *model.TrustRevealCard) *v1.TrustRevealCard {
	if card == nil {
		return nil
	}
	return toProtoTrustCard(*card)
}

func toProtoTrustCard(card model.TrustRevealCard) *v1.TrustRevealCard {
	return &v1.TrustRevealCard{
		Id:               card.ID,
		LotId:            card.LotID,
		Type:             v1.TrustCardType(v1.TrustCardType_value[string(card.Type)]),
		Title:            card.Title,
		Content:          card.Content,
		ImageUrl:         card.ImageURL,
		Revealed:         card.Revealed,
		RevealedAtUnixMs: card.RevealedAtUnixMs,
	}
}

func toProtoLot(lot *model.Lot) *v1.Lot {
	if lot == nil {
		return nil
	}
	return &v1.Lot{
		Id:              lot.ID,
		RoomId:          lot.RoomID,
		Title:           lot.Title,
		Description:     lot.Description,
		ImageUrl:        lot.ImageURL,
		Status:          v1.LotStatus(v1.LotStatus_value[string(lot.Status)]),
		Rule:            toProtoBidRule(lot.Rule),
		CurrentPrice:    toProtoMoney(lot.CurrentPrice),
		LeadingUserId:   lot.LeadingUserID,
		LeadingNickname: lot.LeadingNickname,
		StartedAtUnixMs: lot.StartedAtUnixMs,
		EndsAtUnixMs:    lot.EndsAtUnixMs,
		SettledAtUnixMs: lot.SettledAtUnixMs,
		WinnerUserId:    lot.WinnerUserID,
		WinnerNickname:  lot.WinnerNickname,
		FinalPrice:      toProtoMoney(lot.FinalPrice),
		Version:         lot.Version,
		TrustCards:      toProtoTrustCards(lot.TrustCards),
		DuelState:       toProtoDuelState(lot.DuelState),
		PlaybookStage:   v1.PlaybookStage(v1.PlaybookStage_value[string(lot.PlaybookStage)]),
	}
}

func toProtoLots(lots []*model.Lot) []*v1.Lot {
	out := make([]*v1.Lot, 0, len(lots))
	for _, lot := range lots {
		out = append(out, toProtoLot(lot))
	}
	return out
}

func toDomainLotStatus(status v1.LotStatus) model.LotStatus {
	if status == v1.LotStatus_LOT_STATUS_UNSPECIFIED {
		return ""
	}
	return model.LotStatus(status.String())
}

func toProtoBidPtr(bid *model.Bid) *v1.Bid {
	if bid == nil {
		return nil
	}
	return &v1.Bid{
		Id:              bid.ID,
		LotId:           bid.LotID,
		UserId:          bid.UserID,
		Nickname:        bid.Nickname,
		Amount:          toProtoMoney(bid.Amount),
		CreatedAtUnixMs: bid.CreatedAtUnixMs,
	}
}

func toProtoBids(bids []model.Bid) []*v1.Bid {
	out := make([]*v1.Bid, 0, len(bids))
	for i := range bids {
		out = append(out, toProtoBidPtr(&bids[i]))
	}
	return out
}

func toProtoRanking(ranking []model.RankingItem) []*v1.RankingItem {
	out := make([]*v1.RankingItem, 0, len(ranking))
	for _, item := range ranking {
		out = append(out, &v1.RankingItem{
			Rank:        item.Rank,
			UserId:      item.UserID,
			Nickname:    item.Nickname,
			Amount:      toProtoMoney(item.Amount),
			BidAtUnixMs: item.BidAtUnixMs,
		})
	}
	return out
}

func toProtoDuelStatePtr(duel *model.DuelState) *v1.DuelState {
	if duel == nil {
		return nil
	}
	return toProtoDuelState(*duel)
}

func toProtoDuelState(duel model.DuelState) *v1.DuelState {
	return &v1.DuelState{
		Active:          duel.Active,
		LotId:           duel.LotID,
		UserAId:         duel.UserAID,
		UserANickname:   duel.UserANickname,
		UserBId:         duel.UserBID,
		UserBNickname:   duel.UserBNickname,
		StartedAtUnixMs: duel.StartedAtUnixMs,
		EndsAtUnixMs:    duel.EndsAtUnixMs,
		ExtendCount:     duel.ExtendCount,
		MaxExtendCount:  duel.MaxExtendCount,
	}
}

func toProtoSnapshot(snapshot *model.RoomSnapshot) *v1.RoomSnapshot {
	if snapshot == nil {
		return nil
	}
	return &v1.RoomSnapshot{
		RoomId:           snapshot.RoomID,
		CurrentLot:       toProtoLot(snapshot.CurrentLot),
		Ranking:          toProtoRanking(snapshot.Ranking),
		RecentBids:       toProtoBids(snapshot.RecentBids),
		PlaybookStage:    v1.PlaybookStage(v1.PlaybookStage_value[string(snapshot.PlaybookStage)]),
		ServerTimeUnixMs: snapshot.ServerTimeUnixMs,
	}
}
