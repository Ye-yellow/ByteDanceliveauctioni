package realtime

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"

	v1 "live-auction-bid/backend/api/auction/service/v1"
	"live-auction-bid/backend/app/auction/service/internal/pkg/auth"
)

var (
	errTicketUnavailable = errors.New("websocket ticket signing secret is not configured")
	errTicketInvalid     = errors.New("invalid websocket ticket")
	errTicketExpired     = errors.New("websocket ticket expired")
)

type wsTicketClaims struct {
	RoomID          string   `json:"room_id"`
	Scope           string   `json:"scope"`
	UserID          string   `json:"user_id"`
	MainAccountID   string   `json:"main_account_id,omitempty"`
	RoleCodes       []string `json:"role_codes,omitempty"`
	PermissionCodes []string `json:"permission_codes,omitempty"`
	IssuedAtUnixMs  int64    `json:"iat_ms"`
	ExpiresAtUnixMs int64    `json:"exp_ms"`
	Nonce           string   `json:"nonce"`
}

type wsTicketCodec struct {
	secret []byte
	ttl    time.Duration
	now    func() time.Time
}

func newWSTicketCodec(cfg Config) wsTicketCodec {
	return wsTicketCodec{
		secret: []byte(strings.TrimSpace(cfg.TicketSecret)),
		ttl:    cfg.TicketTTL,
		now:    time.Now,
	}
}

func (c wsTicketCodec) issue(input wsTicketClaims) (string, int64, error) {
	if len(c.secret) == 0 {
		return "", 0, errTicketUnavailable
	}
	now := c.now()
	input.IssuedAtUnixMs = now.UnixMilli()
	input.ExpiresAtUnixMs = now.Add(c.ttl).UnixMilli()
	input.Nonce = randomNonce()
	payload, err := json.Marshal(input)
	if err != nil {
		return "", 0, err
	}
	encodedPayload := base64.RawURLEncoding.EncodeToString(payload)
	signature := c.sign(encodedPayload)
	return encodedPayload + "." + signature, input.ExpiresAtUnixMs, nil
}

func (c wsTicketCodec) parse(ticket, expectedRoomID, expectedScope string) (wsTicketClaims, error) {
	if len(c.secret) == 0 {
		return wsTicketClaims{}, errTicketUnavailable
	}
	parts := strings.Split(ticket, ".")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return wsTicketClaims{}, errTicketInvalid
	}
	if !hmac.Equal([]byte(parts[1]), []byte(c.sign(parts[0]))) {
		return wsTicketClaims{}, errTicketInvalid
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return wsTicketClaims{}, errTicketInvalid
	}
	var claims wsTicketClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return wsTicketClaims{}, errTicketInvalid
	}
	if claims.RoomID != expectedRoomID || claims.Scope != expectedScope || claims.UserID == "" {
		return wsTicketClaims{}, errTicketInvalid
	}
	if claims.ExpiresAtUnixMs <= c.now().UnixMilli() {
		return wsTicketClaims{}, errTicketExpired
	}
	return claims, nil
}

func (c wsTicketCodec) sign(payload string) string {
	mac := hmac.New(sha256.New, c.secret)
	_, _ = mac.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func randomNonce() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return base64.RawURLEncoding.EncodeToString([]byte(time.Now().Format(time.RFC3339Nano)))
	}
	return base64.RawURLEncoding.EncodeToString(buf)
}

func authContextFromTicketClaims(ticket string, claims wsTicketClaims) auth.AuthContext {
	authClaims := &auth.Claims{
		UserID:          claims.UserID,
		RoleCodes:       append([]string(nil), claims.RoleCodes...),
		PermissionCodes: append([]string(nil), claims.PermissionCodes...),
		MainAccountID:   claims.MainAccountID,
		Status:          v1.UserStatus_USER_STATUS_ACTIVE,
		StatusName:      auth.StatusName(v1.UserStatus_USER_STATUS_ACTIVE),
		TokenType:       "ws_ticket",
	}
	return auth.AuthContext{
		Claims:      authClaims,
		TokenStatus: auth.TokenStatusValid,
		RawToken:    ticket,
		UserID:      authClaims.UserID,
		RoleCodes:   append([]string(nil), authClaims.RoleCodes...),
	}
}
