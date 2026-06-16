package e2e

import (
	"net/http"
	"strings"
	"testing"
)

func TestAuctionDraftAutosaveAndQueueContract(t *testing.T) {
	c := newClient(t)
	merchant := registerMerchant(t, c, "draft_merchant")

	emptyDraft, resp := c.post(t, "/api/lots/drafts", merchant.AccessToken, map[string]any{
		"room_id": optionalRoomID(),
	})
	assertHTTPStatus(t, resp, http.StatusOK)
	assertOK(t, emptyDraft)
	lot := objectField(t, emptyDraft, "lot")
	lotID := stringField(lot, "id")
	if lotID == "" {
		t.Fatalf("draft missing lot id: %+v", emptyDraft)
	}
	if status := stringField(lot, "status"); status != "LOT_STATUS_DRAFT" {
		t.Fatalf("expected draft status, got %q body=%+v", status, emptyDraft)
	}

	incompleteQueue, resp := c.post(t, "/api/lots/"+lotID+"/queue", merchant.AccessToken, map[string]any{})
	assertHTTPStatus(t, resp, http.StatusOK)
	assertResultCode(t, incompleteQueue, resultCodeInvalidArgument)

	patch, resp := c.patch(t, "/api/lots/"+lotID+"/draft", merchant.AccessToken, validLotBodyWith(optionalRoomID(), func(body map[string]any) {
		body["title"] = "E2E 自动保存拍品"
		body["description"] = "自动保存补齐标题、图片、规则和详情"
		body["gallery_image_urls"] = []string{"https://example.com/e2e-gallery-1.jpg"}
		body["category"] = "jade"
		body["tags"] = []string{"e2e", "draft"}
		body["stock"] = 2
	}))
	assertHTTPStatus(t, resp, http.StatusOK)
	assertOK(t, patch)
	patchedLot := objectField(t, patch, "lot")
	if got := stringField(patchedLot, "title"); got != "E2E 自动保存拍品" {
		t.Fatalf("expected patched title, got %q body=%+v", got, patch)
	}
	if tags := stringSliceField(patchedLot, "tags"); !containsString(tags, "draft") {
		t.Fatalf("expected patched tags to include draft, got %+v", patchedLot)
	}

	queued := queueLot(t, c, merchant, lotID)
	queuedLot := objectField(t, queued, "lot")
	if status := stringField(queuedLot, "status"); status != "LOT_STATUS_QUEUED" {
		t.Fatalf("expected queued lot status, got %q body=%+v", status, queued)
	}
	if queueStatus := stringField(queuedLot, "queueStatus", "queue_status"); queueStatus != "LOT_QUEUE_STATUS_QUEUED" {
		t.Fatalf("expected queued queue status, got %q body=%+v", queueStatus, queued)
	}
	position := numberField(t, queued, "queuePosition", "queue_position")
	if position < 1 {
		t.Fatalf("expected positive queue position, got %+v", queued)
	}

	repeated := queueLot(t, c, merchant, lotID)
	if repeatedPosition := numberField(t, repeated, "queuePosition", "queue_position"); repeatedPosition != position {
		t.Fatalf("repeated queue should be idempotent, first=%d repeated=%d body=%+v", position, repeatedPosition, repeated)
	}
}

func TestAuctionDraftQueueRequiresMerchantAuthorizationContract(t *testing.T) {
	c := newClient(t)
	merchant := registerMerchant(t, c, "draft_auth_merchant")
	buyer := registerBuyer(t, c, "draft_auth_buyer")

	unauthCreate, resp := c.post(t, "/api/lots/drafts", "", map[string]any{
		"room_id": optionalRoomID(),
	})
	assertHTTPStatus(t, resp, http.StatusOK)
	assertResultCode(t, unauthCreate, resultCodeLoginRequired)

	buyerCreate, resp := c.post(t, "/api/lots/drafts", buyer.AccessToken, map[string]any{
		"room_id": optionalRoomID(),
	})
	assertHTTPStatus(t, resp, http.StatusOK)
	assertResultCode(t, buyerCreate, resultCodeForbidden)

	lotID, _ := createReadyDraftLot(t, c, merchant, "E2E 草稿队列鉴权拍品")
	unauthQueue, resp := c.post(t, "/api/lots/"+lotID+"/queue", "", map[string]any{})
	assertHTTPStatus(t, resp, http.StatusOK)
	assertResultCode(t, unauthQueue, resultCodeLoginRequired)

	buyerQueue, resp := c.post(t, "/api/lots/"+lotID+"/queue", buyer.AccessToken, map[string]any{})
	assertHTTPStatus(t, resp, http.StatusOK)
	assertResultCode(t, buyerQueue, resultCodeForbidden)
}

