package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/elif-bank/apps/core-ledger/internal/domain"
	"github.com/elif-bank/pkg/hashchain"
)

func setupBranchCashTest() (*BranchCashUsecase, *mockAccountRepo, *mockLedgerRepo) {
	accRepo := newMockAccountRepo()
	ledgerRepo := newMockLedgerRepo()
	txMgr := &mockTxManager{}
	uc := NewBranchCashUsecase(accRepo, ledgerRepo, txMgr)

	accRepo.accounts["acc-cust-01"] = &domain.Account{
		ID:            "acc-cust-01",
		AccountNumber: "1001000012",
		AccountType:   domain.AccountTypeLiability,
		Balance:       100_000_000, // IDR 100,000,000
		HoldBalance:   0,
		Status:        domain.StatusActive,
	}
	accRepo.accounts["acc-vault-01"] = &domain.Account{
		ID:            "acc-vault-01",
		AccountNumber: "GL-VAULT-JKT01",
		AccountType:   domain.AccountTypeAsset,
		Balance:       1_000_000_000, // IDR 1,000,000,000
		HoldBalance:   0,
		Status:        domain.StatusActive,
	}

	return uc, accRepo, ledgerRepo
}

func TestBranchCashUsecase_STP_Withdrawal_UnderThreshold(t *testing.T) {
	// 1. ARRANGE
	uc, accRepo, ledgerRepo := setupBranchCashTest()
	cmd := BranchCashCommand{
		ReferenceID:     "ref-cash-001",
		AccountNumber:   "1001000012",
		VaultAccountID:  "acc-vault-01",
		TransactionType: domain.TxTypeCashWithdrawal,
		Amount:          25_000_000, // <= 50M threshold
		Description:     "Branch counter withdrawal",
		MakerID:         "usr_teller_01",
	}

	// 2. ACT
	res, err := uc.Execute(context.Background(), cmd)

	// 3. ASSERT
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if res.Status != string(domain.TxStatusPosted) {
		t.Errorf("expected POSTED status, got %s", res.Status)
	}
	if accRepo.accounts["acc-cust-01"].Balance != 75_000_000 {
		t.Errorf("expected customer balance 75,000,000, got %d", accRepo.accounts["acc-cust-01"].Balance)
	}
	if accRepo.accounts["acc-vault-01"].Balance != 975_000_000 {
		t.Errorf("expected vault balance 975,000,000, got %d", accRepo.accounts["acc-vault-01"].Balance)
	}
	if len(ledgerRepo.entries) != 2 {
		t.Fatalf("expected 2 balanced ledger entries, got %d", len(ledgerRepo.entries))
	}
	if !hashchain.VerifyHash(ledgerRepo.entries[0].PreviousHash, ledgerRepo.entries[0].AccountID, ledgerRepo.entries[0].Amount, ledgerRepo.entries[0].BalanceAfter, ledgerRepo.entries[0].CreatedAt, ledgerRepo.entries[0].CurrentHash) {
		t.Error("SHA-256 hash chain verification failed")
	}
}

func TestBranchCashUsecase_STP_Deposit_UnderThreshold(t *testing.T) {
	// 1. ARRANGE
	uc, accRepo, ledgerRepo := setupBranchCashTest()
	cmd := BranchCashCommand{
		ReferenceID:     "ref-cash-002",
		AccountNumber:   "1001000012",
		VaultAccountID:  "acc-vault-01",
		TransactionType: domain.TxTypeCashDeposit,
		Amount:          50_000_000, // Exactly 50M -> still STP
		Description:     "Branch counter deposit",
		MakerID:         "usr_teller_01",
	}

	// 2. ACT
	res, err := uc.Execute(context.Background(), cmd)

	// 3. ASSERT
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if res.Status != string(domain.TxStatusPosted) {
		t.Errorf("expected POSTED status at exact 50M boundary, got %s", res.Status)
	}
	if accRepo.accounts["acc-cust-01"].Balance != 150_000_000 {
		t.Errorf("expected customer balance 150,000,000, got %d", accRepo.accounts["acc-cust-01"].Balance)
	}
	if accRepo.accounts["acc-vault-01"].Balance != 1_050_000_000 {
		t.Errorf("expected vault balance 1,050,000,000, got %d", accRepo.accounts["acc-vault-01"].Balance)
	}
	if len(ledgerRepo.entries) != 2 {
		t.Fatalf("expected 2 ledger entries, got %d", len(ledgerRepo.entries))
	}
}

