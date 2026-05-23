package requestctx

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net"
	"net/http"
	"strings"
	"time"
)

const (
	HeaderRequestID   = "X-Request-Id"
	HeaderTraceID     = "X-Trace-Id"
	HeaderServerTime  = "X-Server-Time"
	HeaderClientApp   = "X-Client-App"
	HeaderClientVer   = "X-Client-Version"
	HeaderClientTime  = "X-Client-Time"
	ClientTypeAdmin   = "admin"
	ClientTypeBuyerH5 = "h5"
	ClientTypeUnknown = "unknown"
)

type contextKey struct{}

// RequestContext is the transport-neutral request metadata used by service and
// application layers. Business code should depend on this context instead of
// reading HTTP request objects directly.
type RequestContext struct {
	RequestID    string
	TraceID      string
	ClientType   string
	ClientApp    string
	ClientVer    string
	ClientTime   string
	IP           string
	UserAgent    string
	UserID       string
	UserRole     string
	ServerTimeMs int64
}

func WithRequestContext(ctx context.Context, rc RequestContext) context.Context {
	return context.WithValue(ctx, contextKey{}, rc)
}

func FromContext(ctx context.Context) (RequestContext, bool) {
	rc, ok := ctx.Value(contextKey{}).(RequestContext)
	return rc, ok
}

func RequestID(ctx context.Context) string {
	if rc, ok := FromContext(ctx); ok {
		return rc.RequestID
	}
	return ""
}

func TraceID(ctx context.Context) string {
	if rc, ok := FromContext(ctx); ok {
		if rc.TraceID != "" {
			return rc.TraceID
		}
		return rc.RequestID
	}
	return ""
}

func ClientType(ctx context.Context) string {
	if rc, ok := FromContext(ctx); ok && rc.ClientType != "" {
		return rc.ClientType
	}
	return ClientTypeUnknown
}

func UserID(ctx context.Context) string {
	if rc, ok := FromContext(ctx); ok {
		return rc.UserID
	}
	return ""
}

func UserRole(ctx context.Context) string {
	if rc, ok := FromContext(ctx); ok {
		return rc.UserRole
	}
	return ""
}

func WithUser(ctx context.Context, userID, userRole string) context.Context {
	rc, ok := FromContext(ctx)
	if !ok {
		rc = RequestContext{RequestID: newID(), TraceID: "", ClientType: ClientTypeUnknown, ServerTimeMs: time.Now().UnixMilli()}
		rc.TraceID = rc.RequestID
	}
	rc.UserID = strings.TrimSpace(userID)
	rc.UserRole = strings.TrimSpace(userRole)
	return WithRequestContext(ctx, rc)
}

func Snapshot(ctx context.Context) RequestContext {
	rc, _ := FromContext(ctx)
	if rc.TraceID == "" {
		rc.TraceID = rc.RequestID
	}
	if rc.ClientType == "" {
		rc.ClientType = ClientTypeUnknown
	}
	return rc
}

func ServerTimeMs(ctx context.Context) int64 {
	if rc, ok := FromContext(ctx); ok && rc.ServerTimeMs > 0 {
		return rc.ServerTimeMs
	}
	return time.Now().UnixMilli()
}

func HTTPMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nowMs := time.Now().UnixMilli()
		requestID := strings.TrimSpace(r.Header.Get(HeaderRequestID))
		if requestID == "" {
			requestID = newID()
		}
		traceID := strings.TrimSpace(r.Header.Get(HeaderTraceID))
		if traceID == "" {
			traceID = requestID
		}
		rc := RequestContext{
			RequestID:    requestID,
			TraceID:      traceID,
			ClientType:   inferClientType(r.Header.Get(HeaderClientApp)),
			ClientApp:    strings.TrimSpace(r.Header.Get(HeaderClientApp)),
			ClientVer:    strings.TrimSpace(r.Header.Get(HeaderClientVer)),
			ClientTime:   strings.TrimSpace(r.Header.Get(HeaderClientTime)),
			IP:           clientIP(r),
			UserAgent:    r.UserAgent(),
			ServerTimeMs: nowMs,
		}
		w.Header().Set(HeaderRequestID, requestID)
		w.Header().Set(HeaderTraceID, traceID)
		w.Header().Set(HeaderServerTime, strconvFormatInt(nowMs))
		next.ServeHTTP(w, r.WithContext(WithRequestContext(r.Context(), rc)))
	})
}

func inferClientType(clientApp string) string {
	s := strings.ToLower(strings.TrimSpace(clientApp))
	switch {
	case strings.Contains(s, "admin") || strings.Contains(s, "host"):
		return ClientTypeAdmin
	case strings.Contains(s, "h5") || strings.Contains(s, "buyer"):
		return ClientTypeBuyerH5
	default:
		return ClientTypeUnknown
	}
}

func clientIP(r *http.Request) string {
	for _, header := range []string{"X-Forwarded-For", "X-Real-IP"} {
		value := strings.TrimSpace(r.Header.Get(header))
		if value == "" {
			continue
		}
		if header == "X-Forwarded-For" {
			value = strings.TrimSpace(strings.Split(value, ",")[0])
		}
		if value != "" {
			return value
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil && host != "" {
		return host
	}
	return r.RemoteAddr
}

func newID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return strconvFormatInt(time.Now().UnixNano())
	}
	return hex.EncodeToString(b[:])
}

func strconvFormatInt(v int64) string {
	// Keep strconv localized so the package API stays intentionally tiny.
	return formatInt(v)
}
