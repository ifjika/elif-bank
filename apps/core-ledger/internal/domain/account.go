package domain

import (
	"time"
)

// ==============================================================================
// Domain Entity: Account
// Dokumen Terkait: docs/TSD.md (Seksi 1.1) & docs/PRD.md (Seksi 3.1)
// Fungsi: Mewakili rekening nasabah dan akun internal Chart of Accounts (COA)
// ==============================================================================

// AccountType mewakili klasifikasi akuntansi standar 5 pilar perbankan.
type AccountType string

const (
	AccountTypeAsset     AccountType = "ASSET"     // Aktiva (misal: Kas Fisik Cabang, Giro BI)
	AccountTypeLiability AccountType = "LIABILITY" // Pasiva/Kewajiban (misal: Tabungan & Giro Nasabah)
	AccountTypeEquity    AccountType = "EQUITY"    // Modal Disetor Bank
	AccountTypeRevenue   AccountType = "REVENUE"   // Pendapatan Operasional (Bunga Pinjaman, Fee)
	AccountTypeExpense   AccountType = "EXPENSE"   // Beban Operasional (Bunga Simpanan, IT Ops)
)

// AccountStatus memodelkan Finite State Machine rekening (docs/APP_FLOW.md).
type AccountStatus string

const (
	StatusActive  AccountStatus = "ACTIVE"
	StatusFrozen  AccountStatus = "FROZEN"
	StatusDormant AccountStatus = "DORMANT"
	StatusClosed  AccountStatus = "CLOSED"
)

// Account merepresentasikan entitas rekening dalam memori Go runtime.
type Account struct {
	ID             string        `json:"id"`
	AccountNumber  string        `json:"account_number"`
	CustomerID     *string       `json:"customer_id,omitempty"` // Pointer *string karena bisa nil untuk akun internal bank
	AccountType    AccountType   `json:"account_type"`
	Currency       string        `json:"currency"`
	Balance        int64         `json:"balance"`      // Saldo buku total (satuan terkecil: Rupiah)
	HoldBalance    int64         `json:"hold_balance"` // Saldo tertahan (escrow/jaminan)
	Status         AccountStatus `json:"status"`
	Version        int64         `json:"version"` // Optimistic concurrency version
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`
}

// AvailableBalance menghitung saldo efektif yang bebas digunakan untuk transaksi keluar.
func (a *Account) AvailableBalance() int64 {
	avail := a.Balance - a.HoldBalance
	if avail < 0 {
		return 0
	}
	return avail
}

// CanDebit memverifikasi apakah rekening siap mendebit sejumlah dana keluar.
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

// CanCredit memverifikasi apakah rekening siap menerima mutasi dana masuk.
func (a *Account) CanCredit(amount int64) error {
	if a.Status != StatusActive {
		return ErrAccountNotActive
	}
	if amount <= 0 {
		return ErrInvalidAmount
	}
	return nil
}

// Debit memotong saldo rekening setelah melewati validasi.
func (a *Account) Debit(amount int64) error {
	if err := a.CanDebit(amount); err != nil {
		return err
	}
	a.Balance -= amount
	a.UpdatedAt = time.Now().UTC()
	return nil
}

// Credit menambah saldo rekening setelah melewati validasi.
func (a *Account) Credit(amount int64) error {
	if err := a.CanCredit(amount); err != nil {
		return err
	}
	a.Balance += amount
	a.UpdatedAt = time.Now().UTC()
	return nil
}

// Hold mengunci sebagian saldo efektif untuk pre-otorisasi atau jaminan.
func (a *Account) Hold(amount int64) error {
	if err := a.CanDebit(amount); err != nil {
		return err
	}
	a.HoldBalance += amount
	a.UpdatedAt = time.Now().UTC()
	return nil
}

// ReleaseHold melepaskan kunci saldo tertahan kembali menjadi saldo efektif.
func (a *Account) ReleaseHold(amount int64) error {
	if amount <= 0 || a.HoldBalance < amount {
		return ErrInvalidAmount
	}
	a.HoldBalance -= amount
	a.UpdatedAt = time.Now().UTC()
	return nil
}
