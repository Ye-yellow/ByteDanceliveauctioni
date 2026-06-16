package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

const (
	resultCodeOK                  int32 = 0
	resultCodeInvalidArgument     int32 = 400001
	resultCodeLoginRequired       int32 = 401001
	resultCodeTokenInvalid        int32 = 401003
	resultCodeSessionExpired      int32 = 401004
	resultCodeInvalidCredentials  int32 = 401005
	resultCodeForbidden           int32 = 403001
	resultCodeAccountDisabled     int32 = 403002
	resultCodeUserNotFound        int32 = 404001
	resultCodeUsernameTaken       int32 = 409002
	resultCodeRoomActiveLotExists int32 = 409003
	resultCodeBidTooLow           int32 = 409101
	resultCodeBidNotLive          int32 = 409102
	resultCodeBidEnded            int32 = 409103
	resultCodeBidAlreadyLeading   int32 = 409104
	resultCodeBidCurrencyMismatch int32 = 409105
	resultCodeLotCancelled        int32 = 409107
	resultCodeDepositRequired     int32 = 409109
	resultCodeAddressRequired     int32 = 409110
	resultCodeAddressNotFound     int32 = 409111
)

type e2eClient struct {
	baseURL string
	http    *http.Client
}

type account struct {
	UserID       string
	Username     string
	Password     string
	Nickname     string
	AccessToken  string
	RefreshToken string
}

type apiResult struct {
	Code    int32
	Message string
	TraceID string
}

func newClient(t *testing.T) *e2eClient {
	t.Helper()
	baseURL := strings.TrimRight(strings.TrimSpace(os.Getenv("LIVE_AUCTION_E2E_BASE_URL")), "/")
	if baseURL == "" {
		t.Skip("set LIVE_AUCTION_E2E_BASE_URL to run backend e2e tests")
	}
	timeout := 10 * time.Second
	if raw := strings.TrimSpace(os.Getenv("LIVE_AUCTION_E2E_TIMEOUT")); raw != "" {
		parsed, err := time.ParseDuration(raw)
		if err != nil {
			t.Fatalf("invalid LIVE_AUCTION_E2E_TIMEOUT %q: %v", raw, err)
		}
		timeout = parsed
	}
	return &e2eClient{
		baseURL: baseURL,
		http:    &http.Client{Timeout: timeout},
	}
}

func (c *e2eClient) get(t *testing.T, path string, token string) (map[string]any, *http.Response) {
	t.Helper()
	return c.doJSON(t, http.MethodGet, path, token, nil, nil)
}

func (c *e2eClient) post(t *testing.T, path string, token string, body any) (map[string]any, *http.Response) {
	t.Helper()
	return c.doJSON(t, http.MethodPost, path, token, body, nil)
}

func (c *e2eClient) patch(t *testing.T, path string, token string, body any) (map[string]any, *http.Response) {
	t.Helper()
	return c.doJSON(t, http.MethodPatch, path, token, body, nil)
}

func (c *e2eClient) put(t *testing.T, path string, token string, body any) (map[string]any, *http.Response) {
	t.Helper()
	return c.doJSON(t, http.MethodPut, path, token, body, nil)
}

func (c *e2eClient) delete(t *testing.T, path string, token string) (map[string]any, *http.Response) {
	t.Helper()
	return c.doJSON(t, http.MethodDelete, path, token, nil, nil)
}

func (c *e2eClient) doJSON(t *testing.T, method string, path string, token string, body any, headers map[string]string) (map[string]any, *http.Response) {
	t.Helper()
	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal request body: %v", err)
		}
		reader = bytes.NewReader(payload)
	}
	req, err := http.NewRequest(method, c.baseURL+path, reader)
	if err != nil {
		t.Fatalf("new request %s %s: %v", method, path, err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		t.Fatalf("%s %s failed: %v", method, path, err)
	}
	t.Cleanup(func() {
		_ = resp.Body.Close()
	})
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read response body: %v", err)
	}
	if len(bytes.TrimSpace(raw)) == 0 {
		return map[string]any{}, resp
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("decode response for %s %s status=%d body=%s: %v", method, path, resp.StatusCode, string(raw), err)
	}
	return out, resp
}

func (c *e2eClient) postRaw(t *testing.T, path string, token string, raw string, headers map[string]string) (map[string]any, *http.Response) {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, c.baseURL+path, strings.NewReader(raw))
	if err != nil {
		t.Fatalf("new raw post request %s: %v", path, err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		t.Fatalf("POST %s failed: %v", path, err)
	}
	t.Cleanup(func() {
		_ = resp.Body.Close()
	})
	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read response body: %v", err)
	}
	var out map[string]any
	if err := json.Unmarshal(payload, &out); err != nil {
		t.Fatalf("decode raw post response for %s status=%d body=%s: %v", path, resp.StatusCode, string(payload), err)
	}
	return out, resp
}

func assertHTTPStatus(t *testing.T, resp *http.Response, want int) {
	t.Helper()
	if resp.StatusCode != want {
		t.Fatalf("expected HTTP %d, got %d", want, resp.StatusCode)
	}
}

func requireResult(t *testing.T, body map[string]any) apiResult {
	t.Helper()
	raw, ok := body["result"].(map[string]any)
	if !ok {
		t.Fatalf("response missing result object: %+v", body)
	}
	code, ok := optionalNumberField(raw, "code")
	if !ok {
		code = int64(resultCodeOK)
	}
	return apiResult{
		Code:    int32(code),
		Message: stringField(raw, "message"),
		TraceID: stringField(raw, "traceId", "trace_id"),
	}
}

