package server

import (
	"context"
	"fmt"
	"strings"

	httptransport "github.com/go-kratos/kratos/v2/transport/http"
	v1 "live-auction-bid/backend/api/auction/service/v1"
	"live-auction-bid/backend/app/auction/service/internal/biz/auction"
	"live-auction-bid/backend/app/auction/service/internal/biz/shop"
	"live-auction-bid/backend/app/auction/service/internal/pkg/apperr"
	appsvc "live-auction-bid/backend/app/auction/service/internal/service"
)

type userOrdersHTTPResponse struct {
	Result *v1.ReplyResult  `json:"result"`
	Orders []shop.UserOrder `json:"orders"`
	Total  int64            `json:"total"`
	Page   int              `json:"page"`
	Size   int              `json:"pageSize"`
}

type frequentStoresHTTPResponse struct {
	Result *v1.ReplyResult      `json:"result"`
	Stores []shop.FrequentStore `json:"stores"`
	Total  int64                `json:"total"`
	Limit  int                  `json:"limit"`
}

type userOrderHTTPResponse struct {
	Result  *v1.ReplyResult        `json:"result"`
	Order   *shop.UserOrder        `json:"order,omitempty"`
	Payment *shop.UserOrderPayment `json:"payment,omitempty"`
	Paid    bool                   `json:"paid,omitempty"`
}

func registerOrderHTTP(srv *httptransport.Server, service *appsvc.OrderService) {
	r := srv.Route("/")
	r.GET("/api/orders/me", func(ctx httptransport.Context) error {
		query := userOrderQueryFromHTTP(ctx)
		h := ctx.Middleware(func(ctx context.Context, raw any) (any, error) {
			list, err := service.ListMyOrders(ctx, query)
			if err != nil {
				return userOrdersHTTPResponse{Result: appsvc.ErrorResult(ctx, err), Orders: []shop.UserOrder{}}, nil
			}
			return userOrdersHTTPResponse{Result: appsvc.OKResult(ctx), Orders: list.Orders, Total: list.Total, Page: list.Page, Size: list.PageSize}, nil
		})
		out, err := h(ctx, nil)
		if err != nil {
			return ctx.Result(200, userOrdersHTTPResponse{Result: appsvc.ErrorResult(ctx, err), Orders: []shop.UserOrder{}})
		}
		return ctx.Result(200, out)
	})
	r.GET("/api/orders/me/frequent-stores", func(ctx httptransport.Context) error {
		limit := intQuery(ctx.Query().Get("limit"))
		h := ctx.Middleware(func(ctx context.Context, raw any) (any, error) {
			list, err := service.ListMyFrequentStores(ctx, limit)
			if err != nil {
				return frequentStoresHTTPResponse{Result: appsvc.ErrorResult(ctx, err), Stores: []shop.FrequentStore{}}, nil
			}
			return frequentStoresHTTPResponse{Result: appsvc.OKResult(ctx), Stores: list.Stores, Total: list.Total, Limit: list.Limit}, nil
		})
		out, err := h(ctx, nil)
		if err != nil {
			return ctx.Result(200, frequentStoresHTTPResponse{Result: appsvc.ErrorResult(ctx, err), Stores: []shop.FrequentStore{}})
		}
		return ctx.Result(200, out)
	})
	r.GET("/api/orders/{order_id}", func(ctx httptransport.Context) error {
		orderID := ctx.Vars().Get("order_id")
		h := ctx.Middleware(func(ctx context.Context, raw any) (any, error) {
			order, err := service.GetMyOrder(ctx, orderID)
			if err != nil {
				return userOrderHTTPResponse{Result: appsvc.ErrorResult(ctx, err)}, nil
			}
			return userOrderHTTPResponse{Result: appsvc.OKResult(ctx), Order: order}, nil
		})
		out, err := h(ctx, nil)
		if err != nil {
			return ctx.Result(200, userOrderHTTPResponse{Result: appsvc.ErrorResult(ctx, err)})
		}
		return ctx.Result(200, out)
	})
	r.POST("/api/orders/{order_id}/mock-pay", func(ctx httptransport.Context) error {
		orderID := ctx.Vars().Get("order_id")
		var req auction.MockPayRequest
		if err := ctx.Bind(&req); err != nil {
			return ctx.Result(200, userOrderHTTPResponse{Result: appsvc.ErrorResult(ctx, fmt.Errorf("%w: invalid request body", apperr.ErrInvalidArgument))})
		}
		h := ctx.Middleware(func(ctx context.Context, raw any) (any, error) {
			result, err := service.MockPayOrder(ctx, orderID, req)
			if err != nil {
				return userOrderHTTPResponse{Result: appsvc.ErrorResult(ctx, err)}, nil
			}
			return userOrderHTTPResponse{Result: appsvc.OKResult(ctx), Order: &result.Order, Payment: &result.Payment, Paid: result.Paid}, nil
		})
		out, err := h(ctx, &req)
		if err != nil {
			return ctx.Result(200, userOrderHTTPResponse{Result: appsvc.ErrorResult(ctx, err)})
		}
		return ctx.Result(200, out)
	})
}

func userOrderQueryFromHTTP(ctx httptransport.Context) shop.OrderQuery {
	query := ctx.Query()
	return shop.OrderQuery{
		Source:   shop.OrderSource(strings.TrimSpace(query.Get("source"))),
		Status:   shop.OrderStatus(strings.TrimSpace(query.Get("status"))),
		Query:    strings.TrimSpace(query.Get("q")),
		LotID:    firstNonEmpty(strings.TrimSpace(query.Get("lotId")), strings.TrimSpace(query.Get("lot_id"))),
		Page:     intQuery(query.Get("page")),
		PageSize: intQuery(query.Get("pageSize")),
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
