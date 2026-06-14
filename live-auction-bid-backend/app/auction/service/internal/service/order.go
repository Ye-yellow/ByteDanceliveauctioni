package service

import (
	"context"
	"fmt"
	"strings"

	"live-auction-bid/backend/app/auction/service/internal/biz/auction"
	"live-auction-bid/backend/app/auction/service/internal/biz/shop"
	userbiz "live-auction-bid/backend/app/auction/service/internal/biz/user"
	"live-auction-bid/backend/app/auction/service/internal/pkg/apperr"
	"live-auction-bid/backend/app/auction/service/internal/pkg/auth"
)

type OrderService struct {
	shop    *shop.Usecase
	auction *auction.AuctionUsecase
}

func NewOrderService(shopUsecase *shop.Usecase, auctionUsecase *auction.AuctionUsecase) *OrderService {
	return &OrderService{shop: shopUsecase, auction: auctionUsecase}
}

func (s *OrderService) ListMyOrders(ctx context.Context, query shop.OrderQuery) (shop.UserOrderList, error) {
	claims, err := auth.RequirePermission(ctx, userbiz.PermissionOrderViewOwn)
	if err != nil {
		return shop.UserOrderList{}, err
	}
	return s.shop.ListUserOrders(ctx, claims.UserID, query)
}

func (s *OrderService) ListMyFrequentStores(ctx context.Context, limit int) (shop.FrequentStoreList, error) {
	claims, err := auth.RequirePermission(ctx, userbiz.PermissionOrderViewOwn)
	if err != nil {
		return shop.FrequentStoreList{}, err
	}
	return s.shop.ListFrequentStores(ctx, claims.UserID, limit)
}

func (s *OrderService) GetMyOrder(ctx context.Context, orderID string) (*shop.UserOrder, error) {
	claims, err := auth.RequirePermission(ctx, userbiz.PermissionOrderViewOwn)
	if err != nil {
		return nil, err
	}
	return s.shop.FindUserOrder(ctx, claims.UserID, orderID)
}

func (s *OrderService) MockPayOrder(ctx context.Context, orderID string, req auction.MockPayRequest) (*shop.UserOrderMockPayResult, error) {
	claims, err := auth.RequirePermission(ctx, userbiz.PermissionOrderPay)
	if err != nil {
		return nil, err
	}
	order, err := s.shop.FindUserOrder(ctx, claims.UserID, orderID)
	if err != nil {
		return nil, err
	}
	switch order.Source {
	case shop.OrderSourceAuction:
		return s.mockPayAuctionOrder(ctx, claims.UserID, *order, req)
	case shop.OrderSourceShop:
		return s.mockPayShopOrder(ctx, claims.UserID, *order, req)
	default:
		return nil, fmt.Errorf("%w: unsupported order source %s", apperr.ErrInvalidArgument, order.Source)
	}
}

func (s *OrderService) mockPayAuctionOrder(ctx context.Context, userID string, order shop.UserOrder, req auction.MockPayRequest) (*shop.UserOrderMockPayResult, error) {
	if s.auction == nil {
		return nil, fmt.Errorf("%w: auction usecase is required", apperr.ErrInvalidArgument)
	}
	if req.Amount == 0 {
		req.Amount = order.TotalAmount
	}
	if strings.TrimSpace(req.Currency) == "" {
		req.Currency = order.Currency
	}
	result, err := s.auction.MockPayOrder(ctx, userID, order.ID, req)
	if err != nil {
		return nil, err
	}
	refreshed, err := s.shop.FindUserOrder(ctx, userID, order.ID)
	if err != nil {
		return nil, err
	}
	return &shop.UserOrderMockPayResult{
		Order: *refreshed,
		Payment: shop.UserOrderPayment{
			ID:              result.Payment.ID,
			OrderID:         result.Payment.OrderID,
			Source:          shop.OrderSourceAuction,
			Provider:        "mock",
			UserID:          userID,
			Status:          shop.PaymentStatus(strings.ToLower(string(result.Payment.Status))),
			Amount:          result.Payment.Amount,
			Currency:        result.Payment.Currency,
			CreatedAtUnixMs: result.Payment.CreatedAtUnixMs,
			SucceededAtMs:   result.Payment.SucceededAtMs,
		},
		Paid: result.Paid,
	}, nil
}

func (s *OrderService) mockPayShopOrder(ctx context.Context, userID string, order shop.UserOrder, req auction.MockPayRequest) (*shop.UserOrderMockPayResult, error) {
	if s.shop == nil {
		return nil, fmt.Errorf("%w: shop usecase is required", apperr.ErrInvalidArgument)
	}
	result, err := s.shop.MockPayOrder(ctx, userID, order.ID, shop.MockPayRequest{IdempotencyKey: strings.TrimSpace(req.IdempotencyKey)})
	if err != nil {
		return nil, err
	}
	refreshed, err := s.shop.FindUserOrder(ctx, userID, order.ID)
	if err != nil {
		return nil, err
	}
	return &shop.UserOrderMockPayResult{
		Order: *refreshed,
		Payment: shop.UserOrderPayment{
			ID:              result.Payment.ID,
			OrderID:         result.Payment.OrderID,
			Source:          shop.OrderSourceShop,
			Provider:        "mock",
			UserID:          result.Payment.UserID,
			Status:          result.Payment.Status,
			Amount:          result.Payment.Amount,
			Currency:        result.Payment.Currency,
			IdempotencyKey:  result.Payment.IdempotencyKey,
			CreatedAtUnixMs: result.Payment.CreatedAtUnixMs,
			SucceededAtMs:   result.Payment.SucceededAtMs,
		},
		Paid: result.Paid,
	}, nil
}
