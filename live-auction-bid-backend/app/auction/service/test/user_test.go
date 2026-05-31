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
	if registered.GetId() == "" || registered.GetUsername() != "buyer_one" || !hasRoleForTest(registered, userbiz.RoleBuyer) {
		t.Fatalf("registered user mismatch: %+v", registered)
	}
	if tokens.GetAccessToken() == "" || tokens.GetRefreshToken() == "" {
		t.Fatalf("expected tokens after register, got %+v", tokens)
	}
	if _, _, err := uc.Register(ctx, &v1.RegisterRequest{Username: "buyer_one", Password: "password123", Nickname: "重复"}); !apperr.IsUsernameTaken(err) {
		t.Fatalf("expected duplicate username error, got %v", err)
	}
	if _, _, err := uc.Register(ctx, &v1.RegisterRequest{Username: "buy12", Password: "password123", Nickname: "短账号"}); !apperr.IsInvalidArgument(err) {
		t.Fatalf("expected short buyer username to be rejected, got %v", err)
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
	resetUser, err := uc.ResetPassword(ctx, "buyer_one", "newpassword123")
	if err != nil {
		t.Fatalf("reset buyer password failed: %v", err)
	}
	if resetUser.GetId() != registered.GetId() {
		t.Fatalf("reset user mismatch: %+v", resetUser)
	}
	if _, _, err := uc.Login(ctx, "buyer_one", "password123"); !apperr.IsInvalidCredentials(err) {
		t.Fatalf("expected old password to fail after reset, got %v", err)
	}
	if _, err := uc.RefreshToken(ctx, loginTokens.GetRefreshToken()); !apperr.IsSessionExpired(err) {
		t.Fatalf("expected reset to revoke existing refresh token, got %v", err)
	}
	loggedIn, loginTokens, err = uc.Login(ctx, "buyer_one", "newpassword123")
	if err != nil {
		t.Fatalf("login with reset password failed: %v", err)
	}
	if loggedIn.GetId() != registered.GetId() {
		t.Fatalf("login after reset user mismatch: %+v", loggedIn)
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

	if err := uc.BootstrapMainAccount(ctx, "main", "mainpass123", "主账号"); err != nil {
		t.Fatalf("bootstrap main account failed: %v", err)
	}
	resetBootstrapMain, err := uc.ResetPassword(ctx, "main", "newmainpass123")
	if err != nil {
		t.Fatalf("reset bootstrap main account password failed: %v", err)
	}
	if resetBootstrapMain.GetUsername() != "main" {
		t.Fatalf("reset bootstrap main account mismatch: %+v", resetBootstrapMain)
	}
	if _, _, err := uc.Login(ctx, "main", "newmainpass123"); err != nil {
		t.Fatalf("login with reset bootstrap main account password failed: %v", err)
	}

	mainAccount, mainTokens, err := uc.RegisterMerchant(ctx, &v1.RegisterMerchantRequest{
		Username: "merchant_a",
		Password: "merchantpass123",
	})
	if err != nil {
		t.Fatalf("register main account failed: %v", err)
	}
	if !hasRoleForTest(mainAccount, userbiz.RoleMerchantOwner) ||
		mainAccount.GetMainAccountId() != mainAccount.GetId() ||
		mainAccount.GetStatus() != v1.UserStatus_USER_STATUS_ACTIVE ||
		mainAccount.GetNickname() != "merchant_a" {
		t.Fatalf("main account mismatch: %+v", mainAccount)
	}
	resetMain, err := uc.ResetPassword(ctx, "merchant_a", "newmerchantpass123")
	if err != nil {
		t.Fatalf("reset main account password failed: %v", err)
	}
	if resetMain.GetId() != mainAccount.GetId() {
		t.Fatalf("reset main account mismatch: %+v", resetMain)
	}
	if _, err := uc.RefreshToken(ctx, mainTokens.GetRefreshToken()); !apperr.IsSessionExpired(err) {
		t.Fatalf("expected main account password reset to revoke old refresh token, got %v", err)
	}
	mainAccount, _, err = uc.Login(ctx, "merchant_a", "newmerchantpass123")
	if err != nil {
		t.Fatalf("main account login failed: %v", err)
	}
	mainCtx := auth.WithClaims(ctx, claimsForUser(mainAccount))
	anchor, err := uc.AdminCreateUser(mainCtx, &v1.AdminCreateUserRequest{
		Username: "anchor1",
		Password: "anchorpass123",
		Nickname: "主播",
		RoleCode: userbiz.RoleAnchor,
	})
	if err != nil {
		t.Fatalf("main account create user failed: %v", err)
	}
	if !hasRoleForTest(anchor, userbiz.RoleAnchor) ||
		anchor.GetMainAccountId() != mainAccount.GetId() ||
		anchor.GetCreatedByUserId() != mainAccount.GetId() ||
		anchor.GetStatus() != v1.UserStatus_USER_STATUS_ACTIVE {
		t.Fatalf("expected anchor role, got %+v", anchor)
	}
	if _, err := uc.AdminCreateUser(mainCtx, &v1.AdminCreateUserRequest{Username: "buyer_from_main", Password: "password123", Nickname: "后台买家", RoleCode: userbiz.RoleBuyer}); !apperr.IsInvalidArgument(err) {
		t.Fatalf("expected invalid argument for main account creating buyer, got %v", err)
	}
	if _, err := uc.AdminCreateUser(mainCtx, &v1.AdminCreateUserRequest{Username: "missing_role", Password: "password123", Nickname: "缺少角色"}); !apperr.IsInvalidArgument(err) {
		t.Fatalf("expected invalid argument for missing team user role, got %v", err)
	}
	anchorCtx := auth.WithClaims(ctx, claimsForUser(anchor))
	if _, err := uc.AdminCreateUser(anchorCtx, &v1.AdminCreateUserRequest{Username: "anchor_child", Password: "password123", Nickname: "主播子账号"}); !apperr.IsPermissionDenied(err) {
		t.Fatalf("expected permission denied for anchor team create, got %v", err)
	}
	operator, err := uc.AdminUpdateUserRole(mainCtx, anchor.GetId(), userbiz.RoleOperator)
	if err != nil {
		t.Fatalf("main account update role failed: %v", err)
	}
	if !hasRoleForTest(operator, userbiz.RoleOperator) {
		t.Fatalf("expected operator role, got %+v", operator)
	}
	if _, err := uc.AdminUpdateUserRole(mainCtx, registered.GetId(), userbiz.RoleBuyer); !apperr.IsInvalidArgument(err) {
		t.Fatalf("expected invalid argument for main account updating role to buyer, got %v", err)
	}
	operatorCtx := auth.WithClaims(ctx, claimsForUser(operator))
	if _, err := uc.AdminUpdateUserRole(operatorCtx, registered.GetId(), userbiz.RoleAnchor); !apperr.IsPermissionDenied(err) {
		t.Fatalf("expected permission denied for operator role update, got %v", err)
	}
	buyerCtx := auth.WithClaims(ctx, claimsForUser(registered))
	if _, err := uc.AdminCreateUser(buyerCtx, &v1.AdminCreateUserRequest{Username: "xuser", Password: "password123", Nickname: "x"}); !apperr.IsPermissionDenied(err) {
		t.Fatalf("expected permission denied for buyer team create, got %v", err)
	}
	userList, err := uc.ListUsers(mainCtx, userbiz.ListUsersQuery{RoleCode: userbiz.RoleOperator, Keyword: "anchor", Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("main account list users failed: %v", err)
	}
	if userList.Total != 1 || len(userList.Users) != 1 || !hasRoleForTest(userList.Users[0], userbiz.RoleOperator) {
		t.Fatalf("expected filtered operator user list, got %+v", userList)
	}
	if _, err := uc.ListUsers(mainCtx, userbiz.ListUsersQuery{RoleCode: userbiz.RoleBuyer, Page: 1, PageSize: 10}); !apperr.IsInvalidArgument(err) {
		t.Fatalf("expected invalid argument for main account listing buyer users, got %v", err)
	}
	backofficeList, err := uc.ListUsers(mainCtx, userbiz.ListUsersQuery{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("main account list team users failed: %v", err)
	}
	for _, user := range backofficeList.Users {
		if !isManagedTeamUserForTest(user) || user.GetMainAccountId() != mainAccount.GetId() {
			t.Fatalf("main account list users must only return own subaccounts: %+v", backofficeList)
		}
	}
	if _, err := uc.ListUsers(operatorCtx, userbiz.ListUsersQuery{}); !apperr.IsPermissionDenied(err) {
		t.Fatalf("expected operator list users denied, got %v", err)
	}
	if _, err := uc.ListUsers(buyerCtx, userbiz.ListUsersQuery{}); !apperr.IsPermissionDenied(err) {
		t.Fatalf("expected buyer list users denied, got %v", err)
	}
}

func TestUserUsecaseMainAccountScopeAndStatus(t *testing.T) {
	repo := newTestUserRepo()
	manager := newTestAuthManager(t)
	uc := userbiz.NewUsecase(repo, manager)
	ctx := context.Background()

	mainA, _, err := uc.RegisterMerchant(ctx, &v1.RegisterMerchantRequest{Username: "merchant_a", Password: "merchantpass123"})
	if err != nil {
		t.Fatalf("register main a failed: %v", err)
	}
	mainB, _, err := uc.RegisterMerchant(ctx, &v1.RegisterMerchantRequest{Username: "merchant_b", Password: "merchantpass123"})
	if err != nil {
		t.Fatalf("register main b failed: %v", err)
	}
	ctxA := auth.WithClaims(ctx, claimsForUser(mainA))
	ctxB := auth.WithClaims(ctx, claimsForUser(mainB))

	anchorA, err := uc.AdminCreateUser(ctxA, &v1.AdminCreateUserRequest{Username: "anchor_a", Password: "anchorpass123", Nickname: "Anchor A", RoleCode: userbiz.RoleAnchor})
	if err != nil {
		t.Fatalf("main a create anchor failed: %v", err)
	}
	operatorB, err := uc.AdminCreateUser(ctxB, &v1.AdminCreateUserRequest{Username: "operator_b", Password: "operatorpass123", Nickname: "Operator B", RoleCode: userbiz.RoleOperator})
	if err != nil {
		t.Fatalf("main b create operator failed: %v", err)
	}
	listA, err := uc.ListUsers(ctxA, userbiz.ListUsersQuery{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("main a list failed: %v", err)
	}
	if listA.Total != 1 || listA.Users[0].GetId() != anchorA.GetId() {
		t.Fatalf("main a list leaked or missed scoped subaccounts: %+v", listA)
	}
	if _, err := uc.AdminUpdateUserRole(ctxA, operatorB.GetId(), userbiz.RoleAnchor); !apperr.IsPermissionDenied(err) {
		t.Fatalf("main a must not update main b subaccount role, got %v", err)
	}
	if _, err := uc.AdminUpdateUserStatus(ctxA, operatorB.GetId(), v1.UserStatus_USER_STATUS_DISABLED); !apperr.IsPermissionDenied(err) {
		t.Fatalf("main a must not update main b subaccount status, got %v", err)
	}

	operator, operatorTokens, err := uc.Login(ctx, "operator_b", "operatorpass123")
	if err != nil {
		t.Fatalf("operator b login before disable failed: %v", err)
	}
	disabled, err := uc.AdminUpdateUserStatus(ctxB, operator.GetId(), v1.UserStatus_USER_STATUS_DISABLED)
	if err != nil {
		t.Fatalf("disable operator failed: %v", err)
	}
	if disabled.GetStatus() != v1.UserStatus_USER_STATUS_DISABLED {
		t.Fatalf("expected disabled operator, got %+v", disabled)
	}
	if _, _, err := uc.Login(ctx, "operator_b", "operatorpass123"); !apperr.IsAccountDisabled(err) {
		t.Fatalf("disabled operator login should fail, got %v", err)
	}
	if _, err := uc.RefreshToken(ctx, operatorTokens.GetRefreshToken()); !apperr.IsSessionExpired(err) {
		t.Fatalf("disabled operator refresh token should be revoked, got %v", err)
	}
	if _, err := uc.GetMe(auth.WithClaims(ctx, claimsForUser(disabled))); !apperr.IsAccountDisabled(err) {
		t.Fatalf("disabled operator get me should fail, got %v", err)
	}
	enabled, err := uc.AdminUpdateUserStatus(ctxB, operator.GetId(), v1.UserStatus_USER_STATUS_ACTIVE)
	if err != nil {
		t.Fatalf("enable operator failed: %v", err)
	}
	if enabled.GetStatus() != v1.UserStatus_USER_STATUS_ACTIVE {
		t.Fatalf("expected active operator, got %+v", enabled)
	}
	if _, _, err := uc.Login(ctx, "operator_b", "operatorpass123"); err != nil {
		t.Fatalf("operator login after enable failed: %v", err)
	}
	if _, err := uc.AdminUpdateUserStatus(ctxB, mainB.GetId(), v1.UserStatus_USER_STATUS_DISABLED); !apperr.IsInvalidArgument(err) {
		t.Fatalf("main account cannot be disabled through team status API, got %v", err)
	}
}

func TestAuthManagerAccessTokenAndPasswordHash(t *testing.T) {
	now := time.Unix(1000, 0)
	manager, err := auth.NewManager(auth.Config{Secret: "secret", Issuer: "auction", AccessTTL: time.Minute, RefreshTTL: time.Hour}, auth.WithNow(func() time.Time { return now }))
	if err != nil {
		t.Fatalf("new auth manager failed: %v", err)
	}
	user := &v1.User{Id: "u1", Username: "buyer", Nickname: "买家", RoleCodes: []string{userbiz.RoleBuyer}, PermissionCodes: userbiz.PermissionsForRole(userbiz.RoleBuyer), Status: v1.UserStatus_USER_STATUS_ACTIVE}
	pair, err := manager.IssueTokenPair(user)
	if err != nil {
		t.Fatalf("issue tokens failed: %v", err)
	}
	claims, err := manager.ParseAccessToken(pair.AccessToken)
	if err != nil {
		t.Fatalf("parse access token failed: %v", err)
	}
	if claims.UserID != "u1" || !auth.HasRoleCode(claims, userbiz.RoleBuyer) || !auth.HasPermission(claims, userbiz.PermissionBidPlace) {
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
		if query.RoleCode != "" && !hasRoleForTest(user, query.RoleCode) {
			continue
		}
		if query.RoleCode == "" && !isManagedTeamUserForTest(user) {
			continue
		}
		if query.MainAccountID != "" && user.GetMainAccountId() != query.MainAccountID {
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

func (r *testUserRepo) UpdateUserRole(ctx context.Context, userID string, mainAccountID string, roleCode string, updatedAtUnixMs int64) (*v1.User, error) {
	user, found := r.usersByID[userID]
	if !found {
		return nil, apperr.ErrUserNotFound
	}
	if user.GetMainAccountId() != mainAccountID {
		return nil, apperr.ErrPermissionDenied
	}
	next := proto.Clone(user).(*v1.User)
	next.RoleCodes = []string{userbiz.NormalizeRoleCode(roleCode)}
	next.PermissionCodes = userbiz.PermissionsForRole(roleCode)
	next.UpdatedAtUnixMs = updatedAtUnixMs
	r.usersByID[userID] = next
	return proto.Clone(next).(*v1.User), nil
}

func (r *testUserRepo) UpdateUserStatus(ctx context.Context, userID string, mainAccountID string, status v1.UserStatus, updatedAtUnixMs int64) (*v1.User, error) {
	user, found := r.usersByID[userID]
	if !found {
		return nil, apperr.ErrUserNotFound
	}
	if user.GetMainAccountId() != mainAccountID {
		return nil, apperr.ErrPermissionDenied
	}
	next := proto.Clone(user).(*v1.User)
	next.Status = status
	next.UpdatedAtUnixMs = updatedAtUnixMs
	r.usersByID[userID] = next
	return proto.Clone(next).(*v1.User), nil
}

func (r *testUserRepo) UpdatePasswordByUsername(ctx context.Context, username string, passwordHash string, updatedAtUnixMs int64) (*v1.User, error) {
	userID, found := r.userIDByName[username]
	if !found {
		return nil, apperr.ErrUserNotFound
	}
	user := proto.Clone(r.usersByID[userID]).(*v1.User)
	user.UpdatedAtUnixMs = updatedAtUnixMs
	r.usersByID[userID] = user
	r.passwordByID[userID] = passwordHash
	return proto.Clone(user).(*v1.User), nil
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

func (r *testUserRepo) RevokeSessionsByUserID(ctx context.Context, userID string, revokedAtUnixMs int64) error {
	for id, session := range r.sessionsByID {
		if session.UserID == userID && session.RevokedAtUnixMs == 0 {
			session.RevokedAtUnixMs = revokedAtUnixMs
			r.sessionsByID[id] = session
		}
	}
	return nil
}

func claimsForUser(user *v1.User) *auth.Claims {
	return &auth.Claims{
		UserID:          user.GetId(),
		Username:        user.GetUsername(),
		Nickname:        user.GetNickname(),
		RoleCodes:       append([]string(nil), user.GetRoleCodes()...),
		PermissionCodes: append([]string(nil), user.GetPermissionCodes()...),
		MainAccountID:   user.GetMainAccountId(),
		Status:          user.GetStatus(),
	}
}

func hasRoleForTest(next *v1.User, roleCode string) bool {
	return userbiz.UserHasRole(next, roleCode)
}

func isManagedTeamUserForTest(next *v1.User) bool {
	return hasRoleForTest(next, userbiz.RoleAnchor) || hasRoleForTest(next, userbiz.RoleOperator)
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
