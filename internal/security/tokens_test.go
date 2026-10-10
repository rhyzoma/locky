package security

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestJWTSignIncludesKeyID(t *testing.T) {
	issuedAt := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	issuer := JWT{
		Key:      []byte("a-test-signing-key-that-is-long-enough"),
		KeyID:    "2026-01",
		Issuer:   "locky",
		Audience: "locky",
		Now:      func() time.Time { return issuedAt },
	}

	rawToken, err := issuer.Sign("user-1", "user@example.com", time.Minute)
	if err != nil {
		t.Fatalf("Sign() error = %v", err)
	}

	token, err := jwt.Parse(rawToken, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			t.Fatalf("signing method = %v, want HS256", token.Method.Alg())
		}
		if token.Header["kid"] != "2026-01" {
			t.Fatalf("kid = %v, want %q", token.Header["kid"], "2026-01")
		}
		return issuer.Key, nil
	})
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if !token.Valid {
		t.Fatal("Parse() token is invalid")
	}
}
