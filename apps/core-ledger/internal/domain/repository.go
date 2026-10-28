package domain

import (
	"context"
)

// ==============================================================================
// Domain Ports: Repository & Transaction Manager Interfaces
// Dokumen Terkait: docs/TSD.md (Seksi 3) & docs/ARCHITECTURE.md (Seksi 4)
// Filosofi: Dependency Inversion Principle (DIP) - Use Case bergantung pada Interface,
//           bukan pada implementasi database spesifik (pgx/PostgreSQL).
// ==============================================================================

// AccountRepository mendefinisikan operasi pembacaan dan pembaruan rekening.
type AccountRepository interface {
	// GetByID mengambil data rekening berdasarkan UUID tanpa lock.
	GetByID(ctx context.Context, id string) (*Account, error)

	// GetByAccountNumber mengambil data rekening berdasarkan nomor rekening.
	GetByAccountNumber(ctx context.Context, accNum string) (*Account, error)

	// GetByIDWithLock mengambil data rekening dengan baris terkunci (SELECT ... FOR UPDATE).
	// Wajib dipanggil dalam transaksi untuk mencegah race condition.
	GetByIDWithLock(ctx context.Context, id string) (*Account, error)

	// UpdateBalance memperbarui saldo balance dan hold_balance rekening di database.
	UpdateBalance(ctx context.Context, acc *Account) error
}

// LedgerRepository mendefinisikan operasi penulisan transaksi dan mutasi buku besar.
type LedgerRepository interface {
	// CreateTransaction menyimpan header transaksi finansial (tabel transactions).
	CreateTransaction(ctx context.Context, tx *Transaction) error

	// CreateLedgerEntries menyimpan pasangan mutasi jurnal debit dan kredit (tabel ledger_entries).
	CreateLedgerEntries(ctx context.Context, entries []LedgerEntry) error

	// GetLatestEntryHash mengambil hash SHA-256 mutasi terakhir dari suatu rekening untuk chaining.
	GetLatestEntryHash(ctx context.Context, accountID string) (string, error)

	// CreateOutboxEvent menyimpan event ke tabel outbox_events dalam transaksi database yang sama.
	CreateOutboxEvent(ctx context.Context, eventType, aggregateID string, payload []byte) error
}

// TransactionManager mengabstraksikan transaksi database ACID (BEGIN, COMMIT, ROLLBACK).
// Fungsi `fn` dieksekusi di dalam transaksi. Jika `fn` mengembalikan error, otomatis ROLLBACK.
// Jika `fn` mengembalikan nil, otomatis COMMIT.
type TransactionManager interface {
	ExecuteInTransaction(ctx context.Context, fn func(txCtx context.Context) error) error
}
