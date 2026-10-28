package domain

import (
	"errors"
	"time"
)

// ==============================================================================
// Domain Entity: Transaction
// Related Documents: docs/TSD.md (Section 1.1) & docs/PRD.md (Section 3.3)
// Purpose: Financial transaction header tracking Maker, Checker, and Status
// ==============================================================================

type TransactionType string

const (
	TxTypeTransfer       TransactionType = "TRANSFER"
	TxTypeCashDeposit    TransactionType = "CASH_DEPOSIT"
	TxTypeCashWithdrawal TransactionType = "CASH_WITHDRAWAL"
	TxTypeFee            TransactionType = "FEE"
	TxTypeInterest       TransactionType = "INTEREST"
)

type TransactionStatus string

const (
	TxStatusPosted          TransactionStatus = "POSTED"
	TxStatusPendingApproval TransactionStatus = "PENDING_APPROVAL"
	TxStatusRejected        TransactionStatus = "REJECTED"
)

// Transaction records financial transaction headers.
type Transaction struct {
	ID              string            `json:"id"`
	ReferenceID     string            `json:"reference_id"` // Client idempotency key
	TransactionType TransactionType   `json:"transaction_type"`
	Amount          int64             `json:"amount"` // Transaction amount (integer IDR)
	Description     string            `json:"description"`
	MakerID         string            `json:"maker_id"`             // Initiating Teller or Customer ID
	CheckerID       *string           `json:"checker_id,omitempty"` // Authorizing Supervisor ID (if required)
	Status          TransactionStatus `json:"status"`
	CreatedAt       time.Time         `json:"created_at"`
}

// Approve processes transaction approval by a Supervisor (Checker).
func (t *Transaction) Approve(checkerID string) error {
	if t.Status != TxStatusPendingApproval {
		return errors.New("transaction is not pending approval")
	}
	// Anti-Self-Approval Enforcement (docs/SECURITY_GUIDE.md Section 1.1)
	if t.MakerID == checkerID {
		return errors.New("maker and checker cannot be the same user")
	}
	t.CheckerID = &checkerID
	t.Status = TxStatusPosted
	return nil
}

// Reject rejects transaction by a Supervisor (Checker).
func (t *Transaction) Reject(checkerID string) error {
	if t.Status != TxStatusPendingApproval {
		return errors.New("transaction is not pending approval")
	}
	if t.MakerID == checkerID {
		return errors.New("maker and checker cannot be the same user")
	}
	t.CheckerID = &checkerID
	t.Status = TxStatusRejected
	return nil
}
