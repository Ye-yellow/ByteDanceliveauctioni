package service

import (
	"context"
	"log/slog"
	"sort"
	"strings"

	v1 "live-auction-bid/backend/api/auction/service/v1"
	"live-auction-bid/backend/app/auction/service/internal/aiassistant"
	"live-auction-bid/backend/app/auction/service/internal/biz/auction"
	"live-auction-bid/backend/app/auction/service/internal/searchindex"
)

func (s *AuctionService) SetAIAssistant(assistant *aiassistant.Assistant) *AuctionService {
	if s == nil {
		return nil
	}
	s.ai = assistant
	return s
}

func (s *AuctionService) SetBuyerSearch(index *searchindex.PGVectorIndex, embedder *searchindex.EmbeddingClient) *AuctionService {
	if s == nil {
		return nil
	}
	s.buyerSearch = index
	s.buyerEmbedder = embedder
	return s
}

func (s *AuctionService) ConsultBuyer(ctx context.Context, req *v1.BuyerConsultRequest) (*v1.BuyerConsultReply, error) {
	aiReq := aiassistant.BuyerConsultRequest{
		Query:          req.GetQuery(),
		RoomID:         req.GetRoomId(),
		LotID:          req.GetLotId(),
		Budget:         req.GetBudget(),
		RiskPreference: req.GetRiskPreference(),
	}
	assistant := s.aiAssistant()
	aiCtx := aiassistant.BuyerConsultContext{Candidates: s.buyerCandidates(ctx, aiReq)}
	if strings.TrimSpace(aiReq.RoomID) != "" {
		if snapshot, err := s.auction.Snapshot(ctx, strings.TrimSpace(aiReq.RoomID)); err == nil {
			aiCtx.Snapshot = auction.SnapshotForViewer(snapshot, lotResultViewerFromContext(ctx))
		}
	}
	reply, err := assistant.ConsultBuyer(ctx, aiReq, aiCtx)
	if err != nil {
		return &v1.BuyerConsultReply{Result: ErrorResult(ctx, err)}, nil
	}
	return buyerConsultReplyToProto(ctx, reply), nil
}

func (s *AuctionService) aiAssistant() *aiassistant.Assistant {
	if s != nil && s.ai != nil {
		return s.ai
	}
	return aiassistant.New(aiassistant.Config{Provider: "mock"})
}

func (s *AuctionService) buyerCandidates(ctx context.Context, req aiassistant.BuyerConsultRequest) []aiassistant.LotCandidate {
	vector := s.vectorBuyerCandidates(ctx, req)
	keyword := s.keywordBuyerCandidates(ctx, req)
	return mergeBuyerCandidates(vector, keyword, 8)
}

