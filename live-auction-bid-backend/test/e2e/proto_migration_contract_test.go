package e2e

import (
	"net/http"
	"testing"
)

func TestMigratedShopAndUnifiedOrderProtoRoutesContract(t *testing.T) {
	c := newClient(t)
	buyer := registerBuyer(t, c, "proto_shop_buyer")

	products, resp := c.get(t, "/api/shop/products?page=1&pageSize=3", "")
	assertHTTPStatus(t, resp, http.StatusOK)
	assertOK(t, products)
	productItems := arrayField(t, products, "products")
	if len(productItems) == 0 {
		t.Fatalf("expected seeded shop products, got %+v", products)
	}
	product := anyObject(t, productItems[0])
	productID := stringField(product, "id")
	if productID == "" {
		t.Fatalf("product missing id: %+v", product)
	}
	skus := arrayField(t, product, "skus")
	if len(skus) == 0 {
		t.Fatalf("product missing skus: %+v", product)
	}
	skuID := stringField(anyObject(t, skus[0]), "id")
	if skuID == "" {
		t.Fatalf("sku missing id: %+v", skus[0])
	}

	productDetail, resp := c.get(t, "/api/shop/products/"+productID, "")
	assertHTTPStatus(t, resp, http.StatusOK)
	assertOK(t, productDetail)
	if got := stringField(objectField(t, productDetail, "product"), "id"); got != productID {
		t.Fatalf("product detail id mismatch, want %q got %q body=%+v", productID, got, productDetail)
	}

	missingProduct, resp := c.get(t, "/api/shop/products/not-found-product", "")
	assertHTTPStatus(t, resp, http.StatusOK)
	assertResultCode(t, missingProduct, resultCodeUserNotFound)

	noTokenOrders, resp := c.get(t, "/api/shop/orders", "")
	assertHTTPStatus(t, resp, http.StatusOK)
	assertResultCode(t, noTokenOrders, resultCodeLoginRequired)

	addressID := createDeliveryAddress(t, c, buyer)
	addresses, resp := c.get(t, "/api/shop/addresses", buyer.AccessToken)
	assertHTTPStatus(t, resp, http.StatusOK)
	assertOK(t, addresses)
	if len(arrayField(t, addresses, "addresses")) == 0 {
		t.Fatalf("expected at least one address: %+v", addresses)
	}

	updatedAddress, resp := c.put(t, "/api/shop/addresses/"+addressID, buyer.AccessToken, map[string]any{
		"address": map[string]any{
			"receiverName": "E2E Buyer Updated",
			"phone":        "13800138001",
			"province":     "上海市",
			"city":         "上海市",
			"district":     "浦东新区",
			"street":       "世纪大道",
			"detail":       "E2E 测试地址 2 号",
			"tag":          "work",
			"isDefault":    true,
		},
	})
	assertHTTPStatus(t, resp, http.StatusOK)
	assertOK(t, updatedAddress)
	if got := stringField(objectField(t, updatedAddress, "address"), "receiverName", "receiver_name"); got != "E2E Buyer Updated" {
		t.Fatalf("expected updated receiver, got %q body=%+v", got, updatedAddress)
	}

	defaultAddress, resp := c.post(t, "/api/shop/addresses/"+addressID+"/default", buyer.AccessToken, map[string]any{})
	assertHTTPStatus(t, resp, http.StatusOK)
	assertOK(t, defaultAddress)

	invalidOrder, resp := c.post(t, "/api/shop/orders", buyer.AccessToken, map[string]any{
		"skuId":     skuID,
		"quantity":  0,
		"addressId": addressID,
	})
	assertHTTPStatus(t, resp, http.StatusOK)
	assertResultCode(t, invalidOrder, resultCodeInvalidArgument)

	shopOrder := createShopOrder(t, c, buyer, skuID, addressID)
	shopOrderID := stringField(shopOrder, "id")
	shopPay, resp := c.post(t, "/api/shop/orders/"+shopOrderID+"/mock-pay", buyer.AccessToken, map[string]any{
		"idempotencyKey": uniqueUsername("shop_pay"),
	})
	assertHTTPStatus(t, resp, http.StatusOK)
	assertOK(t, shopPay)
	if paid, _ := shopPay["paid"].(bool); !paid {
		t.Fatalf("expected shop mock pay to succeed: %+v", shopPay)
	}

	unifiedOrder := createShopOrder(t, c, buyer, skuID, addressID)
	unifiedOrderID := stringField(unifiedOrder, "id")
	unifiedList, resp := c.get(t, "/api/orders/me?source=shop&page=1&pageSize=10", buyer.AccessToken)
	assertHTTPStatus(t, resp, http.StatusOK)
	assertOK(t, unifiedList)
	if len(arrayField(t, unifiedList, "orders")) == 0 {
		t.Fatalf("expected unified orders to include shop order: %+v", unifiedList)
	}

	unifiedDetail, resp := c.get(t, "/api/orders/"+unifiedOrderID, buyer.AccessToken)
	assertHTTPStatus(t, resp, http.StatusOK)
	assertOK(t, unifiedDetail)
	if got := stringField(objectField(t, unifiedDetail, "order"), "id"); got != unifiedOrderID {
		t.Fatalf("unified order detail id mismatch, want %q got %q body=%+v", unifiedOrderID, got, unifiedDetail)
	}

	unifiedPay, resp := c.post(t, "/api/orders/"+unifiedOrderID+"/mock-pay", buyer.AccessToken, map[string]any{
		"idempotencyKey": uniqueUsername("unified_pay"),
	})
	assertHTTPStatus(t, resp, http.StatusOK)
	assertOK(t, unifiedPay)
	if paid, _ := unifiedPay["paid"].(bool); !paid {
		t.Fatalf("expected unified mock pay to succeed: %+v", unifiedPay)
	}

	stores, resp := c.get(t, "/api/orders/me/frequent-stores?limit=5", buyer.AccessToken)
	assertHTTPStatus(t, resp, http.StatusOK)
	assertOK(t, stores)
	_ = arrayField(t, stores, "stores")

	deleteAddressID := createDeliveryAddress(t, c, buyer)
	deleted, resp := c.delete(t, "/api/shop/addresses/"+deleteAddressID, buyer.AccessToken)
	assertHTTPStatus(t, resp, http.StatusOK)
	assertOK(t, deleted)
}

