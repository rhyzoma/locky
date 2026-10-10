package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/rhyzoma/locky/internal/domain"
	"github.com/rhyzoma/locky/internal/mocks"
	"github.com/rhyzoma/locky/internal/repository/memory"
	"github.com/rhyzoma/locky/internal/usecase"
)

var _ usecase.PasswordHasher = (*mocks.FakePasswordHasher)(nil)
var _ usecase.TokenIssuer = (*mocks.FakeTokenIssuer)(nil)

func TestServiceRegisterLoginRefreshLogout(t *testing.T) {
	store := memory.NewStore()
	mailer := &recordingMailer{}
	passwords := &mocks.FakePasswordHasher{HashResult: "hashed-password", CompareResult: true}
	tokens := &mocks.FakeTokenIssuer{Token: "access-token"}
	service, err := usecase.NewService(usecase.ServiceConfig{
		Users:              store.Users(),
		Credentials:        store.Credentials(),
		Sessions:           store.Sessions(),
		Outbox:             store,
		Mailer:             mailer,
		Passwords:          passwords,
		Tokens:             tokens,
		VerificationTokens: store.VerificationTokens(),
	})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	ctx := context.Background()
	user, err := service.Register(ctx, " Test@Example.com ", "correct horse battery staple")
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if user.Email != "test@example.com" {
		t.Errorf("Register() email = %q, want %q", user.Email, "test@example.com")
	}
	if err := service.VerifyEmail(ctx, mailer.verificationToken); err != nil {
		t.Fatalf("VerifyEmail() error = %v", err)
	}
	if err := service.VerifyEmail(ctx, mailer.verificationToken); !errors.Is(err, domain.ErrVerificationInvalid) {
		t.Errorf("VerifyEmail() reused token error = %v, want %v", err, domain.ErrVerificationInvalid)
	}
	if passwords.HashCalls != 1 {
		t.Errorf("PasswordHasher.Hash() calls = %d, want %d", passwords.HashCalls, 1)
	}
	pair, _, err := service.Login(ctx, user.Email, "correct horse battery staple")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if passwords.CompareCalls != 1 {
		t.Errorf("PasswordHasher.Compare() calls = %d, want %d", passwords.CompareCalls, 1)
	}
	if tokens.Calls != 1 {
		t.Errorf("TokenIssuer.Sign() calls = %d, want %d", tokens.Calls, 1)
	}
	rotated, err := service.Refresh(ctx, pair.RefreshToken)
	if err != nil {
		t.Fatalf("Refresh() error = %v", err)
	}
	if rotated.RefreshToken == pair.RefreshToken {
		t.Error("Refresh() returned the same refresh token, want rotation")
	}
	if tokens.Calls != 2 {
		t.Errorf("TokenIssuer.Sign() calls after Refresh() = %d, want %d", tokens.Calls, 2)
	}
	if err := service.Logout(ctx, rotated.RefreshToken); err != nil {
		t.Fatalf("Logout() error = %v", err)
	}
	if _, err := service.Refresh(ctx, rotated.RefreshToken); err == nil {
		t.Error("Refresh() after Logout succeeded, want error")
	}
}

type recordingMailer struct {
	verificationToken string
}

func (m *recordingMailer) SendVerification(_ context.Context, _, token string) error {
	m.verificationToken = token
	return nil
}

func (m *recordingMailer) SendPasswordReset(context.Context, string, string) error {
	return nil
}

func TestServiceAutoVerifyEmail(t *testing.T) {
	store := memory.NewStore()
	passwords := &mocks.FakePasswordHasher{HashResult: "hashed-password", CompareResult: true}
	tokens := &mocks.FakeTokenIssuer{Token: "access-token"}
	service, err := usecase.NewService(usecase.ServiceConfig{
		Users:              store.Users(),
		Credentials:        store.Credentials(),
		Sessions:           store.Sessions(),
		Outbox:             store,
		Mailer:             memory.Mailer{},
		Passwords:          passwords,
		Tokens:             tokens,
		VerificationTokens: store.VerificationTokens(),
		AutoVerifyEmail:    true,
	})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	user, err := service.Register(context.Background(), "local@example.com", "correct horse battery staple")
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if !user.EmailVerified || user.Status != domain.UserStatusActive {
		t.Errorf("Register() user = %+v, want verified active user", user)
	}
	if _, _, err := service.Login(context.Background(), user.Email, "correct horse battery staple"); err != nil {
		t.Fatalf("Login() error = %v, want nil", err)
	}
}