func TestQueueLotRejectsIncompleteDraftCasesContract(t *testing.T) {
	c := newClient(t)
	merchant := registerMerchant(t, c, "queue_invalid")

	tests := []struct {
		name            string
		body            map[string]any
		messageContains string
	}{
		{
			name:            "missing title",
			body:            draftBodyWith(func(body map[string]any) { delete(body, "title") }),
			messageContains: "标题",
		},
		{
			name:            "missing image",
			body:            draftBodyWith(func(body map[string]any) { delete(body, "image_url") }),
			messageContains: "图片",
		},
		{
			name: "missing rule",
			body: draftBodyWith(func(body map[string]any) { delete(body, "rule") }),
		},
		{
			name: "min increment must be positive",
			body: draftBodyWith(func(body map[string]any) {
				body["rule"] = validBidRuleWith(func(rule map[string]any) {
					rule["min_increment"] = map[string]any{"amount": 0, "currency": "CNY"}
				})
			}),
			messageContains: "最低加价",
		},
		{
			name: "duration must be at least 60 seconds",
			body: draftBodyWith(func(body map[string]any) {
				body["rule"] = validBidRuleWith(func(rule map[string]any) {
					rule["duration_seconds"] = 30
				})
			}),
			messageContains: "竞拍时长",
		},
		{
			name: "start and increment currency must match",
			body: draftBodyWith(func(body map[string]any) {
				body["rule"] = validBidRuleWith(func(rule map[string]any) {
					rule["min_increment"] = map[string]any{"amount": 1000, "currency": "USD"}
				})
			}),
			messageContains: "币种",
		},
		{
			name: "cap price must be greater than start price",
			body: draftBodyWith(func(body map[string]any) {
				body["rule"] = validBidRuleWith(func(rule map[string]any) {
					rule["cap_price"] = map[string]any{"amount": 10000, "currency": "CNY"}
				})
			}),
		},
		{
			name: "image url must be http or https",
			body: draftBodyWith(func(body map[string]any) {
				body["image_url"] = "ftp://example.com/bad.jpg"
			}),
			messageContains: "参数",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			create, resp := c.post(t, "/api/lots/drafts", merchant.AccessToken, tt.body)
			assertHTTPStatus(t, resp, http.StatusOK)
			assertOK(t, create)
			lotID := stringField(objectField(t, create, "lot"), "id")
			if lotID == "" {
				t.Fatalf("draft missing lot id: %+v", create)
			}

			queued, resp := c.post(t, "/api/lots/"+lotID+"/queue", merchant.AccessToken, map[string]any{})
			assertHTTPStatus(t, resp, http.StatusOK)
			assertResultCode(t, queued, resultCodeInvalidArgument)
			if message := resultMessage(queued); tt.messageContains != "" && !strings.Contains(message, tt.messageContains) {
				t.Fatalf("expected message to contain %q, got %q body=%+v", tt.messageContains, message, queued)
			}
		})
	}
}

func TestPatchLotDraftRejectsInvalidAutosaveCasesContract(t *testing.T) {
	c := newClient(t)
	merchant := registerMerchant(t, c, "patch_invalid")

	tests := []struct {
		name string
		body map[string]any
	}{
		{
			name: "invalid main image url",
			body: validLotBodyWith(optionalRoomID(), func(body map[string]any) {
				body["image_url"] = "temporary-preview://local-file"
			}),
		},
		{
			name: "too many gallery images",
			body: validLotBodyWith(optionalRoomID(), func(body map[string]any) {
				body["gallery_image_urls"] = []string{
					"https://example.com/1.jpg",
					"https://example.com/2.jpg",
					"https://example.com/3.jpg",
					"https://example.com/4.jpg",
					"https://example.com/5.jpg",
					"https://example.com/6.jpg",
					"https://example.com/7.jpg",
				}
			}),
		},
		{
			name: "negative stock",
			body: validLotBodyWith(optionalRoomID(), func(body map[string]any) {
				body["stock"] = -1
			}),
		},
		{
			name: "deposit currency missing",
			body: validLotBodyWith(optionalRoomID(), func(body map[string]any) {
				body["deposit_amount"] = map[string]any{"amount": 1000}
			}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			create, resp := c.post(t, "/api/lots/drafts", merchant.AccessToken, map[string]any{"room_id": optionalRoomID()})
			assertHTTPStatus(t, resp, http.StatusOK)
			assertOK(t, create)
			lotID := stringField(objectField(t, create, "lot"), "id")
			if lotID == "" {
				t.Fatalf("draft missing lot id: %+v", create)
			}

			patch, resp := c.patch(t, "/api/lots/"+lotID+"/draft", merchant.AccessToken, tt.body)
			assertHTTPStatus(t, resp, http.StatusOK)
			assertResultCode(t, patch, resultCodeInvalidArgument)
		})
	}
}

func draftBodyWith(mutate func(map[string]any)) map[string]any {
	body := validLotBody(optionalRoomID())
	body["title"] = "E2E 队列校验拍品"
	body["image_url"] = "https://example.com/e2e-queue.jpg"
	if mutate != nil {
		mutate(body)
	}
	return body
}
