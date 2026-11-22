package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/elif-bank/apps/core-ledger/internal/domain"
	"github.com/elif-bank/pkg/hashchain"
)

// ==============================================================================
// Repository: MySQL 8.0 / TiDB Cloud Serverless Core Ledger Implementation
// Related Documents: deploy/migrations/mysql_tidb/000001_init_bank_schema.up.sql
// ==============================================================================

type txContextKey struct{}

type sqlExecutor interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
}

func getExecutor(ctx context.Context, db *sql.DB) sqlExecutor {
	if tx, ok := ctx.Value(txContextKey{}).(*sql.Tx); ok && tx != nil {
		return tx
	}
	return db
}

// MySQLTransactionManager manages ACID database transactions via context propagation.
type MySQLTransactionManager struct {
	db *sql.DB
}

func NewMySQLTransactionManager(db *sql.DB) *MySQLTransactionManager {
	return &MySQLTransactionManager{db: db}
}

func (m *MySQLTransactionManager) ExecuteInTransaction(ctx context.Context, fn func(txCtx context.Context) error) error {
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin ACID transaction: %w", err)
	}

	txCtx := context.WithValue(ctx, txContextKey{}, tx)
	if err := fn(txCtx); err != nil {
		_ = tx.Rollback()
		return err
	}

	if err := tx.Commit(); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("failed to commit ACID transaction: %w", err)
	}
	return nil
}

// MySQLAccountRepository implements domain.AccountRepository against MySQL/TiDB.
type MySQLAccountRepository struct {
	db *sql.DB
}

func NewMySQLAccountRepository(db *sql.DB) *MySQLAccountRepository {
	return &MySQLAccountRepository{db: db}
}

func (r *MySQLAccountRepository) GetByID(ctx context.Context, id string) (*domain.Account, error) {
	const query = `
		SELECT id, account_number, customer_id, account_type, currency, balance, hold_balance, status, version, created_at, updated_at
		FROM accounts WHERE id = ? LIMIT 1
	`
	return r.scanAccount(getExecutor(ctx, r.db).QueryRowContext(ctx, query, id))
}

func (r *MySQLAccountRepository) GetByAccountNumber(ctx context.Context, accNum string) (*domain.Account, error) {
	const query = `
		SELECT id, account_number, customer_id, account_type, currency, balance, hold_balance, status, version, created_at, updated_at
		FROM accounts WHERE account_number = ? LIMIT 1
	`
	return r.scanAccount(getExecutor(ctx, r.db).QueryRowContext(ctx, query, accNum))
}

func (r *MySQLAccountRepository) GetByIDWithLock(ctx context.Context, id string) (*domain.Account, error) {
	const query = `
		SELECT id, account_number, customer_id, account_type, currency, balance, hold_balance, status, version, created_at, updated_at
		FROM accounts WHERE id = ? FOR UPDATE
	`
	return r.scanAccount(getExecutor(ctx, r.db).QueryRowContext(ctx, query, id))
}

func (r *MySQLAccountRepository) UpdateBalance(ctx context.Context, acc *domain.Account) error {
	const query = `
		UPDATE accounts
		SET balance = ?, hold_balance = ?, version = version + 1, updated_at = ?
		WHERE id = ?
	`
	_, err := getExecutor(ctx, r.db).ExecContext(ctx, query, acc.Balance, acc.HoldBalance, acc.UpdatedAt, acc.ID)
	return err
}

func (r *MySQLAccountRepository) scanAccount(row *sql.Row) (*domain.Account, error) {
	var acc domain.Account
	var custID sql.NullString
	var accType, status string

	err := row.Scan(
		&acc.ID,
		&acc.AccountNumber,
		&custID,
		&accType,
		&acc.Currency,
		&acc.Balance,
		&acc.HoldBalance,
		&status,
		&acc.Version,
		&acc.CreatedAt,
		&acc.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrAccountNotFound
		}
		return nil, err
	}

	if custID.Valid {
		acc.CustomerID = &custID.String
	}
	acc.AccountType = domain.AccountType(accType)
	acc.Status = domain.AccountStatus(status)
	return &acc, nil
}

