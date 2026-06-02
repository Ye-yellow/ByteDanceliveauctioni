package user

import (
	"slices"
	"strings"

	v1 "live-auction-bid/backend/api/auction/service/v1"
)

const (
	RoleMerchantOwner = "merchant_owner"
	RoleAnchor        = "anchor"
	RoleOperator      = "operator"
	RoleBuyer         = "buyer"
)

const (
	PermissionTeamUserCreate        = "team.user.create"
	PermissionTeamUserList          = "team.user.list"
	PermissionTeamUserUpdateRole    = "team.user.update_role"
	PermissionTeamUserUpdateStatus  = "team.user.update_status"
	PermissionTeamUserResetPassword = "team.user.reset_password"
	PermissionLotCreate             = "lot.create"
	PermissionLotUpdate             = "lot.update"
	PermissionLotQueue              = "lot.queue"
	PermissionLotViewAdmin          = "lot.view_admin"
	PermissionAuctionControl        = "auction.control"
	PermissionOrderManage           = "order.manage"
	PermissionRealtimeView          = "realtime.view"
	PermissionUploadImage           = "upload.image"
	PermissionBidPlace              = "bid.place"
	PermissionOrderPay              = "order.pay"
	PermissionOrderViewOwn          = "order.view_own"
)

var rolePermissions = map[string][]string{
	RoleMerchantOwner: {
		PermissionTeamUserCreate,
		PermissionTeamUserList,
		PermissionTeamUserUpdateRole,
		PermissionTeamUserUpdateStatus,
		PermissionTeamUserResetPassword,
		PermissionLotCreate,
		PermissionLotUpdate,
		PermissionLotQueue,
		PermissionLotViewAdmin,
		PermissionAuctionControl,
		PermissionOrderManage,
		PermissionRealtimeView,
		PermissionUploadImage,
	},
	RoleAnchor: {
		PermissionLotViewAdmin,
		PermissionAuctionControl,
		PermissionOrderManage,
		PermissionRealtimeView,
	},
	RoleOperator: {
		PermissionLotCreate,
		PermissionLotUpdate,
		PermissionLotQueue,
		PermissionLotViewAdmin,
		PermissionOrderManage,
		PermissionRealtimeView,
		PermissionUploadImage,
	},
	RoleBuyer: {
		PermissionBidPlace,
		PermissionOrderPay,
		PermissionOrderViewOwn,
	},
}

func DefaultRolePermissions() map[string][]string {
	out := make(map[string][]string, len(rolePermissions))
	for role, permissions := range rolePermissions {
		out[role] = slices.Clone(permissions)
	}
	return out
}

func PermissionsForRole(roleCode string) []string {
	roleCode = NormalizeRoleCode(roleCode)
	return slices.Clone(rolePermissions[roleCode])
}

func NormalizeRoleCode(roleCode string) string {
	return strings.ToLower(strings.TrimSpace(roleCode))
}

func NormalizePermissionCode(permissionCode string) string {
	return strings.ToLower(strings.TrimSpace(permissionCode))
}

func IsKnownRole(roleCode string) bool {
	_, ok := rolePermissions[NormalizeRoleCode(roleCode)]
	return ok
}

func IsManagedTeamRole(roleCode string) bool {
	switch NormalizeRoleCode(roleCode) {
	case RoleAnchor, RoleOperator:
		return true
	default:
		return false
	}
}

func IsBackofficeRole(roleCode string) bool {
	switch NormalizeRoleCode(roleCode) {
	case RoleMerchantOwner, RoleAnchor, RoleOperator:
		return true
	default:
		return false
	}
}

func UserHasRole(user *v1.User, roleCode string) bool {
	if user == nil {
		return false
	}
	roleCode = NormalizeRoleCode(roleCode)
	for _, got := range user.GetRoleCodes() {
		if NormalizeRoleCode(got) == roleCode {
			return true
		}
	}
	return false
}

func UserHasPermission(user *v1.User, permissionCode string) bool {
	if user == nil {
		return false
	}
	permissionCode = NormalizePermissionCode(permissionCode)
	for _, got := range user.GetPermissionCodes() {
		if NormalizePermissionCode(got) == permissionCode {
			return true
		}
	}
	return false
}
