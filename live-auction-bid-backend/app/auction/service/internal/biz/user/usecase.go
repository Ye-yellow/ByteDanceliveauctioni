package user

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"google.golang.org/protobuf/proto"
	v1 "live-auction-bid/backend/api/auction/service/v1"
	"live-auction-bid/backend/app/auction/service/internal/pkg/apperr"
	"live-auction-bid/backend/app/auction/service/internal/pkg/auth"
	"live-auction-bid/backend/app/auction/service/internal/pkg/idgen"
)

type Usecase struct {
	repo Repository
	auth *auth.Manager
	now  func() time.Time
}

func NewUsecase(repo Repository, authManager *auth.Manager) *Usecase {
	return &Usecase{repo: repo, auth: authManager, now: time.Now}
}

func (uc *Usecase) Register(ctx context.Context, req *v1.RegisterRequest) (*v1.User, *v1.AuthTokens, error) {
	username, password, nickname, err := normalizeCredentials(req.GetUsername(), req.GetPassword(), req.GetNickname())
	if err != nil {
		return nil, nil, err
	}
	if len(username) < 6 {
		return nil, nil, fmt.Errorf("%w: username must be 6-64 characters", apperr.ErrInvalidArgument)
	}
	return uc.createUserWithTokens(ctx, username, password, nickname, v1.UserRole_USER_ROLE_BUYER, "", "")
}

func (uc *Usecase) RegisterMerchant(ctx context.Context, req *v1.RegisterMerchantRequest) (*v1.User, *v1.AuthTokens, error) {
	nickname := strings.TrimSpace(req.GetNickname())
	if nickname == "" {
		nickname = strings.TrimSpace(req.GetUsername())
	}
	username, password, nickname, err := normalizeCredentials(req.GetUsername(), req.GetPassword(), nickname)
	if err != nil {
		return nil, nil, err
	}
	return uc.createUserWithTokens(ctx, username, password, nickname, v1.UserRole_USER_ROLE_MAIN_ACCOUNT, "", "")
}

func (uc *Usecase) Login(ctx context.Context, username, password string) (*v1.User, *v1.AuthTokens, error) {
	username = normalizeUsername(username)
	if username == "" || password == "" {
		return nil, nil, apperr.ErrInvalidCredentials
	}
	user, passwordHash, err := uc.repo.FindUserByUsername(ctx, username)
	if err != nil {
		if apperr.IsUserNotFound(err) {
			return nil, nil, apperr.ErrInvalidCredentials
		}
		return nil, nil, err
	}
	if !auth.VerifyPassword(passwordHash, password) {
		return nil, nil, apperr.ErrInvalidCredentials
	}
	if err := ensureUserActive(user); err != nil {
		return nil, nil, err
	}
	tokens, err := uc.issueAndStoreTokens(ctx, user)
	if err != nil {
		return nil, nil, err
	}
	return cloneUser(user), tokens, nil
}

func (uc *Usecase) ResetPassword(ctx context.Context, username, password string) (*v1.User, error) {
	username = normalizeUsername(username)
	if len(username) < 3 || len(username) > 64 {
		return nil, fmt.Errorf("%w: username must be 3-64 characters", apperr.ErrInvalidArgument)
	}
	if len(password) < 8 || len(password) > 128 {
		return nil, fmt.Errorf("%w: password must be 8-128 characters", apperr.ErrInvalidArgument)
	}
	for _, r := range username {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			continue
		}
		return nil, fmt.Errorf("%w: username may contain lowercase letters, digits, underscore, and hyphen", apperr.ErrInvalidArgument)
	}
	_, _, err := uc.repo.FindUserByUsername(ctx, username)
	if err != nil {
		return nil, err
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return nil, err
	}
	nowMs := uc.now().UnixMilli()
	updated, err := uc.repo.UpdatePasswordByUsername(ctx, username, hash, nowMs)
	if err != nil {
		return nil, err
	}
	if err := uc.repo.RevokeSessionsByUserID(ctx, updated.GetId(), nowMs); err != nil {
		return nil, err
	}
	return cloneUser(updated), nil
}

