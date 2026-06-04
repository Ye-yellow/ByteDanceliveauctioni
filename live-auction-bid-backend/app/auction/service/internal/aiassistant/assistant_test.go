package aiassistant

import (
	"context"
	"testing"

	v1 "live-auction-bid/backend/api/auction/service/v1"
)

func TestMockBuyerConsultUsesCandidatesAndBidAdvice(t *testing.T) {
	assistant := New(Config{Provider: "mock"})
	reply, err := assistant.ConsultBuyer(context.Background(), BuyerConsultRequest{
		Query:          "预算两万想看翡翠手镯",
		Budget:         2000000,
		RiskPreference: "steady",
	}, BuyerConsultContext{
		Candidates: []LotCandidate{{
			Type:         "lot",
			Title:        "冰糯翡翠手镯",
			RoomID:       "room-live",
			LotID:        "lot-live",
			Status:       "LOT_STATUS_LIVE",
			CurrentPrice: &v1.Money{Amount: 1800000, Currency: "CNY"},
			Href:         "/m/room/room-live",
			Reason:       "标题命中 翡翠 手镯",
			Lot:          sampleLot("lot-live", "room-live", "冰糯翡翠手镯", v1.LotStatus_LOT_STATUS_LIVE),
		}},
	})
	if err != nil {
		t.Fatalf("mock consult should not fail: %v", err)
	}
	if !reply.FallbackUsed {
		t.Fatal("mock provider should mark fallbackUsed")
	}
	if reply.Intent != "find_auction" || len(reply.Results) != 1 || reply.Results[0].LotID != "lot-live" {
		t.Fatalf("unexpected consult reply: %+v", reply)
	}
	if reply.BidAdvice.NextBidAmount == nil || reply.BidAdvice.NextBidAmount.Amount == 0 {
		t.Fatalf("expected next bid advice, got %+v", reply.BidAdvice)
	}
	if len(reply.Sources) == 0 || reply.Sources[0].LotID != "lot-live" {
		t.Fatalf("expected cited lot source, got %+v", reply.Sources)
	}
}

func TestBuyerReplyFromMapAcceptsLooseModelShape(t *testing.T) {
	reply := buyerReplyFromMap(map[string]any{
		"answer": "found",
		"intent": "find_auction",
		"results": []any{
			map[string]any{
				"type":         "lot",
				"title":        "AI demo lot",
				"roomId":       "room-ai",
				"lotId":        "lot-ai",
				"status":       "LOT_STATUS_LIVE",
				"currentPrice": map[string]any{"amount": float64(1800000), "currency": "CNY"},
				"href":         "/m/room/room-ai",
				"reason":       "matched query",
			},
		},
		"bidAdvice": map[string]any{
			"nextBidAmount":      float64(1820000),
			"maxSuggestedAmount": map[string]any{"amount": float64(2000000), "currency": "CNY"},
			"strategy":           "bid carefully",
			"risks":              "manual confirmation required",
			"confidence":         float64(0.8),
		},
		"sources": []any{"AI demo lot"},
	})

	if reply.Answer != "found" || reply.Intent != "find_auction" {
		t.Fatalf("basic fields not decoded: %+v", reply)
	}
	if len(reply.Results) != 1 || reply.Results[0].CurrentPrice.GetAmount() != 1800000 {
		t.Fatalf("results not decoded: %+v", reply.Results)
	}
	if reply.BidAdvice.NextBidAmount.GetAmount() != 1820000 || len(reply.BidAdvice.Risks) != 1 || reply.BidAdvice.Confidence != 0.8 {
		t.Fatalf("bid advice not decoded: %+v", reply.BidAdvice)
	}
	if len(reply.Sources) != 1 || reply.Sources[0].Title != "AI demo lot" {
		t.Fatalf("sources not decoded: %+v", reply.Sources)
	}
}

