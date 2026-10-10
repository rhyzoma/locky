package usecase

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/rhyzoma/locky/internal/domain"
)

type PasswordHasher interface {
	Hash(password string) (string, error)
	Compare(hash, password string) bool
}

type TokenIssuer interface {
	Sign(userID, email string, lifetime time.Duration) (string, error)
}

type Service struct {
	users              domain.UserRepository
	credentials        domain.CredentialRepository
	sessions           domain.SessionStore
	outbox             domain.OutboxStore
	mailer             domain.Mailer
	passwords          PasswordHasher
	tokens             TokenIssuer
	verificationTokens domain.VerificationTokenStore
	now                func() time.Time
	autoVerify         bool
}

type ServiceConfig struct {
	Users              domain.UserRepository
	Credentials        domain.CredentialRepository
	Sessions           domain.SessionStore
	Outbox             domain.OutboxStore
	Mailer             domain.Mailer
	Passwords          PasswordHasher
	Tokens             TokenIssuer
	VerificationTokens domain.VerificationTokenStore
	Now                func() time.Time
	AutoVerifyEmail    bool
}

func NewService(cfg ServiceConfig) (*Service, error) {
	if cfg.Users == nil || cfg.Credentials == nil || cfg.Sessions == nil || cfg.Outbox == nil || cfg.Mailer == nil {
		return nil, errors.New("locky: all repositories and mailer are required")
	}
	if cfg.Passwords == nil || cfg.Tokens == nil {
		return nil, errors.New("locky: password and token services are required")
	}
	if cfg.VerificationTokens == nil {
		return nil, errors.New("locky: verification token store is required")
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	return &Service{
		users:              cfg.Users,
		credentials:        cfg.Credentials,
		sessions:           cfg.Sessions,
		outbox:             cfg.Outbox,
		mailer:             cfg.Mailer,
		passwords:          cfg.Passwords,
		tokens:             cfg.Tokens,
		verificationTokens: cfg.VerificationTokens,
		now:                cfg.Now,
		autoVerify:         cfg.AutoVerifyEmail,
	}, nil
}

func (s *Service) Register(ctx context.Context, email, password string) (domain.User, error) {
	email = normalizeEmail(email)
	if email == "" || password == "" {
		return domain.User{}, fmt.Errorf("register: email and password are required")
	}
	if _, err := s.users.ByEmail(ctx, email); err == nil {
		return domain.User{}, domain.ErrEmailExists
	} else if !errors.Is(err, domain.ErrUserNotFound) {
		return domain.User{}, fmt.Errorf("check existing user: %w", err)
	}
	hash, err := s.passwords.Hash(password)
	if err != nil {
		return domain.User{}, fmt.Errorf("hash password: %w", err)
	}
	now := s.now()
	user := domain.User{ID: randomID(), Email: email, Status: domain.UserStatusPending, CreatedAt: now, UpdatedAt: now}
	if err := s.users.Create(ctx, user); err != nil {
		return domain.User{}, fmt.Errorf("create user: %w", err)
	}
	if err := s.credentials.Create(ctx, domain.PasswordCredential{UserID: user.ID, PasswordHash: hash, UpdatedAt: now}); err != nil {
		return domain.User{}, fmt.Errorf("create credential: %w", err)
	}
	if s.autoVerify {
		user.EmailVerified = true
		user.Status = domain.UserStatusActive
		user.UpdatedAt = now
		if err := s.users.Update(ctx, user); err != nil {
			return domain.User{}, fmt.Errorf("auto verify user: %w", err)
		}
	} else {
		verification, err := randomToken()
		if err != nil {
			return domain.User{}, fmt.Errorf("generate verification token: %w", err)
		}
		if err := s.verificationTokens.Create(ctx, domain.VerificationToken{
			TokenHash: hashToken(verification),
			UserID:    user.ID,
			ExpiresAt: now.Add(24 * time.Hour),
		}); err != nil {
			return domain.User{}, fmt.Errorf("store verification token: %w", err)
		}
		if err := s.mailer.SendVerification(ctx, user.Email, verification); err != nil {
			return domain.User{}, fmt.Errorf("send verification: %w", err)
		}
	}
	return user, nil
}

func (s *Service) Login(ctx context.Context, email, password string) (domain.TokenPair, domain.User, error) {
	user, err := s.users.ByEmail(ctx, normalizeEmail(email))
	if err != nil {
		return domain.TokenPair{}, domain.User{}, domain.ErrInvalidCredentials
	}
	if user.Status == domain.UserStatusDisabled {
		return domain.TokenPair{}, domain.User{}, domain.ErrUserDisabled
	}
	if !user.EmailVerified {
		return domain.TokenPair{}, domain.User{}, domain.ErrEmailNotVerified
	}
	credential, err := s.credentials.ByUserID(ctx, user.ID)
	if err != nil || !s.passwords.Compare(credential.PasswordHash, password) {
		return domain.TokenPair{}, domain.User{}, domain.ErrInvalidCredentials
	}
	pair, session, err := s.issueTokens(user)
	if err != nil {
		return domain.TokenPair{}, domain.User{}, err
	}
	if err := s.sessions.Create(ctx, session); err != nil {
		return domain.TokenPair{}, domain.User{}, fmt.Errorf("create session: %w", err)
	}
	return pair, user, nil
}

func (s *Service) Refresh(ctx context.Context, refreshToken string) (domain.TokenPair, error) {
	hash := hashToken(refreshToken)
	old, err := s.sessions.ByTokenHash(ctx, hash)
	if err != nil || old.Revoked || !s.now().Before(old.ExpiresAt) {
		return domain.TokenPair{}, domain.ErrTokenInvalid
	}
	user, err := s.users.ByID(ctx, old.UserID)
	if err != nil {
		return domain.TokenPair{}, domain.ErrUserNotFound
	}
	if user.Status == domain.UserStatusDisabled {
		return domain.TokenPair{}, domain.ErrUserDisabled
	}
	pair, next, err := s.issueTokens(user)
	if err != nil {
		return domain.TokenPair{}, err
	}
	if err := s.sessions.Rotate(ctx, hash, next); err != nil {
		return domain.TokenPair{}, fmt.Errorf("rotate session: %w", err)
	}
	return pair, nil
}

func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	if err := s.sessions.Revoke(ctx, hashToken(refreshToken)); err != nil && !errors.Is(err, domain.ErrSessionNotFound) {
		return fmt.Errorf("revoke session: %w", err)
	}
	return nil
}

