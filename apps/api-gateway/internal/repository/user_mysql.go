package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/elif-bank/apps/api-gateway/internal/auth"
)

// ==============================================================================
// Repository: MySQL / TiDB UserRepository Implementation
// Related Documents: deploy/migrations/mysql_tidb/000001_init_bank_schema.up.sql
// ==============================================================================

type MySQLUserRepository struct {
	db *sql.DB
}

func NewMySQLUserRepository(db *sql.DB) *MySQLUserRepository {
	return &MySQLUserRepository{db: db}
}

func (r *MySQLUserRepository) GetByUsername(ctx context.Context, username string) (*auth.User, error) {
	const query = `
		SELECT id, username, password_hash, full_name, role, COALESCE(branch_id, ''), status, created_at
		FROM app_users
		WHERE username = ?
		LIMIT 1
	`
	return r.scanUser(r.db.QueryRowContext(ctx, query, username))
}

func (r *MySQLUserRepository) GetByID(ctx context.Context, id string) (*auth.User, error) {
	const query = `
		SELECT id, username, password_hash, full_name, role, COALESCE(branch_id, ''), status, created_at
		FROM app_users
		WHERE id = ?
		LIMIT 1
	`
	return r.scanUser(r.db.QueryRowContext(ctx, query, id))
}

func (r *MySQLUserRepository) UpdateLastLogin(ctx context.Context, userID string, loginAt time.Time) error {
	const query = `UPDATE app_users SET last_login_at = ? WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, loginAt, userID)
	return err
}

func (r *MySQLUserRepository) scanUser(row *sql.Row) (*auth.User, error) {
	var u auth.User
	var roleStr, statusStr string

	err := row.Scan(
		&u.ID,
		&u.Username,
		&u.PasswordHash,
		&u.FullName,
		&roleStr,
		&u.BranchID,
		&statusStr,
		&u.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, auth.ErrUserNotFound
		}
		return nil, err
	}

	u.Role = auth.UserRole(roleStr)
	u.Status = auth.UserStatus(statusStr)
	return &u, nil
}

// SeedDemoUsersIfEmpty inserts initial branch & operator accounts into MySQL/TiDB
// when app_users is empty. Existing user records are never overwritten.
func SeedDemoUsersIfEmpty(ctx context.Context, db *sql.DB) error {
	var count int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM app_users").Scan(&count); err != nil {
		return fmt.Errorf("failed to check app_users count (ensure 000001_init_bank_schema.up.sql migration is applied): %w", err)
	}
	if count > 0 {
		return nil
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin seed transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	const branchQuery = `
		INSERT IGNORE INTO bank_branches (id, branch_code, branch_name, address)
		VALUES
			('branch-jkt-01', 'JKT01', 'Elif Bank Sudirma1n Main Branch', 'Jl. Jend. Sudirman Kav. 52, Jakarta'),
			('branch-hq', 'HQ001', 'Elif Bank Head Office', 'Menara Elif, Jakarta')
	`
	if _, err := tx.ExecContext(ctx, branchQuery); err != nil {
		return fmt.Errorf("failed to seed bank_branches: %w", err)
	}

	demoHash, err := auth.HashPassword("Password123!")
	if err != nil {
		return fmt.Errorf("failed to hash demo password: %w", err)
	}

	const userQuery = `
		INSERT INTO app_users (id, username, password_hash, full_name, role, branch_id, status)
		VALUES
			('usr_teller_01', 'teller_budi', ?, 'Budi Santoso', 'TELLER', 'branch-jkt-01', 'ACTIVE'),
			('usr_spv_01', 'spv_hendra', ?, 'Hendra Wijaya', 'SUPERVISOR', 'branch-jkt-01', 'ACTIVE'),
			('usr_admin_01', 'admin_jika', ?, 'Jika System Admin', 'ADMIN', 'branch-hq', 'ACTIVE')
	`
	if _, err := tx.ExecContext(ctx, userQuery, demoHash, demoHash, demoHash); err != nil {
		return fmt.Errorf("failed to seed app_users: %w", err)
	}

	return tx.Commit()
}
