package data

import (
	"context"
	"time"

	"gorm.io/gorm/clause"
	userbiz "live-auction-bid/backend/app/auction/service/internal/biz/user"
	"live-auction-bid/backend/app/auction/service/internal/pkg/idgen"
)

func (s *Store) EnsureRBACDefaults(ctx context.Context) error {
	nowMs := time.Now().UnixMilli()
	roles := []AuctionRoleModel{
		{Code: userbiz.RoleMerchantOwner, Name: "商家主账号", Description: "商家/主播空间主账号", System: true, CreatedAtUnixMs: nowMs, UpdatedAtUnixMs: nowMs},
		{Code: userbiz.RoleAnchor, Name: "主播/场控", Description: "直播控场子账号", System: true, CreatedAtUnixMs: nowMs, UpdatedAtUnixMs: nowMs},
		{Code: userbiz.RoleOperator, Name: "运营子账号", Description: "拍品与订单运营子账号", System: true, CreatedAtUnixMs: nowMs, UpdatedAtUnixMs: nowMs},
		{Code: userbiz.RoleBuyer, Name: "买家", Description: "H5 买家账号", System: true, CreatedAtUnixMs: nowMs, UpdatedAtUnixMs: nowMs},
	}
	if err := s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "code"}},
		DoUpdates: clause.AssignmentColumns([]string{"name", "description", "system", "updated_at_unix_ms"}),
	}).Create(&roles).Error; err != nil {
		return err
	}

	permissions := permissionModels(nowMs)
	if err := s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "code"}},
		DoUpdates: clause.AssignmentColumns([]string{"name", "module", "description", "updated_at_unix_ms"}),
	}).Create(&permissions).Error; err != nil {
		return err
	}

	for roleCode, codes := range userbiz.DefaultRolePermissions() {
		for _, permissionCode := range codes {
			rp := AuctionRolePermissionModel{
				ID:              idgen.New("rp"),
				RoleCode:        roleCode,
				PermissionCode:  permissionCode,
				CreatedAtUnixMs: nowMs,
			}
			if err := s.db.WithContext(ctx).Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "role_code"}, {Name: "permission_code"}},
				DoNothing: true,
			}).Create(&rp).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func permissionModels(nowMs int64) []AuctionPermissionModel {
	names := map[string][3]string{
		userbiz.PermissionTeamUserCreate:       {"团队账号创建", "team", "创建主播/运营子账号"},
		userbiz.PermissionTeamUserList:         {"团队账号查询", "team", "查询当前主账号空间团队账号"},
		userbiz.PermissionTeamUserUpdateRole:   {"团队角色调整", "team", "调整主播/运营子账号角色"},
		userbiz.PermissionTeamUserUpdateStatus: {"团队状态调整", "team", "启用或禁用子账号"},
		userbiz.PermissionLotCreate:            {"拍品创建", "lot", "创建拍品或草稿"},
		userbiz.PermissionLotUpdate:            {"拍品编辑", "lot", "编辑拍品资料"},
		userbiz.PermissionLotQueue:             {"拍品入队", "lot", "加入本场队列"},
		userbiz.PermissionLotViewAdmin:         {"后台拍品查看", "lot", "查看后台拍品列表"},
		userbiz.PermissionAuctionControl:       {"直播控场", "auction", "开拍、落锤、取消和 Duel 控制"},
		userbiz.PermissionOrderManage:          {"订单管理", "order", "后台查看和处理订单"},
		userbiz.PermissionRealtimeView:         {"实时状态查看", "realtime", "查看房间实时状态和事件"},
		userbiz.PermissionUploadImage:          {"图片上传", "asset", "上传拍品图片"},
		userbiz.PermissionBidPlace:             {"买家出价", "buyer", "H5 买家出价"},
		userbiz.PermissionOrderPay:             {"买家支付", "buyer", "H5 买家模拟支付"},
		userbiz.PermissionOrderViewOwn:         {"本人订单查看", "buyer", "查看本人订单和出价"},
	}
	permissions := make([]AuctionPermissionModel, 0, len(names))
	for code, meta := range names {
		permissions = append(permissions, AuctionPermissionModel{
			Code:            code,
			Name:            meta[0],
			Module:          meta[1],
			Description:     meta[2],
			CreatedAtUnixMs: nowMs,
			UpdatedAtUnixMs: nowMs,
		})
	}
	return permissions
}