func (s *Service) VerifyEmail(ctx context.Context, token string) error {
	if strings.TrimSpace(token) == "" {
		return domain.ErrVerificationInvalid
	}
	userID, err := s.verificationTokens.Consume(ctx, hashToken(token))
	if err != nil {
		return err
	}
	user, err := s.users.ByID(ctx, userID)
	if err != nil {
		return domain.ErrUserNotFound
	}
	user.EmailVerified = true
	user.Status = domain.UserStatusActive
	user.UpdatedAt = s.now()
	if err := s.users.Update(ctx, user); err != nil {
		return fmt.Errorf("verify email: %w", err)
	}
	return s.appendEvent(ctx, "EmailVerified", user.ID, user)
}

func (s *Service) CurrentUser(ctx context.Context, userID string) (domain.User, error) {
	user, err := s.users.ByID(ctx, userID)
	if err != nil {
		return domain.User{}, err
	}
	return user, nil
}

func (s *Service) issueTokens(user domain.User) (domain.TokenPair, domain.Session, error) {
	refresh, err := randomToken()
	if err != nil {
		return domain.TokenPair{}, domain.Session{}, fmt.Errorf("generate refresh token: %w", err)
	}
	expiresIn := 15 * time.Minute
	access, err := s.tokens.Sign(user.ID, user.Email, expiresIn)
	if err != nil {
		return domain.TokenPair{}, domain.Session{}, fmt.Errorf("sign access token: %w", err)
	}
	return domain.TokenPair{AccessToken: access, RefreshToken: refresh, ExpiresIn: expiresIn}, domain.Session{ID: randomID(), UserID: user.ID, TokenHash: hashToken(refresh), ExpiresAt: s.now().Add(30 * 24 * time.Hour)}, nil
}

func (s *Service) appendEvent(ctx context.Context, eventType, aggregate string, value any) error {
	payload, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshal %s event: %w", eventType, err)
	}
	if err := s.outbox.Append(ctx, domain.OutboxEvent{ID: randomID(), Type: eventType, Aggregate: aggregate, Payload: payload, CreatedAt: s.now()}); err != nil {
		return fmt.Errorf("append %s event: %w", eventType, err)
	}
	return nil
}

func normalizeEmail(email string) string { return strings.ToLower(strings.TrimSpace(email)) }

func randomID() string {
	value, _ := randomToken()
	return value[:8] + "-" + value[8:12] + "-" + value[12:16] + "-" + value[16:20] + "-" + value[20:32]
}

func randomToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
