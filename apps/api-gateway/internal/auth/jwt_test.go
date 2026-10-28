package auth

import (
	"errors"
	"testing"
	"time"
)

func TestTokenManager_GenerateAndVerify_Success(t *testing.T) {
	secret := "super-secure-banking-secret-key-32b"
	manager := NewTokenManager(secret, 15*time.Minute)

	tokenStr, err := manager.GenerateToken("user-1001", "teller_budi", RoleTeller, "branch-jkt-01")
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}
	if tokenStr == "" {
		t.Fatal("expected non-empty token string")
	}

	claims, err := manager.VerifyToken(tokenStr)
	if err != nil {
		t.Fatalf("failed to verify valid token: %v", err)
	}

	if claims.UserID != "user-1001" {
		t.Errorf("expected UserID 'user-1001', got '%s'", claims.UserID)
	}
	if claims.Username != "teller_budi" {
		t.Errorf("expected Username 'teller_budi', got '%s'", claims.Username)
	}
	if claims.Role != RoleTeller {
		t.Errorf("expected Role 'TELLER', got '%s'", claims.Role)
	}
	if claims.BranchID != "branch-jkt-01" {
		t.Errorf("expected BranchID 'branch-jkt-01', got '%s'", claims.BranchID)
	}
}

func TestTokenManager_ExpiredToken(t *testing.T) {
	secret := "super-secure-banking-secret-key-32b"
	// Negative lifespan (-1 second) to trigger immediate token expiration
	manager := NewTokenManager(secret, -1*time.Second)

	tokenStr, err := manager.GenerateToken("user-1002", "customer_ani", RoleCustomer, "")
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	_, err = manager.VerifyToken(tokenStr)
	if !errors.Is(err, ErrExpiredToken) {
		t.Fatalf("expected ErrExpiredToken, got: %v", err)
	}
}

func TestTokenManager_TamperedToken(t *testing.T) {
	secret := "super-secure-banking-secret-key-32b"
	manager := NewTokenManager(secret, 15*time.Minute)

	tokenStr, err := manager.GenerateToken("user-1003", "spv_hendra", RoleSupervisor, "branch-jkt-01")
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	// Modify one character at the end to invalidate the cryptographic signature
	tamperedToken := tokenStr[:len(tokenStr)-4] + "xxxx"

	_, err = manager.VerifyToken(tamperedToken)
	if !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken on tampered token, got: %v", err)
	}
}
