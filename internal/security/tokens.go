package security

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type JWT struct {
	Key      []byte
	KeyID    string
	Issuer   string
	Audience string
	Now      func() time.Time
}

func (j JWT) Sign(userID, email string, lifetime time.Duration) (string, error) {
	now := time.Now
	if j.Now != nil {
		now = j.Now
	}
	issuedAt := now()
	claims := jwt.RegisteredClaims{Subject: userID, Issuer: j.Issuer, Audience: []string{j.Audience}, IssuedAt: jwt.NewNumericDate(issuedAt), ExpiresAt: jwt.NewNumericDate(issuedAt.Add(lifetime))}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"sub": claims.Subject, "iss": claims.Issuer, "aud": claims.Audience, "iat": claims.IssuedAt, "exp": claims.ExpiresAt, "email": email})
	if j.KeyID != "" {
		token.Header["kid"] = j.KeyID
	}
	return token.SignedString(j.Key)
}
