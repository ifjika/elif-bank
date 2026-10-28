package hashchain

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

// ==============================================================================
// Package hashchain: Cryptographic Tamper-Evident Ledger Audit Trail
// Dokumen Terkait: docs/TSD.md (Seksi 2.2) & docs/PRD.md (Seksi 4)
// Formula: SHA256(PreviousHash + AccountID + Amount + BalanceAfter + RFC3339Timestamp)
// ==============================================================================

// GenesisHash adalah nilai hash awal bagi akun baru yang belum memiliki riwayat mutasi.
const GenesisHash = "0000000000000000000000000000000000000000000000000000000000000000"

// ComputeHash menghitung hash SHA-256 berantai untuk satu baris ledger entry.
func ComputeHash(previousHash, accountID string, amount, balanceAfter int64, createdAt time.Time) string {
	if previousHash == "" {
		previousHash = GenesisHash
	}

	// Gunakan format timestamp RFC3339Nano deterministik agar konsisten
	payload := fmt.Sprintf("%s|%s|%d|%d|%s",
		previousHash,
		accountID,
		amount,
		balanceAfter,
		createdAt.UTC().Format(time.RFC3339Nano),
	)

	hasher := sha256.New()
	hasher.Write([]byte(payload))
	return hex.EncodeToString(hasher.Sum(nil))
}

// VerifyHash memvalidasi apakah currentHash pada baris buku besar cocok dengan payload aslinya.
func VerifyHash(previousHash, accountID string, amount, balanceAfter int64, createdAt time.Time, currentHash string) bool {
	expected := ComputeHash(previousHash, accountID, amount, balanceAfter, createdAt)
	return expected == currentHash
}
