package user

import (
	"context"

	v1 "live-auction-bid/backend/api/auction/service/v1"
)

type Session struct {
	ID                     string
	UserID                 string
	RefreshTokenHash       string
	RefreshExpiresAtUnixMs int64
	RevokedAtUnixMs        int64
	CreatedAtUnixMs        int64
}

type Repository interface {
	CreateUser(ctx context.Context, user *v1.User, passwordHash string) error
	FindUserByID(ctx context.Context, userID string) (*v1.User, string, error)
	FindUserByUsername(ctx context.Context, username string) (*v1.User, string, error)
	UpdateUserRole(ctx context.Context, userID string, role v1.UserRole, updatedAtUnixMs int64) (*v1.User, error)
	CreateSession(ctx context.Context, session Session) error
	FindSessionByRefreshHash(ctx context.Context, refreshTokenHash string) (Session, bool, error)
	RevokeSession(ctx context.Context, sessionID string, revokedAtUnixMs int64) error
}
