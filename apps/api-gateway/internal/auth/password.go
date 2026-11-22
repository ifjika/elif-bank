package auth

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

// ==============================================================================
// Module auth: Password Hashing & Verification Engine
// Related Documents: docs/SECURITY_GUIDE.md (Section 3) & docs/IMPLEMENTATION_PLAN.md (4.1.1)
// Purpose: One-way password hashing with Bcrypt (Cost >= 12) and constant-time check
// ==============================================================================

const (
	// DefaultBcryptCost defines the key-stretching cost factor (2^12 = 4,096 rounds).
	DefaultBcryptCost = 12
)

var (
	ErrPasswordTooShort = errors.New("password must be at least 8 characters")
	ErrPasswordMismatch = errors.New("invalid password")
)

// HashPassword generates a salted bcrypt hash of the plaintext password.
func HashPassword(password string) (string, error) {
	if len(password) < 8 {
		return "", ErrPasswordTooShort
	}

	bytes, err := bcrypt.GenerateFromPassword([]byte(password), DefaultBcryptCost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// VerifyPassword compares a plaintext password against a stored bcrypt hash in constant time.
func VerifyPassword(password, hash string) error {
	if password == "" || hash == "" {
		return ErrPasswordMismatch
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		return ErrPasswordMismatch
	}
	return nil
}
