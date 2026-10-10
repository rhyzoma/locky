package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhyzoma/locky/internal/domain"
	"github.com/rhyzoma/locky/internal/repository/postgres/sqlc"
)

type Store struct {
	pool    *pgxpool.Pool
	queries *sqlcgen.Queries
}

func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool, queries: sqlcgen.New(pool)}
}

func (s *Store) Ping(ctx context.Context) error { return s.pool.Ping(ctx) }

func (s *Store) Close() { s.pool.Close() }

type UserRepository struct{ store *Store }

func (s *Store) Users() *UserRepository { return &UserRepository{store: s} }

func (s *UserRepository) Create(ctx context.Context, user domain.User) error {
	id, err := toUUID(user.ID)
	if err != nil {
		return fmt.Errorf("parse user id: %w", err)
	}
	if err := s.store.queries.CreateUser(ctx, sqlcgen.CreateUserParams{
		ID:            id,
		Email:         user.Email,
		EmailVerified: user.EmailVerified,
		Status:        string(user.Status),
		CreatedAt:     toTimestamp(user.CreatedAt),
		UpdatedAt:     toTimestamp(user.UpdatedAt),
	}); err != nil {
		return fmt.Errorf("insert user: %w", err)
	}
	return nil
}

func (s *UserRepository) ByEmail(ctx context.Context, email string) (domain.User, error) {
	user, err := s.store.queries.GetUserByEmail(ctx, email)
	if err != nil {
		return domain.User{}, mapNotFound(err, domain.ErrUserNotFound, "select user")
	}
	return fromSQLUser(user)
}

func (s *UserRepository) ByID(ctx context.Context, id string) (domain.User, error) {
	userID, err := toUUID(id)
	if err != nil {
		return domain.User{}, fmt.Errorf("parse user id: %w", err)
	}
	user, err := s.store.queries.GetUserByID(ctx, userID)
	if err != nil {
		return domain.User{}, mapNotFound(err, domain.ErrUserNotFound, "select user")
	}
	return fromSQLUser(user)
}

func (s *UserRepository) Update(ctx context.Context, user domain.User) error {
	id, err := toUUID(user.ID)
	if err != nil {
		return fmt.Errorf("parse user id: %w", err)
	}
	rows, err := s.store.queries.UpdateUser(ctx, sqlcgen.UpdateUserParams{
		ID:            id,
		Email:         user.Email,
		EmailVerified: user.EmailVerified,
		Status:        string(user.Status),
		UpdatedAt:     toTimestamp(user.UpdatedAt),
	})
	if err != nil {
		return fmt.Errorf("update user: %w", err)
	}
	if rows == 0 {
		return domain.ErrUserNotFound
	}
	return nil
}

var _ domain.UserRepository = (*UserRepository)(nil)

type CredentialRepository struct{ store *Store }

func (s *Store) Credentials() *CredentialRepository { return &CredentialRepository{store: s} }

func (s *CredentialRepository) Create(ctx context.Context, credential domain.PasswordCredential) error {
	userID, err := toUUID(credential.UserID)
	if err != nil {
		return fmt.Errorf("parse credential user id: %w", err)
	}
	if err := s.store.queries.CreateCredential(ctx, sqlcgen.CreateCredentialParams{
		UserID:       userID,
		PasswordHash: credential.PasswordHash,
		UpdatedAt:    toTimestamp(credential.UpdatedAt),
	}); err != nil {
		return fmt.Errorf("insert credential: %w", err)
	}
	return nil
}

func (s *CredentialRepository) ByUserID(ctx context.Context, userID string) (domain.PasswordCredential, error) {
	id, err := toUUID(userID)
	if err != nil {
		return domain.PasswordCredential{}, fmt.Errorf("parse credential user id: %w", err)
	}
	credential, err := s.store.queries.GetCredentialByUserID(ctx, id)
	if err != nil {
		return domain.PasswordCredential{}, mapNotFound(err, domain.ErrUserNotFound, "select credential")
	}
	return domain.PasswordCredential{
		UserID:       fromUUID(credential.UserID),
		PasswordHash: credential.PasswordHash,
		UpdatedAt:    fromTimestamp(credential.UpdatedAt),
	}, nil
}

