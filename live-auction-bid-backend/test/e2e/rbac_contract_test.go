package e2e

import (
	"net/http"
	"testing"
)

func TestRBACMerchantTeamUserContract(t *testing.T) {
	c := newClient(t)
	merchant := registerMerchant(t, c, "merchant")

	tests := []struct {
		name     string
		roleCode string
		wantCode int32
	}{
		{name: "anchor subaccount allowed", roleCode: "anchor", wantCode: resultCodeOK},
		{name: "operator subaccount allowed", roleCode: "operator", wantCode: resultCodeOK},
		{name: "buyer cannot be created from admin", roleCode: "buyer", wantCode: resultCodeInvalidArgument},
		{name: "merchant owner cannot be created as subaccount", roleCode: "merchant_owner", wantCode: resultCodeInvalidArgument},
		{name: "blank role rejected", roleCode: "", wantCode: resultCodeInvalidArgument},
	}

	createdCount := 0
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			created, resp := c.post(t, "/api/admin/users", merchant.AccessToken, map[string]any{
				"username":  uniqueUsername("team"),
				"password":  "teampass123",
				"nickname":  "团队用户",
				"role_code": tt.roleCode,
			})
			assertHTTPStatus(t, resp, http.StatusOK)
			assertResultCode(t, created, tt.wantCode)
			if tt.wantCode != resultCodeOK {
				return
			}
			createdCount++
			createdUser := objectField(t, created, "user")
			if !containsString(stringSliceField(createdUser, "roleCodes", "role_codes"), tt.roleCode) {
				t.Fatalf("created team user should have %q role, got %+v", tt.roleCode, createdUser)
			}
			if got := stringField(createdUser, "mainAccountId", "main_account_id"); got != merchant.UserID {
				t.Fatalf("created team user should be scoped to merchant %q, got %q", merchant.UserID, got)
			}
		})
	}

	list, resp := c.get(t, "/api/admin/users?page=1&pageSize=20", merchant.AccessToken)
	assertHTTPStatus(t, resp, http.StatusOK)
	assertOK(t, list)
	if total := numberField(t, list, "total"); total < int64(createdCount) {
		t.Fatalf("expected at least %d team users in admin list, got %+v", createdCount, list)
	}
}

func TestRBACBuyerCannotUseAdminEndpoints(t *testing.T) {
	c := newClient(t)
	buyer := registerBuyer(t, c, "rbac_buyer")

	noToken, resp := c.get(t, "/api/admin/users", "")
	assertHTTPStatus(t, resp, http.StatusOK)
	assertResultCode(t, noToken, resultCodeLoginRequired)

	forbidden, resp := c.get(t, "/api/admin/users", buyer.AccessToken)
	assertHTTPStatus(t, resp, http.StatusOK)
	assertResultCode(t, forbidden, resultCodeForbidden)
}

func TestRBACTeamUserLifecycleAndIsolationContract(t *testing.T) {
	c := newClient(t)
	merchant := registerMerchant(t, c, "rbac_lifecycle")
	otherMerchant := registerMerchant(t, c, "rbac_other")
	teamUser := createTeamUser(t, c, merchant, "anchor")
	teamUserID := stringField(teamUser, "id")
	if teamUserID == "" {
		t.Fatalf("created team user missing id: %+v", teamUser)
	}

	crossMerchantUpdate, resp := c.post(t, "/api/admin/users/"+teamUserID+"/role", otherMerchant.AccessToken, map[string]any{
		"role_code": "operator",
	})
	assertHTTPStatus(t, resp, http.StatusOK)
	assertResultCode(t, crossMerchantUpdate, resultCodeForbidden)

	invalidRole, resp := c.post(t, "/api/admin/users/"+teamUserID+"/role", merchant.AccessToken, map[string]any{
		"role_code": "buyer",
	})
	assertHTTPStatus(t, resp, http.StatusOK)
	assertResultCode(t, invalidRole, resultCodeInvalidArgument)

	updatedRole, resp := c.post(t, "/api/admin/users/"+teamUserID+"/role", merchant.AccessToken, map[string]any{
		"role_code": "operator",
	})
	assertHTTPStatus(t, resp, http.StatusOK)
	assertOK(t, updatedRole)
	if roles := stringSliceField(objectField(t, updatedRole, "user"), "roleCodes", "role_codes"); !containsString(roles, "operator") {
		t.Fatalf("expected operator role after update, got %+v", updatedRole)
	}

	shortPassword, resp := c.post(t, "/api/admin/users/"+teamUserID+"/reset-password", merchant.AccessToken, map[string]any{
		"password": "short",
	})
	assertHTTPStatus(t, resp, http.StatusOK)
	assertResultCode(t, shortPassword, resultCodeInvalidArgument)

	reset, resp := c.post(t, "/api/admin/users/"+teamUserID+"/reset-password", merchant.AccessToken, map[string]any{
		"password": "newteampass123",
	})
	assertHTTPStatus(t, resp, http.StatusOK)
	assertOK(t, reset)

	disabled, resp := c.post(t, "/api/admin/users/"+teamUserID+"/status", merchant.AccessToken, map[string]any{
		"status": 2,
	})
	assertHTTPStatus(t, resp, http.StatusOK)
	assertOK(t, disabled)
	if got := stringField(objectField(t, disabled, "user"), "status"); got != "USER_STATUS_DISABLED" {
		t.Fatalf("expected disabled status, got %q body=%+v", got, disabled)
	}

	disabledLogin, resp := c.post(t, "/api/users/login", "", map[string]any{
		"username": stringField(teamUser, "username"),
		"password": "newteampass123",
	})
	assertHTTPStatus(t, resp, http.StatusOK)
	assertResultCode(t, disabledLogin, resultCodeAccountDisabled)

	enabled, resp := c.post(t, "/api/admin/users/"+teamUserID+"/status", merchant.AccessToken, map[string]any{
		"status": 1,
	})
	assertHTTPStatus(t, resp, http.StatusOK)
	assertOK(t, enabled)

	activeLogin, resp := c.post(t, "/api/users/login", "", map[string]any{
		"username": stringField(teamUser, "username"),
		"password": "newteampass123",
	})
	assertHTTPStatus(t, resp, http.StatusOK)
	assertOK(t, activeLogin)
}

