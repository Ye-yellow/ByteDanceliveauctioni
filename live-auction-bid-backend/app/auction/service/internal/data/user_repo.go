package data

import (
	"context"
	"errors"
	"slices"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	v1 "live-auction-bid/backend/api/auction/service/v1"
	"live-auction-bid/backend/app/auction/service/internal/biz/user"
	"live-auction-bid/backend/app/auction/service/internal/pkg/apperr"
	"live-auction-bid/backend/app/auction/service/internal/pkg/idgen"
)

func (s *Store) CreateUser(ctx context.Context, next *v1.User, passwordHash string) error {
	if next == nil {
		return errors.New("user is required")
	}
	if len(next.GetRoleCodes()) == 0 {
		return errors.New("user role codes are required")
	}
	model := userToModel(next, passwordHash)
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(model).Error; err != nil {
			if isDuplicateKey(err) {
				return apperr.ErrUsernameTaken
			}
			return err
		}
		for _, roleCode := range next.GetRoleCodes() {
			roleCode = user.NormalizeRoleCode(roleCode)
			if roleCode == "" {
				continue
			}
			row := AuctionUserRoleModel{
				ID:              idgen.New("ur"),
				UserID:          next.GetId(),
				RoleCode:        roleCode,
				MainAccountID:   next.GetMainAccountId(),
				GrantedByUserID: next.GetCreatedByUserId(),
				CreatedAtUnixMs: next.GetCreatedAtUnixMs(),
			}
			if err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "user_id"}, {Name: "role_code"}, {Name: "main_account_id"}},
				DoNothing: true,
			}).Create(&row).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return err
	}
	return nil
}

func (s *Store) FindUserByID(ctx context.Context, userID string) (*v1.User, string, error) {
	if userID == "" {
		return nil, "", errors.New("user id is required")
	}
	var model AuctionUserModel
	if err := s.db.WithContext(ctx).Where("id = ?", userID).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, "", apperr.ErrUserNotFound
		}
		return nil, "", err
	}
	next, err := s.modelToUser(ctx, &model)
	return next, model.PasswordHash, err
}

func (s *Store) FindUserByUsername(ctx context.Context, username string) (*v1.User, string, error) {
	if username == "" {
		return nil, "", errors.New("username is required")
	}
	var model AuctionUserModel
	if err := s.db.WithContext(ctx).Where("username = ?", username).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, "", apperr.ErrUserNotFound
		}
		return nil, "", err
	}
	next, err := s.modelToUser(ctx, &model)
	return next, model.PasswordHash, err
}

func (s *Store) ListUsers(ctx context.Context, query user.ListUsersQuery) (user.ListUsersResult, error) {
	query.Page, query.PageSize = normalizeUserPagination(query.Page, query.PageSize)
	db := s.db.WithContext(ctx).Model(&AuctionUserModel{})
	roleCode := user.NormalizeRoleCode(query.RoleCode)
	if roleCode != "" {
		db = db.Joins("JOIN auction_user_roles ON auction_user_roles.user_id = auction_users.id AND auction_user_roles.role_code = ?", roleCode)
	} else {
		db = db.Joins("JOIN auction_user_roles ON auction_user_roles.user_id = auction_users.id AND auction_user_roles.role_code IN ?", []string{
			user.RoleAnchor,
			user.RoleOperator,
		})
	}
	if query.MainAccountID != "" {
		db = db.Where("auction_users.main_account_id = ?", query.MainAccountID)
	}
	if query.Status != v1.UserStatus_USER_STATUS_UNSPECIFIED {
		db = db.Where("auction_users.status = ?", int32(query.Status))
	}
	if keyword := strings.TrimSpace(query.Keyword); keyword != "" {
		like := "%" + strings.ToLower(keyword) + "%"
		db = db.Where("LOWER(auction_users.id) LIKE ? OR LOWER(auction_users.username) LIKE ? OR LOWER(auction_users.nickname) LIKE ?", like, like, like)
	}
	var total int64
	if err := db.Session(&gorm.Session{}).Distinct("auction_users.id").Count(&total).Error; err != nil {
		return user.ListUsersResult{}, err
	}
	var models []AuctionUserModel
	if err := db.
		Distinct("auction_users.*").
		Order("auction_users.created_at_unix_ms DESC").
		Order("auction_users.id ASC").
		Offset((query.Page - 1) * query.PageSize).
		Limit(query.PageSize).
		Find(&models).Error; err != nil {
		return user.ListUsersResult{}, err
	}
	users := make([]*v1.User, 0, len(models))
	for i := range models {
		next, err := s.modelToUser(ctx, &models[i])
		if err != nil {
			return user.ListUsersResult{}, err
		}
		users = append(users, next)
	}
	return user.ListUsersResult{Users: users, Total: total, Page: query.Page, PageSize: query.PageSize}, nil
}

