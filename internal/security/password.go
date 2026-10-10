package security

import "golang.org/x/crypto/bcrypt"

type BcryptPasswords struct{}

func (BcryptPasswords) Hash(password string) (string, error) {
	value, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(value), err
}

func (BcryptPasswords) Compare(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
