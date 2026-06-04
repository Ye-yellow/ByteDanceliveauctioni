package service

import (
	"context"
	"sort"
	"strings"

	v1 "live-auction-bid/backend/api/auction/service/v1"
	"live-auction-bid/backend/app/auction/service/internal/aiassistant"
	"live-auction-bid/backend/app/auction/service/internal/biz/auction"
	userbiz "live-auction-bid/backend/app/auction/service/internal/biz/user"
	"live-auction-bid/backend/app/auction/service/internal/pkg/auth"
)

func (s *AuctionService) SetAIAssistant(assistant *aiassistant.Assistant) *AuctionService {
	if s == nil {
		return nil
	}
	s.ai = assistant
	return s
}

func (s *AuctionService) ConsultBuyer(ctx context.Context, req aiassistant.BuyerConsultRequest) (aiassistant.BuyerConsultReply, error) {
	assistant := s.aiAssistant()
	aiCtx := aiassistant.BuyerConsultContext{Candidates: s.buyerCandidates(ctx, req)}
	if strings.TrimSpace(req.RoomID) != "" {
		if snapshot, err := s.auction.Snapshot(ctx, strings.TrimSpace(req.RoomID)); err == nil {
			aiCtx.Snapshot = auction.SnapshotForViewer(snapshot, lotResultViewerFromContext(ctx))
		}
	}
	reply, err := assistant.ConsultBuyer(ctx, req, aiCtx)
	if err != nil {
		return aiassistant.BuyerConsultReply{Result: ErrorResult(ctx, err)}, nil
	}
	reply.Result = okResult(ctx)
	return reply, nil
}

func (s *AuctionService) AssistMerchant(ctx context.Context, req aiassistant.MerchantAssistRequest) (aiassistant.MerchantAssistReply, error) {
	page := strings.ToLower(strings.TrimSpace(req.Page))
	var permission string
	switch page {
	case "create", "auction-create":
		permission = userbiz.PermissionLotCreate
	case "control", "live-control", "realtime-control":
		permission = userbiz.PermissionAuctionControl
	default:
		permission = userbiz.PermissionLotViewAdmin
	}
	_, mainAccountID, err := requirePermissionMainAccount(ctx, permission)
	if err != nil {
		return aiassistant.MerchantAssistReply{Result: ErrorResult(ctx, err)}, nil
	}
	aiCtx := aiassistant.MerchantAssistContext{RoomID: strings.TrimSpace(req.RoomID)}
	if aiCtx.RoomID != "" {
		if err := s.auction.ValidateRoomInMainAccount(ctx, aiCtx.RoomID, mainAccountID); err != nil {
			return aiassistant.MerchantAssistReply{Result: ErrorResult(ctx, err)}, nil
		}
		if snapshot, err := s.auction.Snapshot(ctx, aiCtx.RoomID); err == nil {
			aiCtx.Snapshot = auction.SnapshotForViewer(snapshot, lotResultViewerFromContext(ctx))
			aiCtx.CurrentLot = aiCtx.Snapshot.GetCurrentLot()
			aiCtx.RankingSize = len(aiCtx.Snapshot.GetRanking())
		}
	}
	if req.LotID != "" && aiCtx.CurrentLot == nil {
		if lot, err := s.auction.GetLot(ctx, req.LotID); err == nil && lot.GetMainAccountId() == mainAccountID {
			aiCtx.CurrentLot = auction.LotForViewer(lot, lotResultViewerFromContext(ctx))
		}
	}
	reply, err := s.aiAssistant().AssistMerchant(ctx, req, aiCtx)
	if err != nil {
		return aiassistant.MerchantAssistReply{Result: ErrorResult(ctx, err)}, nil
	}
	reply.Result = okResult(ctx)
	return reply, nil
}

func (s *AuctionService) aiAssistant() *aiassistant.Assistant {
	if s != nil && s.ai != nil {
		return s.ai
	}
	return aiassistant.New(aiassistant.Config{Provider: "mock"})
}

