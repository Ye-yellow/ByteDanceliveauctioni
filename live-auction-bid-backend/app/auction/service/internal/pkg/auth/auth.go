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
	"slices"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
	"golang.org/x/crypto/bcrypt"
	v1 "live-auction-bid/backend/api/auction/service/v1"
	"live-auction-bid/backend/app/auction/service/internal/pkg/apperr"
	"live-auction-bid/backend/app/auction/service/internal/pkg/requestctx"
)

const (
	DefaultAccessTTL  = 15 * time.Minute
	DefaultRefreshTTL = 7 * 24 * time.Hour

	RoleMerchantOwner = "merchant_owner"
)

type contextKey struct{}
type authContextKey struct{}

type TokenStatus string

const (
	TokenStatusNone    TokenStatus = "none"
	TokenStatusValid   TokenStatus = "valid"
	TokenStatusExpired TokenStatus = "expired"
	TokenStatusInvalid TokenStatus = "invalid"
)

// AuthContext captures authentication state for both public and protected routes.
// Middleware always writes it, so downstream code can distinguish anonymous, expired,
// invalid, and valid sessions without parsing HTTP headers.
type AuthContext struct {
	Claims      *Claims
	TokenStatus TokenStatus
	RawToken    string
	UserID      string
	RoleCodes   []string
}

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
	UserID          string        `json:"sub"`
	Username        string        `json:"username"`
	Nickname        string        `json:"nickname"`
	AvatarURL       string        `json:"avatar_url,omitempty"`
	RoleCodes       []string      `json:"role_codes"`
	PermissionCodes []string      `json:"permission_codes"`
	MainAccountID   string        `json:"main_account_id,omitempty"`
	Status          v1.UserStatus `json:"-"`
	StatusName      string        `json:"status"`
	TokenType       string        `json:"typ"`
	Issuer          string        `json:"iss"`
	IssuedAt        int64         `json:"iat"`
	ExpiresAt       int64         `json:"exp"`
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
	if len(user.GetRoleCodes()) == 0 {
		return TokenPair{}, fmt.Errorf("%w: user role codes are required", apperr.ErrInvalidArgument)
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
	if claims.TokenType != "access" || claims.UserID == "" || len(claims.RoleCodes) == 0 {
		return nil, apperr.ErrInvalidToken
	}
	if m.issuer != "" && claims.Issuer != m.issuer {
		return nil, apperr.ErrInvalidToken
	}
	if claims.ExpiresAt <= m.now().Unix() {
		return nil, apperr.ErrTokenExpired
	}
	claims.RoleCodes = normalizeCodes(claims.RoleCodes)
	claims.PermissionCodes = normalizeCodes(claims.PermissionCodes)
	status, ok := StatusFromName(claims.StatusName)
	if !ok {
		return nil, apperr.ErrInvalidToken
	}
	claims.Status = status
	return &claims, nil
}

func (m *Manager) Middleware() middleware.Middleware {
	return func(next middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (any, error) {
			authCtx := m.AuthContextFromBearer(bearerAuthorization(ctx))
			if authCtx.TokenStatus == TokenStatusValid {
				ctx = WithClaims(ctx, authCtx.Claims)
			}
			ctx = WithAuthContext(ctx, authCtx)
			return next(ctx, req)
		}
	}
}

func (m *Manager) WithAuthContextFromBearer(ctx context.Context, authorization string) context.Context {
	authCtx := m.AuthContextFromBearer(authorization)
	if authCtx.TokenStatus == TokenStatusValid {
		ctx = WithClaims(ctx, authCtx.Claims)
	}
	return WithAuthContext(ctx, authCtx)
}

func (m *Manager) AuthContextFromBearer(authorization string) AuthContext {
	authCtx := AuthContext{TokenStatus: TokenStatusNone}
	token := BearerToken(authorization)
	if token == "" {
		return authCtx
	}
	authCtx.RawToken = token
	claims, err := m.ParseAccessToken(token)
	switch {
	case err == nil:
		authCtx.Claims = claims
		authCtx.TokenStatus = TokenStatusValid
		authCtx.UserID = claims.UserID
		authCtx.RoleCodes = slices.Clone(claims.RoleCodes)
	case apperr.IsTokenExpired(err):
		authCtx.TokenStatus = TokenStatusExpired
	default:
		authCtx.TokenStatus = TokenStatusInvalid
	}
	return authCtx
}