func TestMockMerchantAssistantSuggestsSafeActions(t *testing.T) {
	assistant := New(Config{Provider: "mock"})
	lot := sampleLot("lot-control", "room-control", "翡翠控场拍品", v1.LotStatus_LOT_STATUS_LIVE)
	lot.TrustCards = []*v1.TrustRevealCard{{Id: "card-1", LotId: lot.Id, Title: "证书卡", Content: "天然翡翠证书", Revealed: false}}
	reply, err := assistant.AssistMerchant(context.Background(), MerchantAssistRequest{Page: "control", RoomID: lot.RoomId, LotID: lot.Id}, MerchantAssistContext{
		RoomID:      lot.RoomId,
		CurrentLot:  lot,
		RankingSize: 2,
	})
	if err != nil {
		t.Fatalf("mock merchant assistant should not fail: %v", err)
	}
	if !reply.FallbackUsed {
		t.Fatal("mock provider should mark fallbackUsed")
	}
	if len(reply.RecommendedActions) == 0 {
		t.Fatalf("expected safe recommended actions, got %+v", reply)
	}
	for _, action := range reply.RecommendedActions {
		switch action.Type {
		case ActionRevealTrustCard, ActionStartDuel, ActionNavigate, ActionCopyText:
		default:
			t.Fatalf("unsafe action type leaked: %+v", action)
		}
	}
}

func TestMerchantReplyFromMapAcceptsLooseModelShape(t *testing.T) {
	reply := merchantReplyFromMap(map[string]any{
		"answer":    "ok",
		"checklist": []any{"fill title", map[string]any{"label": "add image", "status": "missing", "reason": "main image is required"}},
		"nextSteps": "wait for merchant input",
		"recommendedActions": []any{
			map[string]any{"type": ActionStartDuel, "label": "start duel", "reason": "top2 close", "enabled": true},
		},
		"draftSuggestions": map[string]any{
			"title":       "AI demo lot",
			"description": "demo description",
			"tags":        []any{"jade", "demo"},
			"trustCards": []any{
				map[string]any{"type": "TRUST_CARD_TYPE_CERTIFICATE", "title": "certificate", "content": "show certificate"},
			},
		},
		"warnings": "manual confirmation required",
	})

	if reply.Answer != "ok" {
		t.Fatalf("answer not decoded: %+v", reply)
	}
	if len(reply.Checklist) != 2 || reply.Checklist[0].Label != "fill title" || reply.Checklist[1].Status != "missing" {
		t.Fatalf("checklist not decoded: %+v", reply.Checklist)
	}
	if len(reply.NextSteps) != 1 || reply.NextSteps[0] != "wait for merchant input" {
		t.Fatalf("nextSteps not decoded: %+v", reply.NextSteps)
	}
	if len(reply.RecommendedActions) != 1 || reply.RecommendedActions[0].Type != ActionStartDuel {
		t.Fatalf("actions not decoded: %+v", reply.RecommendedActions)
	}
	if reply.DraftSuggestions.TitleSuggestion != "AI demo lot" || len(reply.DraftSuggestions.Tags) != 2 || len(reply.DraftSuggestions.TrustCards) != 1 {
		t.Fatalf("draft suggestions not decoded: %+v", reply.DraftSuggestions)
	}
	if len(reply.Warnings) != 1 {
		t.Fatalf("warnings not decoded: %+v", reply.Warnings)
	}
}

func sampleLot(id, roomID, title string, status v1.LotStatus) *v1.Lot {
	return &v1.Lot{
		Id:          id,
		RoomId:      roomID,
		Title:       title,
		Description: "适合直播竞拍演示的高价值非标品",
		ImageUrl:    "https://example.com/lot.jpg",
		Status:      status,
		Rule: &v1.BidRule{
			StartPrice:      &v1.Money{Amount: 1000000, Currency: "CNY"},
			MinIncrement:    &v1.Money{Amount: 100000, Currency: "CNY"},
			DurationSeconds: 300,
		},
		CurrentPrice: &v1.Money{Amount: 1800000, Currency: "CNY"},
	}
}
