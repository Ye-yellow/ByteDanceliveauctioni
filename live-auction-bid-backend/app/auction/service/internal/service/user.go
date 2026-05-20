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
		return &v1.RegisterReply{Result: ErrorResult(err)}, nil
	}
	return &v1.RegisterReply{Result: okResult(), User: user, Tokens: tokens}, nil
}

func (s *UserService) Login(ctx context.Context, req *v1.LoginRequest) (*v1.LoginReply, error) {
	user, tokens, err := s.users.Login(ctx, req.GetUsername(), req.GetPassword())
	if err != nil {
		return &v1.LoginReply{Result: ErrorResult(err)}, nil
	}
	return &v1.LoginReply{Result: okResult(), User: user, Tokens: tokens}, nil
}

func (s *UserService) RefreshToken(ctx context.Context, req *v1.RefreshTokenRequest) (*v1.RefreshTokenReply, error) {
	tokens, err := s.users.RefreshToken(ctx, req.GetRefreshToken())
	if err != nil {
		return &v1.RefreshTokenReply{Result: ErrorResult(err)}, nil
	}
	return &v1.RefreshTokenReply{Result: okResult(), Tokens: tokens}, nil
}

func (s *UserService) Logout(ctx context.Context, req *v1.LogoutRequest) (*v1.LogoutReply, error) {
	if err := s.users.Logout(ctx, req.GetRefreshToken()); err != nil {
		return &v1.LogoutReply{Result: ErrorResult(err)}, nil
	}
	return &v1.LogoutReply{Result: okResult()}, nil
}

func (s *UserService) GetMe(ctx context.Context, req *v1.GetMeRequest) (*v1.GetMeReply, error) {
	user, err := s.users.GetMe(ctx)
	if err != nil {
		return &v1.GetMeReply{Result: ErrorResult(err)}, nil
	}
	return &v1.GetMeReply{Result: okResult(), User: user}, nil
}

func (s *UserService) AdminCreateUser(ctx context.Context, req *v1.AdminCreateUserRequest) (*v1.AdminCreateUserReply, error) {
	user, err := s.users.AdminCreateUser(ctx, req)
	if err != nil {
		return &v1.AdminCreateUserReply{Result: ErrorResult(err)}, nil
	}
	return &v1.AdminCreateUserReply{Result: okResult(), User: user}, nil
}

func (s *UserService) AdminUpdateUserRole(ctx context.Context, req *v1.AdminUpdateUserRoleRequest) (*v1.AdminUpdateUserRoleReply, error) {
	user, err := s.users.AdminUpdateUserRole(ctx, req.GetUserId(), req.GetRole())
	if err != nil {
		return &v1.AdminUpdateUserRoleReply{Result: ErrorResult(err)}, nil
	}
	return &v1.AdminUpdateUserRoleReply{Result: okResult(), User: user}, nil
}
