package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// ==============================================================================
// Module auth: JWT Token Management & Cryptographic Claims
// Dokumen Terkait: docs/SECURITY_GUIDE.md (Seksi 1) & docs/INTEGRATION.md (Seksi 2.1)
// Fungsi: Menerbitkan dan memvalidasi token JWT untuk Nasabah, Maker, dan Checker
// ==============================================================================

var (
	ErrInvalidToken = errors.New("invalid or tampered authentication token")
	ErrExpiredToken = errors.New("authentication token has expired")
)

// UserRole memetakan hak akses peran dalam ekosistem perbankan.
type UserRole string

const (
	RoleCustomer   UserRole = "CUSTOMER"
	RoleTeller     UserRole = "TELLER"     // Maker kas cabang
	RoleSupervisor UserRole = "SUPERVISOR" // Checker otorisasi kas cabang
	RoleAdmin      UserRole = "ADMIN"
)

// BankClaims adalah payload data yang tertanam di dalam token JWT.
// Menggunakan struct embedding jwt.RegisteredClaims untuk standar RFC 7519.
type BankClaims struct {
	UserID   string   `json:"user_id"`
	Username string   `json:"username"`
	Role     UserRole `json:"role"`
	BranchID string   `json:"branch_id,omitempty"`
	jwt.RegisteredClaims
}

// TokenManager mengelola siklus pembuatan dan verifikasi token.
type TokenManager struct {
	secretKey     []byte
	tokenDuration time.Duration
}

// NewTokenManager membuat instance TokenManager baru.
func NewTokenManager(secretKey string, tokenDuration time.Duration) *TokenManager {
	return &TokenManager{
		secretKey:     []byte(secretKey),
		tokenDuration: tokenDuration,
	}
}

// GenerateToken membuat token JWT baru bertanda tangan kriptografis HMAC-SHA256.
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

	// Gunakan algoritma HMAC SHA-256 (SigningMethodHS256)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedString, err := token.SignedString(m.secretKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return signedString, nil
}

// VerifyToken memvalidasi keaslian signature dan masa berlaku token JWT.
func (m *TokenManager) VerifyToken(tokenString string) (*BankClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &BankClaims{}, func(t *jwt.Token) (interface{}, error) {
		// Proteksi Algorithm Confusion Attack:
		// Wajib memastikan bahwa algoritma penandatanganan adalah HMAC SHA-256 asli
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
