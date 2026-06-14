package server

import (
	"context"
	"fmt"
	"strings"

	httptransport "github.com/go-kratos/kratos/v2/transport/http"
	"live-auction-bid/backend/app/auction/service/internal/biz/shop"
	"live-auction-bid/backend/app/auction/service/internal/pkg/apperr"
	appsvc "live-auction-bid/backend/app/auction/service/internal/service"
)

type shopProductsHTTPResponse struct {
	Result   any            `json:"result"`
	Products []shop.Product `json:"products"`
	Total    int64          `json:"total"`
	Page     int            `json:"page"`
	Size     int            `json:"pageSize"`
}

type shopProductHTTPResponse struct {
	Result  any           `json:"result"`
	Product *shop.Product `json:"product,omitempty"`
}

type shopAddressesHTTPResponse struct {
	Result    any                    `json:"result"`
	Addresses []shop.DeliveryAddress `json:"addresses"`
}

type shopAddressHTTPResponse struct {
	Result  any                   `json:"result"`
	Address *shop.DeliveryAddress `json:"address,omitempty"`
}

type shopOrdersHTTPResponse struct {
	Result any          `json:"result"`
	Orders []shop.Order `json:"orders"`
	Total  int64        `json:"total"`
	Page   int          `json:"page"`
	Size   int          `json:"pageSize"`
}

type shopOrderHTTPResponse struct {
	Result  any           `json:"result"`
	Order   *shop.Order   `json:"order,omitempty"`
	Payment *shop.Payment `json:"payment,omitempty"`
	Paid    bool          `json:"paid,omitempty"`
}

