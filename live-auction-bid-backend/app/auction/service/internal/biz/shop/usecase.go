package shop

import (
	"context"
	"fmt"
	"strings"

	"live-auction-bid/backend/app/auction/service/internal/pkg/apperr"
)

type Usecase struct {
	repo Repository
}

func NewUsecase(repo Repository) *Usecase {
	return &Usecase{repo: repo}
}

func (uc *Usecase) ListProducts(ctx context.Context, query ProductQuery) (ProductList, error) {
	if uc == nil || uc.repo == nil {
		return ProductList{}, fmt.Errorf("%w: shop repository is required", apperr.ErrInvalidArgument)
	}
	query.Page, query.PageSize = NormalizePagination(query.Page, query.PageSize)
	query.Query = strings.TrimSpace(query.Query)
	query.Category = strings.TrimSpace(query.Category)
	return uc.repo.ListProducts(ctx, query)
}

func (uc *Usecase) GetProduct(ctx context.Context, productID string) (*Product, error) {
	if uc == nil || uc.repo == nil {
		return nil, fmt.Errorf("%w: shop repository is required", apperr.ErrInvalidArgument)
	}
	productID = strings.TrimSpace(productID)
	if productID == "" {
		return nil, fmt.Errorf("%w: product id is required", apperr.ErrInvalidArgument)
	}
	return uc.repo.FindProductByID(ctx, productID)
}

func (uc *Usecase) ListDeliveryAddresses(ctx context.Context, userID string) ([]DeliveryAddress, error) {
	if uc == nil || uc.repo == nil {
		return nil, fmt.Errorf("%w: shop repository is required", apperr.ErrInvalidArgument)
	}
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, apperr.ErrUnauthenticated
	}
	return uc.repo.ListDeliveryAddresses(ctx, userID)
}

func (uc *Usecase) CreateDeliveryAddress(ctx context.Context, userID string, input DeliveryAddressInput) (*DeliveryAddress, error) {
	if uc == nil || uc.repo == nil {
		return nil, fmt.Errorf("%w: shop repository is required", apperr.ErrInvalidArgument)
	}
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, apperr.ErrUnauthenticated
	}
	input = NormalizeDeliveryAddressInput(input)
	if err := ValidateDeliveryAddressInput(input); err != nil {
		return nil, err
	}
	return uc.repo.CreateDeliveryAddress(ctx, userID, input)
}

func (uc *Usecase) UpdateDeliveryAddress(ctx context.Context, userID, addressID string, input DeliveryAddressInput) (*DeliveryAddress, error) {
	if uc == nil || uc.repo == nil {
		return nil, fmt.Errorf("%w: shop repository is required", apperr.ErrInvalidArgument)
	}
	userID = strings.TrimSpace(userID)
	addressID = strings.TrimSpace(addressID)
	if userID == "" {
		return nil, apperr.ErrUnauthenticated
	}
	if addressID == "" {
		return nil, fmt.Errorf("%w: address id is required", apperr.ErrAddressRequired)
	}
	input = NormalizeDeliveryAddressInput(input)
	if err := ValidateDeliveryAddressInput(input); err != nil {
		return nil, err
	}
	return uc.repo.UpdateDeliveryAddress(ctx, userID, addressID, input)
}

func (uc *Usecase) DeleteDeliveryAddress(ctx context.Context, userID, addressID string) error {
	if uc == nil || uc.repo == nil {
		return fmt.Errorf("%w: shop repository is required", apperr.ErrInvalidArgument)
	}
	userID = strings.TrimSpace(userID)
	addressID = strings.TrimSpace(addressID)
	if userID == "" {
		return apperr.ErrUnauthenticated
	}
	if addressID == "" {
		return fmt.Errorf("%w: address id is required", apperr.ErrAddressRequired)
	}
	return uc.repo.DeleteDeliveryAddress(ctx, userID, addressID)
}

func (uc *Usecase) SetDefaultDeliveryAddress(ctx context.Context, userID, addressID string) ([]DeliveryAddress, error) {
	if uc == nil || uc.repo == nil {
		return nil, fmt.Errorf("%w: shop repository is required", apperr.ErrInvalidArgument)
	}
	userID = strings.TrimSpace(userID)
	addressID = strings.TrimSpace(addressID)
	if userID == "" {
		return nil, apperr.ErrUnauthenticated
	}
	if addressID == "" {
		return nil, fmt.Errorf("%w: address id is required", apperr.ErrAddressRequired)
	}
	return uc.repo.SetDefaultDeliveryAddress(ctx, userID, addressID)
}

