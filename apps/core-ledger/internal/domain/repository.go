package domain

import (
	"context"
)

// ==============================================================================
// Domain Ports: Repository & Transaction Manager Interfaces
// Related Documents: docs/TSD.md (Section 3) & docs/ARCHITECTURE.md (Section 4)
// Philosophy: Dependency Inversion Principle (DIP) - Use Cases depend on Interfaces,
//             never on concrete database implementations (e.g., MySQL/TiDB driver).
// ==============================================================================

// AccountRepository defines account read and update operations.
type AccountRepository interface {
	// GetByID retrieves account data by UUID without row locking.
	GetByID(ctx context.Context, id string) (*Account, error)

	// GetByAccountNumber retrieves account data by its account number.
	GetByAccountNumber(ctx context.Context, accNum string) (*Account, error)

	// GetByIDWithLock retrieves account data with pessimistic row locking (SELECT ... FOR UPDATE).
	// Must be invoked inside a database transaction to prevent race conditions.
	GetByIDWithLock(ctx context.Context, id string) (*Account, error)

	// UpdateBalance updates account balance and hold_balance in the database.
	UpdateBalance(ctx context.Context, acc *Account) error
}

// LedgerRepository defines financial transaction and ledger mutation operations.
type LedgerRepository interface {
	// CreateTransaction saves the financial transaction header (transactions table).
	CreateTransaction(ctx context.Context, tx *Transaction) error

	// CreateLedgerEntries persists paired debit and credit journal entries (ledger_entries table).
	CreateLedgerEntries(ctx context.Context, entries []LedgerEntry) error

	// GetLatestEntryHash retrieves the latest SHA-256 hash of an account for hash chaining.
	GetLatestEntryHash(ctx context.Context, accountID string) (string, error)

	// CreateOutboxEvent stores an event in the outbox_events table within the same database transaction.
	CreateOutboxEvent(ctx context.Context, eventType, aggregateID string, payload []byte) error
}

// TransactionManager abstracts ACID database transactions (BEGIN, COMMIT, ROLLBACK).
// If `fn` returns an error, the transaction automatically rolls back.
// If `fn` returns nil, the transaction commits.
type TransactionManager interface {
	ExecuteInTransaction(ctx context.Context, fn func(txCtx context.Context) error) error
}