func (s *CredentialRepository) Update(ctx context.Context, credential domain.PasswordCredential) error {
	userID, err := toUUID(credential.UserID)
	if err != nil {
		return fmt.Errorf("parse credential user id: %w", err)
	}
	if err := s.store.queries.UpdateCredential(ctx, sqlcgen.UpdateCredentialParams{
		UserID:       userID,
		PasswordHash: credential.PasswordHash,
		UpdatedAt:    toTimestamp(credential.UpdatedAt),
	}); err != nil {
		return fmt.Errorf("update credential: %w", err)
	}
	return nil
}

var _ domain.CredentialRepository = (*CredentialRepository)(nil)

type SessionStore struct{ store *Store }

func (s *Store) Sessions() *SessionStore { return &SessionStore{store: s} }

func (s *SessionStore) Create(ctx context.Context, session domain.Session) error {
	if err := createSession(ctx, s.store.queries, session); err != nil {
		return fmt.Errorf("insert session: %w", err)
	}
	return nil
}

func (s *SessionStore) ByTokenHash(ctx context.Context, tokenHash string) (domain.Session, error) {
	session, err := s.store.queries.GetSessionByTokenHash(ctx, tokenHash)
	if err != nil {
		return domain.Session{}, mapNotFound(err, domain.ErrSessionNotFound, "select session")
	}
	return domain.Session{
		ID:        fromUUID(session.ID),
		UserID:    fromUUID(session.UserID),
		TokenHash: session.TokenHash,
		ExpiresAt: fromTimestamp(session.ExpiresAt),
		Revoked:   session.Revoked,
	}, nil
}

func (s *SessionStore) Rotate(ctx context.Context, oldTokenHash string, next domain.Session) error {
	tx, err := s.store.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin session rotation: %w", err)
	}
	defer tx.Rollback(ctx)

	queries := s.store.queries.WithTx(tx)
	rows, err := queries.RevokeSessionForRotation(ctx, oldTokenHash)
	if err != nil {
		return fmt.Errorf("revoke old session: %w", err)
	}
	if rows != 1 {
		return domain.ErrTokenInvalid
	}
	if err := createSession(ctx, queries, next); err != nil {
		return fmt.Errorf("insert rotated session: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit session rotation: %w", err)
	}
	return nil
}

func (s *SessionStore) Revoke(ctx context.Context, tokenHash string) error {
	rows, err := s.store.queries.RevokeSession(ctx, tokenHash)
	if err != nil {
		return fmt.Errorf("revoke session: %w", err)
	}
	if rows == 0 {
		return domain.ErrSessionNotFound
	}
	return nil
}

func (s *SessionStore) RevokeAll(ctx context.Context, userID string) error {
	id, err := toUUID(userID)
	if err != nil {
		return fmt.Errorf("parse session user id: %w", err)
	}
	if err := s.store.queries.RevokeAllSessions(ctx, id); err != nil {
		return fmt.Errorf("revoke user sessions: %w", err)
	}
	return nil
}

var _ domain.SessionStore = (*SessionStore)(nil)

type VerificationTokenStore struct{ store *Store }

func (s *Store) VerificationTokens() *VerificationTokenStore {
	return &VerificationTokenStore{store: s}
}

func (s *VerificationTokenStore) Create(ctx context.Context, token domain.VerificationToken) error {
	userID, err := toUUID(token.UserID)
	if err != nil {
		return fmt.Errorf("parse verification user id: %w", err)
	}
	if err := s.store.queries.CreateVerificationToken(ctx, sqlcgen.CreateVerificationTokenParams{
		TokenHash: token.TokenHash,
		UserID:    userID,
		ExpiresAt: toTimestamp(token.ExpiresAt),
	}); err != nil {
		return fmt.Errorf("insert verification token: %w", err)
	}
	return nil
}

