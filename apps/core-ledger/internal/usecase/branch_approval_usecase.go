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
// Use Case: BranchApprovalUsecase (Phase 5.2)
// Related Documents: docs/IMPLEMENTATION_PLAN.md (5.2) & docs/SECURITY_GUIDE.md
// Feature: Four-Eyes Principle Supervisor Approval/Rejection for Cash > 50M IDR
// ==============================================================================

type ApprovalAction string

const (
	ActionApprove ApprovalAction = "APPROVE"
	ActionReject  ApprovalAction = "REJECT"
)

var (
	ErrInvalidApprovalAction = errors.New("action must be APPROVE or REJECT")
	ErrMissingTargetAccounts = errors.New("pending transaction is missing target or vault account metadata")
)

type BranchApprovalCommand struct {
	TransactionID string
	CheckerID     string
	Action        ApprovalAction
	Notes         string
}

type BranchApprovalResult struct {
	TransactionID    string    `json:"transaction_id"`
	Status           string    `json:"status"`
	CheckerID        string    `json:"checker_id"`
	BalanceAfter     int64     `json:"balance_after"`
	HoldBalanceAfter int64     `json:"hold_balance_after"`
	ApprovedAt       time.Time `json:"approved_at"`
}

type BranchApprovalUsecase struct {
	accountRepo domain.AccountRepository
	ledgerRepo  domain.LedgerRepository
	txManager   domain.TransactionManager
}

func NewBranchApprovalUsecase(
	accRepo domain.AccountRepository,
	ledgerRepo domain.LedgerRepository,
	txMgr domain.TransactionManager,
) *BranchApprovalUsecase {
	return &BranchApprovalUsecase{
		accountRepo: accRepo,
		ledgerRepo:  ledgerRepo,
		txManager:   txMgr,
	}
}