func TestRBACListUsersFiltersAndPaginationContract(t *testing.T) {
	c := newClient(t)
	merchant := registerMerchant(t, c, "rbac_list")
	_ = createTeamUser(t, c, merchant, "anchor")
	_ = createTeamUser(t, c, merchant, "operator")

	anchorList, resp := c.get(t, "/api/admin/users?page=1&pageSize=1&roleCode=anchor", merchant.AccessToken)
	assertHTTPStatus(t, resp, http.StatusOK)
	assertOK(t, anchorList)
	if page := numberField(t, anchorList, "page"); page != 1 {
		t.Fatalf("expected page 1, got %+v", anchorList)
	}
	if size := numberField(t, anchorList, "pageSize", "size"); size != 1 {
		t.Fatalf("expected pageSize 1, got %+v", anchorList)
	}
	if total := numberField(t, anchorList, "total"); total < 1 {
		t.Fatalf("expected at least one anchor, got %+v", anchorList)
	}
	if users := arrayField(t, anchorList, "users"); len(users) > 1 {
		t.Fatalf("pageSize=1 should return at most one user, got %+v", anchorList)
	}

	invalidRoleFilter, resp := c.get(t, "/api/admin/users?roleCode=buyer", merchant.AccessToken)
	assertHTTPStatus(t, resp, http.StatusOK)
	assertResultCode(t, invalidRoleFilter, resultCodeInvalidArgument)

	activeList, resp := c.get(t, "/api/admin/users?status=USER_STATUS_ACTIVE&page=1&pageSize=5", merchant.AccessToken)
	assertHTTPStatus(t, resp, http.StatusOK)
	assertOK(t, activeList)
	if total := numberField(t, activeList, "total"); total < 1 {
		t.Fatalf("expected at least one active team user, got %+v", activeList)
	}

	emptyList, resp := c.get(t, "/api/admin/users?keyword="+uniqueUsername("not_found")+"&page=1&pageSize=5", merchant.AccessToken)
	assertHTTPStatus(t, resp, http.StatusOK)
	assertOK(t, emptyList)
	if total := numberField(t, emptyList, "total"); total != 0 {
		t.Fatalf("expected keyword query to return empty result, got %+v", emptyList)
	}
	if users := arrayField(t, emptyList, "users"); len(users) != 0 {
		t.Fatalf("expected empty users array for empty keyword query, got %+v", emptyList)
	}

	normalized, resp := c.get(t, "/api/admin/users?page=0&pageSize=200", merchant.AccessToken)
	assertHTTPStatus(t, resp, http.StatusOK)
	assertOK(t, normalized)
	if page := numberField(t, normalized, "page"); page != 1 {
		t.Fatalf("expected invalid page to normalize to 1, got %+v", normalized)
	}
	if size := numberField(t, normalized, "pageSize", "size"); size != 100 {
		t.Fatalf("expected pageSize to cap at 100, got %+v", normalized)
	}
}
