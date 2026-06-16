package e2e

import (
	"net/http"
	"testing"
)

func TestAuctionProtectedOperationsRequireExpectedPermissions(t *testing.T) {
	c := newClient(t)

	unauthCreate, resp := c.post(t, "/api/lots", "", validLotBody("auth-room"))
	assertHTTPStatus(t, resp, http.StatusOK)
	assertResultCode(t, unauthCreate, resultCodeLoginRequired)

	buyer := registerBuyer(t, c, "auction_buyer")
	buyerCreate, resp := c.post(t, "/api/lots", buyer.AccessToken, validLotBody("buyer-room"))
	assertHTTPStatus(t, resp, http.StatusOK)
	assertResultCode(t, buyerCreate, resultCodeForbidden)

	unauthBid, resp := c.post(t, "/api/lots/missing-lot/bid", "", bidBody(11000, "CNY", "missing-auth"))
	assertHTTPStatus(t, resp, http.StatusOK)
	assertResultCode(t, unauthBid, resultCodeLoginRequired)

	merchant := registerMerchant(t, c, "auction_no_bid")
	merchantBid, resp := c.post(t, "/api/lots/missing-lot/bid", merchant.AccessToken, bidBody(11000, "CNY", "merchant-bid"))
	assertHTTPStatus(t, resp, http.StatusOK)
	assertResultCode(t, merchantBid, resultCodeForbidden)

	buyerStart, resp := c.post(t, "/api/lots/missing-lot/start", buyer.AccessToken, map[string]any{})
	assertHTTPStatus(t, resp, http.StatusOK)
	assertResultCode(t, buyerStart, resultCodeForbidden)
}

func TestAuctionBuyerBidUsesTokenIdentityAndIdempotencyContract(t *testing.T) {
	c := newClient(t)
	merchant := registerMerchant(t, c, "auction_merchant")
	buyer := registerBuyer(t, c, "auction_bidder")
	otherBuyer := registerBuyer(t, c, "auction_other_bidder")
	lotID, _ := createReadyDraftLot(t, c, merchant, "E2E 出价矩阵拍品")
	_ = startLot(t, c, merchant, lotID)
	holdDeposit(t, c, buyer, lotID)
	holdDeposit(t, c, otherBuyer, lotID)

	idempotencyKey := uniqueUsername("bid_key")
	bid, resp := placeBid(t, c, buyer, lotID, 11000, "CNY", idempotencyKey)
	assertHTTPStatus(t, resp, http.StatusOK)
	assertOK(t, bid)
	if accepted, _ := bid["accepted"].(bool); !accepted {
		t.Fatalf("expected accepted bid, got %+v", bid)
	}
	bidObject := objectField(t, bid, "bid")
	if got := stringField(bidObject, "userId", "user_id"); got != buyer.UserID {
		t.Fatalf("bidder identity should come from token, want %q got %q body=%+v", buyer.UserID, got, bid)
	}

	replayed, resp := placeBid(t, c, buyer, lotID, 11000, "CNY", idempotencyKey)
	assertHTTPStatus(t, resp, http.StatusOK)
	assertOK(t, replayed)
	if accepted, _ := replayed["accepted"].(bool); !accepted {
		t.Fatalf("expected idempotent replay to be accepted, got %+v", replayed)
	}
	if replayedID := stringField(objectField(t, replayed, "bid"), "id"); replayedID != stringField(bidObject, "id") {
		t.Fatalf("expected replayed bid id %q, got %q body=%+v", stringField(bidObject, "id"), replayedID, replayed)
	}

	tests := []struct {
		name  string
		token string
		body  map[string]any
		want  int32
	}{
		{
			name:  "missing idempotency key",
			token: buyer.AccessToken,
			body:  map[string]any{"amount": map[string]any{"amount": 12000, "currency": "CNY"}},
			want:  resultCodeInvalidArgument,
		},
		{
			name:  "current leader cannot immediately outbid self",
			token: buyer.AccessToken,
			body:  bidBody(12000, "CNY", uniqueUsername("leader_repeat")),
			want:  resultCodeBidAlreadyLeading,
		},
		{
			name:  "other buyer amount lower than current plus increment",
			token: otherBuyer.AccessToken,
			body:  bidBody(11500, "CNY", uniqueUsername("too_low")),
			want:  resultCodeBidTooLow,
		},
		{
			name:  "other buyer currency mismatch",
			token: otherBuyer.AccessToken,
			body:  bidBody(13000, "USD", uniqueUsername("currency")),
			want:  resultCodeBidCurrencyMismatch,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, resp := c.post(t, "/api/lots/"+lotID+"/bid", tt.token, tt.body)
			assertHTTPStatus(t, resp, http.StatusOK)
			assertResultCode(t, body, tt.want)
			if accepted, _ := body["accepted"].(bool); accepted {
				t.Fatalf("rejected bid should not be accepted: %+v", body)
			}
		})
	}
}

