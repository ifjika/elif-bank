package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/elif-bank/apps/core-ledger/internal/domain"
	"github.com/elif-bank/pkg/hashchain"
)

// ==============================================================================
// Use Case: BranchCashUsecase (Phase 5.1)
// Related Documents: docs/IMPLEMENTATION_PLAN.md (5.1) & docs/PRD.md (Section 3.3)
// Feature: Over-the-Counter Cash Deposit/Withdrawal with Maker-Checker Threshold
// ==============================================================================

const BranchCashApprovalThreshold int64 = 50_000_000

var ErrInvalidBranchTxType = errors.New("transaction type must be CASH_DEPOSIT or CASH_WITHDRAWAL")

type BranchCashCommand struct {
	ReferenceID     string
	AccountNumber   string
	VaultAccountID  string
	TransactionType domain.TransactionType
	Amount          int64
	Description     string
	MakerID         string
}

type BranchCashResult struct {
	TransactionID    string    `json:"transaction_id"`
	ReferenceID      string    `json:"reference_id"`
	TransactionType  string    `json:"transaction_type"`
	Amount           int64     `json:"amount"`
	BalanceAfter     int64     `json:"balance_after"`
	HoldBalanceAfter int64     `json:"hold_balance_after"`
	Status           string    `json:"status"`
	CreatedAt        time.Time `json:"created_at"`
}

type BranchCashUsecase struct {
	accountRepo domain.AccountRepository
	ledgerRepo  domain.LedgerRepository
	txManager   domain.TransactionManager
}

func NewBranchCashUsecase(
	accRepo domain.AccountRepository,
	ledgerRepo domain.LedgerRepository,
	txMgr domain.TransactionManager,
) *BranchCashUsecase {
	return &BranchCashUsecase{
		accountRepo: accRepo,
		ledgerRepo:  ledgerRepo,
		txManager:   txMgr,
	}
}