// MySQLLedgerRepository implements domain.LedgerRepository and backoffice query views.
type MySQLLedgerRepository struct {
	db *sql.DB
}

func NewMySQLLedgerRepository(db *sql.DB) *MySQLLedgerRepository {
	return &MySQLLedgerRepository{db: db}
}

func (r *MySQLLedgerRepository) CreateTransaction(ctx context.Context, tx *domain.Transaction) error {
	const query = `
		INSERT INTO transactions (
			id, reference_id, transaction_type, amount, currency, business_date,
			description, target_account_id, vault_account_id, maker_id, status, created_at
		) VALUES (?, ?, ?, ?, 'IDR', ?, ?, ?, ?, ?, ?, ?)
	`
	businessDate := tx.CreatedAt.UTC().Format("2006-01-02")
	_, err := getExecutor(ctx, r.db).ExecContext(
		ctx,
		query,
		tx.ID,
		tx.ReferenceID,
		string(tx.TransactionType),
		tx.Amount,
		businessDate,
		tx.Description,
		tx.TargetAccountID,
		tx.VaultAccountID,
		tx.MakerID,
		string(tx.Status),
		tx.CreatedAt,
	)
	return err
}

func (r *MySQLLedgerRepository) GetTransactionByIDWithLock(ctx context.Context, id string) (*domain.Transaction, error) {
	const query = `
		SELECT id, reference_id, transaction_type, amount, description,
		       target_account_id, vault_account_id, maker_id, checker_id,
		       checker_notes, status, approved_at, created_at
		FROM transactions
		WHERE id = ?
		FOR UPDATE
	`
	var tx domain.Transaction
	var txType, status string
	var targetAccID, vaultAccID, checkerID, checkerNotes sql.NullString
	var approvedAt sql.NullTime

	err := getExecutor(ctx, r.db).QueryRowContext(ctx, query, id).Scan(
		&tx.ID,
		&tx.ReferenceID,
		&txType,
		&tx.Amount,
		&tx.Description,
		&targetAccID,
		&vaultAccID,
		&tx.MakerID,
		&checkerID,
		&checkerNotes,
		&status,
		&approvedAt,
		&tx.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrTxNotFound
		}
		return nil, err
	}

	tx.TransactionType = domain.TransactionType(txType)
	tx.Status = domain.TransactionStatus(status)
	if targetAccID.Valid {
		tx.TargetAccountID = &targetAccID.String
	}
	if vaultAccID.Valid {
		tx.VaultAccountID = &vaultAccID.String
	}
	if checkerID.Valid {
		tx.CheckerID = &checkerID.String
	}
	if checkerNotes.Valid {
		tx.CheckerNotes = &checkerNotes.String
	}
	if approvedAt.Valid {
		tx.ApprovedAt = &approvedAt.Time
	}
	return &tx, nil
}

func (r *MySQLLedgerRepository) UpdateTransactionStatus(ctx context.Context, tx *domain.Transaction) error {
	const query = `
		UPDATE transactions
		SET status = ?, checker_id = ?, checker_notes = ?, approved_at = ?
		WHERE id = ?
	`
	_, err := getExecutor(ctx, r.db).ExecContext(
		ctx,
		query,
		string(tx.Status),
		tx.CheckerID,
		tx.CheckerNotes,
		tx.ApprovedAt,
		tx.ID,
	)
	return err
}

func (r *MySQLLedgerRepository) CreateLedgerEntries(ctx context.Context, entries []domain.LedgerEntry) error {
	const query = `
		INSERT INTO ledger_entries (
			transaction_id, account_id, entry_type, amount, balance_after,
			previous_hash, current_hash, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`
	exec := getExecutor(ctx, r.db)
	for _, entry := range entries {
		if _, err := exec.ExecContext(
			ctx,
			query,
			entry.TransactionID,
			entry.AccountID,
			string(entry.EntryType),
			entry.Amount,
			entry.BalanceAfter,
			entry.PreviousHash,
			entry.CurrentHash,
			entry.CreatedAt,
		); err != nil {
			return err
		}
	}
	return nil
}

