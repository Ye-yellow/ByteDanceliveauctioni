package memory

import (
	"context"
	"fmt"

	domain "live-auction-bid/backend/internal/domain/auction"
)

// StubAI is the open-source-AI integration seam. Replace with Ollama/Qwen/Llama HTTP client later.
type StubAI struct{}

func (StubAI) OnBid(ctx context.Context, lot *domain.Lot, bid domain.Bid) string {
	return fmt.Sprintf("%s 出价 %.2f，当前领先！还有机会，喜欢就别错过。", bid.Nickname, float64(bid.Amount)/100)
}

func (StubAI) SuggestStartPrice(ctx context.Context, title, description string, referencePrice domain.Money) domain.Money {
	if referencePrice > 0 {
		return referencePrice * 8 / 10
	}
	return 10000
}
