package data

import (
	"context"
	"errors"
	"strings"

	"gorm.io/gorm"
	v1 "live-auction-bid/backend/api/auction/service/v1"
	"live-auction-bid/backend/app/auction/service/internal/biz/user"
	"live-auction-bid/backend/app/auction/service/internal/pkg/apperr"
)

func (s *Store) CreateUser(ctx context.Context, user *v1.User, passwordHash string) error {
	if user == nil {
		return errors.New("user is required")
	}
	model := userToModel(user, passwordHash)
	if err := s.db.WithContext(ctx).Create(model).Error; err != nil {
		if isDuplicateKey(err) {
			return apperr.ErrUsernameTaken
		}
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
	return modelToUser(&model), model.PasswordHash, nil
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
	return modelToUser(&model), model.PasswordHash, nil
}

func (s *Store) ListUsers(ctx context.Context, query user.ListUsersQuery) (user.ListUsersResult, error) {
	query.Page, query.PageSize = normalizeUserPagination(query.Page, query.PageSize)
	db := s.db.WithContext(ctx).Model(&AuctionUserModel{})
	if query.Role != v1.UserRole_USER_ROLE_UNSPECIFIED {
		db = db.Where("role = ?", int32(query.Role))
	}
	if keyword := strings.TrimSpace(query.Keyword); keyword != "" {
		like := "%" + strings.ToLower(keyword) + "%"
		db = db.Where("LOWER(id) LIKE ? OR LOWER(username) LIKE ? OR LOWER(nickname) LIKE ?", like, like, like)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return user.ListUsersResult{}, err
	}
	var models []AuctionUserModel
	if err := db.
		Order("created_at_unix_ms DESC").
		Order("id ASC").
		Offset((query.Page - 1) * query.PageSize).
		Limit(query.PageSize).
		Find(&models).Error; err != nil {
		return user.ListUsersResult{}, err
	}
	users := make([]*v1.User, 0, len(models))
	for i := range models {
		users = append(users, modelToUser(&models[i]))
	}
	return user.ListUsersResult{Users: users, Total: total, Page: query.Page, PageSize: query.PageSize}, nil
}

func (s *Store) UpdateUserRole(ctx context.Context, userID string, role v1.UserRole, updatedAtUnixMs int64) (*v1.User, error) {
	if userID == "" {
		return nil, errors.New("user id is required")
	}
	result := s.db.WithContext(ctx).
		Model(&AuctionUserModel{}).
		Where("id = ?", userID).
		Updates(map[string]any{"role": int32(role), "updated_at_unix_ms": updatedAtUnixMs})
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, apperr.ErrUserNotFound
	}
	user, _, err := s.FindUserByID(ctx, userID)
	return user, err
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

func userToModel(user *v1.User, passwordHash string) *AuctionUserModel {
	return &AuctionUserModel{
		ID:              user.Id,
		Username:        user.Username,
		Nickname:        user.Nickname,
		PasswordHash:    passwordHash,
		Role:            int32(user.Role),
		CreatedAtUnixMs: user.CreatedAtUnixMs,
		UpdatedAtUnixMs: user.UpdatedAtUnixMs,
	}
}

func modelToUser(model *AuctionUserModel) *v1.User {
	return &v1.User{
		Id:              model.ID,
		Username:        model.Username,
		Nickname:        model.Nickname,
		Role:            v1.UserRole(model.Role),
		CreatedAtUnixMs: model.CreatedAtUnixMs,
		UpdatedAtUnixMs: model.UpdatedAtUnixMs,
	}
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