func (u *BranchApprovalUsecase) Execute(ctx context.Context, cmd BranchApprovalCommand) (*BranchApprovalResult, error) {
	if cmd.TransactionID == "" || cmd.CheckerID == "" {
		return nil, errors.New("transaction_id and checker_id are required")
	}
	if cmd.Action != ActionApprove && cmd.Action != ActionReject {
		return nil, ErrInvalidApprovalAction
	}

	var result *BranchApprovalResult

	err := u.txManager.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		txHeader, err := u.ledgerRepo.GetTransactionByIDWithLock(txCtx, cmd.TransactionID)
		if err != nil {
			return fmt.Errorf("failed to lock transaction %s: %w", cmd.TransactionID, err)
		}
		if txHeader == nil {
			return domain.ErrTxNotFound
		}
		if txHeader.Status != domain.TxStatusPendingApproval {
			return domain.ErrTxNotPendingApproval
		}
		if txHeader.TargetAccountID == nil || txHeader.VaultAccountID == nil {
			return ErrMissingTargetAccounts
		}

		// Deterministic lexicographical row-locking to prevent cyclic deadlocks
		firstID, secondID := sortIDs(*txHeader.TargetAccountID, *txHeader.VaultAccountID)
		accFirst, err := u.accountRepo.GetByIDWithLock(txCtx, firstID)
		if err != nil {
			return fmt.Errorf("failed to lock account %s: %w", firstID, err)
		}
		accSecond, err := u.accountRepo.GetByIDWithLock(txCtx, secondID)
		if err != nil {
			return fmt.Errorf("failed to lock account %s: %w", secondID, err)
		}

		var customerAcc, vaultAcc *domain.Account
		if accFirst.ID == *txHeader.TargetAccountID {
			customerAcc, vaultAcc = accFirst, accSecond
		} else {
			customerAcc, vaultAcc = accSecond, accFirst
		}
		if customerAcc == nil || vaultAcc == nil {
			return domain.ErrAccountNotFound
		}

		now := time.Now().UTC()

		// Branch A: REJECT
		if cmd.Action == ActionReject {
			if err := txHeader.Reject(cmd.CheckerID, cmd.Notes, now); err != nil {
				return err
			}

			if txHeader.TransactionType == domain.TxTypeCashWithdrawal {
				if err := customerAcc.ReleaseHold(txHeader.Amount); err != nil {
					return err
				}
				if err := u.accountRepo.UpdateBalance(txCtx, customerAcc); err != nil {
					return err
				}
			}

			if err := u.ledgerRepo.UpdateTransactionStatus(txCtx, txHeader); err != nil {
				return err
			}

			outboxPayload := fmt.Sprintf(`{"transaction_id":"%s","status":"REJECTED","checker_id":"%s"}`,
				txHeader.ID, cmd.CheckerID)
			if err := u.ledgerRepo.CreateOutboxEvent(txCtx, "BRANCH_CASH_REJECTED", txHeader.ID, []byte(outboxPayload)); err != nil {
				return err
			}

			result = &BranchApprovalResult{
				TransactionID:    txHeader.ID,
				Status:           string(txHeader.Status),
				CheckerID:        cmd.CheckerID,
				BalanceAfter:     customerAcc.Balance,
				HoldBalanceAfter: customerAcc.HoldBalance,
				ApprovedAt:       now,
			}
			return nil
		}

		// Branch B: APPROVE (Enforce Four-Eyes Principle + Double-Entry Mutation)
		if err := txHeader.Approve(cmd.CheckerID, cmd.Notes, now); err != nil {
			return err
		}

		var customerEntryType, vaultEntryType domain.EntryType
		if txHeader.TransactionType == domain.TxTypeCashWithdrawal {
			if err := customerAcc.ReleaseHold(txHeader.Amount); err != nil {
				return err
			}
			if err := customerAcc.Debit(txHeader.Amount); err != nil {
				return err
			}
			if err := vaultAcc.Debit(txHeader.Amount); err != nil {
				return err
			}
			customerEntryType = domain.EntryTypeDebit // Liability decreases
			vaultEntryType = domain.EntryTypeCredit   // Asset decreases
		} else {
			if err := customerAcc.Credit(txHeader.Amount); err != nil {
				return err
			}
			if err := vaultAcc.Credit(txHeader.Amount); err != nil {
				return err
			}
			customerEntryType = domain.EntryTypeCredit // Liability increases
			vaultEntryType = domain.EntryTypeDebit     // Asset increases
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
				Amount:        txHeader.Amount,
				BalanceAfter:  customerAcc.Balance,
				PreviousHash:  prevHashCust,
				CurrentHash:   hashchain.ComputeHash(prevHashCust, customerAcc.ID, txHeader.Amount, customerAcc.Balance, now),
				CreatedAt:     now,
			},
			{
				TransactionID: txHeader.ID,
				AccountID:     vaultAcc.ID,
				EntryType:     vaultEntryType,
				Amount:        txHeader.Amount,
				BalanceAfter:  vaultAcc.Balance,
				PreviousHash:  prevHashVault,
				CurrentHash:   hashchain.ComputeHash(prevHashVault, vaultAcc.ID, txHeader.Amount, vaultAcc.Balance, now),
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
		if err := u.ledgerRepo.UpdateTransactionStatus(txCtx, txHeader); err != nil {
			return err
		}

		outboxPayload := fmt.Sprintf(`{"transaction_id":"%s","status":"POSTED","checker_id":"%s","amount":%d}`,
			txHeader.ID, cmd.CheckerID, txHeader.Amount)
		if err := u.ledgerRepo.CreateOutboxEvent(txCtx, "BRANCH_CASH_APPROVED", txHeader.ID, []byte(outboxPayload)); err != nil {
			return err
		}

		result = &BranchApprovalResult{
			TransactionID:    txHeader.ID,
			Status:           string(txHeader.Status),
			CheckerID:        cmd.CheckerID,
			BalanceAfter:     customerAcc.Balance,
			HoldBalanceAfter: customerAcc.HoldBalance,
			ApprovedAt:       now,
		}
		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}
