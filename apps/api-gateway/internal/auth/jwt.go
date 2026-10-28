package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// ==============================================================================
// Module auth: JWT Token Management & Cryptographic Claims
// Related Documents: docs/SECURITY_GUIDE.md (Section 1) & docs/INTEGRATION.md (Section 2.1)
// Purpose: Issue and validate JWT tokens for Customers, Makers, and Checkers
// ==============================================================================

var (
	ErrInvalidToken = errors.New("invalid or tampered authentication token")
	ErrExpiredToken = errors.New("authentication token has expired")
)

// UserRole maps role-based access permissions within the banking ecosystem.
type UserRole string

const (
	RoleCustomer   UserRole = "CUSTOMER"
	RoleTeller     UserRole = "TELLER"     // Branch cash Maker
	RoleSupervisor UserRole = "SUPERVISOR" // Branch cash authorization Checker
	RoleAdmin      UserRole = "ADMIN"
)

// BankClaims represents the embedded claim payload in JWT tokens.
// Uses struct embedding with jwt.RegisteredClaims per RFC 7519.
type BankClaims struct {
	UserID   string   `json:"user_id"`
	Username string   `json:"username"`
	Role     UserRole `json:"role"`
	BranchID string   `json:"branch_id,omitempty"`
	jwt.RegisteredClaims
}

// TokenManager manages token generation and verification lifecycle.
type TokenManager struct {
	secretKey     []byte
	tokenDuration time.Duration
}

// NewTokenManager creates a new TokenManager instance.
func NewTokenManager(secretKey string, tokenDuration time.Duration) *TokenManager {
	return &TokenManager{
		secretKey:     []byte(secretKey),
		tokenDuration: tokenDuration,
	}
}

// GenerateToken generates a new JWT token signed with HMAC-SHA256.
func (m *TokenManager) GenerateToken(userID, username string, role UserRole, branchID string) (string, error) {
	now := time.Now().UTC()
	claims := BankClaims{
		UserID:   userID,
		Username: username,
		Role:     role,
		BranchID: branchID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(m.tokenDuration)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "aethelcore-auth-gateway",
		},
	}

	// Use HMAC SHA-256 algorithm (SigningMethodHS256)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedString, err := token.SignedString(m.secretKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return signedString, nil
}

// VerifyToken validates cryptographic signature and expiration of JWT token.
func (m *TokenManager) VerifyToken(tokenString string) (*BankClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &BankClaims{}, func(t *jwt.Token) (interface{}, error) {
		// Algorithm Confusion Attack Protection:
		// Ensure signing method is strictly HMAC SHA-256
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected token signing algorithm: %v", t.Header["alg"])
		}
		return m.secretKey, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*BankClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}