func (s *Store) UpdateUserRole(ctx context.Context, userID string, mainAccountID string, roleCode string, updatedAtUnixMs int64) (*v1.User, error) {
	if userID == "" {
		return nil, errors.New("user id is required")
	}
	roleCode = user.NormalizeRoleCode(roleCode)
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&AuctionUserModel{}).
			Joins("JOIN auction_user_roles ON auction_user_roles.user_id = auction_users.id AND auction_user_roles.role_code IN ?", []string{user.RoleAnchor, user.RoleOperator}).
			Where("auction_users.id = ? AND auction_users.main_account_id = ?", userID, mainAccountID).
			Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			return apperr.ErrPermissionDenied
		}
		if err := tx.Where("user_id = ? AND role_code IN ?", userID, []string{user.RoleAnchor, user.RoleOperator}).Delete(&AuctionUserRoleModel{}).Error; err != nil {
			return err
		}
		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}, {Name: "role_code"}, {Name: "main_account_id"}},
			DoNothing: true,
		}).Create(&AuctionUserRoleModel{
			ID:              idgen.New("ur"),
			UserID:          userID,
			RoleCode:        roleCode,
			MainAccountID:   mainAccountID,
			GrantedByUserID: mainAccountID,
			CreatedAtUnixMs: updatedAtUnixMs,
		}).Error; err != nil {
			return err
		}
		return tx.Model(&AuctionUserModel{}).Where("id = ?", userID).Update("updated_at_unix_ms", updatedAtUnixMs).Error
	}); err != nil {
		return nil, err
	}
	next, _, err := s.FindUserByID(ctx, userID)
	return next, err
}

func (s *Store) UpdateUserStatus(ctx context.Context, userID string, mainAccountID string, status v1.UserStatus, updatedAtUnixMs int64) (*v1.User, error) {
	if userID == "" {
		return nil, errors.New("user id is required")
	}
	result := s.db.WithContext(ctx).
		Model(&AuctionUserModel{}).
		Where("id = ? AND main_account_id = ? AND EXISTS (SELECT 1 FROM auction_user_roles WHERE auction_user_roles.user_id = auction_users.id AND auction_user_roles.role_code IN ?)", userID, mainAccountID, []string{user.RoleAnchor, user.RoleOperator}).
		Updates(map[string]any{"status": int32(status), "updated_at_unix_ms": updatedAtUnixMs})
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, apperr.ErrPermissionDenied
	}
	next, _, err := s.FindUserByID(ctx, userID)
	return next, err
}

func (s *Store) UpdatePasswordByUsername(ctx context.Context, username string, passwordHash string, updatedAtUnixMs int64) (*v1.User, error) {
	if username == "" || passwordHash == "" {
		return nil, errors.New("username and password hash are required")
	}
	result := s.db.WithContext(ctx).
		Model(&AuctionUserModel{}).
		Where("username = ?", username).
		Updates(map[string]any{"password_hash": passwordHash, "updated_at_unix_ms": updatedAtUnixMs})
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, apperr.ErrUserNotFound
	}
	next, _, err := s.FindUserByUsername(ctx, username)
	return next, err
}

