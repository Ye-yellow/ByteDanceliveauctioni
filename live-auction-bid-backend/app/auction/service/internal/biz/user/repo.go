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

type ListUsersQuery struct {
	Page          int           `json:"page"`
	PageSize      int           `json:"pageSize"`
	RoleCode      string        `json:"roleCode,omitempty"`
	Status        v1.UserStatus `json:"status,omitempty"`
	Keyword       string        `json:"keyword,omitempty"`
	MainAccountID string        `json:"mainAccountId,omitempty"`
}

type ListUsersResult struct {
	Users    []*v1.User `json:"users"`
	Total    int64      `json:"total"`
	Page     int        `json:"page"`
	PageSize int        `json:"pageSize"`
}

type Repository interface {
	CreateUser(ctx context.Context, user *v1.User, passwordHash string) error
	FindUserByID(ctx context.Context, userID string) (*v1.User, string, error)
	FindUserByUsername(ctx context.Context, username string) (*v1.User, string, error)
	ListUsers(ctx context.Context, query ListUsersQuery) (ListUsersResult, error)
	UpdatePasswordByUsername(ctx context.Context, username string, passwordHash string, updatedAtUnixMs int64) (*v1.User, error)
	UpdatePasswordByUserID(ctx context.Context, userID string, mainAccountID string, passwordHash string, updatedAtUnixMs int64) (*v1.User, error)
	UpdateUserRole(ctx context.Context, userID string, mainAccountID string, roleCode string, updatedAtUnixMs int64) (*v1.User, error)
	UpdateUserStatus(ctx context.Context, userID string, mainAccountID string, status v1.UserStatus, updatedAtUnixMs int64) (*v1.User, error)
	CreateSession(ctx context.Context, session Session) error
	FindSessionByRefreshHash(ctx context.Context, refreshTokenHash string) (Session, bool, error)
	RevokeSession(ctx context.Context, sessionID string, revokedAtUnixMs int64) error
	RevokeSessionsByUserID(ctx context.Context, userID string, revokedAtUnixMs int64) error
}
