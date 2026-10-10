package grpcserver

import (
	"errors"

	"github.com/rhyzoma/locky/internal/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func mapError(err error) error {
	switch {
	case errors.Is(err, domain.ErrInvalidCredentials), errors.Is(err, domain.ErrTokenInvalid):
		return status.Error(codes.Unauthenticated, err.Error())
	case errors.Is(err, domain.ErrVerificationInvalid), errors.Is(err, domain.ErrVerificationExpired):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, domain.ErrEmailExists):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, domain.ErrUserNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, domain.ErrUserDisabled):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, domain.ErrEmailNotVerified):
		return status.Error(codes.FailedPrecondition, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}