func (u *BranchCashUsecase) Execute(ctx context.Context, cmd BranchCashCommand) (*BranchCashResult, error) {
	if cmd.Amount <= 0 {
		return nil, domain.ErrInvalidAmount
	}
	if cmd.TransactionType != domain.TxTypeCashDeposit && cmd.TransactionType != domain.TxTypeCashWithdrawal {
		return nil, ErrInvalidBranchTxType
	}
	if cmd.ReferenceID == "" || cmd.MakerID == "" {
		return nil, errors.New("reference_id and maker_id are required")
	}

	var result *BranchCashResult

	err := u.txManager.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		lookupAcc, err := u.accountRepo.GetByAccountNumber(txCtx, cmd.AccountNumber)
		if err != nil {
			return fmt.Errorf("failed to lookup customer account: %w", err)
		}
		if lookupAcc == nil {
			return domain.ErrAccountNotFound
		}

		// Deterministic lexicographical row-locking to prevent cyclic deadlocks
		firstID, secondID := sortIDs(lookupAcc.ID, cmd.VaultAccountID)
		accFirst, err := u.accountRepo.GetByIDWithLock(txCtx, firstID)
		if err != nil {
			return fmt.Errorf("failed to lock account %s: %w", firstID, err)
		}
		accSecond, err := u.accountRepo.GetByIDWithLock(txCtx, secondID)
		if err != nil {
			return fmt.Errorf("failed to lock account %s: %w", secondID, err)
		}

		var customerAcc, vaultAcc *domain.Account
		if accFirst.ID == lookupAcc.ID {
			customerAcc, vaultAcc = accFirst, accSecond
		} else {
			customerAcc, vaultAcc = accSecond, accFirst
		}
		if customerAcc == nil || vaultAcc == nil {
			return domain.ErrAccountNotFound
		}

		now := time.Now().UTC()

		// Branch A: > 50M IDR requires Supervisor Dual-Authorization (PENDING_APPROVAL)
		if cmd.Amount > BranchCashApprovalThreshold {
			if cmd.TransactionType == domain.TxTypeCashWithdrawal {
				if err := customerAcc.Hold(cmd.Amount); err != nil {
					return err
				}
				if err := u.accountRepo.UpdateBalance(txCtx, customerAcc); err != nil {
					return fmt.Errorf("failed to lock hold balance: %w", err)
				}
			} else {
				if err := customerAcc.CanCredit(cmd.Amount); err != nil {
					return err
				}
			}

			txHeader := &domain.Transaction{
				ID:              cmd.ReferenceID,
				ReferenceID:     cmd.ReferenceID,
				TransactionType: cmd.TransactionType,
				Amount:          cmd.Amount,
				Description:     cmd.Description,
				TargetAccountID: &customerAcc.ID,
				VaultAccountID:  &vaultAcc.ID,
				MakerID:         cmd.MakerID,
				Status:          domain.TxStatusPendingApproval,
				CreatedAt:       now,
			}
			if err := u.ledgerRepo.CreateTransaction(txCtx, txHeader); err != nil {
				return fmt.Errorf("failed to persist pending transaction: %w", err)
			}

			result = &BranchCashResult{
				TransactionID:    txHeader.ID,
				ReferenceID:      txHeader.ReferenceID,
				TransactionType:  string(txHeader.TransactionType),
				Amount:           txHeader.Amount,
				BalanceAfter:     customerAcc.Balance,
				HoldBalanceAfter: customerAcc.HoldBalance,
				Status:           string(txHeader.Status),
				CreatedAt:        now,
			}
			return nil
		}

		// Branch B: <= 50M IDR executes Straight-Through Processing (POSTED)
		var customerEntryType, vaultEntryType domain.EntryType
		if cmd.TransactionType == domain.TxTypeCashWithdrawal {
			if err := customerAcc.Debit(cmd.Amount); err != nil {
				return err
			}
			if err := vaultAcc.Debit(cmd.Amount); err != nil {
				return err
			}
			customerEntryType = domain.EntryTypeDebit // Liability decreases
			vaultEntryType = domain.EntryTypeCredit   // Asset decreases
		} else {
			if err := customerAcc.Credit(cmd.Amount); err != nil {
				return err
			}
			if err := vaultAcc.Credit(cmd.Amount); err != nil {
				return err
			}
			customerEntryType = domain.EntryTypeCredit // Liability increases
			vaultEntryType = domain.EntryTypeDebit     // Asset increases
		}

		txHeader := &domain.Transaction{
			ID:              cmd.ReferenceID,
			ReferenceID:     cmd.ReferenceID,
			TransactionType: cmd.TransactionType,
			Amount:          cmd.Amount,
			Description:     cmd.Description,
			MakerID:         cmd.MakerID,
			Status:          domain.TxStatusPosted,
			CreatedAt:       now,
		}
		if err := u.ledgerRepo.CreateTransaction(txCtx, txHeader); err != nil {
			return fmt.Errorf("failed to persist transaction header: %w", err)
		}

		prevHashCust, err := u.ledgerRepo.GetLatestEntryHash(txCtx, customerAcc.ID)
		if err != nil {
			return err
		}
		prevHashVault, err := u.ledgerRepo.GetLatestEntryHash(txCtx, vaultAcc.ID)
		if err != nil {
			return err
		}

		entries := []domain.LedgerEntry{
			{
				TransactionID: txHeader.ID,
				AccountID:     customerAcc.ID,
				EntryType:     customerEntryType,
				Amount:        cmd.Amount,
				BalanceAfter:  customerAcc.Balance,
				PreviousHash:  prevHashCust,
				CurrentHash:   hashchain.ComputeHash(prevHashCust, customerAcc.ID, cmd.Amount, customerAcc.Balance, now),
				CreatedAt:     now,
			},
			{
				TransactionID: txHeader.ID,
				AccountID:     vaultAcc.ID,
				EntryType:     vaultEntryType,
				Amount:        cmd.Amount,
				BalanceAfter:  vaultAcc.Balance,
				PreviousHash:  prevHashVault,
				CurrentHash:   hashchain.ComputeHash(prevHashVault, vaultAcc.ID, cmd.Amount, vaultAcc.Balance, now),
				CreatedAt:     now,
			},
		}
		if err := u.ledgerRepo.CreateLedgerEntries(txCtx, entries); err != nil {
			return fmt.Errorf("failed to persist ledger entries: %w", err)
		}

		if err := u.accountRepo.UpdateBalance(txCtx, customerAcc); err != nil {
			return err
		}
		if err := u.accountRepo.UpdateBalance(txCtx, vaultAcc); err != nil {
			return err
		}

		outboxPayload := fmt.Sprintf(`{"transaction_id":"%s","type":"%s","amount":%d,"account":"%s"}`,
			txHeader.ID, cmd.TransactionType, cmd.Amount, customerAcc.AccountNumber)
		if err := u.ledgerRepo.CreateOutboxEvent(txCtx, "BRANCH_CASH_POSTED", txHeader.ID, []byte(outboxPayload)); err != nil {
			return err
		}

		result = &BranchCashResult{
			TransactionID:    txHeader.ID,
			ReferenceID:      txHeader.ReferenceID,
			TransactionType:  string(txHeader.TransactionType),
			Amount:           txHeader.Amount,
			BalanceAfter:     customerAcc.Balance,
			HoldBalanceAfter: customerAcc.HoldBalance,
			Status:           string(txHeader.Status),
			CreatedAt:        now,
		}
		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}