func (s *Store) UpdatePasswordByUserID(ctx context.Context, userID string, mainAccountID string, passwordHash string, updatedAtUnixMs int64) (*v1.User, error) {
	if userID == "" || mainAccountID == "" || passwordHash == "" {
		return nil, errors.New("user id, main account id, and password hash are required")
	}
	result := s.db.WithContext(ctx).
		Model(&AuctionUserModel{}).
		Where("id = ? AND main_account_id = ? AND EXISTS (SELECT 1 FROM auction_user_roles WHERE auction_user_roles.user_id = auction_users.id AND auction_user_roles.role_code IN ?)", userID, mainAccountID, []string{user.RoleAnchor, user.RoleOperator}).
		Updates(map[string]any{"password_hash": passwordHash, "updated_at_unix_ms": updatedAtUnixMs})
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, apperr.ErrPermissionDenied
	}
	next, _, err := s.FindUserByID(ctx, userID)
	return next, err
}

func (s *Store) CreateSession(ctx context.Context, session user.Session) error {
	if session.ID == "" || session.UserID == "" || session.RefreshTokenHash == "" {
		return errors.New("session id, user id and refresh token hash are required")
	}
	return s.db.WithContext(ctx).Create(sessionToModel(session)).Error
}

func (s *Store) FindSessionByRefreshHash(ctx context.Context, refreshTokenHash string) (user.Session, bool, error) {
	if refreshTokenHash == "" {
		return user.Session{}, false, errors.New("refresh token hash is required")
	}
	var model AuctionUserSessionModel
	if err := s.db.WithContext(ctx).Where("refresh_token_hash = ?", refreshTokenHash).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return user.Session{}, false, nil
		}
		return user.Session{}, false, err
	}
	return modelToSession(&model), true, nil
}

func (s *Store) RevokeSession(ctx context.Context, sessionID string, revokedAtUnixMs int64) error {
	if sessionID == "" {
		return errors.New("session id is required")
	}
	return s.db.WithContext(ctx).
		Model(&AuctionUserSessionModel{}).
		Where("id = ? AND revoked_at_unix_ms = 0", sessionID).
		Update("revoked_at_unix_ms", revokedAtUnixMs).
		Error
}

func (s *Store) RevokeSessionsByUserID(ctx context.Context, userID string, revokedAtUnixMs int64) error {
	if userID == "" {
		return errors.New("user id is required")
	}
	return s.db.WithContext(ctx).
		Model(&AuctionUserSessionModel{}).
		Where("user_id = ? AND revoked_at_unix_ms = 0", userID).
		Update("revoked_at_unix_ms", revokedAtUnixMs).
		Error
}

func userToModel(next *v1.User, passwordHash string) *AuctionUserModel {
	avatarURL := strings.TrimSpace(next.GetAvatarUrl())
	if avatarURL == "" {
		avatarURL = user.AvatarURLForUserID(next.GetId())
	}
	return &AuctionUserModel{
		ID:              next.Id,
		Username:        next.Username,
		Nickname:        next.Nickname,
		AvatarURL:       avatarURL,
		PasswordHash:    passwordHash,
		MainAccountID:   next.MainAccountId,
		CreatedByUserID: next.CreatedByUserId,
		Status:          int32(effectiveUserModelStatus(next.Status)),
		CreatedAtUnixMs: next.CreatedAtUnixMs,
		UpdatedAtUnixMs: next.UpdatedAtUnixMs,
	}
}

