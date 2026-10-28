package domain

import "errors"

// ==============================================================================
// Domain Sentinel Errors
// Dokumen Terkait: docs/TSD.md & docs/INTEGRATION.md (Seksi 1.3)
// Fungsi: Error bisnis spesifik tanpa ketergantungan framework eksternal
// ==============================================================================

var (
	// ErrInsufficientFunds dikembalikan jika saldo efektif kurang dari nominal debit.
	ErrInsufficientFunds = errors.New("insufficient available balance")

	// ErrAccountNotActive dikembalikan jika rekening sumber/tujuan berstatus FROZEN, DORMANT, atau CLOSED.
	ErrAccountNotActive = errors.New("account is not active")

	// ErrInvalidAmount dikembalikan jika nominal transaksi <= 0.
	ErrInvalidAmount = errors.New("transaction amount must be greater than zero")

	// ErrSameAccountTransfer dikembalikan jika rekening asal dan tujuan identik.
	ErrSameAccountTransfer = errors.New("source and destination account cannot be the same")

	// ErrAccountNotFound dikembalikan jika rekening tidak terdaftar di buku besar.
	ErrAccountNotFound = errors.New("account not found")

	// ErrInvalidEntryType dikembalikan jika tipe jurnal bukan 'DR' atau 'CR'.
	ErrInvalidEntryType = errors.New("invalid ledger entry type, must be DR or CR")
)
