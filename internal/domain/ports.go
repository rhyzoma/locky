package domain

import "context"

type UserRepository interface {
	Create(ctx context.Context, user User) error
	ByEmail(ctx context.Context, email string) (User, error)
	ByID(ctx context.Context, id string) (User, error)
	Update(ctx context.Context, user User) error
}

type CredentialRepository interface {
	Create(ctx context.Context, credential PasswordCredential) error
	ByUserID(ctx context.Context, userID string) (PasswordCredential, error)
	Update(ctx context.Context, credential PasswordCredential) error
}

type SessionStore interface {
	Create(ctx context.Context, session Session) error
	ByTokenHash(ctx context.Context, tokenHash string) (Session, error)
	Rotate(ctx context.Context, oldTokenHash string, next Session) error
	Revoke(ctx context.Context, tokenHash string) error
	RevokeAll(ctx context.Context, userID string) error
}

type VerificationTokenStore interface {
	Create(ctx context.Context, token VerificationToken) error
	Consume(ctx context.Context, tokenHash string) (string, error)
}

type OutboxStore interface {
	Append(ctx context.Context, event OutboxEvent) error
}

type Mailer interface {
	SendVerification(ctx context.Context, email, token string) error
	SendPasswordReset(ctx context.Context, email, token string) error
}
