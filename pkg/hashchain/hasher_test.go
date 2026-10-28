package hashchain

import (
	"testing"
	"time"
)

func TestHashChain_DeterministicAndVerify(t *testing.T) {
	now := time.Date(2026, 10, 28, 14, 0, 0, 0, time.UTC)
	hash1 := ComputeHash(GenesisHash, "acc-001", 100000, 100000, now)

	if len(hash1) != 64 {
		t.Fatalf("expected 64 characters hex SHA-256 string, got len=%d", len(hash1))
	}

	// Verifikasi hash cocok
	if !VerifyHash(GenesisHash, "acc-001", 100000, 100000, now, hash1) {
		t.Error("VerifyHash failed on valid payload")
	}

	// Verifikasi jika data dimanipulasi (tampered)
	if VerifyHash(GenesisHash, "acc-001", 99999, 100000, now, hash1) {
		t.Error("VerifyHash should have failed on tampered amount")
	}
}
