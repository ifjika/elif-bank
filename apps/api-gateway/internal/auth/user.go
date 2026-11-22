package auth

import (
	"context"
	"errors"
	"time"
)

// ==============================================================================
// Module auth: User Domain & Repository Ports
// Related Documents: docs/TSD.md (app_users) & docs/IMPLEMENTATION_PLAN.md (4.1.2)
// ==============================================================================

type UserStatus string

const (
	UserStatusActive    UserStatus = "ACTIVE"
	UserStatusSuspended UserStatus = "SUSPENDED"
)

var (
	ErrUserNotFound       = errors.New("user not found")
	ErrInvalidCredentials = errors.New("invalid username or password")
	ErrUserSuspended      = errors.New("user account is suspended")
	ErrEmptyCredentials   = errors.New("username and password are required")
)

// User represents an authenticated operator or customer record from app_users.
type User struct {
	ID           string     `json:"user_id"`
	Username     string     `json:"username"`
	PasswordHash string     `json:"-"` // Never serialize password hash to JSON
	FullName     string     `json:"full_name"`
	Role         UserRole   `json:"role"`
	BranchID     string     `json:"branch_id,omitempty"`
	Status       UserStatus `json:"status"`
	CreatedAt    time.Time  `json:"created_at"`
}

// UserRepository defines persistence operations for querying app_users.
type UserRepository interface {
	GetByUsername(ctx context.Context, username string) (*User, error)
	GetByID(ctx context.Context, id string) (*User, error)
	UpdateLastLogin(ctx context.Context, userID string, loginAt time.Time) error
}

// RefreshTokenStore tracks stateful refresh token validity for revocation on logout or compromise.
type RefreshTokenStore interface {
	Save(ctx context.Context, tokenID, userID string, expiresAt time.Time) error
	IsValid(ctx context.Context, tokenID string) (bool, error)
	Revoke(ctx context.Context, tokenID string) error
}