func registerShopHTTP(srv *httptransport.Server, service *appsvc.ShopService) {
	r := srv.Route("/")
	r.GET("/api/shop/products", func(ctx httptransport.Context) error {
		query := shopProductQueryFromHTTP(ctx)
		list, err := service.ListProducts(ctx, query)
		if err != nil {
			return ctx.Result(200, shopProductsHTTPResponse{Result: appsvc.ErrorResult(ctx, err), Products: []shop.Product{}})
		}
		return ctx.Result(200, shopProductsHTTPResponse{Result: appsvc.OKResult(ctx), Products: list.Products, Total: list.Total, Page: list.Page, Size: list.PageSize})
	})
	r.GET("/api/shop/products/{product_id}", func(ctx httptransport.Context) error {
		product, err := service.GetProduct(ctx, ctx.Vars().Get("product_id"))
		if err != nil {
			return ctx.Result(200, shopProductHTTPResponse{Result: appsvc.ErrorResult(ctx, err)})
		}
		return ctx.Result(200, shopProductHTTPResponse{Result: appsvc.OKResult(ctx), Product: product})
	})
	r.GET("/api/shop/addresses", func(ctx httptransport.Context) error {
		h := ctx.Middleware(func(ctx context.Context, raw any) (any, error) {
			addresses, err := service.ListDeliveryAddresses(ctx)
			if err != nil {
				return shopAddressesHTTPResponse{Result: appsvc.ErrorResult(ctx, err), Addresses: []shop.DeliveryAddress{}}, nil
			}
			return shopAddressesHTTPResponse{Result: appsvc.OKResult(ctx), Addresses: addresses}, nil
		})
		out, err := h(ctx, nil)
		if err != nil {
			return ctx.Result(200, shopAddressesHTTPResponse{Result: appsvc.ErrorResult(ctx, err), Addresses: []shop.DeliveryAddress{}})
		}
		return ctx.Result(200, out)
	})
	r.POST("/api/shop/addresses", func(ctx httptransport.Context) error {
		var req shop.DeliveryAddressInput
		if err := ctx.Bind(&req); err != nil {
			return ctx.Result(200, shopAddressHTTPResponse{Result: appsvc.ErrorResult(ctx, fmt.Errorf("%w: invalid request body", apperr.ErrInvalidArgument))})
		}
		h := ctx.Middleware(func(ctx context.Context, raw any) (any, error) {
			address, err := service.CreateDeliveryAddress(ctx, req)
			if err != nil {
				return shopAddressHTTPResponse{Result: appsvc.ErrorResult(ctx, err)}, nil
			}
			return shopAddressHTTPResponse{Result: appsvc.OKResult(ctx), Address: address}, nil
		})
		out, err := h(ctx, &req)
		if err != nil {
			return ctx.Result(200, shopAddressHTTPResponse{Result: appsvc.ErrorResult(ctx, err)})
		}
		return ctx.Result(200, out)
	})
	r.PUT("/api/shop/addresses/{address_id}", func(ctx httptransport.Context) error {
		var req shop.DeliveryAddressInput
		if err := ctx.Bind(&req); err != nil {
			return ctx.Result(200, shopAddressHTTPResponse{Result: appsvc.ErrorResult(ctx, fmt.Errorf("%w: invalid request body", apperr.ErrInvalidArgument))})
		}
		addressID := ctx.Vars().Get("address_id")
		h := ctx.Middleware(func(ctx context.Context, raw any) (any, error) {
			address, err := service.UpdateDeliveryAddress(ctx, addressID, req)
			if err != nil {
				return shopAddressHTTPResponse{Result: appsvc.ErrorResult(ctx, err)}, nil
			}
			return shopAddressHTTPResponse{Result: appsvc.OKResult(ctx), Address: address}, nil
		})
		out, err := h(ctx, &req)
		if err != nil {
			return ctx.Result(200, shopAddressHTTPResponse{Result: appsvc.ErrorResult(ctx, err)})
		}
		return ctx.Result(200, out)
	})
	r.DELETE("/api/shop/addresses/{address_id}", func(ctx httptransport.Context) error {
		addressID := ctx.Vars().Get("address_id")
		h := ctx.Middleware(func(ctx context.Context, raw any) (any, error) {
			if err := service.DeleteDeliveryAddress(ctx, addressID); err != nil {
				return shopAddressHTTPResponse{Result: appsvc.ErrorResult(ctx, err)}, nil
			}
			return shopAddressHTTPResponse{Result: appsvc.OKResult(ctx)}, nil
		})
		out, err := h(ctx, nil)
		if err != nil {
			return ctx.Result(200, shopAddressHTTPResponse{Result: appsvc.ErrorResult(ctx, err)})
		}
		return ctx.Result(200, out)
	})
	r.POST("/api/shop/addresses/{address_id}/default", func(ctx httptransport.Context) error {
		addressID := ctx.Vars().Get("address_id")
		h := ctx.Middleware(func(ctx context.Context, raw any) (any, error) {
			addresses, err := service.SetDefaultDeliveryAddress(ctx, addressID)
			if err != nil {
				return shopAddressesHTTPResponse{Result: appsvc.ErrorResult(ctx, err), Addresses: []shop.DeliveryAddress{}}, nil
			}
			return shopAddressesHTTPResponse{Result: appsvc.OKResult(ctx), Addresses: addresses}, nil
		})
		out, err := h(ctx, nil)
		if err != nil {
			return ctx.Result(200, shopAddressesHTTPResponse{Result: appsvc.ErrorResult(ctx, err), Addresses: []shop.DeliveryAddress{}})
		}
		return ctx.Result(200, out)
	})
	r.POST("/api/shop/orders", func(ctx httptransport.Context) error {
		var req shop.CreateOrderRequest
		if err := ctx.Bind(&req); err != nil {
			return ctx.Result(200, shopOrderHTTPResponse{Result: appsvc.ErrorResult(ctx, fmt.Errorf("%w: invalid request body", apperr.ErrInvalidArgument))})
		}
		h := ctx.Middleware(func(ctx context.Context, raw any) (any, error) {
			order, err := service.CreateOrder(ctx, req)
			if err != nil {
				return shopOrderHTTPResponse{Result: appsvc.ErrorResult(ctx, err)}, nil
			}
			return shopOrderHTTPResponse{Result: appsvc.OKResult(ctx), Order: order}, nil
		})
		out, err := h(ctx, &req)
		if err != nil {
			return ctx.Result(200, shopOrderHTTPResponse{Result: appsvc.ErrorResult(ctx, err)})
		}
		return ctx.Result(200, out)
	})
	r.GET("/api/shop/orders", func(ctx httptransport.Context) error {
		query := shopOrderQueryFromHTTP(ctx)
		h := ctx.Middleware(func(ctx context.Context, raw any) (any, error) {
			list, err := service.ListMyOrders(ctx, query)
			if err != nil {
				return shopOrdersHTTPResponse{Result: appsvc.ErrorResult(ctx, err), Orders: []shop.Order{}}, nil
			}
			return shopOrdersHTTPResponse{Result: appsvc.OKResult(ctx), Orders: list.Orders, Total: list.Total, Page: list.Page, Size: list.PageSize}, nil
		})
		out, err := h(ctx, nil)
		if err != nil {
			return ctx.Result(200, shopOrdersHTTPResponse{Result: appsvc.ErrorResult(ctx, err), Orders: []shop.Order{}})
		}
		return ctx.Result(200, out)
	})
	r.POST("/api/shop/orders/{order_id}/mock-pay", func(ctx httptransport.Context) error {
		var req shop.MockPayRequest
		if err := ctx.Bind(&req); err != nil {
			return ctx.Result(200, shopOrderHTTPResponse{Result: appsvc.ErrorResult(ctx, fmt.Errorf("%w: invalid request body", apperr.ErrInvalidArgument))})
		}
		orderID := ctx.Vars().Get("order_id")
		h := ctx.Middleware(func(ctx context.Context, raw any) (any, error) {
			result, err := service.MockPayOrder(ctx, orderID, req)
			if err != nil {
				return shopOrderHTTPResponse{Result: appsvc.ErrorResult(ctx, err)}, nil
			}
			return shopOrderHTTPResponse{Result: appsvc.OKResult(ctx), Order: &result.Order, Payment: &result.Payment, Paid: result.Paid}, nil
		})
		out, err := h(ctx, &req)
		if err != nil {
			return ctx.Result(200, shopOrderHTTPResponse{Result: appsvc.ErrorResult(ctx, err)})
		}
		return ctx.Result(200, out)
	})
}

func shopProductQueryFromHTTP(ctx httptransport.Context) shop.ProductQuery {
	query := ctx.Query()
	return shop.ProductQuery{
		Query:    strings.TrimSpace(query.Get("q")),
		Category: strings.TrimSpace(query.Get("category")),
		Page:     intQuery(query.Get("page")),
		PageSize: intQuery(query.Get("pageSize")),
	}
}

func shopOrderQueryFromHTTP(ctx httptransport.Context) shop.OrderQuery {
	query := ctx.Query()
	return shop.OrderQuery{
		Status:   shop.OrderStatus(strings.TrimSpace(query.Get("status"))),
		Query:    strings.TrimSpace(query.Get("q")),
		Page:     intQuery(query.Get("page")),
		PageSize: intQuery(query.Get("pageSize")),
	}
}
