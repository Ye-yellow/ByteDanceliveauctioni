package server

import (
	"context"

	httptransport "github.com/go-kratos/kratos/v2/transport/http"
	"live-auction-bid/backend/app/auction/service/internal/aiassistant"
	appsvc "live-auction-bid/backend/app/auction/service/internal/service"
)

func registerAIHTTP(srv *httptransport.Server, service *appsvc.AuctionService) {
	r := srv.Route("/")
	r.POST("/api/ai/buyer/consult", func(ctx httptransport.Context) error {
		var req aiassistant.BuyerConsultRequest
		if err := ctx.Bind(&req); err != nil {
			return ctx.Result(200, aiassistant.BuyerConsultReply{Result: appsvc.ErrorResult(ctx, err)})
		}
		h := ctx.Middleware(func(ctx context.Context, raw any) (any, error) {
			reply, err := service.ConsultBuyer(ctx, req)
			if err != nil {
				return aiassistant.BuyerConsultReply{Result: appsvc.ErrorResult(ctx, err)}, nil
			}
			return reply, nil
		})
		out, err := h(ctx, &req)
		if err != nil {
			return ctx.Result(200, aiassistant.BuyerConsultReply{Result: appsvc.ErrorResult(ctx, err)})
		}
		return ctx.Result(200, out)
	})
	r.POST("/api/ai/merchant/assistant", func(ctx httptransport.Context) error {
		var req aiassistant.MerchantAssistRequest
		if err := ctx.Bind(&req); err != nil {
			return ctx.Result(200, aiassistant.MerchantAssistReply{Result: appsvc.ErrorResult(ctx, err)})
		}
		h := ctx.Middleware(func(ctx context.Context, raw any) (any, error) {
			reply, err := service.AssistMerchant(ctx, req)
			if err != nil {
				return aiassistant.MerchantAssistReply{Result: appsvc.ErrorResult(ctx, err)}, nil
			}
			return reply, nil
		})
		out, err := h(ctx, &req)
		if err != nil {
			return ctx.Result(200, aiassistant.MerchantAssistReply{Result: appsvc.ErrorResult(ctx, err)})
		}
		return ctx.Result(200, out)
	})
}
