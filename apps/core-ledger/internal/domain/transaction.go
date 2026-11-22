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

var (
	ErrTxNotFound            = errors.New("transaction not found")
	ErrTxNotPendingApproval  = errors.New("transaction is not pending approval")
	ErrSelfApprovalForbidden = errors.New("maker and checker cannot be the same user")
)

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
	TargetAccountID *string           `json:"target_account_id,omitempty"` // Customer account for pending branch ops
	VaultAccountID  *string           `json:"vault_account_id,omitempty"`  // Branch GL vault account
	MakerID         string            `json:"maker_id"`                    // Initiating Teller or Customer ID
	CheckerID       *string           `json:"checker_id,omitempty"`        // Authorizing Supervisor ID
	CheckerNotes    *string           `json:"checker_notes,omitempty"`     // Supervisor verification notes
	Status          TransactionStatus `json:"status"`
	ApprovedAt      *time.Time        `json:"approved_at,omitempty"`
	CreatedAt       time.Time         `json:"created_at"`
}

// Approve processes transaction approval by a Supervisor (Checker).
func (t *Transaction) Approve(checkerID, notes string, now time.Time) error {
	if t.Status != TxStatusPendingApproval {
		return ErrTxNotPendingApproval
	}
	// Anti-Self-Approval Enforcement (docs/SECURITY_GUIDE.md Section 1.1)
	if t.MakerID == checkerID {
		return ErrSelfApprovalForbidden
	}
	t.CheckerID = &checkerID
	if notes != "" {
		t.CheckerNotes = &notes
	}
	t.Status = TxStatusPosted
	t.ApprovedAt = &now
	return nil
}

// Reject rejects transaction by a Supervisor (Checker).
func (t *Transaction) Reject(checkerID, notes string, now time.Time) error {
	if t.Status != TxStatusPendingApproval {
		return ErrTxNotPendingApproval
	}
	if t.MakerID == checkerID {
		return ErrSelfApprovalForbidden
	}
	t.CheckerID = &checkerID
	if notes != "" {
		t.CheckerNotes = &notes
	}
	t.Status = TxStatusRejected
	t.ApprovedAt = &now
	return nil
}
