package grpcserver

import (
	lockyv1 "github.com/rhyzoma/locky/api/gen/locky/v1"
	"github.com/rhyzoma/locky/internal/domain"
)

func toProtoUser(user domain.User) *lockyv1.User {
	return &lockyv1.User{
		Id:            user.ID,
		Email:         user.Email,
		EmailVerified: user.EmailVerified,
		Status:        string(user.Status),
	}
}

func toProtoTokens(tokens domain.TokenPair) *lockyv1.TokenResponse {
	return &lockyv1.TokenResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		ExpiresIn:    int64(tokens.ExpiresIn.Seconds()),
	}
}