func (uc *Usecase) RefreshToken(ctx context.Context, refreshToken string) (*v1.AuthTokens, error) {
	refreshToken = strings.TrimSpace(refreshToken)
	if refreshToken == "" {
		return nil, fmt.Errorf("%w: refresh token is required", apperr.ErrInvalidArgument)
	}
	session, found, err := uc.repo.FindSessionByRefreshHash(ctx, auth.HashRefreshToken(refreshToken))
	if err != nil {
		return nil, err
	}
	nowMs := uc.now().UnixMilli()
	if !found || session.RevokedAtUnixMs != 0 || session.RefreshExpiresAtUnixMs <= nowMs {
		return nil, apperr.ErrSessionExpired
	}
	user, _, err := uc.repo.FindUserByID(ctx, session.UserID)
	if err != nil {
		return nil, err
	}
	if err := ensureUserActive(user); err != nil {
		return nil, err
	}
	if err := uc.repo.RevokeSession(ctx, session.ID, nowMs); err != nil {
		return nil, err
	}
	return uc.issueAndStoreTokens(ctx, user)
}

func (uc *Usecase) Logout(ctx context.Context, refreshToken string) error {
	refreshToken = strings.TrimSpace(refreshToken)
	if refreshToken == "" {
		return nil
	}
	session, found, err := uc.repo.FindSessionByRefreshHash(ctx, auth.HashRefreshToken(refreshToken))
	if err != nil {
		return err
	}
	if !found || session.RevokedAtUnixMs != 0 {
		return nil
	}
	return uc.repo.RevokeSession(ctx, session.ID, uc.now().UnixMilli())
}

func (uc *Usecase) GetMe(ctx context.Context) (*v1.User, error) {
	claims, err := auth.RequireUser(ctx)
	if err != nil {
		return nil, err
	}
	user, _, err := uc.repo.FindUserByID(ctx, claims.UserID)
	if err != nil {
		return nil, err
	}
	if err := ensureUserActive(user); err != nil {
		return nil, err
	}
	return cloneUser(user), nil
}

func (uc *Usecase) AdminCreateUser(ctx context.Context, req *v1.AdminCreateUserRequest) (*v1.User, error) {
	claims, err := auth.RequireRole(ctx, v1.UserRole_USER_ROLE_MAIN_ACCOUNT)
	if err != nil {
		return nil, err
	}
	username, password, nickname, err := normalizeCredentials(req.GetUsername(), req.GetPassword(), req.GetNickname())
	if err != nil {
		return nil, err
	}
	role := req.GetRole()
	if !isManagedTeamRole(role) {
		return nil, fmt.Errorf("%w: main account can only create team subaccounts", apperr.ErrInvalidArgument)
	}
	user, _, err := uc.createUser(ctx, username, password, nickname, role, mainAccountIDFromClaims(claims), claims.UserID)
	if err != nil {
		return nil, err
	}
	return cloneUser(user), nil
}

func (uc *Usecase) AdminUpdateUserRole(ctx context.Context, userID string, role v1.UserRole) (*v1.User, error) {
	claims, err := auth.RequireRole(ctx, v1.UserRole_USER_ROLE_MAIN_ACCOUNT)
	if err != nil {
		return nil, err
	}
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, fmt.Errorf("%w: user id is required", apperr.ErrInvalidArgument)
	}
	if !isManagedTeamRole(role) {
		return nil, fmt.Errorf("%w: main account can only update team subaccount roles", apperr.ErrInvalidArgument)
	}
	mainAccountID := mainAccountIDFromClaims(claims)
	target, _, err := uc.repo.FindUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if err := ensureManagedSubaccountInScope(target, mainAccountID); err != nil {
		return nil, err
	}
	user, err := uc.repo.UpdateUserRole(ctx, userID, mainAccountID, role, uc.now().UnixMilli())
	if err != nil {
		return nil, err
	}
	return cloneUser(user), nil
}

func (uc *Usecase) AdminUpdateUserStatus(ctx context.Context, userID string, status v1.UserStatus) (*v1.User, error) {
	claims, err := auth.RequireRole(ctx, v1.UserRole_USER_ROLE_MAIN_ACCOUNT)
	if err != nil {
		return nil, err
	}
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, fmt.Errorf("%w: user id is required", apperr.ErrInvalidArgument)
	}
	if status != v1.UserStatus_USER_STATUS_ACTIVE && status != v1.UserStatus_USER_STATUS_DISABLED {
		return nil, fmt.Errorf("%w: status must be active or disabled", apperr.ErrInvalidArgument)
	}
	mainAccountID := mainAccountIDFromClaims(claims)
	target, _, err := uc.repo.FindUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if err := ensureManagedSubaccountInScope(target, mainAccountID); err != nil {
		return nil, err
	}
	nowMs := uc.now().UnixMilli()
	updated, err := uc.repo.UpdateUserStatus(ctx, userID, mainAccountID, status, nowMs)
	if err != nil {
		return nil, err
	}
	if status == v1.UserStatus_USER_STATUS_DISABLED {
		if err := uc.repo.RevokeSessionsByUserID(ctx, userID, nowMs); err != nil {
			return nil, err
		}
	}
	return cloneUser(updated), nil
}

