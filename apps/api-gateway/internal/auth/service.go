package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
)

// ==============================================================================
// Module auth: Authentication Usecase (Login, Silent Refresh, Logout)
// Related Documents: docs/IMPLEMENTATION_PLAN.md (4.1.3) & docs/SECURITY_GUIDE.md
// ==============================================================================

const RefreshTokenTTL = 7 * 24 * time.Hour

// LoginResult represents the output of a successful authentication.
type LoginResult struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"-"` // Transmitted strictly via HttpOnly cookie, never in JSON body
	User         *User  `json:"user"`
}

// RefreshResult represents the output of a silent token refresh.
type RefreshResult struct {
	AccessToken string `json:"access_token"`
	User        *User  `json:"user"`
}

// AuthService orchestrates credential verification, dual-token issuance, and session revocation.
type AuthService struct {
	userRepo            UserRepository
	refreshStore        RefreshTokenStore
	accessTokenManager  *TokenManager
	refreshTokenManager *TokenManager
}

// NewAuthService creates a new AuthService instance.
func NewAuthService(
	userRepo UserRepository,
	refreshStore RefreshTokenStore,
	accessMgr *TokenManager,
	refreshMgr *TokenManager,
) *AuthService {
	return &AuthService{
		userRepo:            userRepo,
		refreshStore:        refreshStore,
		accessTokenManager:  accessMgr,
		refreshTokenManager: refreshMgr,
	}
}

// Login validates credentials and issues a short-lived Access Token and stateful Refresh Token.
func (s *AuthService) Login(ctx context.Context, username, password string) (*LoginResult, error) {
	username = strings.TrimSpace(username)
	if username == "" || password == "" {
		return nil, ErrEmptyCredentials
	}

	user, err := s.userRepo.GetByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			// Return generic credential error to prevent username enumeration
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("failed to fetch user: %w", err)
	}

	if user.Status == UserStatusSuspended {
		return nil, ErrUserSuspended
	}

	if err := VerifyPassword(password, user.PasswordHash); err != nil {
		return nil, ErrInvalidCredentials
	}

	accessToken, err := s.accessTokenManager.GenerateToken(user.ID, user.Username, user.Role, user.BranchID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	tokenID, err := generateSecureTokenID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token ID: %w", err)
	}

	refreshToken, err := s.refreshTokenManager.GenerateTokenWithID(tokenID, user.ID, user.Username, user.Role, user.BranchID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	expiresAt := time.Now().UTC().Add(RefreshTokenTTL)
	if err := s.refreshStore.Save(ctx, tokenID, user.ID, expiresAt); err != nil {
		return nil, fmt.Errorf("failed to persist refresh session: %w", err)
	}

	// Best-effort audit timestamp update on app_users.last_login_at
	_ = s.userRepo.UpdateLastLogin(ctx, user.ID, time.Now().UTC())

	return &LoginResult{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         user,
	}, nil
}

// Refresh validates an HttpOnly refresh token and issues a fresh Access Token.
func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (*RefreshResult, error) {
	if strings.TrimSpace(refreshToken) == "" {
		return nil, ErrInvalidToken
	}

	claims, err := s.refreshTokenManager.VerifyToken(refreshToken)
	if err != nil {
		return nil, err
	}

	if claims.ID == "" {
		return nil, ErrInvalidToken
	}

	valid, err := s.refreshStore.IsValid(ctx, claims.ID)
	if err != nil || !valid {
		return nil, ErrInvalidToken
	}

	// Re-check user status in DB in case account was suspended after login
	user, err := s.userRepo.GetByID(ctx, claims.UserID)
	if err != nil {
		return nil, ErrInvalidToken
	}
	if user.Status == UserStatusSuspended {
		_ = s.refreshStore.Revoke(ctx, claims.ID)
		return nil, ErrUserSuspended
	}

	accessToken, err := s.accessTokenManager.GenerateToken(user.ID, user.Username, user.Role, user.BranchID)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh access token: %w", err)
	}

	return &RefreshResult{
		AccessToken: accessToken,
		User:        user,
	}, nil
}

// Logout revokes the stateful refresh token so it cannot be reused.
func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	if strings.TrimSpace(refreshToken) == "" {
		return nil
	}

	claims, err := s.refreshTokenManager.VerifyToken(refreshToken)
	if err != nil || claims.ID == "" {
		return nil
	}

	return s.refreshStore.Revoke(ctx, claims.ID)
}

func generateSecureTokenID() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
