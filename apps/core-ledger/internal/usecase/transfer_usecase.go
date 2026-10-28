package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/elif-bank/apps/core-ledger/internal/domain"
	"github.com/elif-bank/pkg/hashchain"
)

// ==============================================================================
// Use Case: TransferUsecase
// Related Documents: docs/TSD.md (Section 2.1 & 2.2) & docs/PRD.md (Section 3.2)
// Feature: Double-Entry Atomic Transfer with Lexicographical Row-Locking
// ==============================================================================

// TransferCommand represents Data Transfer Object (DTO) for transfer execution.
type TransferCommand struct {
	ReferenceID          string `json:"reference_id"` // Unique client idempotency key
	SourceAccountID      string `json:"source_account_id"`
	DestinationAccountID string `json:"destination_account_id"`
	Amount               int64  `json:"amount"` // IDR amount (must be > 0)
	Description          string `json:"description"`
	MakerID              string `json:"maker_id"`
}

// TransferResult contains execution result of committed transfer.
type TransferResult struct {
	TransactionID string    `json:"transaction_id"`
	ReferenceID   string    `json:"reference_id"`
	Amount        int64     `json:"amount"`
	SourceBalance int64     `json:"source_balance_after"`
	DestBalance   int64     `json:"dest_balance_after"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
}

type TransferUsecase struct {
	accountRepo domain.AccountRepository
	ledgerRepo  domain.LedgerRepository
	txManager   domain.TransactionManager
}

// NewTransferUsecase is the constructor (Dependency Injection).
func NewTransferUsecase(
	accRepo domain.AccountRepository,
	ledgerRepo domain.LedgerRepository,
	txMgr domain.TransactionManager,
) *TransferUsecase {
	return &TransferUsecase{
		accountRepo: accRepo,
		ledgerRepo:  ledgerRepo,
		txManager:   txMgr,
	}
}

// Execute performs atomic ACID double-entry funds transfer.
func (u *TransferUsecase) Execute(ctx context.Context, cmd TransferCommand) (*TransferResult, error) {
	// 1. Initial DTO validation
	if cmd.Amount <= 0 {
		return nil, domain.ErrInvalidAmount
	}
	if cmd.SourceAccountID == cmd.DestinationAccountID {
		return nil, domain.ErrSameAccountTransfer
	}

	var result *TransferResult

	// 2. Execute within an ACID database transaction (docs/TSD.md)
	err := u.txManager.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		// 3. Lexicographical Locking: Sort account IDs alphabetically
		// Guarantees zero deadlocks when two accounts transfer to each other simultaneously
		firstID, secondID := sortIDs(cmd.SourceAccountID, cmd.DestinationAccountID)

		// 4. Lock first account row, then second account row (SELECT ... FOR UPDATE)
		accFirst, err := u.accountRepo.GetByIDWithLock(txCtx, firstID)
		if err != nil {
			return fmt.Errorf("failed to lock account %s: %w", firstID, err)
		}
		if accFirst == nil {
			return domain.ErrAccountNotFound
		}

		accSecond, err := u.accountRepo.GetByIDWithLock(txCtx, secondID)
		if err != nil {
			return fmt.Errorf("failed to lock account %s: %w", secondID, err)
		}
		if accSecond == nil {
			return domain.ErrAccountNotFound
		}

		// Identify source and destination accounts
		var sourceAcc, destAcc *domain.Account
		if accFirst.ID == cmd.SourceAccountID {
			sourceAcc, destAcc = accFirst, accSecond
		} else {
			sourceAcc, destAcc = accSecond, accFirst
		}

		// 5. In-Memory Domain Balance Mutations
		if err := sourceAcc.Debit(cmd.Amount); err != nil {
			return err
		}
		if err := destAcc.Credit(cmd.Amount); err != nil {
			return err
		}

		// 6. Create Transaction Header
		now := time.Now().UTC()
		txHeader := &domain.Transaction{
			ID:              cmd.ReferenceID,
			ReferenceID:     cmd.ReferenceID,
			TransactionType: domain.TxTypeTransfer,
			Amount:          cmd.Amount,
			Description:     cmd.Description,
			MakerID:         cmd.MakerID,
			Status:          domain.TxStatusPosted,
			CreatedAt:       now,
		}

		if err := u.ledgerRepo.CreateTransaction(txCtx, txHeader); err != nil {
			return fmt.Errorf("failed to persist transaction header: %w", err)
		}

		// 7. Retrieve Prior Hashes for SHA-256 Audit Chaining
		prevHashSource, err := u.ledgerRepo.GetLatestEntryHash(txCtx, sourceAcc.ID)
		if err != nil {
			return fmt.Errorf("failed to get previous hash for source: %w", err)
		}

		prevHashDest, err := u.ledgerRepo.GetLatestEntryHash(txCtx, destAcc.ID)
		if err != nil {
			return fmt.Errorf("failed to get previous hash for destination: %w", err)
		}

		// 8. Compute New Hashes (docs/TSD.md Section 2.2)
		currentHashSource := hashchain.ComputeHash(prevHashSource, sourceAcc.ID, cmd.Amount, sourceAcc.Balance, now)
		currentHashDest := hashchain.ComputeHash(prevHashDest, destAcc.ID, cmd.Amount, destAcc.Balance, now)

		// 9. Build Balanced Double-Entry Journal Rows (Debit == Credit)
		entries := []domain.LedgerEntry{
			{
				TransactionID: txHeader.ID,
				AccountID:     sourceAcc.ID,
				EntryType:     domain.EntryTypeDebit, // Source account decreases (Debit Liability)
				Amount:        cmd.Amount,
				BalanceAfter:  sourceAcc.Balance,
				PreviousHash:  prevHashSource,
				CurrentHash:   currentHashSource,
				CreatedAt:     now,
			},
			{
				TransactionID: txHeader.ID,
				AccountID:     destAcc.ID,
				EntryType:     domain.EntryTypeCredit, // Destination account increases (Credit Liability)
				Amount:        cmd.Amount,
				BalanceAfter:  destAcc.Balance,
				PreviousHash:  prevHashDest,
				CurrentHash:   currentHashDest,
				CreatedAt:     now,
			},
		}

		if err := u.ledgerRepo.CreateLedgerEntries(txCtx, entries); err != nil {
			return fmt.Errorf("failed to persist ledger entries: %w", err)
		}

		// 10. Persist Account Balances to Database
		if err := u.accountRepo.UpdateBalance(txCtx, sourceAcc); err != nil {
			return fmt.Errorf("failed to update source balance: %w", err)
		}
		if err := u.accountRepo.UpdateBalance(txCtx, destAcc); err != nil {
			return fmt.Errorf("failed to update destination balance: %w", err)
		}

		// 11. Write Transactional Outbox Event for Kafka Publishing
		outboxPayload := fmt.Sprintf(`{"transaction_id":"%s","amount":%d,"source":"%s","destination":"%s"}`,
			txHeader.ID, cmd.Amount, sourceAcc.AccountNumber, destAcc.AccountNumber)
		if err := u.ledgerRepo.CreateOutboxEvent(txCtx, "TRANSACTION_POSTED", txHeader.ID, []byte(outboxPayload)); err != nil {
			return fmt.Errorf("failed to record outbox event: %w", err)
		}

		result = &TransferResult{
			TransactionID: txHeader.ID,
			ReferenceID:   txHeader.ReferenceID,
			Amount:        txHeader.Amount,
			SourceBalance: sourceAcc.Balance,
			DestBalance:   destAcc.Balance,
			Status:        string(txHeader.Status),
			CreatedAt:     now,
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}

// sortIDs lexicographically sorts two account IDs to prevent cyclic deadlocks.
func sortIDs(idA, idB string) (string, string) {
	if idA < idB {
		return idA, idB
	}
	return idB, idA
}