func TestBranchCashUsecase_HighValue_Withdrawal_LocksHoldBalance(t *testing.T) {
	// 1. ARRANGE
	uc, accRepo, ledgerRepo := setupBranchCashTest()
	cmd := BranchCashCommand{
		ReferenceID:     "ref-cash-003",
		AccountNumber:   "1001000012",
		VaultAccountID:  "acc-vault-01",
		TransactionType: domain.TxTypeCashWithdrawal,
		Amount:          75_000_000, // > 50M threshold
		Description:     "Corporate withdrawal",
		MakerID:         "usr_teller_01",
	}

	// 2. ACT
	res, err := uc.Execute(context.Background(), cmd)

	// 3. ASSERT
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if res.Status != string(domain.TxStatusPendingApproval) {
		t.Errorf("expected PENDING_APPROVAL status, got %s", res.Status)
	}
	cust := accRepo.accounts["acc-cust-01"]
	if cust.Balance != 100_000_000 {
		t.Errorf("book balance must remain 100,000,000 while pending, got %d", cust.Balance)
	}
	if cust.HoldBalance != 75_000_000 || cust.AvailableBalance() != 25_000_000 {
		t.Errorf("expected hold=75M and available=25M, got hold=%d avail=%d", cust.HoldBalance, cust.AvailableBalance())
	}
	if len(ledgerRepo.entries) != 0 {
		t.Errorf("no ledger entries should be posted before supervisor approval, got %d", len(ledgerRepo.entries))
	}
}

func TestBranchCashUsecase_HighValue_Deposit_DoesNotCreditUntilApproved(t *testing.T) {
	// 1. ARRANGE
	uc, accRepo, ledgerRepo := setupBranchCashTest()
	cmd := BranchCashCommand{
		ReferenceID:     "ref-cash-004",
		AccountNumber:   "1001000012",
		VaultAccountID:  "acc-vault-01",
		TransactionType: domain.TxTypeCashDeposit,
		Amount:          80_000_000, // > 50M threshold
		Description:     "Large corporate deposit",
		MakerID:         "usr_teller_01",
	}

	// 2. ACT
	res, err := uc.Execute(context.Background(), cmd)

	// 3. ASSERT
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if res.Status != string(domain.TxStatusPendingApproval) {
		t.Errorf("expected PENDING_APPROVAL status, got %s", res.Status)
	}
	cust := accRepo.accounts["acc-cust-01"]
	if cust.Balance != 100_000_000 || cust.HoldBalance != 0 {
		t.Errorf("deposit pending approval must NOT touch balance or hold_balance, got bal=%d hold=%d", cust.Balance, cust.HoldBalance)
	}
	if len(ledgerRepo.entries) != 0 {
		t.Errorf("no ledger entries should exist while pending approval, got %d", len(ledgerRepo.entries))
	}
}

func TestBranchCashUsecase_InsufficientFunds(t *testing.T) {
	// 1. ARRANGE
	uc, _, _ := setupBranchCashTest()
	cmd := BranchCashCommand{
		ReferenceID:     "ref-cash-005",
		AccountNumber:   "1001000012",
		VaultAccountID:  "acc-vault-01",
		TransactionType: domain.TxTypeCashWithdrawal,
		Amount:          150_000_000, // Exceeds 100M balance
		MakerID:         "usr_teller_01",
	}

	// 2. ACT
	_, err := uc.Execute(context.Background(), cmd)

	// 3. ASSERT
	if !errors.Is(err, domain.ErrInsufficientFunds) {
		t.Fatalf("expected ErrInsufficientFunds, got: %v", err)
	}
}
