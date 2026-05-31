package test

import (
	v1 "live-auction-bid/backend/api/auction/service/v1"
	userbiz "live-auction-bid/backend/app/auction/service/internal/biz/user"
	"live-auction-bid/backend/app/auction/service/internal/pkg/auth"
)

func claimsForRoleCode(userID, username, nickname, roleCode, mainAccountID string) *auth.Claims {
	return &auth.Claims{
		UserID:          userID,
		Username:        username,
		Nickname:        nickname,
		RoleCodes:       []string{roleCode},
		PermissionCodes: userbiz.PermissionsForRole(roleCode),
		MainAccountID:   mainAccountID,
		Status:          v1.UserStatus_USER_STATUS_ACTIVE,
	}
}

func buyerUserForTest(id, username, nickname string) *v1.User {
	return &v1.User{
		Id:              id,
		Username:        username,
		Nickname:        nickname,
		RoleCodes:       []string{userbiz.RoleBuyer},
		PermissionCodes: userbiz.PermissionsForRole(userbiz.RoleBuyer),
		Status:          v1.UserStatus_USER_STATUS_ACTIVE,
	}
}