func TestAuctionDepositHoldPreconditionsAndIdempotencyContract(t *testing.T) {
	c := newClient(t)
	merchant := registerMerchant(t, c, "auction_deposit_merchant")
	buyer := registerBuyer(t, c, "auction_deposit_buyer")
	lotID, _ := createReadyDraftLot(t, c, merchant, "E2E 押金前置拍品")

	noToken, resp := c.post(t, "/api/lots/"+lotID+"/deposit-holds/mock-pay", "", map[string]any{
		"addressId":      "missing-address",
		"idempotencyKey": uniqueUsername("deposit_no_token"),
	})
	assertHTTPStatus(t, resp, http.StatusOK)
	assertResultCode(t, noToken, resultCodeLoginRequired)

	merchantHold, resp := c.post(t, "/api/lots/"+lotID+"/deposit-holds/mock-pay", merchant.AccessToken, map[string]any{
		"addressId":      "missing-address",
		"idempotencyKey": uniqueUsername("deposit_merchant"),
	})
	assertHTTPStatus(t, resp, http.StatusOK)
	assertResultCode(t, merchantHold, resultCodeForbidden)

	missingAddressID, resp := c.post(t, "/api/lots/"+lotID+"/deposit-holds/mock-pay", buyer.AccessToken, map[string]any{
		"idempotencyKey": uniqueUsername("deposit_missing_address"),
	})
	assertHTTPStatus(t, resp, http.StatusOK)
	assertResultCode(t, missingAddressID, resultCodeAddressRequired)

	addressNotFound, resp := c.post(t, "/api/lots/"+lotID+"/deposit-holds/mock-pay", buyer.AccessToken, map[string]any{
		"addressId":      "missing-address",
		"idempotencyKey": uniqueUsername("deposit_address_not_found"),
	})
	assertHTTPStatus(t, resp, http.StatusOK)
	assertResultCode(t, addressNotFound, resultCodeAddressNotFound)

	_ = startLot(t, c, merchant, lotID)
	bidWithoutDeposit, resp := placeBid(t, c, buyer, lotID, 11000, "CNY", uniqueUsername("bid_without_deposit"))
	assertHTTPStatus(t, resp, http.StatusOK)
	assertResultCode(t, bidWithoutDeposit, resultCodeDepositRequired)
	if accepted, _ := bidWithoutDeposit["accepted"].(bool); accepted {
		t.Fatalf("bid without deposit should not be accepted: %+v", bidWithoutDeposit)
	}

	addressID := createDeliveryAddress(t, c, buyer)
	key := uniqueUsername("deposit_replay")
	first, resp := c.post(t, "/api/lots/"+lotID+"/deposit-holds/mock-pay", buyer.AccessToken, map[string]any{
		"addressId":      addressID,
		"idempotencyKey": key,
	})
	assertHTTPStatus(t, resp, http.StatusOK)
	assertOK(t, first)
	if paid, _ := first["paid"].(bool); !paid {
		t.Fatalf("expected first deposit hold to be paid: %+v", first)
	}
	firstHoldID := stringField(objectField(t, first, "depositHold"), "id")
	if firstHoldID == "" {
		t.Fatalf("expected first deposit hold id: %+v", first)
	}

	replayed, resp := c.post(t, "/api/lots/"+lotID+"/deposit-holds/mock-pay", buyer.AccessToken, map[string]any{
		"addressId":      addressID,
		"idempotencyKey": key,
	})
	assertHTTPStatus(t, resp, http.StatusOK)
	assertOK(t, replayed)
	if paid, _ := replayed["paid"].(bool); !paid {
		t.Fatalf("expected replayed deposit hold to be paid: %+v", replayed)
	}
	if replayedHoldID := stringField(objectField(t, replayed, "depositHold"), "id"); replayedHoldID != firstHoldID {
		t.Fatalf("expected replayed deposit hold id %q, got %q body=%+v", firstHoldID, replayedHoldID, replayed)
	}
}

