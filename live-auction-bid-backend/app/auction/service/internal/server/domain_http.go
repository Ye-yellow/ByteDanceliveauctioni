package server

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	httptransport "github.com/go-kratos/kratos/v2/transport/http"
	v1 "live-auction-bid/backend/api/auction/service/v1"
	"live-auction-bid/backend/app/auction/service/internal/biz/auction"
	userbiz "live-auction-bid/backend/app/auction/service/internal/biz/user"
	"live-auction-bid/backend/app/auction/service/internal/pkg/apperr"
	appsvc "live-auction-bid/backend/app/auction/service/internal/service"
)

type lotResultHTTPResponse struct {
	Result       *v1.ReplyResult       `json:"result"`
	Lot          *v1.Lot               `json:"lot,omitempty"`
	AuctionState auction.AuctionState  `json:"auctionState,omitempty"`
	Order        *auction.OrderSummary `json:"order,omitempty"`
}

type listOrdersHTTPResponse struct {
	Result *v1.ReplyResult        `json:"result"`
	Orders []auction.OrderSummary `json:"orders"`
	Total  int64                  `json:"total"`
	Page   int                    `json:"page"`
	Size   int                    `json:"pageSize"`
}

type listBidRecordsHTTPResponse struct {
	Result *v1.ReplyResult     `json:"result"`
	Bids   []auction.BidRecord `json:"bids"`
	Total  int64               `json:"total"`
	Page   int                 `json:"page"`
	Size   int                 `json:"pageSize"`
}

type listLotsHTTPResponse struct {
	Result *v1.ReplyResult `json:"result"`
	Lots   []*v1.Lot       `json:"lots"`
	Total  int64           `json:"total"`
	Page   int             `json:"page"`
	Size   int             `json:"pageSize"`
}

type listUsersHTTPResponse struct {
	Result *v1.ReplyResult `json:"result"`
	Users  []*v1.User      `json:"users"`
	Total  int64           `json:"total"`
	Page   int             `json:"page"`
	Size   int             `json:"pageSize"`
}

type mockPayHTTPResponse struct {
	Result  *v1.ReplyResult         `json:"result"`
	Order   *auction.OrderSummary   `json:"order,omitempty"`
	Payment *auction.PaymentSummary `json:"payment,omitempty"`
	Paid    bool                    `json:"paid"`
}

