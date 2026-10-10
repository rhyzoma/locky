package grpcserver

import (
	"context"

	lockyv1 "github.com/rhyzoma/locky/api/gen/locky/v1"
)

func (s *Server) Register(ctx context.Context, req *lockyv1.RegisterRequest) (*lockyv1.RegisterResponse, error) {
	user, err := s.service.Register(ctx, req.GetEmail(), req.GetPassword())
	if err != nil {
		return nil, mapError(err)
	}
	return &lockyv1.RegisterResponse{User: toProtoUser(user)}, nil
}

func (s *Server) Login(ctx context.Context, req *lockyv1.LoginRequest) (*lockyv1.LoginResponse, error) {
	tokens, user, err := s.service.Login(ctx, req.GetEmail(), req.GetPassword())
	if err != nil {
		return nil, mapError(err)
	}
	return &lockyv1.LoginResponse{
		User:   toProtoUser(user),
		Tokens: toProtoTokens(tokens),
	}, nil
}

func (s *Server) RefreshToken(ctx context.Context, req *lockyv1.RefreshTokenRequest) (*lockyv1.TokenResponse, error) {
	tokens, err := s.service.Refresh(ctx, req.GetRefreshToken())
	if err != nil {
		return nil, mapError(err)
	}
	return toProtoTokens(tokens), nil
}

func (s *Server) Logout(ctx context.Context, req *lockyv1.LogoutRequest) (*lockyv1.Empty, error) {
	if err := s.service.Logout(ctx, req.GetRefreshToken()); err != nil {
		return nil, mapError(err)
	}
	return &lockyv1.Empty{}, nil
}

func (s *Server) VerifyEmail(ctx context.Context, req *lockyv1.VerifyEmailRequest) (*lockyv1.Empty, error) {
	if err := s.service.VerifyEmail(ctx, req.GetToken()); err != nil {
		return nil, mapError(err)
	}
	return &lockyv1.Empty{}, nil
}

func (s *Server) GetCurrentUser(ctx context.Context, req *lockyv1.GetCurrentUserRequest) (*lockyv1.User, error) {
	user, err := s.service.CurrentUser(ctx, req.GetUserId())
	if err != nil {
		return nil, mapError(err)
	}
	return toProtoUser(user), nil
}