func TestAuctionBidRejectsNotLiveAndCancelledLotsContract(t *testing.T) {
	c := newClient(t)
	merchant := registerMerchant(t, c, "auction_state_merchant")
	buyer := registerBuyer(t, c, "auction_state_buyer")

	draftLotID, _ := createReadyDraftLot(t, c, merchant, "E2E 未开拍拍品")
	holdDeposit(t, c, buyer, draftLotID)
	notLive, resp := placeBid(t, c, buyer, draftLotID, 11000, "CNY", uniqueUsername("not_live"))
	assertHTTPStatus(t, resp, http.StatusOK)
	assertResultCode(t, notLive, resultCodeBidNotLive)

	cancelLotID, _ := createReadyDraftLot(t, c, merchant, "E2E 取消后出价拍品")
	_ = startLot(t, c, merchant, cancelLotID)
	holdDeposit(t, c, buyer, cancelLotID)
	cancelled, resp := c.post(t, "/api/lots/"+cancelLotID+"/cancel", merchant.AccessToken, map[string]any{
		"reason": "主播临时取消",
	})
	assertHTTPStatus(t, resp, http.StatusOK)
	assertOK(t, cancelled)

	afterCancel, resp := placeBid(t, c, buyer, cancelLotID, 11000, "CNY", uniqueUsername("cancelled_bid"))
	assertHTTPStatus(t, resp, http.StatusOK)
	assertResultCode(t, afterCancel, resultCodeLotCancelled)
}

func TestAuctionControlRejectsInvalidStateTransitionsContract(t *testing.T) {
	c := newClient(t)
	merchant := registerMerchant(t, c, "auction_control")
	lotID, _ := createReadyDraftLot(t, c, merchant, "E2E 控制台状态拍品")
	_ = startLot(t, c, merchant, lotID)

	startAgain, resp := c.post(t, "/api/lots/"+lotID+"/start", merchant.AccessToken, map[string]any{})
	assertHTTPStatus(t, resp, http.StatusOK)
	assertResultCode(t, startAgain, resultCodeInvalidArgument)

	settleWithoutBid, resp := c.post(t, "/api/lots/"+lotID+"/settle", merchant.AccessToken, map[string]any{})
	assertHTTPStatus(t, resp, http.StatusOK)
	assertResultCode(t, settleWithoutBid, resultCodeInvalidArgument)

	cancelWithoutReason, resp := c.post(t, "/api/lots/"+lotID+"/cancel", merchant.AccessToken, map[string]any{})
	assertHTTPStatus(t, resp, http.StatusOK)
	assertResultCode(t, cancelWithoutReason, resultCodeInvalidArgument)
}

func validLotBody(roomID string) map[string]any {
	return map[string]any{
		"room_id":   roomID,
		"title":     "E2E 合同测试拍品",
		"image_url": "https://example.com/e2e-lot.jpg",
		"rule":      validBidRuleWith(nil),
	}
}
