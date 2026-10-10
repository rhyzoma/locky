package domain

import "errors"

var (
	ErrInvalidCredentials  = errors.New("locky: invalid credentials")
	ErrEmailExists         = errors.New("locky: email already exists")
	ErrUserNotFound        = errors.New("locky: user not found")
	ErrSessionNotFound     = errors.New("locky: session not found")
	ErrSessionRevoked      = errors.New("locky: session revoked")
	ErrUserDisabled        = errors.New("locky: user disabled")
	ErrEmailNotVerified    = errors.New("locky: email not verified")
	ErrTokenInvalid        = errors.New("locky: invalid token")
	ErrVerificationInvalid = errors.New("locky: invalid verification token")
	ErrVerificationExpired = errors.New("locky: verification token expired")
)