func registerDomainHTTP(srv *httptransport.Server, service *appsvc.AuctionService, users *appsvc.UserService) {
	r := srv.Route("/")
	r.GET("/api/lots/{lot_id}/result", func(ctx httptransport.Context) error {
		lotID := ctx.Vars().Get("lot_id")
		h := ctx.Middleware(func(ctx context.Context, req any) (any, error) {
			result, err := service.GetLotResult(ctx, lotID)
			if err != nil {
				return lotResultHTTPResponse{Result: appsvc.ErrorResult(ctx, err)}, nil
			}
			return lotResultHTTPResponse{Result: appsvc.OKResult(ctx), Lot: result.Lot, AuctionState: result.AuctionState, Order: result.Order}, nil
		})
		out, err := h(ctx, nil)
		if err != nil {
			return ctx.Result(200, lotResultHTTPResponse{Result: appsvc.ErrorResult(ctx, err)})
		}
		return ctx.Result(200, out)
	})
	r.GET("/api/me/orders", func(ctx httptransport.Context) error {
		query := orderQueryFromHTTP(ctx)
		h := ctx.Middleware(func(ctx context.Context, req any) (any, error) {
			orders, err := service.ListMyOrdersPage(ctx, query)
			if err != nil {
				return listOrdersHTTPResponse{Result: appsvc.ErrorResult(ctx, err), Orders: []auction.OrderSummary{}}, nil
			}
			return listOrdersHTTPResponse{Result: appsvc.OKResult(ctx), Orders: orders.Orders, Total: orders.Total, Page: orders.Page, Size: orders.PageSize}, nil
		})
		out, err := h(ctx, nil)
		if err != nil {
			return ctx.Result(200, listOrdersHTTPResponse{Result: appsvc.ErrorResult(ctx, err), Orders: []auction.OrderSummary{}})
		}
		return ctx.Result(200, out)
	})
	r.GET("/api/me/bids", func(ctx httptransport.Context) error {
		query := bidRecordQueryFromHTTP(ctx)
		h := ctx.Middleware(func(ctx context.Context, req any) (any, error) {
			bids, err := service.ListMyBids(ctx, query)
			if err != nil {
				return listBidRecordsHTTPResponse{Result: appsvc.ErrorResult(ctx, err), Bids: []auction.BidRecord{}}, nil
			}
			return listBidRecordsHTTPResponse{Result: appsvc.OKResult(ctx), Bids: bids.Bids, Total: bids.Total, Page: bids.Page, Size: bids.PageSize}, nil
		})
		out, err := h(ctx, nil)
		if err != nil {
			return ctx.Result(200, listBidRecordsHTTPResponse{Result: appsvc.ErrorResult(ctx, err), Bids: []auction.BidRecord{}})
		}
		return ctx.Result(200, out)
	})
	r.GET("/api/admin/orders", func(ctx httptransport.Context) error {
		query := orderQueryFromHTTP(ctx)
		h := ctx.Middleware(func(ctx context.Context, req any) (any, error) {
			orders, err := service.ListOrders(ctx, query)
			if err != nil {
				return listOrdersHTTPResponse{Result: appsvc.ErrorResult(ctx, err), Orders: []auction.OrderSummary{}}, nil
			}
			return listOrdersHTTPResponse{Result: appsvc.OKResult(ctx), Orders: orders.Orders, Total: orders.Total, Page: orders.Page, Size: orders.PageSize}, nil
		})
		out, err := h(ctx, nil)
		if err != nil {
			return ctx.Result(200, listOrdersHTTPResponse{Result: appsvc.ErrorResult(ctx, err), Orders: []auction.OrderSummary{}})
		}
		return ctx.Result(200, out)
	})
	r.GET("/api/admin/lots", func(ctx httptransport.Context) error {
		query := lotQueryFromHTTP(ctx)
		h := ctx.Middleware(func(ctx context.Context, req any) (any, error) {
			lots, err := service.ListAdminLots(ctx, query)
			if err != nil {
				return listLotsHTTPResponse{Result: appsvc.ErrorResult(ctx, err), Lots: []*v1.Lot{}}, nil
			}
			return listLotsHTTPResponse{Result: appsvc.OKResult(ctx), Lots: lots.Lots, Total: lots.Total, Page: lots.Page, Size: lots.PageSize}, nil
		})
		out, err := h(ctx, nil)
		if err != nil {
			return ctx.Result(200, listLotsHTTPResponse{Result: appsvc.ErrorResult(ctx, err), Lots: []*v1.Lot{}})
		}
		return ctx.Result(200, out)
	})
	r.GET("/api/admin/users", func(ctx httptransport.Context) error {
		query := userQueryFromHTTP(ctx)
		h := ctx.Middleware(func(ctx context.Context, req any) (any, error) {
			list, err := users.ListUsers(ctx, query)
			if err != nil {
				return listUsersHTTPResponse{Result: appsvc.ErrorResult(ctx, err), Users: []*v1.User{}}, nil
			}
			return listUsersHTTPResponse{Result: appsvc.OKResult(ctx), Users: list.Users, Total: list.Total, Page: list.Page, Size: list.PageSize}, nil
		})
		out, err := h(ctx, nil)
		if err != nil {
			return ctx.Result(200, listUsersHTTPResponse{Result: appsvc.ErrorResult(ctx, err), Users: []*v1.User{}})
		}
		return ctx.Result(200, out)
	})
	r.POST("/api/orders/{order_id}/mock-pay", func(ctx httptransport.Context) error {
		orderID := ctx.Vars().Get("order_id")
		var req auction.MockPayRequest
		if err := ctx.Bind(&req); err != nil {
			return ctx.Result(200, mockPayHTTPResponse{Result: appsvc.ErrorResult(ctx, fmt.Errorf("%w: invalid request body", apperr.ErrInvalidArgument))})
		}
		h := ctx.Middleware(func(ctx context.Context, raw any) (any, error) {
			result, err := service.MockPayOrder(ctx, orderID, req)
			if err != nil {
				return mockPayHTTPResponse{Result: appsvc.ErrorResult(ctx, err)}, nil
			}
			return mockPayHTTPResponse{Result: appsvc.OKResult(ctx), Order: &result.Order, Payment: &result.Payment, Paid: result.Paid}, nil
		})
		out, err := h(ctx, &req)
		if err != nil {
			return ctx.Result(200, mockPayHTTPResponse{Result: appsvc.ErrorResult(ctx, err)})
		}
		return ctx.Result(200, out)
	})
}