func (s *Store) modelToUser(ctx context.Context, model *AuctionUserModel) (*v1.User, error) {
	roleCodes, err := s.userRoleCodes(ctx, model.ID)
	if err != nil {
		return nil, err
	}
	permissionCodes, err := s.userPermissionCodes(ctx, model.ID)
	if err != nil {
		return nil, err
	}
	avatarURL := strings.TrimSpace(model.AvatarURL)
	if avatarURL == "" {
		avatarURL = user.AvatarURLForUserID(model.ID)
	}
	return &v1.User{
		Id:              model.ID,
		Username:        model.Username,
		Nickname:        model.Nickname,
		AvatarUrl:       avatarURL,
		MainAccountId:   model.MainAccountID,
		CreatedByUserId: model.CreatedByUserID,
		Status:          effectiveUserModelStatus(v1.UserStatus(model.Status)),
		CreatedAtUnixMs: model.CreatedAtUnixMs,
		UpdatedAtUnixMs: model.UpdatedAtUnixMs,
		RoleCodes:       roleCodes,
		PermissionCodes: permissionCodes,
	}, nil
}

func (s *Store) userRoleCodes(ctx context.Context, userID string) ([]string, error) {
	var codes []string
	if err := s.db.WithContext(ctx).Model(&AuctionUserRoleModel{}).
		Where("user_id = ?", userID).
		Order("role_code ASC").
		Pluck("role_code", &codes).Error; err != nil {
		return nil, err
	}
	return codes, nil
}

func (s *Store) userPermissionCodes(ctx context.Context, userID string) ([]string, error) {
	var rolePermissions []string
	if err := s.db.WithContext(ctx).Model(&AuctionRolePermissionModel{}).
		Joins("JOIN auction_user_roles ON auction_user_roles.role_code = auction_role_permissions.role_code AND auction_user_roles.user_id = ?", userID).
		Pluck("auction_role_permissions.permission_code", &rolePermissions).Error; err != nil {
		return nil, err
	}
	var userAllows []string
	if err := s.db.WithContext(ctx).Model(&AuctionUserPermissionModel{}).
		Where("user_id = ? AND effect = ?", userID, "allow").
		Pluck("permission_code", &userAllows).Error; err != nil {
		return nil, err
	}
	var userDenies []string
	if err := s.db.WithContext(ctx).Model(&AuctionUserPermissionModel{}).
		Where("user_id = ? AND effect = ?", userID, "deny").
		Pluck("permission_code", &userDenies).Error; err != nil {
		return nil, err
	}
	deny := make(map[string]struct{}, len(userDenies))
	for _, code := range userDenies {
		deny[code] = struct{}{}
	}
	all := append(rolePermissions, userAllows...)
	slices.Sort(all)
	out := make([]string, 0, len(all))
	for _, code := range all {
		if _, denied := deny[code]; denied {
			continue
		}
		if len(out) == 0 || out[len(out)-1] != code {
			out = append(out, code)
		}
	}
	return out, nil
}

func effectiveUserModelStatus(status v1.UserStatus) v1.UserStatus {
	if status == v1.UserStatus_USER_STATUS_UNSPECIFIED {
		return v1.UserStatus_USER_STATUS_ACTIVE
	}
	return status
}

func sessionToModel(session user.Session) *AuctionUserSessionModel {
	return &AuctionUserSessionModel{
		ID:                     session.ID,
		UserID:                 session.UserID,
		RefreshTokenHash:       session.RefreshTokenHash,
		RefreshExpiresAtUnixMs: session.RefreshExpiresAtUnixMs,
		RevokedAtUnixMs:        session.RevokedAtUnixMs,
		CreatedAtUnixMs:        session.CreatedAtUnixMs,
	}
}

func modelToSession(model *AuctionUserSessionModel) user.Session {
	return user.Session{
		ID:                     model.ID,
		UserID:                 model.UserID,
		RefreshTokenHash:       model.RefreshTokenHash,
		RefreshExpiresAtUnixMs: model.RefreshExpiresAtUnixMs,
		RevokedAtUnixMs:        model.RevokedAtUnixMs,
		CreatedAtUnixMs:        model.CreatedAtUnixMs,
	}
}

func isDuplicateKey(err error) bool {
	return strings.Contains(strings.ToLower(err.Error()), "duplicate")
}

func normalizeUserPagination(page, pageSize int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}