func (m *Manager) signAccessToken(user *v1.User, issuedAt, expiresAt time.Time) (string, error) {
	claims := Claims{
		UserID:          user.GetId(),
		Username:        user.GetUsername(),
		Nickname:        user.GetNickname(),
		AvatarURL:       user.GetAvatarUrl(),
		RoleCodes:       normalizeCodes(user.GetRoleCodes()),
		PermissionCodes: normalizeCodes(user.GetPermissionCodes()),
		MainAccountID:   user.GetMainAccountId(),
		StatusName:      StatusName(effectiveStatus(user.GetStatus())),
		TokenType:       "access",
		Issuer:          m.issuer,
		IssuedAt:        issuedAt.Unix(),
		ExpiresAt:       expiresAt.Unix(),
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
	ctx = context.WithValue(ctx, contextKey{}, claims)
	if claims != nil {
		ctx = WithAuthContext(ctx, AuthContext{Claims: claims, TokenStatus: TokenStatusValid, UserID: claims.UserID, RoleCodes: slices.Clone(claims.RoleCodes)})
	}
	return ctx
}

func WithAuthContext(ctx context.Context, authCtx AuthContext) context.Context {
	if authCtx.TokenStatus == TokenStatusValid && authCtx.Claims != nil {
		authCtx.UserID = authCtx.Claims.UserID
		authCtx.RoleCodes = slices.Clone(authCtx.Claims.RoleCodes)
		ctx = requestctx.WithUser(ctx, authCtx.UserID, primaryRoleCode(authCtx.RoleCodes))
	}
	return context.WithValue(ctx, authContextKey{}, authCtx)
}

func AuthContextFromContext(ctx context.Context) (AuthContext, bool) {
	authCtx, ok := ctx.Value(authContextKey{}).(AuthContext)
	return authCtx, ok
}

func ClaimsFromContext(ctx context.Context) (*Claims, bool) {
	if authCtx, ok := AuthContextFromContext(ctx); ok && authCtx.Claims != nil && authCtx.TokenStatus == TokenStatusValid {
		return authCtx.Claims, true
	}
	claims, ok := ctx.Value(contextKey{}).(*Claims)
	return claims, ok && claims != nil
}

func RequireUser(ctx context.Context) (*Claims, error) {
	if authCtx, ok := AuthContextFromContext(ctx); ok {
		switch authCtx.TokenStatus {
		case TokenStatusValid:
			if authCtx.Claims != nil {
				if effectiveStatus(authCtx.Claims.Status) == v1.UserStatus_USER_STATUS_DISABLED {
					return nil, apperr.ErrAccountDisabled
				}
				return authCtx.Claims, nil
			}
			return nil, apperr.ErrInvalidToken
		case TokenStatusExpired:
			return nil, apperr.ErrTokenExpired
		case TokenStatusInvalid:
			return nil, apperr.ErrInvalidToken
		case TokenStatusNone:
			return nil, apperr.ErrUnauthenticated
		}
	}
	claims, ok := ClaimsFromContext(ctx)
	if !ok {
		return nil, apperr.ErrUnauthenticated
	}
	if effectiveStatus(claims.Status) == v1.UserStatus_USER_STATUS_DISABLED {
		return nil, apperr.ErrAccountDisabled
	}
	return claims, nil
}

func RequirePermission(ctx context.Context, permissionCode string) (*Claims, error) {
	claims, err := RequireUser(ctx)
	if err != nil {
		return nil, err
	}
	if HasPermission(claims, permissionCode) {
		return claims, nil
	}
	return nil, apperr.ErrPermissionDenied
}

func HasPermission(claims *Claims, permissionCode string) bool {
	if claims == nil {
		return false
	}
	permissionCode = normalizeCode(permissionCode)
	for _, got := range claims.PermissionCodes {
		if normalizeCode(got) == permissionCode {
			return true
		}
	}
	return false
}

func HasAnyPermission(claims *Claims, permissionCodes ...string) bool {
	for _, permissionCode := range permissionCodes {
		if HasPermission(claims, permissionCode) {
			return true
		}
	}
	return false
}

func HasRoleCode(claims *Claims, roleCode string) bool {
	if claims == nil {
		return false
	}
	roleCode = normalizeCode(roleCode)
	for _, got := range claims.RoleCodes {
		if normalizeCode(got) == roleCode {
			return true
		}
	}
	return false
}

func EffectiveMainAccountID(claims *Claims) string {
	if claims == nil {
		return ""
	}
	if strings.TrimSpace(claims.MainAccountID) != "" {
		return strings.TrimSpace(claims.MainAccountID)
	}
	if HasRoleCode(claims, RoleMerchantOwner) {
		return strings.TrimSpace(claims.UserID)
	}
	return ""
}

func StatusName(status v1.UserStatus) string {
	switch status {
	case v1.UserStatus_USER_STATUS_ACTIVE:
		return "active"
	case v1.UserStatus_USER_STATUS_DISABLED:
		return "disabled"
	default:
		return "unspecified"
	}
}

func StatusFromName(name string) (v1.UserStatus, bool) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "", "active":
		return v1.UserStatus_USER_STATUS_ACTIVE, true
	case "disabled":
		return v1.UserStatus_USER_STATUS_DISABLED, true
	case "unspecified":
		return v1.UserStatus_USER_STATUS_UNSPECIFIED, true
	default:
		return v1.UserStatus_USER_STATUS_UNSPECIFIED, false
	}
}

func effectiveStatus(status v1.UserStatus) v1.UserStatus {
	if status == v1.UserStatus_USER_STATUS_UNSPECIFIED {
		return v1.UserStatus_USER_STATUS_ACTIVE
	}
	return status
}

func bearerAuthorization(ctx context.Context) string {
	tr, ok := transport.FromServerContext(ctx)
	if !ok || tr.RequestHeader() == nil {
		return ""
	}
	value := tr.RequestHeader().Get("Authorization")
	if value == "" {
		value = tr.RequestHeader().Get("authorization")
	}
	return value
}

func BearerToken(value string) string {
	const prefix = "Bearer "
	if !strings.HasPrefix(value, prefix) {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(value, prefix))
}

func normalizeCodes(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		next := normalizeCode(value)
		if next == "" {
			continue
		}
		if _, ok := seen[next]; ok {
			continue
		}
		seen[next] = struct{}{}
		out = append(out, next)
	}
	slices.Sort(out)
	return out
}

func normalizeCode(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func primaryRoleCode(roleCodes []string) string {
	if len(roleCodes) == 0 {
		return ""
	}
	for _, preferred := range []string{RoleMerchantOwner, "anchor", "operator", "buyer"} {
		for _, got := range roleCodes {
			if normalizeCode(got) == preferred {
				return preferred
			}
		}
	}
	return normalizeCode(roleCodes[0])
}