func (s *AuctionService) keywordBuyerCandidates(ctx context.Context, req aiassistant.BuyerConsultRequest) []aiassistant.LotCandidate {
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
					CurrentPrice: startSearchPriceMoney(lot),
					Href:         "/m/room/" + lot.GetRoomId(),
					Reason:       reason,
					ImageURL:     lot.GetImageUrl(),
					Score:        score,
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

func (s *AuctionService) vectorBuyerCandidates(ctx context.Context, req aiassistant.BuyerConsultRequest) []aiassistant.LotCandidate {
	if s == nil || s.buyerSearch == nil || s.buyerEmbedder == nil || !s.buyerEmbedder.Configured() {
		return nil
	}
	query := strings.TrimSpace(req.Query)
	if query == "" {
		return nil
	}
	embeddings, err := s.buyerEmbedder.Embed(ctx, []string{query})
	if err != nil || len(embeddings) == 0 {
		if err != nil {
			slog.Warn("buyer vector search embedding failed", "error", err)
		}
		return nil
	}
	docs, err := s.buyerSearch.Search(ctx, searchindex.SearchQuery{
		Vector: embeddings[0],
		RoomID: strings.TrimSpace(req.RoomID),
		LotID:  strings.TrimSpace(req.LotID),
		Limit:  20,
	})
	if err != nil {
		slog.Warn("buyer vector search failed", "error", err)
		return nil
	}
	roomNames := s.publicVisibleRoomNames(ctx)
	out := make([]aiassistant.LotCandidate, 0, len(docs))
	seen := map[string]bool{}
	for rank, doc := range docs {
		if doc.LotID == "" || seen[doc.LotID] {
			continue
		}
		if req.RoomID != "" && doc.RoomID != req.RoomID {
			continue
		}
		if req.LotID != "" && doc.LotID != req.LotID {
			continue
		}
		if _, ok := roomNames[doc.RoomID]; !ok {
			continue
		}
		lot, err := s.auction.GetLot(ctx, doc.LotID)
		if err != nil || lot == nil || !auction.IsPublicVisibleLotStatus(lot.GetStatus()) {
			continue
		}
		seen[doc.LotID] = true
		score := 80 - rank
		if price := currentSearchPrice(lot); req.Budget > 0 && price > 0 && price <= req.Budget {
			score += 4
		}
		out = append(out, aiassistant.LotCandidate{
			Type:         "lot",
			Title:        lot.GetTitle(),
			RoomID:       lot.GetRoomId(),
			LotID:        lot.GetId(),
			Status:       lot.GetStatus().String(),
			CurrentPrice: startSearchPriceMoney(lot),
			Href:         "/m/room/" + lot.GetRoomId(),
			Reason:       "语义匹配你的描述",
			ImageURL:     lot.GetImageUrl(),
			Score:        score,
		})
	}
	return out
}

func (s *AuctionService) publicVisibleRoomNames(ctx context.Context) map[string]string {
	rooms, err := s.auction.ListRooms(ctx, auction.RoomQuery{PublicOnly: true, PublicVisibleOnly: true})
	if err != nil {
		return map[string]string{}
	}
	out := make(map[string]string, len(rooms))
	for _, room := range rooms {
		out[room.ID] = room.Name
	}
	return out
}

func mergeBuyerCandidates(primary, secondary []aiassistant.LotCandidate, limit int) []aiassistant.LotCandidate {
	if limit <= 0 {
		limit = 8
	}
	byLotID := make(map[string]int)
	out := make([]aiassistant.LotCandidate, 0, len(primary)+len(secondary))
	add := func(candidate aiassistant.LotCandidate) {
		if strings.TrimSpace(candidate.LotID) == "" {
			return
		}
		if pos, ok := byLotID[candidate.LotID]; ok {
			if candidate.Score > out[pos].Score {
				out[pos].Score = candidate.Score
			}
			if out[pos].Reason == "" {
				out[pos].Reason = candidate.Reason
			}
			return
		}
		byLotID[candidate.LotID] = len(out)
		out = append(out, candidate)
	}
	for _, candidate := range primary {
		add(candidate)
	}
	for _, candidate := range secondary {
		add(candidate)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Score != out[j].Score {
			return out[i].Score > out[j].Score
		}
		return out[i].LotID < out[j].LotID
	})
	if len(out) > limit {
		return out[:limit]
	}
	return out
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

func currentSearchPrice(lot *v1.Lot) int64 {
	if lot == nil {
		return 0
	}
	if price := lot.GetCurrentPrice(); price != nil && price.GetAmount() > 0 {
		return price.GetAmount()
	}
	return lot.GetRule().GetStartPrice().GetAmount()
}

func startSearchPriceMoney(lot *v1.Lot) *v1.Money {
	if lot == nil {
		return nil
	}
	if price := lot.GetRule().GetStartPrice(); price != nil {
		return &v1.Money{Amount: price.GetAmount(), Currency: price.GetCurrency()}
	}
	return nil
}

func buyerConsultReplyToProto(ctx context.Context, reply aiassistant.BuyerConsultReply) *v1.BuyerConsultReply {
	out := &v1.BuyerConsultReply{
		Answer:       reply.Answer,
		Intent:       reply.Intent,
		FallbackUsed: reply.FallbackUsed,
		Result:       okResult(ctx),
	}
	for _, result := range reply.Results {
		out.Results = append(out.Results, &v1.BuyerConsultResult{
			Type:         result.Type,
			Title:        result.Title,
			RoomId:       result.RoomID,
			LotId:        result.LotID,
			Status:       result.Status,
			CurrentPrice: result.CurrentPrice,
			Href:         result.Href,
			Reason:       result.Reason,
			ImageUrl:     result.ImageURL,
		})
	}
	for _, source := range reply.Sources {
		out.Sources = append(out.Sources, &v1.BuyerConsultSource{
			Type:   source.Type,
			Title:  source.Title,
			RoomId: source.RoomID,
			LotId:  source.LotID,
		})
	}
	return out
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
