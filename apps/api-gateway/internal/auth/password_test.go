package auth

import (
	"errors"
	"testing"
)

func TestPassword_HashAndVerify_Success(t *testing.T) {
	// 1. ARRANGE
	rawPassword := "BankElif2026!"

	// 2. ACT
	hash, err := HashPassword(rawPassword)

	// 3. ASSERT
	if err != nil {
		t.Fatalf("expected no error hashing password, got: %v", err)
	}
	if hash == "" || hash == rawPassword {
		t.Fatal("hash must not be empty or equal to plaintext password")
	}

	// Verify unique salt produces different hashes for same input
	hash2, _ := HashPassword(rawPassword)
	if hash == hash2 {
		t.Error("expected unique salt per hash invocation")
	}

	if err := VerifyPassword(rawPassword, hash); err != nil {
		t.Errorf("expected VerifyPassword to succeed, got: %v", err)
	}
}

func TestPassword_Verify_Mismatch(t *testing.T) {
	// 1. ARRANGE
	hash, _ := HashPassword("CorrectPassword123!")

	// 2. ACT
	err := VerifyPassword("WrongPassword123!", hash)

	// 3. ASSERT
	if !errors.Is(err, ErrPasswordMismatch) {
		t.Fatalf("expected ErrPasswordMismatch, got: %v", err)
	}
}

func TestPassword_TooShort(t *testing.T) {
	// 1. ARRANGE & 2. ACT
	_, err := HashPassword("short")

	// 3. ASSERT
	if !errors.Is(err, ErrPasswordTooShort) {
		t.Fatalf("expected ErrPasswordTooShort, got: %v", err)
	}
}
