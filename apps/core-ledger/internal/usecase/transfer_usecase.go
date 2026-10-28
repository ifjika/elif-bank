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
// Dokumen Terkait: docs/TSD.md (Seksi 2.1 & 2.2) & docs/PRD.md (Seksi 3.2)
// Fitur: Double-Entry Atomic Transfer dengan Lexicographical Row-Locking
// ==============================================================================

// TransferCommand merepresentasikan Data Transfer Object (DTO) untuk eksekusi transfer.
type TransferCommand struct {
	ReferenceID            string `json:"reference_id"` // Kunci Idempotensi unik
	SourceAccountID        string `json:"source_account_id"`
	DestinationAccountID   string `json:"destination_account_id"`
	Amount                 int64  `json:"amount"` // Nominal Rupiah (wajib > 0)
	Description            string `json:"description"`
	MakerID                string `json:"maker_id"`
}

// TransferResult memuat hasil eksekusi transfer yang berhasil dibukukan.
type TransferResult struct {
	TransactionID   string    `json:"transaction_id"`
	ReferenceID     string    `json:"reference_id"`
	Amount          int64     `json:"amount"`
	SourceBalance   int64     `json:"source_balance_after"`
	DestBalance     int64     `json:"dest_balance_after"`
	Status          string    `json:"status"`
	CreatedAt       time.Time `json:"created_at"`
}

type TransferUsecase struct {
	accountRepo domain.AccountRepository
	ledgerRepo  domain.LedgerRepository
	txManager   domain.TransactionManager
}

// NewTransferUsecase adalah konstruktor Go (Dependency Injection).
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

// Execute menjalankan transfer dana atomik berstandar ACID.
func (u *TransferUsecase) Execute(ctx context.Context, cmd TransferCommand) (*TransferResult, error) {
	// 1. Validasi Awal DTO
	if cmd.Amount <= 0 {
		return nil, domain.ErrInvalidAmount
	}
	if cmd.SourceAccountID == cmd.DestinationAccountID {
		return nil, domain.ErrSameAccountTransfer
	}

	var result *TransferResult

	// 2. Jalankan seluruh proses di dalam satu transaksi database ACID (docs/TSD.md)
	err := u.txManager.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		// 3. Algoritma Lexicographical Locking: Tentukan urutan penguncian berdasarkan abjad ID
		// Hal ini menjamin TIDAK AKAN PERNAH terjadi deadlock saat dua rekening saling mentransfer
		firstID, secondID := sortIDs(cmd.SourceAccountID, cmd.DestinationAccountID)

		// 4. Kunci baris akun pertama, lalu akun kedua (SELECT ... FOR UPDATE)
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

		// Identifikasi mana akun sumber dan mana akun tujuan
		var sourceAcc, destAcc *domain.Account
		if accFirst.ID == cmd.SourceAccountID {
			sourceAcc, destAcc = accFirst, accSecond
		} else {
			sourceAcc, destAcc = accSecond, accFirst
		}

		// 5. Eksekusi Mutasi Saldo di Memori (Domain Methods)
		if err := sourceAcc.Debit(cmd.Amount); err != nil {
			return err
		}
		if err := destAcc.Credit(cmd.Amount); err != nil {
			return err
		}

		// 6. Buat Header Transaksi Finansial
		now := time.Now().UTC()
		txHeader := &domain.Transaction{
			ID:              cmd.ReferenceID, // Menggunakan reference ID sebagai identitas utama atau bisa UUID
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

		// 7. Ambil Hash Mutasi Sebelumnya untuk Membentuk SHA-256 Audit Chain
		prevHashSource, err := u.ledgerRepo.GetLatestEntryHash(txCtx, sourceAcc.ID)
		if err != nil {
			return fmt.Errorf("failed to get previous hash for source: %w", err)
		}

		prevHashDest, err := u.ledgerRepo.GetLatestEntryHash(txCtx, destAcc.ID)
		if err != nil {
			return fmt.Errorf("failed to get previous hash for destination: %w", err)
		}

		// 8. Hitung Hash Rantai Mutasi Baru (docs/TSD.md Seksi 2.2)
		currentHashSource := hashchain.ComputeHash(prevHashSource, sourceAcc.ID, cmd.Amount, sourceAcc.Balance, now)
		currentHashDest := hashchain.ComputeHash(prevHashDest, destAcc.ID, cmd.Amount, destAcc.Balance, now)

		// 9. Bentuk Sepasang Entri Jurnal Berpasangan (Double-Entry Invariant: Debit == Kredit)
		entries := []domain.LedgerEntry{
			{
				TransactionID: txHeader.ID,
				AccountID:     sourceAcc.ID,
				EntryType:     domain.EntryTypeDebit, // Akun sumber berkurang (Debit Liability)
				Amount:        cmd.Amount,
				BalanceAfter:  sourceAcc.Balance,
				PreviousHash:  prevHashSource,
				CurrentHash:   currentHashSource,
				CreatedAt:     now,
			},
			{
				TransactionID: txHeader.ID,
				AccountID:     destAcc.ID,
				EntryType:     domain.EntryTypeCredit, // Akun tujuan bertambah (Kredit Liability)
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

		// 10. Update Saldo Akun di Database
		if err := u.accountRepo.UpdateBalance(txCtx, sourceAcc); err != nil {
			return fmt.Errorf("failed to update source balance: %w", err)
		}
		if err := u.accountRepo.UpdateBalance(txCtx, destAcc); err != nil {
			return fmt.Errorf("failed to update destination balance: %w", err)
		}

		// 11. Simpan Event ke Outbox untuk Notifikasi Kafka
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

// sortIDs mengurutkan dua ID akun secara alfabetis untuk mencegah cyclic deadlock.
func sortIDs(idA, idB string) (string, string) {
	if idA < idB {
		return idA, idB
	}
	return idB, idA
}
