package repository

import (
	"context"
	"sync"
	"time"

	"github.com/elif-bank/apps/api-gateway/internal/auth"
)

// ==============================================================================
// Repository: In-Memory Auth Stores (Testing & Standalone Local Execution)
// ==============================================================================

type InMemoryUserRepository struct {
	mu    sync.RWMutex
	users map[string]*auth.User
}

func NewInMemoryUserRepository(seedDemoUsers bool) *InMemoryUserRepository {
	repo := &InMemoryUserRepository{
		users: make(map[string]*auth.User),
	}
	if seedDemoUsers {
		// Pre-hash default demo password using Bcrypt cost >= 12
		demoHash, _ := auth.HashPassword("Password123!")
		now := time.Now().UTC()

		repo.AddUser(&auth.User{
			ID:           "usr_teller_01",
			Username:     "teller_budi",
			PasswordHash: demoHash,
			FullName:     "Budi Santoso",
			Role:         auth.RoleTeller,
			BranchID:     "branch-jkt-01",
			Status:       auth.UserStatusActive,
			CreatedAt:    now,
		})
		repo.AddUser(&auth.User{
			ID:           "usr_spv_01",
			Username:     "spv_hendra",
			PasswordHash: demoHash,
			FullName:     "Hendra Wijaya",
			Role:         auth.RoleSupervisor,
			BranchID:     "branch-jkt-01",
			Status:       auth.UserStatusActive,
			CreatedAt:    now,
		})
		repo.AddUser(&auth.User{
			ID:           "usr_admin_01",
			Username:     "admin_jika",
			PasswordHash: demoHash,
			FullName:     "Jika System Admin",
			Role:         auth.RoleAdmin,
			BranchID:     "branch-hq",
			Status:       auth.UserStatusActive,
			CreatedAt:    now,
		})
	}
	return repo
}

func (r *InMemoryUserRepository) AddUser(u *auth.User) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.users[u.ID] = u
}

func (r *InMemoryUserRepository) GetByUsername(ctx context.Context, username string) (*auth.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, u := range r.users {
		if u.Username == username {
			copyUser := *u
			return &copyUser, nil
		}
	}
	return nil, auth.ErrUserNotFound
}

func (r *InMemoryUserRepository) GetByID(ctx context.Context, id string) (*auth.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	u, ok := r.users[id]
	if !ok {
		return nil, auth.ErrUserNotFound
	}
	copyUser := *u
	return &copyUser, nil
}

func (r *InMemoryUserRepository) UpdateLastLogin(ctx context.Context, userID string, loginAt time.Time) error {
	return nil
}

type refreshRecord struct {
	userID    string
	expiresAt time.Time
}

type InMemoryRefreshTokenStore struct {
	mu     sync.RWMutex
	tokens map[string]refreshRecord
}

func NewInMemoryRefreshTokenStore() *InMemoryRefreshTokenStore {
	return &InMemoryRefreshTokenStore{
		tokens: make(map[string]refreshRecord),
	}
}

func (s *InMemoryRefreshTokenStore) Save(ctx context.Context, tokenID, userID string, expiresAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tokens[tokenID] = refreshRecord{
		userID:    userID,
		expiresAt: expiresAt,
	}
	return nil
}

func (s *InMemoryRefreshTokenStore) IsValid(ctx context.Context, tokenID string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rec, ok := s.tokens[tokenID]
	if !ok {
		return false, nil
	}
	if time.Now().UTC().After(rec.expiresAt) {
		return false, nil
	}
	return true, nil
}

func (s *InMemoryRefreshTokenStore) Revoke(ctx context.Context, tokenID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.tokens, tokenID)
	return nil
}
