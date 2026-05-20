package auth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
	"golang.org/x/crypto/bcrypt"
	v1 "live-auction-bid/backend/api/auction/service/v1"
	"live-auction-bid/backend/app/auction/service/internal/pkg/apperr"
)

const (
	DefaultAccessTTL  = 15 * time.Minute
	DefaultRefreshTTL = 7 * 24 * time.Hour
)

type contextKey struct{}

type Config struct {
	Secret     string
	Issuer     string
	AccessTTL  time.Duration
	RefreshTTL time.Duration
}

type Manager struct {
	secret     []byte
	issuer     string
	accessTTL  time.Duration
	refreshTTL time.Duration
	now        func() time.Time
}

type Option func(*Manager)

func WithNow(now func() time.Time) Option {
	return func(m *Manager) {
		if now != nil {
			m.now = now
		}
	}
}

type Claims struct {
	UserID    string      `json:"sub"`
	Username  string      `json:"username"`
	Nickname  string      `json:"nickname"`
	Role      v1.UserRole `json:"-"`
	RoleName  string      `json:"role"`
	TokenType string      `json:"typ"`
	Issuer    string      `json:"iss"`
	IssuedAt  int64       `json:"iat"`
	ExpiresAt int64       `json:"exp"`
}

type TokenPair struct {
	AccessToken        string
	RefreshToken       string
	AccessExpiresAtMs  int64
	RefreshExpiresAtMs int64
}

func NewManager(cfg Config, opts ...Option) (*Manager, error) {
	if strings.TrimSpace(cfg.Secret) == "" {
		return nil, errors.New("jwt secret is required")
	}
	if cfg.AccessTTL <= 0 {
		cfg.AccessTTL = DefaultAccessTTL
	}
	if cfg.RefreshTTL <= 0 {
		cfg.RefreshTTL = DefaultRefreshTTL
	}
	m := &Manager{
		secret:     []byte(cfg.Secret),
		issuer:     cfg.Issuer,
		accessTTL:  cfg.AccessTTL,
		refreshTTL: cfg.RefreshTTL,
		now:        time.Now,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(m)
		}
	}
	return m, nil
}

func (m *Manager) AccessTTL() time.Duration {
	return m.accessTTL
}

func (m *Manager) RefreshTTL() time.Duration {
	return m.refreshTTL
}

func (m *Manager) IssueTokenPair(user *v1.User) (TokenPair, error) {
	if user == nil || user.GetId() == "" {
		return TokenPair{}, fmt.Errorf("%w: user is required", apperr.ErrInvalidArgument)
	}
	now := m.now()
	accessExpiresAt := now.Add(m.accessTTL)
	refreshExpiresAt := now.Add(m.refreshTTL)
	access, err := m.signAccessToken(user, now, accessExpiresAt)
	if err != nil {
		return TokenPair{}, err
	}
	refresh, err := NewOpaqueToken()
	if err != nil {
		return TokenPair{}, err
	}
	return TokenPair{
		AccessToken:        access,
		RefreshToken:       refresh,
		AccessExpiresAtMs:  accessExpiresAt.UnixMilli(),
		RefreshExpiresAtMs: refreshExpiresAt.UnixMilli(),
	}, nil
}

func (m *Manager) ParseAccessToken(token string) (*Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, apperr.ErrInvalidToken
	}
	signed := parts[0] + "." + parts[1]
	if !hmac.Equal([]byte(parts[2]), []byte(m.sign(signed))) {
		return nil, apperr.ErrInvalidToken
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, apperr.ErrInvalidToken
	}
	var claims Claims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, apperr.ErrInvalidToken
	}
	if claims.TokenType != "access" || claims.UserID == "" {
		return nil, apperr.ErrInvalidToken
	}
	if m.issuer != "" && claims.Issuer != m.issuer {
		return nil, apperr.ErrInvalidToken
	}
	if claims.ExpiresAt <= m.now().Unix() {
		return nil, apperr.ErrInvalidToken
	}
	role, ok := RoleFromName(claims.RoleName)
	if !ok {
		return nil, apperr.ErrInvalidToken
	}
	claims.Role = role
	return &claims, nil
}

func (m *Manager) Middleware() middleware.Middleware {
	return func(next middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (any, error) {
			token := bearerToken(ctx)
			if token != "" {
				if claims, err := m.ParseAccessToken(token); err == nil {
					ctx = WithClaims(ctx, claims)
				}
			}
			return next(ctx, req)
		}
	}
}

func (m *Manager) signAccessToken(user *v1.User, issuedAt, expiresAt time.Time) (string, error) {
	claims := Claims{
		UserID:    user.GetId(),
		Username:  user.GetUsername(),
		Nickname:  user.GetNickname(),
		RoleName:  RoleName(user.GetRole()),
		TokenType: "access",
		Issuer:    m.issuer,
		IssuedAt:  issuedAt.Unix(),
		ExpiresAt: expiresAt.Unix(),
	}
	header := map[string]string{"alg": "HS256", "typ": "JWT"}
	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", err
	}
	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	signed := base64.RawURLEncoding.EncodeToString(headerJSON) + "." + base64.RawURLEncoding.EncodeToString(claimsJSON)
	return signed + "." + m.sign(signed), nil
}

func (m *Manager) sign(value string) string {
	mac := hmac.New(sha256.New, m.secret)
	_, _ = mac.Write([]byte(value))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func VerifyPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

func NewOpaqueToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func HashRefreshToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func WithClaims(ctx context.Context, claims *Claims) context.Context {
	return context.WithValue(ctx, contextKey{}, claims)
}

func ClaimsFromContext(ctx context.Context) (*Claims, bool) {
	claims, ok := ctx.Value(contextKey{}).(*Claims)
	return claims, ok && claims != nil
}

func RequireUser(ctx context.Context) (*Claims, error) {
	claims, ok := ClaimsFromContext(ctx)
	if !ok {
		return nil, apperr.ErrUnauthenticated
	}
	return claims, nil
}

func RequireRole(ctx context.Context, roles ...v1.UserRole) (*Claims, error) {
	claims, err := RequireUser(ctx)
	if err != nil {
		return nil, err
	}
	for _, role := range roles {
		if claims.Role == role {
			return claims, nil
		}
	}
	return nil, apperr.ErrPermissionDenied
}

func RoleName(role v1.UserRole) string {
	switch role {
	case v1.UserRole_USER_ROLE_BUYER:
		return "buyer"
	case v1.UserRole_USER_ROLE_ANCHOR:
		return "anchor"
	case v1.UserRole_USER_ROLE_OPERATOR:
		return "operator"
	case v1.UserRole_USER_ROLE_ADMIN:
		return "admin"
	default:
		return "unspecified"
	}
}

func RoleFromName(name string) (v1.UserRole, bool) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "buyer":
		return v1.UserRole_USER_ROLE_BUYER, true
	case "anchor":
		return v1.UserRole_USER_ROLE_ANCHOR, true
	case "operator":
		return v1.UserRole_USER_ROLE_OPERATOR, true
	case "admin":
		return v1.UserRole_USER_ROLE_ADMIN, true
	default:
		return v1.UserRole_USER_ROLE_UNSPECIFIED, false
	}
}

func bearerToken(ctx context.Context) string {
	tr, ok := transport.FromServerContext(ctx)
	if !ok || tr.RequestHeader() == nil {
		return ""
	}
	value := tr.RequestHeader().Get("Authorization")
	if value == "" {
		value = tr.RequestHeader().Get("authorization")
	}
	const prefix = "Bearer "
	if !strings.HasPrefix(value, prefix) {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(value, prefix))
}
