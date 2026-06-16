package e2e

import (
	"net/http"
	"testing"
)

func TestHTTPHealthAndUnauthenticatedResultContract(t *testing.T) {
	c := newClient(t)

	health, resp := c.get(t, "/healthz", "")
	assertHTTPStatus(t, resp, http.StatusOK)
	if ok, _ := health["ok"].(bool); !ok {
		t.Fatalf("expected healthz ok response, got %+v", health)
	}

	body, resp := c.doJSON(t, http.MethodGet, "/api/users/me", "", nil, map[string]string{
		"X-Request-Id": "e2e-request-id",
		"X-Trace-Id":   "e2e-trace-id",
	})
	assertHTTPStatus(t, resp, http.StatusOK)
	result := assertResultCode(t, body, resultCodeLoginRequired)
	if result.Message != "login required" {
		t.Fatalf("unexpected unauthenticated message: %+v", result)
	}
	if result.TraceID != "e2e-trace-id" {
		t.Fatalf("expected trace id to be carried through result, got %+v", result)
	}
	if got := resp.Header.Get("X-Request-Id"); got != "e2e-request-id" {
		t.Fatalf("expected request id response header, got %q", got)
	}
	if got := resp.Header.Get("X-Trace-Id"); got != "e2e-trace-id" {
		t.Fatalf("expected trace id response header, got %q", got)
	}
}

func TestHTTPInvalidJSONMapsToInvalidArgument(t *testing.T) {
	c := newClient(t)

	body, resp := c.postRaw(t, "/api/shop/addresses", "", "{", nil)
	assertHTTPStatus(t, resp, http.StatusOK)
	assertResultCode(t, body, resultCodeInvalidArgument)
}

func TestHTTPCORSPreflightContract(t *testing.T) {
	c := newClient(t)

	_, resp := c.doJSON(t, http.MethodOptions, "/api/users/me", "", nil, map[string]string{
		"Origin":                         "http://localhost:5173",
		"Access-Control-Request-Method":  "GET",
		"Access-Control-Request-Headers": "Authorization,Content-Type",
	})
	assertHTTPStatus(t, resp, http.StatusNoContent)
	if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "http://localhost:5173" {
		t.Fatalf("expected allowed CORS origin, got %q", got)
	}
	if got := resp.Header.Get("Access-Control-Allow-Headers"); got == "" {
		t.Fatalf("expected CORS allow headers to be exposed")
	}
}

func TestHTTPMissingResourceUsesResultEnvelopeContract(t *testing.T) {
	c := newClient(t)

	body, resp := c.get(t, "/api/lots/missing-lot/result", "")
	assertHTTPStatus(t, resp, http.StatusOK)
	assertResultCode(t, body, resultCodeUserNotFound)
	if _, ok := body["result"].(map[string]any); !ok {
		t.Fatalf("missing resource should still use result envelope: %+v", body)
	}
}