func (uc *Usecase) ListUsers(ctx context.Context, query ListUsersQuery) (ListUsersResult, error) {
	claims, err := auth.RequireRole(ctx, v1.UserRole_USER_ROLE_MAIN_ACCOUNT)
	if err != nil {
		return ListUsersResult{}, err
	}
	query.Page, query.PageSize = normalizePagination(query.Page, query.PageSize)
	if query.Role != v1.UserRole_USER_ROLE_UNSPECIFIED && !isManagedTeamRole(query.Role) {
		return ListUsersResult{}, fmt.Errorf("%w: team users query only supports subaccount roles", apperr.ErrInvalidArgument)
	}
	query.MainAccountID = mainAccountIDFromClaims(claims)
	result, err := uc.repo.ListUsers(ctx, query)
	if err != nil {
		return ListUsersResult{}, err
	}
	users := make([]*v1.User, 0, len(result.Users))
	for _, user := range result.Users {
		users = append(users, cloneUser(user))
	}
	result.Users = users
	return result, nil
}

func (uc *Usecase) BootstrapMainAccount(ctx context.Context, username, password, nickname string) error {
	username, password, nickname, err := normalizeCredentials(username, password, nickname)
	if err != nil {
		return err
	}
	existing, _, err := uc.repo.FindUserByUsername(ctx, username)
	if err == nil {
		if existing.GetRole() == v1.UserRole_USER_ROLE_MAIN_ACCOUNT {
			return nil
		}
		return fmt.Errorf("bootstrap main account username %q already exists with role %s", username, existing.GetRole())
	}
	if !apperr.IsUserNotFound(err) {
		return err
	}
	_, _, err = uc.createUser(ctx, username, password, nickname, v1.UserRole_USER_ROLE_MAIN_ACCOUNT, "", "")
	return err
}

func (uc *Usecase) createUserWithTokens(ctx context.Context, username, password, nickname string, role v1.UserRole, mainAccountID string, createdByUserID string) (*v1.User, *v1.AuthTokens, error) {
	user, _, err := uc.createUser(ctx, username, password, nickname, role, mainAccountID, createdByUserID)
	if err != nil {
		return nil, nil, err
	}
	tokens, err := uc.issueAndStoreTokens(ctx, user)
	if err != nil {
		return nil, nil, err
	}
	return cloneUser(user), tokens, nil
}

func (uc *Usecase) createUser(ctx context.Context, username, password, nickname string, role v1.UserRole, mainAccountID string, createdByUserID string) (*v1.User, string, error) {
	if !isValidRole(role) {
		return nil, "", fmt.Errorf("%w: invalid user role", apperr.ErrInvalidArgument)
	}
	if isManagedTeamRole(role) && strings.TrimSpace(mainAccountID) == "" {
		return nil, "", fmt.Errorf("%w: main account id is required for team subaccounts", apperr.ErrInvalidArgument)
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return nil, "", err
	}
	nowMs := uc.now().UnixMilli()
	userID := idgen.NewUserID()
	mainAccountID = strings.TrimSpace(mainAccountID)
	createdByUserID = strings.TrimSpace(createdByUserID)
	if role == v1.UserRole_USER_ROLE_MAIN_ACCOUNT {
		mainAccountID = userID
		createdByUserID = ""
	}
	if role == v1.UserRole_USER_ROLE_BUYER {
		mainAccountID = ""
		createdByUserID = ""
	}
	user := &v1.User{
		Id:              userID,
		Username:        username,
		Nickname:        nickname,
		Role:            role,
		CreatedAtUnixMs: nowMs,
		UpdatedAtUnixMs: nowMs,
		MainAccountId:   mainAccountID,
		CreatedByUserId: createdByUserID,
		Status:          v1.UserStatus_USER_STATUS_ACTIVE,
	}
	if err := uc.repo.CreateUser(ctx, user, hash); err != nil {
		if errors.Is(err, apperr.ErrUsernameTaken) {
			return nil, "", err
		}
		return nil, "", err
	}
	return user, hash, nil
}

