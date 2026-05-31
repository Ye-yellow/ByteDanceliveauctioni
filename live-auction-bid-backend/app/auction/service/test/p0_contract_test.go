package test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"

	"live-auction-bid/backend/app/auction/service/internal/pkg/apperr"
	"live-auction-bid/backend/app/auction/service/internal/pkg/auth"
	"live-auction-bid/backend/app/auction/service/internal/pkg/requestctx"
	appsvc "live-auction-bid/backend/app/auction/service/internal/service"
)

func TestP0ErrorResultDoesNotExposeInternalDetailsAndCarriesTraceID(t *testing.T) {
	ctx := requestctx.WithRequestContext(context.Background(), requestctx.RequestContext{RequestID: "req-1", TraceID: "trace-1"})
	result := appsvc.ErrorResult(ctx, errors.New("sql: password=secret redis dial tcp"))

	if result.GetCode() != appsvc.ResultCodeInternalError {
		t.Fatalf("expected internal error code, got %d", result.GetCode())
	}
	if result.GetMessage() != appsvc.MessageInternalError {
		t.Fatalf("internal error message must be sanitized, got %q", result.GetMessage())
	}
	if result.GetTraceId() != "trace-1" {
		t.Fatalf("expected trace id from request context, got %q", result.GetTraceId())
	}
}

func TestP0AuthContextDistinguishesNoneInvalidExpiredAndValid(t *testing.T) {
	now := time.Unix(1700000000, 0)
	manager, err := auth.NewManager(auth.Config{Secret: "unit-test-secret", Issuer: "test", AccessTTL: time.Minute}, auth.WithNow(func() time.Time { return now }))
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	user := buyerUserForTest("u1", "buyer", "买家")
	tokens, err := manager.IssueTokenPair(user)
	if err != nil {
		t.Fatalf("issue tokens: %v", err)
	}

	validCtx := runAuthMiddleware(t, manager, "Bearer "+tokens.AccessToken)
	claims, err := auth.RequireUser(validCtx)
	if err != nil || claims.UserID != "u1" {
		t.Fatalf("expected valid auth context, claims=%+v err=%v", claims, err)
	}

	missingCtx := runAuthMiddleware(t, manager, "")
	if _, err := auth.RequireUser(missingCtx); !apperr.IsUnauthenticated(err) {
		t.Fatalf("missing token should be unauthenticated, got %v", err)
	}

	invalidCtx := runAuthMiddleware(t, manager, "Bearer tampered.token.value")
	if _, err := auth.RequireUser(invalidCtx); !apperr.IsInvalidToken(err) {
		t.Fatalf("invalid token should be invalid token, got %v", err)
	}

	expiredManager, err := auth.NewManager(auth.Config{Secret: "unit-test-secret", Issuer: "test", AccessTTL: time.Minute}, auth.WithNow(func() time.Time { return now.Add(2 * time.Minute) }))
	if err != nil {
		t.Fatalf("new expired manager: %v", err)
	}
	expiredCtx := runAuthMiddleware(t, expiredManager, "Bearer "+tokens.AccessToken)
	if _, err := auth.RequireUser(expiredCtx); !apperr.IsTokenExpired(err) {
		t.Fatalf("expired token should be token expired, got %v", err)
	}
	if result := appsvc.ErrorResult(expiredCtx, apperr.ErrTokenExpired); result.GetCode() != appsvc.ResultCodeTokenExpired {
		t.Fatalf("expired token should map to auth refresh code 401002, got %+v", result)
	}
	if result := appsvc.ErrorResult(expiredCtx, apperr.ErrSessionExpired); result.GetCode() != appsvc.ResultCodeSessionExpired || result.GetMessage() != appsvc.MessageSessionExpired {
		t.Fatalf("session expired should map to stable session code/message, got %+v", result)
	}
}

func TestP0RequestContextHTTPMiddlewareWritesCorrelationHeaders(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set(requestctx.HeaderRequestID, "client-req-1")
	req.Header.Set(requestctx.HeaderClientApp, "admin-web")
	rr := httptest.NewRecorder()

	requestctx.HTTPMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rc, ok := requestctx.FromContext(r.Context())
		if !ok {
			t.Fatal("request context missing")
		}
		if rc.RequestID != "client-req-1" || rc.TraceID != "client-req-1" || rc.ClientType != requestctx.ClientTypeAdmin {
			t.Fatalf("unexpected request context: %+v", rc)
		}
	})).ServeHTTP(rr, req)

	if rr.Header().Get(requestctx.HeaderRequestID) != "client-req-1" {
		t.Fatalf("request id response header missing: %q", rr.Header().Get(requestctx.HeaderRequestID))
	}
	if rr.Header().Get(requestctx.HeaderTraceID) != "client-req-1" || rr.Header().Get(requestctx.HeaderServerTime) == "" {
		t.Fatalf("trace/server-time response headers missing: trace=%q server=%q", rr.Header().Get(requestctx.HeaderTraceID), rr.Header().Get(requestctx.HeaderServerTime))
	}
}

func runAuthMiddleware(t *testing.T, manager *auth.Manager, authorization string) context.Context {
	t.Helper()
	ctx := transport.NewServerContext(context.Background(), headerTransport{header: transportHeader{"Authorization": authorization}})
	var captured context.Context
	handler := manager.Middleware()(func(ctx context.Context, req any) (any, error) {
		captured = ctx
		return nil, nil
	})
	if _, err := handler(ctx, nil); err != nil {
		t.Fatalf("auth middleware returned error: %v", err)
	}
	return captured
}

var _ middleware.Handler = func(context.Context, any) (any, error) { return nil, nil }

type headerTransport struct{ header transportHeader }

func (h headerTransport) Kind() transport.Kind            { return transport.KindHTTP }
func (h headerTransport) Endpoint() string                { return "" }
func (h headerTransport) Operation() string               { return "" }
func (h headerTransport) RequestHeader() transport.Header { return h.header }
func (h headerTransport) ReplyHeader() transport.Header   { return transportHeader{} }

type transportHeader map[string]string

func (h transportHeader) Get(key string) string        { return h[key] }
func (h transportHeader) Set(key string, value string) { h[key] = value }
func (h transportHeader) Add(key string, value string) { h[key] = value }
func (h transportHeader) Keys() []string {
	keys := make([]string, 0, len(h))
	for k := range h {
		keys = append(keys, k)
	}
	return keys
}
func (h transportHeader) Values(key string) []string {
	if value := h.Get(key); value != "" {
		return []string{value}
	}
	return nil
}