func TestMigratedAuctionQueryProtoRoutesContract(t *testing.T) {
	c := newClient(t)
	merchant := registerMerchant(t, c, "proto_query_merchant")
	buyer := registerBuyer(t, c, "proto_query_buyer")
	lotID, _ := createReadyDraftLot(t, c, merchant, "E2E Proto 查询迁移拍品")

	result, resp := c.get(t, "/api/lots/"+lotID+"/result", buyer.AccessToken)
	assertHTTPStatus(t, resp, http.StatusOK)
	assertOK(t, result)
	if got := stringField(objectField(t, result, "lot"), "id"); got != lotID {
		t.Fatalf("lot result id mismatch, want %q got %q body=%+v", lotID, got, result)
	}
	if state := stringField(result, "auctionState", "auction_state"); state == "" {
		t.Fatalf("lot result missing auction state: %+v", result)
	}

	myOrders, resp := c.get(t, "/api/me/orders?page=1&pageSize=5", buyer.AccessToken)
	assertHTTPStatus(t, resp, http.StatusOK)
	assertOK(t, myOrders)
	_ = arrayField(t, myOrders, "orders")

	myBids, resp := c.get(t, "/api/me/bids?page=1&pageSize=5", buyer.AccessToken)
	assertHTTPStatus(t, resp, http.StatusOK)
	assertOK(t, myBids)
	_ = arrayField(t, myBids, "bids")

	noTokenAdminRooms, resp := c.get(t, "/api/admin/rooms", "")
	assertHTTPStatus(t, resp, http.StatusOK)
	assertResultCode(t, noTokenAdminRooms, resultCodeLoginRequired)

	buyerAdminLots, resp := c.get(t, "/api/admin/lots", buyer.AccessToken)
	assertHTTPStatus(t, resp, http.StatusOK)
	assertResultCode(t, buyerAdminLots, resultCodeForbidden)

	adminLots, resp := c.get(t, "/api/admin/lots?page=1&pageSize=10", merchant.AccessToken)
	assertHTTPStatus(t, resp, http.StatusOK)
	assertOK(t, adminLots)
	if len(arrayField(t, adminLots, "lots")) == 0 {
		t.Fatalf("expected admin lots to include created lot: %+v", adminLots)
	}

	adminOrders, resp := c.get(t, "/api/admin/orders?page=1&pageSize=10", merchant.AccessToken)
	assertHTTPStatus(t, resp, http.StatusOK)
	assertOK(t, adminOrders)
	_ = arrayField(t, adminOrders, "orders")

	adminRooms, resp := c.get(t, "/api/admin/rooms", merchant.AccessToken)
	assertHTTPStatus(t, resp, http.StatusOK)
	assertOK(t, adminRooms)
	_ = arrayField(t, adminRooms, "rooms")

	publicRooms, resp := c.get(t, "/api/rooms", "")
	assertHTTPStatus(t, resp, http.StatusOK)
	assertOK(t, publicRooms)
	_ = arrayField(t, publicRooms, "rooms")

	suggestions, resp := c.get(t, "/api/ai/buyer/suggestions?limit=3", "")
	assertHTTPStatus(t, resp, http.StatusOK)
	assertOK(t, suggestions)
	if len(arrayField(t, suggestions, "suggestions")) == 0 {
		t.Fatalf("expected buyer suggestions: %+v", suggestions)
	}
}

func createShopOrder(t *testing.T, c *e2eClient, buyer account, skuID string, addressID string) map[string]any {
	t.Helper()
	body, resp := c.post(t, "/api/shop/orders", buyer.AccessToken, map[string]any{
		"skuId":          skuID,
		"quantity":       1,
		"addressId":      addressID,
		"idempotencyKey": uniqueUsername("shop_order"),
	})
	assertHTTPStatus(t, resp, http.StatusOK)
	assertOK(t, body)
	order := objectField(t, body, "order")
	if stringField(order, "id") == "" {
		t.Fatalf("shop order missing id: %+v", body)
	}
	return order
}

func anyObject(t *testing.T, value any) map[string]any {
	t.Helper()
	object, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("value is not an object: %T %v", value, value)
	}
	return object
}
