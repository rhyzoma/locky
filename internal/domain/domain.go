package domain

import (
	"time"
)

type UserStatus string

const (
	UserStatusPending  UserStatus = "pending"
	UserStatusActive   UserStatus = "active"
	UserStatusDisabled UserStatus = "disabled"
)

type User struct {
	ID            string     `json:"id"`
	Email         string     `json:"email"`
	EmailVerified bool       `json:"email_verified"`
	Status        UserStatus `json:"status"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

type PasswordCredential struct {
	UserID       string
	PasswordHash string
	UpdatedAt    time.Time
}

type Session struct {
	ID        string
	UserID    string
	TokenHash string
	ExpiresAt time.Time
	Revoked   bool
}

type VerificationToken struct {
	TokenHash string
	UserID    string
	ExpiresAt time.Time
	Used      bool
}

type TokenPair struct {
	AccessToken  string        `json:"access_token"`
	RefreshToken string        `json:"refresh_token"`
	ExpiresIn    time.Duration `json:"expires_in"`
}

type OutboxEvent struct {
	ID        string
	Type      string
	Aggregate string
	Payload   []byte
	CreatedAt time.Time
}
