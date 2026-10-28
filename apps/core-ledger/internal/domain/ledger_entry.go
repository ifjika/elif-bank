package domain

import (
	"time"
)

// ==============================================================================
// Domain Entity: LedgerEntry
// Dokumen Terkait: docs/TSD.md (Seksi 1.1 & 2.2) & docs/PRD.md (Seksi 3.2)
// Fungsi: Baris jurnal mutasi pembukuan berpasangan (Double-Entry General Ledger)
// ==============================================================================

type EntryType string

const (
	EntryTypeDebit  EntryType = "DR" // Debit
	EntryTypeCredit EntryType = "CR" // Kredit
)

// LedgerEntry merepresentasikan satu baris mutasi spesifik pada suatu akun.
type LedgerEntry struct {
	ID            int64     `json:"id"`
	TransactionID string    `json:"transaction_id"`
	AccountID     string    `json:"account_id"`
	EntryType     EntryType `json:"entry_type"` // 'DR' atau 'CR'
	Amount        int64     `json:"amount"`     // Wajib > 0
	BalanceAfter  int64     `json:"balance_after"`
	PreviousHash  string    `json:"previous_hash"` // SHA-256 dari baris akun sebelumnya
	CurrentHash   string    `json:"current_hash"`  // SHA-256(prev + account + amount + balanceAfter)
	CreatedAt     time.Time `json:"created_at"`
}

// Validate memeriksa kelayakan entri jurnal sebelum disimpan ke database.
func (e *LedgerEntry) Validate() error {
	if e.EntryType != EntryTypeDebit && e.EntryType != EntryTypeCredit {
		return ErrInvalidEntryType
	}
	if e.Amount <= 0 {
		return ErrInvalidAmount
	}
	return nil
}

// IsDebit mengembalikan true jika entri bertipe Debit.
func (e *LedgerEntry) IsDebit() bool {
	return e.EntryType == EntryTypeDebit
}

// IsCredit mengembalikan true jika entri bertipe Kredit.
func (e *LedgerEntry) IsCredit() bool {
	return e.EntryType == EntryTypeCredit
}
