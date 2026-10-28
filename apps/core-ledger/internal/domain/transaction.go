package domain

import (
	"errors"
	"time"
)

// ==============================================================================
// Domain Entity: Transaction
// Dokumen Terkait: docs/TSD.md (Seksi 1.1) & docs/PRD.md (Seksi 3.3)
// Fungsi: Header transaksi finansial yang mencatat Maker, Checker, dan Status
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

// Transaction mencatat riwayat induk transaksi finansial.
type Transaction struct {
	ID              string            `json:"id"`
	ReferenceID     string            `json:"reference_id"` // Kunci idempotensi
	TransactionType TransactionType   `json:"transaction_type"`
	Amount          int64             `json:"amount"` // Nominal transaksi (Rupiah integer)
	Description     string            `json:"description"`
	MakerID         string            `json:"maker_id"`            // Identitas Teller atau Nasabah pembuat
	CheckerID       *string           `json:"checker_id,omitempty"` // Identitas Supervisor (jika butuh approval)
	Status          TransactionStatus `json:"status"`
	CreatedAt       time.Time         `json:"created_at"`
}

// Approve memproses persetujuan transaksi oleh Supervisor (Checker).
func (t *Transaction) Approve(checkerID string) error {
	if t.Status != TxStatusPendingApproval {
		return errors.New("transaction is not pending approval")
	}
	// Aturan Anti-Self-Approval (docs/SECURITY_GUIDE.md Seksi 1.1)
	if t.MakerID == checkerID {
		return errors.New("maker and checker cannot be the same user")
	}
	t.CheckerID = &checkerID
	t.Status = TxStatusPosted
	return nil
}

// Reject menolak transaksi oleh Supervisor (Checker).
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
