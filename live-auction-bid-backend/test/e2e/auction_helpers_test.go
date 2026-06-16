package e2e

import (
	"net/http"
	"testing"
)

func createReadyDraftLot(t *testing.T, c *e2eClient, merchant account, title string) (string, map[string]any) {
	t.Helper()
	create, resp := c.post(t, "/api/lots/drafts", merchant.AccessToken, map[string]any{
		"room_id": optionalRoomID(),
	})
	assertHTTPStatus(t, resp, http.StatusOK)
	assertOK(t, create)
	lotID := stringField(objectField(t, create, "lot"), "id")
	if lotID == "" {
		t.Fatalf("draft missing lot id: %+v", create)
	}

	patchBody := validLotBody(optionalRoomID())
	patchBody["title"] = title
	patch, resp := c.patch(t, "/api/lots/"+lotID+"/draft", merchant.AccessToken, patchBody)
	assertHTTPStatus(t, resp, http.StatusOK)
	assertOK(t, patch)
	return lotID, objectField(t, patch, "lot")
}

func startLot(t *testing.T, c *e2eClient, merchant account, lotID string) map[string]any {
	t.Helper()
	start, resp := c.post(t, "/api/lots/"+lotID+"/start", merchant.AccessToken, map[string]any{})
	assertHTTPStatus(t, resp, http.StatusOK)
	assertOK(t, start)
	return objectField(t, start, "lot")
}

func queueLot(t *testing.T, c *e2eClient, merchant account, lotID string) map[string]any {
	t.Helper()
	queued, resp := c.post(t, "/api/lots/"+lotID+"/queue", merchant.AccessToken, map[string]any{})
	assertHTTPStatus(t, resp, http.StatusOK)
	assertOK(t, queued)
	return queued
}

func placeBid(t *testing.T, c *e2eClient, buyer account, lotID string, amount int64, currency string, idempotencyKey string) (map[string]any, *http.Response) {
	t.Helper()
	return c.post(t, "/api/lots/"+lotID+"/bid", buyer.AccessToken, bidBody(amount, currency, idempotencyKey))
}

func createDeliveryAddress(t *testing.T, c *e2eClient, buyer account) string {
	t.Helper()
	body, resp := c.post(t, "/api/shop/addresses", buyer.AccessToken, map[string]any{
		"address": map[string]any{
			"receiverName": "E2E Buyer",
			"phone":        "13800138000",
			"province":     "上海市",
			"city":         "上海市",
			"district":     "浦东新区",
			"street":       "世纪大道",
			"detail":       "E2E 测试地址 1 号",
			"tag":          "e2e",
			"isDefault":    true,
		},
	})
	assertHTTPStatus(t, resp, http.StatusOK)
	assertOK(t, body)
	addressID := stringField(objectField(t, body, "address"), "id")
	if addressID == "" {
		t.Fatalf("delivery address missing id: %+v", body)
	}
	return addressID
}

func holdDeposit(t *testing.T, c *e2eClient, buyer account, lotID string) {
	t.Helper()
	addressID := createDeliveryAddress(t, c, buyer)
	body, resp := c.post(t, "/api/lots/"+lotID+"/deposit-holds/mock-pay", buyer.AccessToken, map[string]any{
		"addressId":      addressID,
		"idempotencyKey": uniqueUsername("deposit_key"),
	})
	assertHTTPStatus(t, resp, http.StatusOK)
	assertOK(t, body)
	if paid, _ := body["paid"].(bool); !paid {
		t.Fatalf("expected deposit hold to be paid, got %+v", body)
	}
}

func bidBody(amount int64, currency string, idempotencyKey string) map[string]any {
	return map[string]any{
		"amount":          map[string]any{"amount": amount, "currency": currency},
		"idempotency_key": idempotencyKey,
	}
}

func validLotBodyWith(roomID string, mutate func(map[string]any)) map[string]any {
	body := validLotBody(roomID)
	if mutate != nil {
		mutate(body)
	}
	return body
}

func validBidRuleWith(mutate func(map[string]any)) map[string]any {
	rule := map[string]any{
		"start_price":               map[string]any{"amount": 10000, "currency": "CNY"},
		"min_increment":             map[string]any{"amount": 1000, "currency": "CNY"},
		"duration_seconds":          300,
		"anti_snipe_window_seconds": 15,
		"anti_snipe_extend_seconds": 15,
		"max_extend_count":          3,
	}
	if mutate != nil {
		mutate(rule)
	}
	return rule
}

func createTeamUser(t *testing.T, c *e2eClient, merchant account, roleCode string) map[string]any {
	t.Helper()
	body, resp := c.post(t, "/api/admin/users", merchant.AccessToken, map[string]any{
		"username":  uniqueUsername(roleCode),
		"password":  "teampass123",
		"nickname":  roleCode + "用户",
		"role_code": roleCode,
	})
	assertHTTPStatus(t, resp, http.StatusOK)
	assertOK(t, body)
	return objectField(t, body, "user")
}
