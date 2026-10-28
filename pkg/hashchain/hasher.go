package hashchain

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

// ==============================================================================
// Package hashchain: Cryptographic Tamper-Evident Ledger Audit Trail
// Related Documents: docs/TSD.md (Section 2.2) & docs/PRD.md (Section 4)
// Formula: SHA256(PreviousHash + AccountID + Amount + BalanceAfter + RFC3339Timestamp)
// ==============================================================================

// GenesisHash is the initial seed hash for newly opened accounts with no prior history.
const GenesisHash = "0000000000000000000000000000000000000000000000000000000000000000"

// ComputeHash calculates the chained SHA-256 hash for a single ledger entry row.
func ComputeHash(previousHash, accountID string, amount, balanceAfter int64, createdAt time.Time) string {
	if previousHash == "" {
		previousHash = GenesisHash
	}

	// Use deterministic RFC3339Nano timestamp formatting for consistency
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

// VerifyHash validates whether the currentHash on a ledger row matches its payload.
func VerifyHash(previousHash, accountID string, amount, balanceAfter int64, createdAt time.Time, currentHash string) bool {
	expected := ComputeHash(previousHash, accountID, amount, balanceAfter, createdAt)
	return expected == currentHash
}
