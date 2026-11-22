package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/elif-bank/apps/core-ledger/internal/domain"
	"github.com/elif-bank/pkg/hashchain"
)

func setupBranchApprovalTest(t *testing.T, txType domain.TransactionType, amount int64) (*BranchApprovalUsecase, *mockAccountRepo, *mockLedgerRepo, string) {
	t.Helper()
	cashUc, accRepo, ledgerRepo := setupBranchCashTest()
	txMgr := &mockTxManager{}
	approvalUc := NewBranchApprovalUsecase(accRepo, ledgerRepo, txMgr)

	// Seed a pending > 50M transaction created by usr_teller_01
	res, err := cashUc.Execute(context.Background(), BranchCashCommand{
		ReferenceID:     "tx-pending-001",
		AccountNumber:   "1001000012",
		VaultAccountID:  "acc-vault-01",
		TransactionType: txType,
		Amount:          amount,
		Description:     "High-value branch operation",
		MakerID:         "usr_teller_01",
	})
	if err != nil {
		t.Fatalf("failed to create pending branch transaction: %v", err)
	}
	return approvalUc, accRepo, ledgerRepo, res.TransactionID
}

func TestBranchApprovalUsecase_Approve_Withdrawal_Success(t *testing.T) {
	// 1. ARRANGE
	uc, accRepo, ledgerRepo, txID := setupBranchApprovalTest(t, domain.TxTypeCashWithdrawal, 75_000_000)

	// 2. ACT
	res, err := uc.Execute(context.Background(), BranchApprovalCommand{
		TransactionID: txID,
		CheckerID:     "usr_spv_01",
		Action:        ActionApprove,
		Notes:         "Verified physical ID & passbook",
	})

	// 3. ASSERT
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if res.Status != string(domain.TxStatusPosted) {
		t.Errorf("expected POSTED status, got %s", res.Status)
	}
	cust := accRepo.accounts["acc-cust-01"]
	if cust.Balance != 25_000_000 || cust.HoldBalance != 0 {
		t.Errorf("expected balance=25M hold=0, got balance=%d hold=%d", cust.Balance, cust.HoldBalance)
	}
	if accRepo.accounts["acc-vault-01"].Balance != 925_000_000 {
		t.Errorf("expected vault balance 925M, got %d", accRepo.accounts["acc-vault-01"].Balance)
	}
	if len(ledgerRepo.entries) != 2 {
		t.Fatalf("expected 2 balanced ledger entries, got %d", len(ledgerRepo.entries))
	}
	if !hashchain.VerifyHash(ledgerRepo.entries[0].PreviousHash, ledgerRepo.entries[0].AccountID, ledgerRepo.entries[0].Amount, ledgerRepo.entries[0].BalanceAfter, ledgerRepo.entries[0].CreatedAt, ledgerRepo.entries[0].CurrentHash) {
		t.Error("SHA-256 hash chain verification failed")
	}
}

func TestBranchApprovalUsecase_Approve_Deposit_Success(t *testing.T) {
	// 1. ARRANGE
	uc, accRepo, ledgerRepo, txID := setupBranchApprovalTest(t, domain.TxTypeCashDeposit, 80_000_000)

	// 2. ACT
	res, err := uc.Execute(context.Background(), BranchApprovalCommand{
		TransactionID: txID,
		CheckerID:     "usr_spv_01",
		Action:        ActionApprove,
		Notes:         "Cash counted & verified",
	})

	// 3. ASSERT
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if res.Status != string(domain.TxStatusPosted) {
		t.Errorf("expected POSTED, got %s", res.Status)
	}
	if accRepo.accounts["acc-cust-01"].Balance != 180_000_000 {
		t.Errorf("expected customer balance 180M, got %d", accRepo.accounts["acc-cust-01"].Balance)
	}
	if accRepo.accounts["acc-vault-01"].Balance != 1_080_000_000 {
		t.Errorf("expected vault balance 1.08B, got %d", accRepo.accounts["acc-vault-01"].Balance)
	}
	if len(ledgerRepo.entries) != 2 {
		t.Fatalf("expected 2 ledger entries, got %d", len(ledgerRepo.entries))
	}
}

func TestBranchApprovalUsecase_Reject_Withdrawal_ReleasesHold(t *testing.T) {
	// 1. ARRANGE
	uc, accRepo, ledgerRepo, txID := setupBranchApprovalTest(t, domain.TxTypeCashWithdrawal, 75_000_000)

	// 2. ACT
	res, err := uc.Execute(context.Background(), BranchApprovalCommand{
		TransactionID: txID,
		CheckerID:     "usr_spv_01",
		Action:        ActionReject,
		Notes:         "Signature mismatch",
	})

	// 3. ASSERT
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if res.Status != string(domain.TxStatusRejected) {
		t.Errorf("expected REJECTED, got %s", res.Status)
	}
	cust := accRepo.accounts["acc-cust-01"]
	if cust.Balance != 100_000_000 || cust.HoldBalance != 0 || cust.AvailableBalance() != 100_000_000 {
		t.Errorf("expected hold released and full 100M available, got bal=%d hold=%d", cust.Balance, cust.HoldBalance)
	}
	if len(ledgerRepo.entries) != 0 {
		t.Errorf("expected 0 ledger entries on rejected transaction, got %d", len(ledgerRepo.entries))
	}
}

func TestBranchApprovalUsecase_SelfApproval_Forbidden(t *testing.T) {
	// 1. ARRANGE
	uc, accRepo, _, txID := setupBranchApprovalTest(t, domain.TxTypeCashWithdrawal, 75_000_000)

	// 2. ACT - Same Teller (usr_teller_01) attempts to approve their own transaction
	_, err := uc.Execute(context.Background(), BranchApprovalCommand{
		TransactionID: txID,
		CheckerID:     "usr_teller_01",
		Action:        ActionApprove,
	})

	// 3. ASSERT
	if !errors.Is(err, domain.ErrSelfApprovalForbidden) {
		t.Fatalf("expected ErrSelfApprovalForbidden, got: %v", err)
	}
	if accRepo.accounts["acc-cust-01"].HoldBalance != 75_000_000 {
		t.Error("hold_balance must remain untouched when self-approval is rejected")
	}
}

func TestBranchApprovalUsecase_AlreadyProcessed_Fails(t *testing.T) {
	// 1. ARRANGE
	uc, _, _, txID := setupBranchApprovalTest(t, domain.TxTypeCashWithdrawal, 75_000_000)
	_, err := uc.Execute(context.Background(), BranchApprovalCommand{
		TransactionID: txID,
		CheckerID:     "usr_spv_01",
		Action:        ActionApprove,
	})
	if err != nil {
		t.Fatalf("first approval failed: %v", err)
	}

	// 2. ACT - Attempt second approval on same transaction
	_, err = uc.Execute(context.Background(), BranchApprovalCommand{
		TransactionID: txID,
		CheckerID:     "usr_spv_02",
		Action:        ActionApprove,
	})

	// 3. ASSERT
	if !errors.Is(err, domain.ErrTxNotPendingApproval) {
		t.Fatalf("expected ErrTxNotPendingApproval on double-approval attempt, got: %v", err)
	}
}
