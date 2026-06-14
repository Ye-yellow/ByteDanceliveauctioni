package service

import (
	"context"

	"live-auction-bid/backend/app/auction/service/internal/biz/shop"
	"live-auction-bid/backend/app/auction/service/internal/pkg/auth"
)

type ShopService struct {
	shop *shop.Usecase
}

func NewShopService(uc *shop.Usecase) *ShopService {
	return &ShopService{shop: uc}
}

func (s *ShopService) ListProducts(ctx context.Context, query shop.ProductQuery) (shop.ProductList, error) {
	return s.shop.ListProducts(ctx, query)
}

func (s *ShopService) GetProduct(ctx context.Context, productID string) (*shop.Product, error) {
	return s.shop.GetProduct(ctx, productID)
}

func (s *ShopService) ListDeliveryAddresses(ctx context.Context) ([]shop.DeliveryAddress, error) {
	claims, err := auth.RequireUser(ctx)
	if err != nil {
		return nil, err
	}
	return s.shop.ListDeliveryAddresses(ctx, claims.UserID)
}

func (s *ShopService) CreateDeliveryAddress(ctx context.Context, input shop.DeliveryAddressInput) (*shop.DeliveryAddress, error) {
	claims, err := auth.RequireUser(ctx)
	if err != nil {
		return nil, err
	}
	return s.shop.CreateDeliveryAddress(ctx, claims.UserID, input)
}

func (s *ShopService) UpdateDeliveryAddress(ctx context.Context, addressID string, input shop.DeliveryAddressInput) (*shop.DeliveryAddress, error) {
	claims, err := auth.RequireUser(ctx)
	if err != nil {
		return nil, err
	}
	return s.shop.UpdateDeliveryAddress(ctx, claims.UserID, addressID, input)
}

func (s *ShopService) DeleteDeliveryAddress(ctx context.Context, addressID string) error {
	claims, err := auth.RequireUser(ctx)
	if err != nil {
		return err
	}
	return s.shop.DeleteDeliveryAddress(ctx, claims.UserID, addressID)
}

func (s *ShopService) SetDefaultDeliveryAddress(ctx context.Context, addressID string) ([]shop.DeliveryAddress, error) {
	claims, err := auth.RequireUser(ctx)
	if err != nil {
		return nil, err
	}
	return s.shop.SetDefaultDeliveryAddress(ctx, claims.UserID, addressID)
}

func (s *ShopService) CreateOrder(ctx context.Context, req shop.CreateOrderRequest) (*shop.Order, error) {
	claims, err := auth.RequireUser(ctx)
	if err != nil {
		return nil, err
	}
	return s.shop.CreateOrder(ctx, shop.UserRef{ID: claims.UserID, Nickname: claims.Nickname}, req)
}

func (s *ShopService) ListMyOrders(ctx context.Context, query shop.OrderQuery) (shop.OrderList, error) {
	claims, err := auth.RequireUser(ctx)
	if err != nil {
		return shop.OrderList{}, err
	}
	return s.shop.ListMyOrders(ctx, claims.UserID, query)
}

func (s *ShopService) MockPayOrder(ctx context.Context, orderID string, req shop.MockPayRequest) (*shop.MockPayResult, error) {
	claims, err := auth.RequireUser(ctx)
	if err != nil {
		return nil, err
	}
	return s.shop.MockPayOrder(ctx, claims.UserID, orderID, req)
}
