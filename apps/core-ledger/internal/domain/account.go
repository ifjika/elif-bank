package domain

import (
	"time"
)

// ==============================================================================
// Domain Entity: Account
// Related Documents: docs/TSD.md (Section 1.1) & docs/PRD.md (Section 3.1)
// Purpose: Models customer accounts and internal Chart of Accounts (COA)
// ==============================================================================

// AccountType represents the standard 5-pillar accounting classification in banking.
type AccountType string

const (
	AccountTypeAsset     AccountType = "ASSET"     // Assets (e.g., Branch Vault Cash, Central Bank Reserves)
	AccountTypeLiability AccountType = "LIABILITY" // Liabilities (e.g., Customer Savings & Checking Accounts)
	AccountTypeEquity    AccountType = "EQUITY"    // Bank Paid-in Capital
	AccountTypeRevenue   AccountType = "REVENUE"   // Operational Revenue (Loan Interest, Fee Income)
	AccountTypeExpense   AccountType = "EXPENSE"   // Operational Expenses (Deposit Interest, IT Operations)
)

// AccountStatus models the Account Finite State Machine (docs/APP_FLOW.md).
type AccountStatus string

const (
	StatusActive  AccountStatus = "ACTIVE"
	StatusFrozen  AccountStatus = "FROZEN"
	StatusDormant AccountStatus = "DORMANT"
	StatusClosed  AccountStatus = "CLOSED"
)

// Account represents an account entity in Go runtime memory.
type Account struct {
	ID             string        `json:"id"`
	AccountNumber  string        `json:"account_number"`
	CustomerID     *string       `json:"customer_id,omitempty"` // Pointer *string; can be nil for internal bank accounts
	AccountType    AccountType   `json:"account_type"`
	Currency       string        `json:"currency"`
	Balance        int64         `json:"balance"`      // Total ledger balance (smallest unit: IDR whole integer)
	HoldBalance    int64         `json:"hold_balance"` // Reserved balance (escrow/collateral)
	Status         AccountStatus `json:"status"`
	Version        int64         `json:"version"` // Optimistic concurrency version
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`
}

// AvailableBalance computes effective spendable funds for outgoing transactions.
func (a *Account) AvailableBalance() int64 {
	avail := a.Balance - a.HoldBalance
	if avail < 0 {
		return 0
	}
	return avail
}

// CanDebit verifies whether the account is eligible to debit outgoing funds.
func (a *Account) CanDebit(amount int64) error {
	if a.Status != StatusActive {
		return ErrAccountNotActive
	}
	if amount <= 0 {
		return ErrInvalidAmount
	}
	if a.AvailableBalance() < amount {
		return ErrInsufficientFunds
	}
	return nil
}

// CanCredit verifies whether the account is eligible to receive incoming funds.
func (a *Account) CanCredit(amount int64) error {
	if a.Status != StatusActive {
		return ErrAccountNotActive
	}
	if amount <= 0 {
		return ErrInvalidAmount
	}
	return nil
}

// Debit deducts balance from the account after passing domain validation.
func (a *Account) Debit(amount int64) error {
	if err := a.CanDebit(amount); err != nil {
		return err
	}
	a.Balance -= amount
	a.UpdatedAt = time.Now().UTC()
	return nil
}

// Credit adds balance to the account after passing domain validation.
func (a *Account) Credit(amount int64) error {
	if err := a.CanCredit(amount); err != nil {
		return err
	}
	a.Balance += amount
	a.UpdatedAt = time.Now().UTC()
	return nil
}

// Hold locks a portion of effective balance for pre-authorizations or collateral.
func (a *Account) Hold(amount int64) error {
	if err := a.CanDebit(amount); err != nil {
		return err
	}
	a.HoldBalance += amount
	a.UpdatedAt = time.Now().UTC()
	return nil
}

// ReleaseHold releases reserved funds back into available balance.
func (a *Account) ReleaseHold(amount int64) error {
	if amount <= 0 || a.HoldBalance < amount {
		return ErrInvalidAmount
	}
	a.HoldBalance -= amount
	a.UpdatedAt = time.Now().UTC()
	return nil
}
