package e2e

import (
	"net/http"
	"strings"
	"testing"
)

func TestAuthRegisterLoginRefreshLogoutContract(t *testing.T) {
	c := newClient(t)
	buyer := registerBuyer(t, c, "buyer")

	me, resp := c.get(t, "/api/users/me", buyer.AccessToken)
	assertHTTPStatus(t, resp, http.StatusOK)
	assertOK(t, me)
	user := objectField(t, me, "user")
	if got := stringField(user, "username"); got != buyer.Username {
		t.Fatalf("expected me username %q, got %q", buyer.Username, got)
	}

	wrongPassword, resp := c.post(t, "/api/users/login", "", map[string]any{
		"username": buyer.Username,
		"password": "wrong-password",
	})
	assertHTTPStatus(t, resp, http.StatusOK)
	assertResultCode(t, wrongPassword, resultCodeInvalidCredentials)

	refreshed, resp := c.post(t, "/api/users/refresh", "", map[string]any{
		"refresh_token": buyer.RefreshToken,
	})
	assertHTTPStatus(t, resp, http.StatusOK)
	assertOK(t, refreshed)
	refreshedTokens := objectField(t, refreshed, "tokens")
	nextRefresh := stringField(refreshedTokens, "refreshToken", "refresh_token")
	if nextRefresh == "" || nextRefresh == buyer.RefreshToken {
		t.Fatalf("expected refresh token rotation, original=%q next=%q", buyer.RefreshToken, nextRefresh)
	}

	reused, resp := c.post(t, "/api/users/refresh", "", map[string]any{
		"refresh_token": buyer.RefreshToken,
	})
	assertHTTPStatus(t, resp, http.StatusOK)
	assertResultCode(t, reused, resultCodeSessionExpired)

	logout, resp := c.post(t, "/api/users/logout", "", map[string]any{
		"refresh_token": nextRefresh,
	})
	assertHTTPStatus(t, resp, http.StatusOK)
	assertOK(t, logout)

	afterLogout, resp := c.post(t, "/api/users/refresh", "", map[string]any{
		"refresh_token": nextRefresh,
	})
	assertHTTPStatus(t, resp, http.StatusOK)
	assertResultCode(t, afterLogout, resultCodeSessionExpired)
}

func TestAuthDuplicateUsernameAndResetPasswordContract(t *testing.T) {
	c := newClient(t)
	buyer := registerBuyer(t, c, "buyer_reset")

	duplicate, resp := c.post(t, "/api/users/register", "", map[string]any{
		"username": buyer.Username,
		"password": "password123",
		"nickname": "duplicate",
	})
	assertHTTPStatus(t, resp, http.StatusOK)
	assertResultCode(t, duplicate, resultCodeUsernameTaken)

	reset, resp := c.post(t, "/api/users/reset-password", "", map[string]any{
		"username": buyer.Username,
		"password": "newpassword123",
	})
	assertHTTPStatus(t, resp, http.StatusOK)
	assertOK(t, reset)

	oldLogin, resp := c.post(t, "/api/users/login", "", map[string]any{
		"username": buyer.Username,
		"password": buyer.Password,
	})
	assertHTTPStatus(t, resp, http.StatusOK)
	assertResultCode(t, oldLogin, resultCodeInvalidCredentials)

	_ = login(t, c, buyer.Username, "newpassword123")
}

func TestAuthRegisterValidationContract(t *testing.T) {
	c := newClient(t)

	tests := []struct {
		name string
		body map[string]any
	}{
		{
			name: "buyer username shorter than public minimum",
			body: map[string]any{
				"username": "abc",
				"password": "password123",
				"nickname": "短用户名",
			},
		},
		{
			name: "username contains unsupported character",
			body: map[string]any{
				"username": uniqueUsername("bad") + "$",
				"password": "password123",
				"nickname": "非法字符",
			},
		},
		{
			name: "password too short",
			body: map[string]any{
				"username": uniqueUsername("shortpw"),
				"password": "short",
				"nickname": "短密码",
			},
		},
		{
			name: "nickname missing",
			body: map[string]any{
				"username": uniqueUsername("nonick"),
				"password": "password123",
				"nickname": "   ",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, resp := c.post(t, "/api/users/register", "", tt.body)
			assertHTTPStatus(t, resp, http.StatusOK)
			assertResultCode(t, body, resultCodeInvalidArgument)
		})
	}
}

func TestAuthLoginFailureAndUsernameNormalizationContract(t *testing.T) {
	c := newClient(t)
	buyer := registerBuyer(t, c, "login_case")

	tests := []struct {
		name string
		body map[string]any
	}{
		{
			name: "unknown username",
			body: map[string]any{"username": uniqueUsername("missing_login"), "password": "password123"},
		},
		{
			name: "wrong password",
			body: map[string]any{"username": buyer.Username, "password": "wrong-password"},
		},
		{
			name: "blank username",
			body: map[string]any{"username": " ", "password": "password123"},
		},
		{
			name: "blank password",
			body: map[string]any{"username": buyer.Username, "password": ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, resp := c.post(t, "/api/users/login", "", tt.body)
			assertHTTPStatus(t, resp, http.StatusOK)
			assertResultCode(t, body, resultCodeInvalidCredentials)
		})
	}

	normalized, resp := c.post(t, "/api/users/login", "", map[string]any{
		"username": "  " + strings.ToUpper(buyer.Username) + "  ",
		"password": buyer.Password,
	})
	assertHTTPStatus(t, resp, http.StatusOK)
	assertOK(t, normalized)
	user := objectField(t, normalized, "user")
	if got := stringField(user, "username"); got != buyer.Username {
		t.Fatalf("expected normalized username %q, got %q body=%+v", buyer.Username, got, normalized)
	}
}

func TestAuthRefreshLogoutAndBearerFailureContract(t *testing.T) {
	c := newClient(t)

	tests := []struct {
		name string
		body map[string]any
		want int32
	}{
		{
			name: "empty refresh token",
			body: map[string]any{"refresh_token": ""},
			want: resultCodeInvalidArgument,
		},
		{
			name: "unknown refresh token",
			body: map[string]any{"refresh_token": "not-a-real-refresh-token"},
			want: resultCodeSessionExpired,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, resp := c.post(t, "/api/users/refresh", "", tt.body)
			assertHTTPStatus(t, resp, http.StatusOK)
			assertResultCode(t, body, tt.want)
		})
	}

	logoutEmpty, resp := c.post(t, "/api/users/logout", "", map[string]any{"refresh_token": ""})
	assertHTTPStatus(t, resp, http.StatusOK)
	assertOK(t, logoutEmpty)

	meWithBadBearer, resp := c.get(t, "/api/users/me", "not-a-jwt")
	assertHTTPStatus(t, resp, http.StatusOK)
	assertResultCode(t, meWithBadBearer, resultCodeTokenInvalid)
}
