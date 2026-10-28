package domain

import "errors"

// ==============================================================================
// Domain Sentinel Errors
// Related Documents: docs/TSD.md & docs/INTEGRATION.md (Section 1.3)
// Purpose: Pure business errors decoupled from external frameworks
// ==============================================================================

var (
	// ErrInsufficientFunds is returned when available balance is less than required debit amount.
	ErrInsufficientFunds = errors.New("insufficient available balance")

	// ErrAccountNotActive is returned when source/destination account is FROZEN, DORMANT, or CLOSED.
	ErrAccountNotActive = errors.New("account is not active")

	// ErrInvalidAmount is returned when transaction amount is <= 0.
	ErrInvalidAmount = errors.New("transaction amount must be greater than zero")

	// ErrSameAccountTransfer is returned when source and destination accounts are identical.
	ErrSameAccountTransfer = errors.New("source and destination account cannot be the same")

	// ErrAccountNotFound is returned when an account is not found in the ledger.
	ErrAccountNotFound = errors.New("account not found")

	// ErrInvalidEntryType is returned when entry type is neither 'DR' nor 'CR'.
	ErrInvalidEntryType = errors.New("invalid ledger entry type, must be DR or CR")
)