func (uc *Usecase) CreateOrder(ctx context.Context, user UserRef, req CreateOrderRequest) (*Order, error) {
	if uc == nil || uc.repo == nil {
		return nil, fmt.Errorf("%w: shop repository is required", apperr.ErrInvalidArgument)
	}
	user.ID = strings.TrimSpace(user.ID)
	user.Nickname = strings.TrimSpace(user.Nickname)
	if user.ID == "" {
		return nil, apperr.ErrUnauthenticated
	}
	if err := ValidateCreateOrderRequest(req); err != nil {
		return nil, err
	}
	req.SKUID = strings.TrimSpace(req.SKUID)
	req.AddressID = strings.TrimSpace(req.AddressID)
	req.IdempotencyKey = strings.TrimSpace(req.IdempotencyKey)
	return uc.repo.CreateOrder(ctx, user, req)
}

func (uc *Usecase) ListMyOrders(ctx context.Context, userID string, query OrderQuery) (OrderList, error) {
	if uc == nil || uc.repo == nil {
		return OrderList{}, fmt.Errorf("%w: shop repository is required", apperr.ErrInvalidArgument)
	}
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return OrderList{}, apperr.ErrUnauthenticated
	}
	query.Page, query.PageSize = NormalizePagination(query.Page, query.PageSize)
	query.UserID = userID
	query.Query = strings.TrimSpace(query.Query)
	return uc.repo.ListShopOrders(ctx, query)
}

func (uc *Usecase) ListUserOrders(ctx context.Context, userID string, query OrderQuery) (UserOrderList, error) {
	if uc == nil || uc.repo == nil {
		return UserOrderList{}, fmt.Errorf("%w: shop repository is required", apperr.ErrInvalidArgument)
	}
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return UserOrderList{}, apperr.ErrUnauthenticated
	}
	query.Page, query.PageSize = NormalizePagination(query.Page, query.PageSize)
	query.UserID = userID
	query.Query = strings.TrimSpace(query.Query)
	query.LotID = strings.TrimSpace(query.LotID)
	return uc.repo.ListUserOrders(ctx, query)
}

func (uc *Usecase) ListFrequentStores(ctx context.Context, userID string, limit int) (FrequentStoreList, error) {
	if uc == nil || uc.repo == nil {
		return FrequentStoreList{}, fmt.Errorf("%w: shop repository is required", apperr.ErrInvalidArgument)
	}
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return FrequentStoreList{}, apperr.ErrUnauthenticated
	}
	return uc.repo.ListFrequentStores(ctx, FrequentStoreQuery{
		UserID: userID,
		Limit:  NormalizeFrequentStoreLimit(limit),
	})
}

func (uc *Usecase) FindUserOrder(ctx context.Context, userID, orderID string) (*UserOrder, error) {
	if uc == nil || uc.repo == nil {
		return nil, fmt.Errorf("%w: shop repository is required", apperr.ErrInvalidArgument)
	}
	userID = strings.TrimSpace(userID)
	orderID = strings.TrimSpace(orderID)
	if userID == "" {
		return nil, apperr.ErrUnauthenticated
	}
	if orderID == "" {
		return nil, fmt.Errorf("%w: order id is required", apperr.ErrInvalidArgument)
	}
	return uc.repo.FindUserOrder(ctx, userID, orderID)
}

func (uc *Usecase) MockPayOrder(ctx context.Context, userID, orderID string, req MockPayRequest) (*MockPayResult, error) {
	if uc == nil || uc.repo == nil {
		return nil, fmt.Errorf("%w: shop repository is required", apperr.ErrInvalidArgument)
	}
	userID = strings.TrimSpace(userID)
	orderID = strings.TrimSpace(orderID)
	req.IdempotencyKey = strings.TrimSpace(req.IdempotencyKey)
	if userID == "" {
		return nil, apperr.ErrUnauthenticated
	}
	if orderID == "" {
		return nil, fmt.Errorf("%w: order id is required", apperr.ErrInvalidArgument)
	}
	if req.IdempotencyKey == "" {
		req.IdempotencyKey = "shop-pay-" + orderID
	}
	return uc.repo.MockPayOrder(ctx, userID, orderID, req)
}
