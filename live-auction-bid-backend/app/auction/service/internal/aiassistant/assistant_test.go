package aiassistant

import (
	"context"
	"testing"

	v1 "live-auction-bid/backend/api/auction/service/v1"
)

func TestMockBuyerConsultUsesCandidates(t *testing.T) {
	assistant := New(Config{Provider: "mock"})
	reply, err := assistant.ConsultBuyer(context.Background(), BuyerConsultRequest{
		Query:          "想看翡翠手镯",
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
				"title":        "精选翡翠拍品",
				"roomId":       "room-ai",
				"lotId":        "lot-ai",
				"status":       "LOT_STATUS_LIVE",
				"currentPrice": map[string]any{"amount": float64(1800000), "currency": "CNY"},
				"href":         "/m/room/room-ai",
				"reason":       "matched query",
			},
		},
		"sources": []any{"精选翡翠拍品"},
	})

	if reply.Answer != "found" || reply.Intent != "find_auction" {
		t.Fatalf("basic fields not decoded: %+v", reply)
	}
	if len(reply.Results) != 1 || reply.Results[0].CurrentPrice.GetAmount() != 1800000 {
		t.Fatalf("results not decoded: %+v", reply.Results)
	}
	if len(reply.Sources) != 1 || reply.Sources[0].Title != "精选翡翠拍品" {
		t.Fatalf("sources not decoded: %+v", reply.Sources)
	}
}
