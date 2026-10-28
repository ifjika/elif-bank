package domain

import (
	"time"
)

// ==============================================================================
// Domain Entity: LedgerEntry
// Related Documents: docs/TSD.md (Section 1.1 & 2.2) & docs/PRD.md (Section 3.2)
// Purpose: Double-Entry General Ledger mutation journal row
// ==============================================================================

type EntryType string

const (
	EntryTypeDebit  EntryType = "DR" // Debit
	EntryTypeCredit EntryType = "CR" // Credit
)

// LedgerEntry represents a single mutation row on a specific account.
type LedgerEntry struct {
	ID            int64     `json:"id"`
	TransactionID string    `json:"transaction_id"`
	AccountID     string    `json:"account_id"`
	EntryType     EntryType `json:"entry_type"` // 'DR' or 'CR'
	Amount        int64     `json:"amount"`     // Must be > 0
	BalanceAfter  int64     `json:"balance_after"`
	PreviousHash  string    `json:"previous_hash"` // SHA-256 chain from prior account row
	CurrentHash   string    `json:"current_hash"`  // SHA-256(prev + account + amount + balanceAfter)
	CreatedAt     time.Time `json:"created_at"`
}

// Validate checks journal entry integrity prior to database persistence.
func (e *LedgerEntry) Validate() error {
	if e.EntryType != EntryTypeDebit && e.EntryType != EntryTypeCredit {
		return ErrInvalidEntryType
	}
	if e.Amount <= 0 {
		return ErrInvalidAmount
	}
	return nil
}

// IsDebit returns true if entry is a Debit.
func (e *LedgerEntry) IsDebit() bool {
	return e.EntryType == EntryTypeDebit
}

// IsCredit returns true if entry is a Credit.
func (e *LedgerEntry) IsCredit() bool {
	return e.EntryType == EntryTypeCredit
}
