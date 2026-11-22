package auth_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/elif-bank/apps/api-gateway/internal/auth"
	"github.com/elif-bank/apps/api-gateway/internal/repository"
)

func setupTestAuthService(t *testing.T) (*auth.AuthService, *repository.InMemoryUserRepository) {
	t.Helper()
	userRepo := repository.NewInMemoryUserRepository(false)
	refreshStore := repository.NewInMemoryRefreshTokenStore()

	hash, err := auth.HashPassword("SecretPass123!")
	if err != nil {
		t.Fatalf("failed to hash test password: %v", err)
	}

	userRepo.AddUser(&auth.User{
		ID:           "usr_001",
		Username:     "teller_budi",
		PasswordHash: hash,
		FullName:     "Budi Santoso",
		Role:         auth.RoleTeller,
		BranchID:     "branch-01",
		Status:       auth.UserStatusActive,
	})
	userRepo.AddUser(&auth.User{
		ID:           "usr_002",
		Username:     "suspended_user",
		PasswordHash: hash,
		FullName:     "Suspended Operator",
		Role:         auth.RoleTeller,
		BranchID:     "branch-01",
		Status:       auth.UserStatusSuspended,
	})

	accessMgr := auth.NewTokenManager("access-secret-key-32-bytes-minimum!", 15*time.Minute)
	refreshMgr := auth.NewTokenManager("refresh-secret-key-32-bytes-minimum", 7*24*time.Hour)

	svc := auth.NewAuthService(userRepo, refreshStore, accessMgr, refreshMgr)
	return svc, userRepo
}

func TestAuthService_Login_Success(t *testing.T) {
	// 1. ARRANGE
	svc, _ := setupTestAuthService(t)

	// 2. ACT
	res, err := svc.Login(context.Background(), "teller_budi", "SecretPass123!")

	// 3. ASSERT
	if err != nil {
		t.Fatalf("expected successful login, got error: %v", err)
	}
	if res.AccessToken == "" || res.RefreshToken == "" {
		t.Fatal("expected non-empty access and refresh tokens")
	}
	if res.User.Role != auth.RoleTeller {
		t.Errorf("expected role TELLER, got %s", res.User.Role)
	}
}

func TestAuthService_Login_WrongPassword(t *testing.T) {
	// 1. ARRANGE
	svc, _ := setupTestAuthService(t)

	// 2. ACT
	_, err := svc.Login(context.Background(), "teller_budi", "WrongPassword!")

	// 3. ASSERT
	if !errors.Is(err, auth.ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got: %v", err)
	}
}

func TestAuthService_Login_UnknownUser(t *testing.T) {
	// 1. ARRANGE
	svc, _ := setupTestAuthService(t)

	// 2. ACT
	_, err := svc.Login(context.Background(), "non_existent", "SecretPass123!")

	// 3. ASSERT
	if !errors.Is(err, auth.ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials (anti-enumeration), got: %v", err)
	}
}

func TestAuthService_Login_SuspendedUser(t *testing.T) {
	// 1. ARRANGE
	svc, _ := setupTestAuthService(t)

	// 2. ACT
	_, err := svc.Login(context.Background(), "suspended_user", "SecretPass123!")

	// 3. ASSERT
	if !errors.Is(err, auth.ErrUserSuspended) {
		t.Fatalf("expected ErrUserSuspended, got: %v", err)
	}
}

func TestAuthService_Login_EmptyCredentials(t *testing.T) {
	// 1. ARRANGE
	svc, _ := setupTestAuthService(t)

	// 2. ACT
	_, err := svc.Login(context.Background(), "", "")

	// 3. ASSERT
	if !errors.Is(err, auth.ErrEmptyCredentials) {
		t.Fatalf("expected ErrEmptyCredentials, got: %v", err)
	}
}

func TestAuthService_RefreshAndLogout_Lifecycle(t *testing.T) {
	// 1. ARRANGE
	svc, _ := setupTestAuthService(t)
	loginRes, err := svc.Login(context.Background(), "teller_budi", "SecretPass123!")
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}

	// 2. ACT - Silent Refresh
	refreshRes, err := svc.Refresh(context.Background(), loginRes.RefreshToken)
	if err != nil {
		t.Fatalf("expected refresh to succeed, got: %v", err)
	}
	if refreshRes.AccessToken == "" {
		t.Fatal("expected new access token from refresh")
	}

	// 2b. ACT - Logout (Revoke Refresh Token)
	if err := svc.Logout(context.Background(), loginRes.RefreshToken); err != nil {
		t.Fatalf("logout failed: %v", err)
	}

	// 3. ASSERT - Refresh after logout must fail
	_, err = svc.Refresh(context.Background(), loginRes.RefreshToken)
	if !errors.Is(err, auth.ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken after logout revocation, got: %v", err)
	}
}