func orderQueryFromHTTP(ctx httptransport.Context) auction.OrderQuery {
	query := ctx.Query()
	return auction.OrderQuery{
		Page:          intQuery(query.Get("page")),
		PageSize:      intQuery(query.Get("pageSize")),
		Status:        auction.OrderStatus(strings.TrimSpace(query.Get("status"))),
		PaymentStatus: auction.PaymentStatus(strings.TrimSpace(query.Get("paymentStatus"))),
		LotID:         strings.TrimSpace(query.Get("lotId")),
		Buyer:         strings.TrimSpace(query.Get("buyer")),
	}
}

func bidRecordQueryFromHTTP(ctx httptransport.Context) auction.BidRecordQuery {
	query := ctx.Query()
	return auction.BidRecordQuery{
		Page:     intQuery(query.Get("page")),
		PageSize: intQuery(query.Get("pageSize")),
		LotID:    strings.TrimSpace(query.Get("lotId")),
	}
}

func lotQueryFromHTTP(ctx httptransport.Context) auction.LotQuery {
	query := ctx.Query()
	return auction.LotQuery{
		Page:     intQuery(query.Get("page")),
		PageSize: intQuery(query.Get("pageSize")),
		Status:   lotStatusFromString(query.Get("status")),
		View:     strings.TrimSpace(query.Get("view")),
		Keyword:  strings.TrimSpace(query.Get("keyword")),
		RoomID:   strings.TrimSpace(query.Get("roomId")),
	}
}

func userQueryFromHTTP(ctx httptransport.Context) userbiz.ListUsersQuery {
	query := ctx.Query()
	return userbiz.ListUsersQuery{
		Page:     intQuery(query.Get("page")),
		PageSize: intQuery(query.Get("pageSize")),
		Role:     userRoleFromString(query.Get("role")),
		Keyword:  strings.TrimSpace(query.Get("keyword")),
	}
}

func intQuery(value string) int {
	next, _ := strconv.Atoi(strings.TrimSpace(value))
	return next
}

func lotStatusFromString(value string) v1.LotStatus {
	value = strings.TrimSpace(value)
	if value == "" {
		return v1.LotStatus_LOT_STATUS_UNSPECIFIED
	}
	if numeric, err := strconv.Atoi(value); err == nil {
		return v1.LotStatus(numeric)
	}
	key := strings.ToUpper(value)
	if !strings.HasPrefix(key, "LOT_STATUS_") {
		key = "LOT_STATUS_" + key
	}
	switch key {
	case "LOT_STATUS_SCHEDULED":
		return v1.LotStatus_LOT_STATUS_SCHEDULED
	case "LOT_STATUS_EXTENDED":
		return v1.LotStatus_LOT_STATUS_EXTENDED
	case "LOT_STATUS_SOLD":
		return v1.LotStatus_LOT_STATUS_SOLD
	case "LOT_STATUS_FAILED":
		return v1.LotStatus_LOT_STATUS_FAILED
	}
	return v1.LotStatus(v1.LotStatus_value[key])
}

func userRoleFromString(value string) v1.UserRole {
	value = strings.TrimSpace(value)
	if value == "" {
		return v1.UserRole_USER_ROLE_UNSPECIFIED
	}
	if numeric, err := strconv.Atoi(value); err == nil {
		return v1.UserRole(numeric)
	}
	key := strings.ToUpper(value)
	if !strings.HasPrefix(key, "USER_ROLE_") {
		key = "USER_ROLE_" + key
	}
	return v1.UserRole(v1.UserRole_value[key])
}