func (r *MySQLLedgerRepository) GetLatestEntryHash(ctx context.Context, accountID string) (string, error) {
	const query = `
		SELECT current_hash FROM ledger_entries
		WHERE account_id = ?
		ORDER BY id DESC
		LIMIT 1
	`
	var hash string
	err := getExecutor(ctx, r.db).QueryRowContext(ctx, query, accountID).Scan(&hash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return hashchain.GenesisHash, nil
		}
		return "", err
	}
	return hash, nil
}

func (r *MySQLLedgerRepository) CreateOutboxEvent(ctx context.Context, eventType, aggregateID string, payload []byte) error {
	const query = `
		INSERT INTO outbox_events (id, aggregate_type, aggregate_id, event_type, payload, status, created_at)
		VALUES (?, 'TRANSACTION', ?, ?, ?, 'PENDING', ?)
	`
	eventID := fmt.Sprintf("evt_%d", time.Now().UnixNano())
	_, err := getExecutor(ctx, r.db).ExecContext(ctx, query, eventID, aggregateID, eventType, string(payload), time.Now().UTC())
	return err
}

// PendingBranchTx represents a branch transaction awaiting Supervisor approval.
type PendingBranchTx struct {
	ID              string    `json:"id"`
	AccountNumber   string    `json:"account_number"`
	CustomerName    string    `json:"customer_name"`
	TransactionType string    `json:"transaction_type"`
	Amount          int64     `json:"amount"`
	MakerName       string    `json:"maker_name"`
	Notes           string    `json:"notes"`
	CreatedAt       time.Time `json:"created_at"`
}

