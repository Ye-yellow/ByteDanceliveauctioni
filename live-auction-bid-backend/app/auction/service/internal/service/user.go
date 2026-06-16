package service

import (
	"context"

	v1 "live-auction-bid/backend/api/auction/service/v1"
	"live-auction-bid/backend/app/auction/service/internal/biz/user"
)

type UserService struct {
	v1.UnimplementedUserServiceServer
	users *user.Usecase
}

func NewUserService(users *user.Usecase) *UserService {
	return &UserService{users: users}
}

func (s *UserService) Register(ctx context.Context, req *v1.RegisterRequest) (*v1.RegisterReply, error) {
	user, tokens, err := s.users.Register(ctx, req)
	if err != nil {
		return &v1.RegisterReply{Result: ErrorResult(ctx, err)}, nil
	}
	return &v1.RegisterReply{Result: okResult(ctx), User: user, Tokens: tokens}, nil
}

func (s *UserService) RegisterMerchant(ctx context.Context, req *v1.RegisterMerchantRequest) (*v1.RegisterMerchantReply, error) {
	user, tokens, err := s.users.RegisterMerchant(ctx, req)
	if err != nil {
		return &v1.RegisterMerchantReply{Result: ErrorResult(ctx, err)}, nil
	}
	return &v1.RegisterMerchantReply{Result: okResult(ctx), User: user, Tokens: tokens}, nil
}

func (s *UserService) Login(ctx context.Context, req *v1.LoginRequest) (*v1.LoginReply, error) {
	user, tokens, err := s.users.Login(ctx, req.GetUsername(), req.GetPassword())
	if err != nil {
		return &v1.LoginReply{Result: ErrorResult(ctx, err)}, nil
	}
	return &v1.LoginReply{Result: okResult(ctx), User: user, Tokens: tokens}, nil
}

func (s *UserService) ResetPassword(ctx context.Context, req *v1.ResetPasswordRequest) (*v1.ResetPasswordReply, error) {
	user, err := s.users.ResetPassword(ctx, req.GetUsername(), req.GetPassword())
	if err != nil {
		return &v1.ResetPasswordReply{Result: ErrorResult(ctx, err)}, nil
	}
	return &v1.ResetPasswordReply{Result: okResult(ctx), User: user}, nil
}

func (s *UserService) RefreshToken(ctx context.Context, req *v1.RefreshTokenRequest) (*v1.RefreshTokenReply, error) {
	tokens, err := s.users.RefreshToken(ctx, req.GetRefreshToken())
	if err != nil {
		return &v1.RefreshTokenReply{Result: ErrorResult(ctx, err)}, nil
	}
	return &v1.RefreshTokenReply{Result: okResult(ctx), Tokens: tokens}, nil
}

func (s *UserService) Logout(ctx context.Context, req *v1.LogoutRequest) (*v1.LogoutReply, error) {
	if err := s.users.Logout(ctx, req.GetRefreshToken()); err != nil {
		return &v1.LogoutReply{Result: ErrorResult(ctx, err)}, nil
	}
	return &v1.LogoutReply{Result: okResult(ctx)}, nil
}

func (s *UserService) GetMe(ctx context.Context, req *v1.GetMeRequest) (*v1.GetMeReply, error) {
	user, err := s.users.GetMe(ctx)
	if err != nil {
		return &v1.GetMeReply{Result: ErrorResult(ctx, err)}, nil
	}
	return &v1.GetMeReply{Result: okResult(ctx), User: user}, nil
}

func (s *UserService) AdminCreateUser(ctx context.Context, req *v1.AdminCreateUserRequest) (*v1.AdminCreateUserReply, error) {
	user, err := s.users.AdminCreateUser(ctx, req)
	if err != nil {
		return &v1.AdminCreateUserReply{Result: ErrorResult(ctx, err)}, nil
	}
	return &v1.AdminCreateUserReply{Result: okResult(ctx), User: user}, nil
}

func (s *UserService) AdminUpdateUserRole(ctx context.Context, req *v1.AdminUpdateUserRoleRequest) (*v1.AdminUpdateUserRoleReply, error) {
	user, err := s.users.AdminUpdateUserRole(ctx, req.GetUserId(), req.GetRoleCode())
	if err != nil {
		return &v1.AdminUpdateUserRoleReply{Result: ErrorResult(ctx, err)}, nil
	}
	return &v1.AdminUpdateUserRoleReply{Result: okResult(ctx), User: user}, nil
}

func (s *UserService) AdminUpdateUserStatus(ctx context.Context, req *v1.AdminUpdateUserStatusRequest) (*v1.AdminUpdateUserStatusReply, error) {
	user, err := s.users.AdminUpdateUserStatus(ctx, req.GetUserId(), req.GetStatus())
	if err != nil {
		return &v1.AdminUpdateUserStatusReply{Result: ErrorResult(ctx, err)}, nil
	}
	return &v1.AdminUpdateUserStatusReply{Result: okResult(ctx), User: user}, nil
}

func (s *UserService) AdminResetUserPassword(ctx context.Context, req *v1.AdminResetUserPasswordRequest) (*v1.AdminResetUserPasswordReply, error) {
	user, err := s.users.AdminResetUserPassword(ctx, req.GetUserId(), req.GetPassword())
	if err != nil {
		return &v1.AdminResetUserPasswordReply{Result: ErrorResult(ctx, err)}, nil
	}
	return &v1.AdminResetUserPasswordReply{Result: okResult(ctx), User: user}, nil
}

func (s *UserService) ListUsers(ctx context.Context, req *v1.ListUsersRequest) (*v1.ListUsersReply, error) {
	list, err := s.users.ListUsers(ctx, user.ListUsersQuery{
		Page:     int(req.GetPage()),
		PageSize: int(req.GetPageSize()),
		RoleCode: req.GetRoleCode(),
		Status:   req.GetStatus(),
		Keyword:  req.GetKeyword(),
	})
	if err != nil {
		return &v1.ListUsersReply{Result: ErrorResult(ctx, err), Users: []*v1.User{}}, nil
	}
	return &v1.ListUsersReply{
		Result:   okResult(ctx),
		Users:    list.Users,
		Total:    list.Total,
		Page:     int32(list.Page),
		PageSize: int32(list.PageSize),
	}, nil
}