func (uc *Usecase) issueAndStoreTokens(ctx context.Context, user *v1.User) (*v1.AuthTokens, error) {
	if err := ensureUserActive(user); err != nil {
		return nil, err
	}
	pair, err := uc.auth.IssueTokenPair(user)
	if err != nil {
		return nil, err
	}
	nowMs := uc.now().UnixMilli()
	session := Session{
		ID:                     idgen.New("sess"),
		UserID:                 user.GetId(),
		RefreshTokenHash:       auth.HashRefreshToken(pair.RefreshToken),
		RefreshExpiresAtUnixMs: pair.RefreshExpiresAtMs,
		CreatedAtUnixMs:        nowMs,
	}
	if err := uc.repo.CreateSession(ctx, session); err != nil {
		return nil, err
	}
	return &v1.AuthTokens{
		AccessToken:            pair.AccessToken,
		RefreshToken:           pair.RefreshToken,
		AccessExpiresAtUnixMs:  pair.AccessExpiresAtMs,
		RefreshExpiresAtUnixMs: pair.RefreshExpiresAtMs,
	}, nil
}

func normalizeCredentials(username, password, nickname string) (string, string, string, error) {
	username = normalizeUsername(username)
	nickname = strings.TrimSpace(nickname)
	if len(username) < 3 || len(username) > 64 {
		return "", "", "", fmt.Errorf("%w: username must be 3-64 characters", apperr.ErrInvalidArgument)
	}
	if len(password) < 8 || len(password) > 128 {
		return "", "", "", fmt.Errorf("%w: password must be 8-128 characters", apperr.ErrInvalidArgument)
	}
	if nickname == "" || len([]rune(nickname)) > 128 {
		return "", "", "", fmt.Errorf("%w: nickname is required and must be at most 128 characters", apperr.ErrInvalidArgument)
	}
	for _, r := range username {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			continue
		}
		return "", "", "", fmt.Errorf("%w: username may contain lowercase letters, digits, underscore, and hyphen", apperr.ErrInvalidArgument)
	}
	return username, password, nickname, nil
}

func normalizeUsername(username string) string {
	return strings.ToLower(strings.TrimSpace(username))
}

func isValidRole(role v1.UserRole) bool {
	switch role {
	case v1.UserRole_USER_ROLE_BUYER,
		v1.UserRole_USER_ROLE_ANCHOR,
		v1.UserRole_USER_ROLE_OPERATOR,
		v1.UserRole_USER_ROLE_MAIN_ACCOUNT:
		return true
	default:
		return false
	}
}

func isManagedTeamRole(role v1.UserRole) bool {
	switch role {
	case v1.UserRole_USER_ROLE_ANCHOR,
		v1.UserRole_USER_ROLE_OPERATOR:
		return true
	default:
		return false
	}
}

func normalizePagination(page, pageSize int) (int, int) {
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

func mainAccountIDFromClaims(claims *auth.Claims) string {
	if claims == nil {
		return ""
	}
	if strings.TrimSpace(claims.MainAccountID) != "" {
		return strings.TrimSpace(claims.MainAccountID)
	}
	if claims.Role == v1.UserRole_USER_ROLE_MAIN_ACCOUNT {
		return strings.TrimSpace(claims.UserID)
	}
	return ""
}

func ensureManagedSubaccountInScope(user *v1.User, mainAccountID string) error {
	if user == nil {
		return apperr.ErrUserNotFound
	}
	if !isManagedTeamRole(user.GetRole()) {
		return fmt.Errorf("%w: only team subaccounts can be managed here", apperr.ErrInvalidArgument)
	}
	if strings.TrimSpace(mainAccountID) == "" || user.GetMainAccountId() != strings.TrimSpace(mainAccountID) {
		return apperr.ErrPermissionDenied
	}
	return nil
}

func ensureUserActive(user *v1.User) error {
	if user == nil {
		return apperr.ErrUserNotFound
	}
	if user.GetStatus() == v1.UserStatus_USER_STATUS_DISABLED {
		return apperr.ErrAccountDisabled
	}
	return nil
}

func cloneUser(user *v1.User) *v1.User {
	if user == nil {
		return nil
	}
	return proto.Clone(user).(*v1.User)
}