func (s *AuctionService) buyerCandidates(ctx context.Context, req aiassistant.BuyerConsultRequest) []aiassistant.LotCandidate {
	rooms, err := s.auction.ListRooms(ctx, auction.RoomQuery{PublicOnly: true, PublicVisibleOnly: true})
	if err != nil {
		return nil
	}
	roomNames := make(map[string]string, len(rooms))
	for _, room := range rooms {
		roomNames[room.ID] = room.Name
	}
	candidates := make([]aiassistant.LotCandidate, 0)
	seen := map[string]bool{}
	for _, room := range rooms {
		if req.RoomID != "" && room.ID != req.RoomID {
			continue
		}
		for _, status := range []v1.LotStatus{v1.LotStatus_LOT_STATUS_QUEUED, v1.LotStatus_LOT_STATUS_LIVE} {
			lots, err := s.auction.ListLots(ctx, room.ID, status)
			if err != nil {
				continue
			}
			for _, lot := range lots {
				if lot == nil || seen[lot.GetId()] || !auction.IsPublicVisibleLotStatus(lot.GetStatus()) {
					continue
				}
				if req.LotID != "" && lot.GetId() != req.LotID {
					continue
				}
				seen[lot.GetId()] = true
				score, reason := scoreLot(req, lot, roomNames[room.ID])
				if strings.TrimSpace(req.Query) != "" && score <= 0 {
					continue
				}
				candidates = append(candidates, aiassistant.LotCandidate{
					Type:         "lot",
					Title:        lot.GetTitle(),
					RoomID:       lot.GetRoomId(),
					LotID:        lot.GetId(),
					Status:       lot.GetStatus().String(),
					CurrentPrice: lot.GetCurrentPrice(),
					Href:         "/m/room/" + lot.GetRoomId(),
					Reason:       reason,
					Score:        score,
					Lot:          auction.LotForViewer(lot, auction.LotResultViewer{}),
				})
			}
		}
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].Score != candidates[j].Score {
			return candidates[i].Score > candidates[j].Score
		}
		return candidates[i].LotID < candidates[j].LotID
	})
	if len(candidates) > 8 {
		return candidates[:8]
	}
	return candidates
}

func scoreLot(req aiassistant.BuyerConsultRequest, lot *v1.Lot, roomName string) (int, string) {
	query := strings.ToLower(strings.TrimSpace(req.Query))
	haystack := strings.ToLower(strings.Join([]string{
		lot.GetTitle(),
		lot.GetDescription(),
		lot.GetCategory(),
		strings.Join(lot.GetTags(), " "),
		roomName,
	}, " "))
	score := 0
	matches := make([]string, 0)
	for _, token := range queryTokens(query) {
		if token != "" && strings.Contains(haystack, token) {
			score += 5
			matches = append(matches, token)
		}
	}
	switch lot.GetStatus() {
	case v1.LotStatus_LOT_STATUS_LIVE, v1.LotStatus_LOT_STATUS_EXTENDED:
		score += 3
	case v1.LotStatus_LOT_STATUS_QUEUED:
		score += 2
	}
	current := lot.GetCurrentPrice().GetAmount()
	if current <= 0 {
		current = lot.GetRule().GetStartPrice().GetAmount()
	}
	if req.Budget > 0 && current > 0 && current <= req.Budget {
		score += 2
	}
	if len(matches) == 0 {
		return score, "公开可见拍品，可进入直播间查看"
	}
	return score, "命中：" + strings.Join(matches, "、")
}

func queryTokens(query string) []string {
	query = strings.NewReplacer("，", " ", "。", " ", ",", " ", ".", " ", "的", " ", "想", " ", "看", " ", "找", " ").Replace(query)
	fields := strings.Fields(query)
	out := make([]string, 0, len(fields)+4)
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if field == "" || len([]rune(field)) <= 1 {
			continue
		}
		out = append(out, field)
	}
	for _, keyword := range []string{"翡翠", "手镯", "珠宝", "玉", "和田玉", "奢侈品", "收藏"} {
		if strings.Contains(query, strings.ToLower(keyword)) {
			out = append(out, strings.ToLower(keyword))
		}
	}
	return out
}

func _authClaimsForAI(ctx context.Context) *auth.Claims {
	claims, _ := auth.ClaimsFromContext(ctx)
	return claims
}