func (s *VerificationTokenStore) Consume(ctx context.Context, tokenHash string) (string, error) {
	userID, err := s.store.queries.ConsumeVerificationToken(ctx, tokenHash)
	if errors.Is(err, pgx.ErrNoRows) {
		expired, expiryErr := s.store.queries.VerificationTokenExpired(ctx, tokenHash)
		if expiryErr == nil && expired {
			return "", domain.ErrVerificationExpired
		}
		return "", domain.ErrVerificationInvalid
	}
	if err != nil {
		return "", fmt.Errorf("consume verification token: %w", err)
	}
	return fromUUID(userID), nil
}

var _ domain.VerificationTokenStore = (*VerificationTokenStore)(nil)

type OutboxStore struct{ store *Store }

func (s *Store) Outbox() *OutboxStore { return &OutboxStore{store: s} }

func (s *OutboxStore) Append(ctx context.Context, event domain.OutboxEvent) error {
	id, err := toUUID(event.ID)
	if err != nil {
		return fmt.Errorf("parse outbox id: %w", err)
	}
	if err := s.store.queries.AppendOutboxEvent(ctx, sqlcgen.AppendOutboxEventParams{
		ID:        id,
		Type:      event.Type,
		Aggregate: event.Aggregate,
		Payload:   event.Payload,
		CreatedAt: toTimestamp(event.CreatedAt),
	}); err != nil {
		return fmt.Errorf("insert outbox event: %w", err)
	}
	return nil
}

var _ domain.OutboxStore = (*OutboxStore)(nil)

func NewPool(ctx context.Context, dsn string, maxConns int32) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse database config: %w", err)
	}
	if maxConns > 0 {
		config.MaxConns = maxConns
	}
	config.MinConns = 1
	config.MaxConnLifetime = 30 * time.Minute
	config.MaxConnIdleTime = 5 * time.Minute
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("open database pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return pool, nil
}

func createSession(ctx context.Context, queries *sqlcgen.Queries, session domain.Session) error {
	id, err := toUUID(session.ID)
	if err != nil {
		return fmt.Errorf("parse session id: %w", err)
	}
	userID, err := toUUID(session.UserID)
	if err != nil {
		return fmt.Errorf("parse session user id: %w", err)
	}
	return queries.CreateSession(ctx, sqlcgen.CreateSessionParams{
		TokenHash: session.TokenHash,
		ID:        id,
		UserID:    userID,
		ExpiresAt: toTimestamp(session.ExpiresAt),
		Revoked:   session.Revoked,
	})
}

func fromSQLUser(user sqlcgen.User) (domain.User, error) {
	if !user.ID.Valid || !user.CreatedAt.Valid || !user.UpdatedAt.Valid {
		return domain.User{}, errors.New("invalid user row")
	}
	return domain.User{
		ID:            fromUUID(user.ID),
		Email:         user.Email,
		EmailVerified: user.EmailVerified,
		Status:        domain.UserStatus(user.Status),
		CreatedAt:     fromTimestamp(user.CreatedAt),
		UpdatedAt:     fromTimestamp(user.UpdatedAt),
	}, nil
}

func toUUID(value string) (pgtype.UUID, error) {
	parsed, err := uuid.Parse(value)
	if err != nil {
		return pgtype.UUID{}, err
	}
	return pgtype.UUID{Bytes: parsed, Valid: true}, nil
}

func fromUUID(value pgtype.UUID) string {
	if !value.Valid {
		return ""
	}
	return uuid.UUID(value.Bytes).String()
}

func toTimestamp(value time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: value, Valid: true}
}

func fromTimestamp(value pgtype.Timestamptz) time.Time { return value.Time }

func mapNotFound(err, notFound error, context string) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return notFound
	}
	return fmt.Errorf("%s: %w", context, err)
}