func assertResultCode(t *testing.T, body map[string]any, want int32) apiResult {
	t.Helper()
	result := requireResult(t, body)
	if result.Code != want {
		t.Fatalf("expected result code %d, got %+v body=%+v", want, result, body)
	}
	return result
}

func assertOK(t *testing.T, body map[string]any) {
	t.Helper()
	result := assertResultCode(t, body, resultCodeOK)
	if result.Message != "" && result.Message != "ok" {
		t.Fatalf("expected ok result message, got %+v", result)
	}
}

func registerBuyer(t *testing.T, c *e2eClient, prefix string) account {
	t.Helper()
	username := uniqueUsername(prefix)
	password := "password123"
	body, resp := c.post(t, "/api/users/register", "", map[string]any{
		"username": username,
		"password": password,
		"nickname": username,
	})
	assertHTTPStatus(t, resp, http.StatusOK)
	assertOK(t, body)
	return accountFromAuthBody(t, username, password, body)
}

func registerMerchant(t *testing.T, c *e2eClient, prefix string) account {
	t.Helper()
	username := uniqueUsername(prefix)
	password := "merchantpass123"
	body, resp := c.post(t, "/api/merchants/register", "", map[string]any{
		"username": username,
		"password": password,
		"nickname": username,
	})
	assertHTTPStatus(t, resp, http.StatusOK)
	assertOK(t, body)
	return accountFromAuthBody(t, username, password, body)
}

func login(t *testing.T, c *e2eClient, username, password string) account {
	t.Helper()
	body, resp := c.post(t, "/api/users/login", "", map[string]any{
		"username": username,
		"password": password,
	})
	assertHTTPStatus(t, resp, http.StatusOK)
	assertOK(t, body)
	return accountFromAuthBody(t, username, password, body)
}

func accountFromAuthBody(t *testing.T, username, password string, body map[string]any) account {
	t.Helper()
	user, ok := body["user"].(map[string]any)
	if !ok {
		t.Fatalf("auth response missing user: %+v", body)
	}
	tokens, ok := body["tokens"].(map[string]any)
	if !ok {
		t.Fatalf("auth response missing tokens: %+v", body)
	}
	next := account{
		UserID:       stringField(user, "id"),
		Username:     stringField(user, "username"),
		Nickname:     stringField(user, "nickname"),
		Password:     password,
		AccessToken:  stringField(tokens, "accessToken", "access_token"),
		RefreshToken: stringField(tokens, "refreshToken", "refresh_token"),
	}
	if next.Username == "" {
		next.Username = username
	}
	if next.UserID == "" || next.AccessToken == "" || next.RefreshToken == "" {
		t.Fatalf("auth response missing user id or tokens: %+v", body)
	}
	return next
}

func uniqueUsername(prefix string) string {
	clean := strings.ToLower(strings.TrimSpace(prefix))
	if clean == "" {
		clean = "e2e"
	}
	clean = strings.NewReplacer("-", "_", ".", "_").Replace(clean)
	return fmt.Sprintf("%s_%d", clean, time.Now().UnixNano())
}

func optionalRoomID() string {
	return strings.TrimSpace(os.Getenv("LIVE_AUCTION_E2E_ROOM_ID"))
}

func resultMessage(body map[string]any) string {
	if result, ok := body["result"].(map[string]any); ok {
		return stringField(result, "message")
	}
	return ""
}

func stringField(values map[string]any, names ...string) string {
	for _, name := range names {
		if value, ok := values[name]; ok {
			if s, ok := value.(string); ok {
				return s
			}
		}
	}
	return ""
}

func numberField(t *testing.T, values map[string]any, names ...string) int64 {
	t.Helper()
	for _, name := range names {
		value, ok := values[name]
		if !ok {
			continue
		}
		switch typed := value.(type) {
		case float64:
			return int64(typed)
		case int64:
			return typed
		case int:
			return int64(typed)
		case string:
			parsed, err := strconv.ParseInt(typed, 10, 64)
			if err != nil {
				t.Fatalf("field %q is not numeric: %T %v", name, value, value)
			}
			return parsed
		default:
			t.Fatalf("field %q is not numeric: %T %v", name, value, value)
		}
	}
	t.Fatalf("missing numeric field %v in %+v", names, values)
	return 0
}

func optionalNumberField(values map[string]any, names ...string) (int64, bool) {
	for _, name := range names {
		value, ok := values[name]
		if !ok {
			continue
		}
		switch typed := value.(type) {
		case float64:
			return int64(typed), true
		case int64:
			return typed, true
		case int:
			return int64(typed), true
		case string:
			parsed, err := strconv.ParseInt(typed, 10, 64)
			if err == nil {
				return parsed, true
			}
		}
	}
	return 0, false
}

func objectField(t *testing.T, values map[string]any, name string) map[string]any {
	t.Helper()
	value, ok := values[name]
	if !ok {
		t.Fatalf("missing object field %q in %+v", name, values)
	}
	object, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("field %q is not an object: %T %v", name, value, value)
	}
	return object
}

func arrayField(t *testing.T, values map[string]any, names ...string) []any {
	t.Helper()
	for _, name := range names {
		value, ok := values[name]
		if !ok {
			continue
		}
		array, ok := value.([]any)
		if !ok {
			t.Fatalf("field %q is not an array: %T %v", name, value, value)
		}
		return array
	}
	t.Fatalf("missing array field %v in %+v", names, values)
	return nil
}

func stringSliceField(values map[string]any, names ...string) []string {
	for _, name := range names {
		raw, ok := values[name].([]any)
		if !ok {
			continue
		}
		out := make([]string, 0, len(raw))
		for _, item := range raw {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
