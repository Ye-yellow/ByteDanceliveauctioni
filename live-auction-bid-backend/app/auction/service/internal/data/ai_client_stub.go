package data

import (
	"context"
	"fmt"

	biz "live-auction-bid/backend/app/auction/service/internal/biz"
)

// StubAI is the open-source-AI integration seam. Replace with Ollama/Qwen/Llama HTTP client later.
type StubAI struct{}

func (StubAI) OnBid(ctx context.Context, lot *biz.Lot, bid biz.Bid) string {
	return fmt.Sprintf("%s 出价 %.2f，当前领先！还有机会，喜欢就别错过。", bid.Nickname, float64(bid.Amount)/100)
}

func (StubAI) SuggestStartPrice(ctx context.Context, title, description string, referencePrice biz.Money) biz.Money {
	if referencePrice > 0 {
		return referencePrice * 8 / 10
	}
	return 10000
}

func (StubAI) GenerateLine(ctx context.Context, req biz.AtmosphereRequest) (string, error) {
	if req.LatestBid != nil {
		return fmt.Sprintf("%s 刚刚出价 %.2f，直播间热度继续上升！", req.LatestBid.Nickname, float64(req.LatestBid.Amount)/100), nil
	}
	return "竞拍正在进行，喜欢这件拍品的朋友可以准备出价。", nil
}
