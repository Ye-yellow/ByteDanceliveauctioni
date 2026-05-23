package test

import (
	"context"
	"errors"
	"sort"
	"strings"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"
	v1 "live-auction-bid/backend/api/auction/service/v1"
	userbiz "live-auction-bid/backend/app/auction/service/internal/biz/user"
	"live-auction-bid/backend/app/auction/service/internal/pkg/apperr"
	"live-auction-bid/backend/app/auction/service/internal/pkg/auth"
)

func TestUserUsecaseRegisterLoginRefreshLogoutAndAdminFlow(t *testing.T) {
	repo := newTestUserRepo()
	manager := newTestAuthManager(t)
	uc := userbiz.NewUsecase(repo, manager)
	ctx := context.Background()

	registered, tokens, err := uc.Register(ctx, &v1.RegisterRequest{
		Username: "Buyer_One",
		Password: "password123",
		Nickname: "买家一号",
	})
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}
	if registered.GetId() == "" || registered.GetUsername() != "buyer_one" || registered.GetRole() != v1.UserRole_USER_ROLE_BUYER {
		t.Fatalf("registered user mismatch: %+v", registered)
	}
	if tokens.GetAccessToken() == "" || tokens.GetRefreshToken() == "" {
		t.Fatalf("expected tokens after register, got %+v", tokens)
	}
	if _, _, err := uc.Register(ctx, &v1.RegisterRequest{Username: "buyer_one", Password: "password123", Nickname: "重复"}); !apperr.IsUsernameTaken(err) {
		t.Fatalf("expected duplicate username error, got %v", err)
	}

	loggedIn, loginTokens, err := uc.Login(ctx, "buyer_one", "password123")
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	if loggedIn.GetId() != registered.GetId() || loginTokens.GetRefreshToken() == tokens.GetRefreshToken() {
		t.Fatalf("login result mismatch: user=%+v tokens=%+v", loggedIn, loginTokens)
	}
	if _, _, err := uc.Login(ctx, "buyer_one", "wrong-password"); !apperr.IsInvalidCredentials(err) {
		t.Fatalf("expected invalid credentials, got %v", err)
	}

	refreshed, err := uc.RefreshToken(ctx, loginTokens.GetRefreshToken())
	if err != nil {
		t.Fatalf("refresh failed: %v", err)
	}
	if refreshed.GetRefreshToken() == "" || refreshed.GetRefreshToken() == loginTokens.GetRefreshToken() {
		t.Fatalf("expected refresh token rotation, got %+v", refreshed)
	}
	if _, err := uc.RefreshToken(ctx, loginTokens.GetRefreshToken()); !apperr.IsSessionExpired(err) {
		t.Fatalf("expected old refresh token to fail, got %v", err)
	}
	if err := uc.Logout(ctx, refreshed.GetRefreshToken()); err != nil {
		t.Fatalf("logout failed: %v", err)
	}
	if _, err := uc.RefreshToken(ctx, refreshed.GetRefreshToken()); !apperr.IsSessionExpired(err) {
		t.Fatalf("expected logged out refresh token to fail, got %v", err)
	}

	if err := uc.BootstrapAdmin(ctx, "admin", "adminpass123", "管理员"); err != nil {
		t.Fatalf("bootstrap admin failed: %v", err)
	}
	admin, _, err := uc.Login(ctx, "admin", "adminpass123")
	if err != nil {
		t.Fatalf("admin login failed: %v", err)
	}
	adminCtx := auth.WithClaims(ctx, &auth.Claims{UserID: admin.Id, Username: admin.Username, Nickname: admin.Nickname, Role: admin.Role})
	anchor, err := uc.AdminCreateUser(adminCtx, &v1.AdminCreateUserRequest{
		Username: "anchor1",
		Password: "anchorpass123",
		Nickname: "主播",
		Role:     v1.UserRole_USER_ROLE_ANCHOR,
	})
	if err != nil {
		t.Fatalf("admin create user failed: %v", err)
	}
	if anchor.GetRole() != v1.UserRole_USER_ROLE_ANCHOR {
		t.Fatalf("expected anchor role, got %+v", anchor)
	}
	if _, err := uc.AdminCreateUser(adminCtx, &v1.AdminCreateUserRequest{Username: "buyer_from_admin", Password: "password123", Nickname: "后台买家", Role: v1.UserRole_USER_ROLE_BUYER}); !apperr.IsInvalidArgument(err) {
		t.Fatalf("expected invalid argument for admin creating buyer, got %v", err)
	}
	if _, err := uc.AdminCreateUser(adminCtx, &v1.AdminCreateUserRequest{Username: "missing_role", Password: "password123", Nickname: "缺少角色"}); !apperr.IsInvalidArgument(err) {
		t.Fatalf("expected invalid argument for missing admin user role, got %v", err)
	}
	anchorCtx := auth.WithClaims(ctx, &auth.Claims{UserID: anchor.Id, Username: anchor.Username, Nickname: anchor.Nickname, Role: v1.UserRole_USER_ROLE_ANCHOR})
	if _, err := uc.AdminCreateUser(anchorCtx, &v1.AdminCreateUserRequest{Username: "anchor_child", Password: "password123", Nickname: "主播子账号"}); !apperr.IsPermissionDenied(err) {
		t.Fatalf("expected permission denied for anchor admin create, got %v", err)
	}
	operator, err := uc.AdminUpdateUserRole(adminCtx, anchor.GetId(), v1.UserRole_USER_ROLE_OPERATOR)
	if err != nil {
		t.Fatalf("admin update role failed: %v", err)
	}
	if operator.GetRole() != v1.UserRole_USER_ROLE_OPERATOR {
		t.Fatalf("expected operator role, got %+v", operator)
	}
	if _, err := uc.AdminUpdateUserRole(adminCtx, registered.GetId(), v1.UserRole_USER_ROLE_BUYER); !apperr.IsInvalidArgument(err) {
		t.Fatalf("expected invalid argument for admin updating role to buyer, got %v", err)
	}
	operatorCtx := auth.WithClaims(ctx, &auth.Claims{UserID: operator.Id, Username: operator.Username, Nickname: operator.Nickname, Role: v1.UserRole_USER_ROLE_OPERATOR})
	if _, err := uc.AdminUpdateUserRole(operatorCtx, registered.GetId(), v1.UserRole_USER_ROLE_ANCHOR); !apperr.IsPermissionDenied(err) {
		t.Fatalf("expected permission denied for operator role update, got %v", err)
	}
	buyerCtx := auth.WithClaims(ctx, &auth.Claims{UserID: registered.Id, Username: registered.Username, Nickname: registered.Nickname, Role: registered.Role})
	if _, err := uc.AdminCreateUser(buyerCtx, &v1.AdminCreateUserRequest{Username: "xuser", Password: "password123", Nickname: "x"}); !apperr.IsPermissionDenied(err) {
		t.Fatalf("expected permission denied for buyer admin create, got %v", err)
	}
	userList, err := uc.ListUsers(adminCtx, userbiz.ListUsersQuery{Role: v1.UserRole_USER_ROLE_OPERATOR, Keyword: "anchor", Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("admin list users failed: %v", err)
	}
	if userList.Total != 1 || len(userList.Users) != 1 || userList.Users[0].GetRole() != v1.UserRole_USER_ROLE_OPERATOR {
		t.Fatalf("expected filtered operator user list, got %+v", userList)
	}
	if _, err := uc.ListUsers(adminCtx, userbiz.ListUsersQuery{Role: v1.UserRole_USER_ROLE_BUYER, Page: 1, PageSize: 10}); !apperr.IsInvalidArgument(err) {
		t.Fatalf("expected invalid argument for admin listing buyer users, got %v", err)
	}
	backofficeList, err := uc.ListUsers(adminCtx, userbiz.ListUsersQuery{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("admin list backoffice users failed: %v", err)
	}
	for _, user := range backofficeList.Users {
		if user.GetRole() == v1.UserRole_USER_ROLE_BUYER {
			t.Fatalf("admin list users must not return buyer accounts: %+v", backofficeList)
		}
	}
	if _, err := uc.ListUsers(operatorCtx, userbiz.ListUsersQuery{}); !apperr.IsPermissionDenied(err) {
		t.Fatalf("expected operator list users denied, got %v", err)
	}
	if _, err := uc.ListUsers(buyerCtx, userbiz.ListUsersQuery{}); !apperr.IsPermissionDenied(err) {
		t.Fatalf("expected buyer list users denied, got %v", err)
	}
}

func TestAuthManagerAccessTokenAndPasswordHash(t *testing.T) {
	now := time.Unix(1000, 0)
	manager, err := auth.NewManager(auth.Config{Secret: "secret", Issuer: "auction", AccessTTL: time.Minute, RefreshTTL: time.Hour}, auth.WithNow(func() time.Time { return now }))
	if err != nil {
		t.Fatalf("new auth manager failed: %v", err)
	}
	user := &v1.User{Id: "u1", Username: "buyer", Nickname: "买家", Role: v1.UserRole_USER_ROLE_BUYER}
	pair, err := manager.IssueTokenPair(user)
	if err != nil {
		t.Fatalf("issue tokens failed: %v", err)
	}
	claims, err := manager.ParseAccessToken(pair.AccessToken)
	if err != nil {
		t.Fatalf("parse access token failed: %v", err)
	}
	if claims.UserID != "u1" || claims.Role != v1.UserRole_USER_ROLE_BUYER || claims.RoleName != "buyer" {
		t.Fatalf("claims mismatch: %+v", claims)
	}
	wrongManager, err := auth.NewManager(auth.Config{Secret: "other", Issuer: "auction"})
	if err != nil {
		t.Fatalf("new wrong manager failed: %v", err)
	}
	if _, err := wrongManager.ParseAccessToken(pair.AccessToken); !apperr.IsInvalidToken(err) {
		t.Fatalf("expected wrong secret to reject token, got %v", err)
	}
	now = now.Add(2 * time.Minute)
	if _, err := manager.ParseAccessToken(pair.AccessToken); !apperr.IsTokenExpired(err) {
		t.Fatalf("expected expired token to fail, got %v", err)
	}
	hash, err := auth.HashPassword("password123")
	if err != nil {
		t.Fatalf("hash password failed: %v", err)
	}
	if !auth.VerifyPassword(hash, "password123") || auth.VerifyPassword(hash, "wrong") {
		t.Fatal("password verify mismatch")
	}
}

type testUserRepo struct {
	usersByID       map[string]*v1.User
	passwordByID    map[string]string
	userIDByName    map[string]string
	sessionsByID    map[string]userbiz.Session
	sessionIDByHash map[string]string
}

func newTestUserRepo() *testUserRepo {
	return &testUserRepo{
		usersByID:       make(map[string]*v1.User),
		passwordByID:    make(map[string]string),
		userIDByName:    make(map[string]string),
		sessionsByID:    make(map[string]userbiz.Session),
		sessionIDByHash: make(map[string]string),
	}
}

func (r *testUserRepo) CreateUser(ctx context.Context, user *v1.User, passwordHash string) error {
	if _, found := r.userIDByName[user.GetUsername()]; found {
		return apperr.ErrUsernameTaken
	}
	r.usersByID[user.GetId()] = proto.Clone(user).(*v1.User)
	r.passwordByID[user.GetId()] = passwordHash
	r.userIDByName[user.GetUsername()] = user.GetId()
	return nil
}

func (r *testUserRepo) FindUserByID(ctx context.Context, userID string) (*v1.User, string, error) {
	user, found := r.usersByID[userID]
	if !found {
		return nil, "", apperr.ErrUserNotFound
	}
	return proto.Clone(user).(*v1.User), r.passwordByID[userID], nil
}

func (r *testUserRepo) FindUserByUsername(ctx context.Context, username string) (*v1.User, string, error) {
	userID, found := r.userIDByName[username]
	if !found {
		return nil, "", apperr.ErrUserNotFound
	}
	return r.FindUserByID(ctx, userID)
}

func (r *testUserRepo) ListUsers(ctx context.Context, query userbiz.ListUsersQuery) (userbiz.ListUsersResult, error) {
	if query.Page <= 0 {
		query.Page = 1
	}
	if query.PageSize <= 0 {
		query.PageSize = 20
	}
	if query.PageSize > 100 {
		query.PageSize = 100
	}
	users := make([]*v1.User, 0, len(r.usersByID))
	keyword := strings.ToLower(strings.TrimSpace(query.Keyword))
	for _, user := range r.usersByID {
		if query.Role != v1.UserRole_USER_ROLE_UNSPECIFIED && user.Role != query.Role {
			continue
		}
		if query.Role == v1.UserRole_USER_ROLE_UNSPECIFIED && user.Role == v1.UserRole_USER_ROLE_BUYER {
			continue
		}
		if keyword != "" && !strings.Contains(strings.ToLower(user.Id+" "+user.Username+" "+user.Nickname), keyword) {
			continue
		}
		users = append(users, proto.Clone(user).(*v1.User))
	}
	sort.Slice(users, func(i, j int) bool { return users[i].Username < users[j].Username })
	total := int64(len(users))
	start := (query.Page - 1) * query.PageSize
	if start >= len(users) {
		return userbiz.ListUsersResult{Users: []*v1.User{}, Total: total, Page: query.Page, PageSize: query.PageSize}, nil
	}
	end := start + query.PageSize
	if end > len(users) {
		end = len(users)
	}
	return userbiz.ListUsersResult{Users: users[start:end], Total: total, Page: query.Page, PageSize: query.PageSize}, nil
}

func (r *testUserRepo) UpdateUserRole(ctx context.Context, userID string, role v1.UserRole, updatedAtUnixMs int64) (*v1.User, error) {
	user, found := r.usersByID[userID]
	if !found {
		return nil, apperr.ErrUserNotFound
	}
	next := proto.Clone(user).(*v1.User)
	next.Role = role
	next.UpdatedAtUnixMs = updatedAtUnixMs
	r.usersByID[userID] = next
	return proto.Clone(next).(*v1.User), nil
}

func (r *testUserRepo) CreateSession(ctx context.Context, session userbiz.Session) error {
	if session.ID == "" || session.RefreshTokenHash == "" {
		return errors.New("invalid session")
	}
	r.sessionsByID[session.ID] = session
	r.sessionIDByHash[session.RefreshTokenHash] = session.ID
	return nil
}

func (r *testUserRepo) FindSessionByRefreshHash(ctx context.Context, refreshTokenHash string) (userbiz.Session, bool, error) {
	sessionID, found := r.sessionIDByHash[refreshTokenHash]
	if !found {
		return userbiz.Session{}, false, nil
	}
	session, found := r.sessionsByID[sessionID]
	return session, found, nil
}

func (r *testUserRepo) RevokeSession(ctx context.Context, sessionID string, revokedAtUnixMs int64) error {
	session, found := r.sessionsByID[sessionID]
	if !found {
		return nil
	}
	session.RevokedAtUnixMs = revokedAtUnixMs
	r.sessionsByID[sessionID] = session
	return nil
}

func newTestAuthManager(t *testing.T) *auth.Manager {
	t.Helper()
	manager, err := auth.NewManager(auth.Config{
		Secret:     "test-secret",
		Issuer:     "auction-test",
		AccessTTL:  time.Minute,
		RefreshTTL: time.Hour,
	})
	if err != nil {
		t.Fatalf("new auth manager failed: %v", err)
	}
	return manager
}