func (r *MySQLLedgerRepository) ListPendingBranchTransactions(ctx context.Context) ([]PendingBranchTx, error) {
	const query = `
		SELECT t.id, COALESCE(a.account_number, '-'), COALESCE(c.full_name, 'Customer'),
		       t.transaction_type, t.amount, COALESCE(u.username, t.maker_id),
		       t.description, t.created_at
		FROM transactions t
		LEFT JOIN accounts a ON t.target_account_id = a.id
		LEFT JOIN customers c ON a.customer_id = c.id
		LEFT JOIN app_users u ON t.maker_id = u.id
		WHERE t.status = 'PENDING_APPROVAL'
		ORDER BY t.created_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []PendingBranchTx
	for rows.Next() {
		var item PendingBranchTx
		if err := rows.Scan(
			&item.ID,
			&item.AccountNumber,
			&item.CustomerName,
			&item.TransactionType,
			&item.Amount,
			&item.MakerName,
			&item.Notes,
			&item.CreatedAt,
		); err != nil {
			return nil, err
		}
		list = append(list, item)
	}
	return list, rows.Err()
}

// AuditLedgerRow represents a persisted double-entry mutation row for audit inspection.
type AuditLedgerRow struct {
	ID            int64     `json:"id"`
	TransactionID string    `json:"transaction_id"`
	AccountNumber string    `json:"account_number"`
	EntryType     string    `json:"entry_type"`
	Amount        int64     `json:"amount"`
	BalanceAfter  int64     `json:"balance_after"`
	PreviousHash  string    `json:"previous_hash"`
	CurrentHash   string    `json:"current_hash"`
	CreatedAt     time.Time `json:"created_at"`
}

func (r *MySQLLedgerRepository) ListRecentLedgerEntries(ctx context.Context, limit int) ([]AuditLedgerRow, error) {
	if limit <= 0 || limit > 100 {
		limit = 25
	}
	const query = `
		SELECT l.id, l.transaction_id, a.account_number, l.entry_type,
		       l.amount, l.balance_after, l.previous_hash, l.current_hash, l.created_at
		FROM ledger_entries l
		JOIN accounts a ON l.account_id = a.id
		ORDER BY l.id DESC
		LIMIT ?
	`
	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []AuditLedgerRow
	for rows.Next() {
		var row AuditLedgerRow
		if err := rows.Scan(
			&row.ID,
			&row.TransactionID,
			&row.AccountNumber,
			&row.EntryType,
			&row.Amount,
			&row.BalanceAfter,
			&row.PreviousHash,
			&row.CurrentHash,
			&row.CreatedAt,
		); err != nil {
			return nil, err
		}
		entries = append(entries, row)
	}
	return entries, rows.Err()
}

// ensureBranchTransactionColumns idempotently adds Maker-Checker branch account columns if missing.
func ensureBranchTransactionColumns(ctx context.Context, db *sql.DB) error {
	columns := []struct {
		name  string
		alter string
	}{
		{
			name:  "target_account_id",
			alter: "ALTER TABLE transactions ADD COLUMN target_account_id CHAR(36) NULL AFTER description",
		},
		{
			name:  "vault_account_id",
			alter: "ALTER TABLE transactions ADD COLUMN vault_account_id CHAR(36) NULL AFTER target_account_id",
		},
	}

	for _, col := range columns {
		var exists int
		err := db.QueryRowContext(
			ctx,
			`SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS
			 WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'transactions' AND COLUMN_NAME = ?`,
			col.name,
		).Scan(&exists)
		if err != nil {
			return fmt.Errorf("failed to inspect column %s on transactions: %w", col.name, err)
		}
		if exists == 0 {
			if _, err := db.ExecContext(ctx, col.alter); err != nil {
				return fmt.Errorf("failed to add column %s on transactions: %w", col.name, err)
			}
		}
	}
	return nil
}

// SeedDemoAccountsIfEmpty provisions branch GL vault cash and sample customer accounts if empty.
func SeedDemoAccountsIfEmpty(ctx context.Context, db *sql.DB) error {
	if err := ensureBranchTransactionColumns(ctx, db); err != nil {
		return err
	}

	var count int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM accounts").Scan(&count); err != nil {
		return fmt.Errorf("failed to check accounts count: %w", err)
	}
	if count > 0 {
		return nil
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	const customerQuery = `
		INSERT IGNORE INTO customers (id, cif_number, full_name, identity_type, identity_number_hash, kyc_status, risk_rating, home_branch_id)
		VALUES
			('cif-001', 'CIF-100001', 'PT Multi Graha Mandiri', 'CORPORATE_NIB', 'hash_nib_001', 'VERIFIED', 'LOW', 'branch-jkt-01'),
			('cif-002', 'CIF-100002', 'Siti Rahayu', 'KTP', 'hash_ktp_002', 'VERIFIED', 'LOW', 'branch-jkt-01')
	`
	if _, err := tx.ExecContext(ctx, customerQuery); err != nil {
		return fmt.Errorf("failed to seed demo customers: %w", err)
	}

	const accountQuery = `
		INSERT IGNORE INTO accounts (id, account_number, customer_id, branch_id, account_type, product_code, is_internal_gl, currency, balance, hold_balance, status)
		VALUES
			('acc-vault-jkt01', 'GL-VAULT-JKT01', NULL, 'branch-jkt-01', 'ASSET', 'GL_VAULT_CASH', TRUE, 'IDR', 1250000000, 0, 'ACTIVE'),
			('acc-cust-001', '1001000012', 'cif-001', 'branch-jkt-01', 'LIABILITY', 'CHECKING', FALSE, 'IDR', 100000000, 0, 'ACTIVE'),
			('acc-cust-002', '1001000045', 'cif-002', 'branch-jkt-01', 'LIABILITY', 'SAVINGS', FALSE, 'IDR', 50000000, 0, 'ACTIVE')
	`
	if _, err := tx.ExecContext(ctx, accountQuery); err != nil {
		return fmt.Errorf("failed to seed demo accounts: %w", err)
	}

	return tx.Commit()
}
