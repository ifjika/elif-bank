package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/elif-bank/apps/core-ledger/internal/domain"
	"github.com/elif-bank/pkg/hashchain"
)

// ==============================================================================
// Unit Test: TransferUsecase (AAA Pattern)
// Dokumen Terkait: docs/TESTING_GUIDE.md (Seksi 2) & docs/PRD.md (Seksi 3.2)
// ==============================================================================

// --- Mock Implementations untuk Interface Domain (In-Memory) ---

type mockTxManager struct{}

func (m *mockTxManager) ExecuteInTransaction(ctx context.Context, fn func(txCtx context.Context) error) error {
	return fn(ctx) // Jalankan langsung fungsi logika di memori
}

type mockAccountRepo struct {
	accounts map[string]*domain.Account
}

func newMockAccountRepo() *mockAccountRepo {
	return &mockAccountRepo{accounts: make(map[string]*domain.Account)}
}

func (m *mockAccountRepo) GetByID(ctx context.Context, id string) (*domain.Account, error) {
	acc, ok := m.accounts[id]
	if !ok {
		return nil, domain.ErrAccountNotFound
	}
	return acc, nil
}

func (m *mockAccountRepo) GetByAccountNumber(ctx context.Context, accNum string) (*domain.Account, error) {
	for _, acc := range m.accounts {
		if acc.AccountNumber == accNum {
			return acc, nil
		}
	}
	return nil, domain.ErrAccountNotFound
}

func (m *mockAccountRepo) GetByIDWithLock(ctx context.Context, id string) (*domain.Account, error) {
	return m.GetByID(ctx, id)
}

func (m *mockAccountRepo) UpdateBalance(ctx context.Context, acc *domain.Account) error {
	m.accounts[acc.ID] = acc
	return nil
}

type mockLedgerRepo struct {
	transactions []*domain.Transaction
	entries      []domain.LedgerEntry
	outboxEvents []string
}

func newMockLedgerRepo() *mockLedgerRepo {
	return &mockLedgerRepo{}
}

func (m *mockLedgerRepo) CreateTransaction(ctx context.Context, tx *domain.Transaction) error {
	m.transactions = append(m.transactions, tx)
	return nil
}

func (m *mockLedgerRepo) CreateLedgerEntries(ctx context.Context, entries []domain.LedgerEntry) error {
	m.entries = append(m.entries, entries...)
	return nil
}

func (m *mockLedgerRepo) GetLatestEntryHash(ctx context.Context, accountID string) (string, error) {
	// Cari mutasi terakhir untuk akun ini
	for i := len(m.entries) - 1; i >= 0; i-- {
		if m.entries[i].AccountID == accountID {
			return m.entries[i].CurrentHash, nil
		}
	}
	return hashchain.GenesisHash, nil
}

func (m *mockLedgerRepo) CreateOutboxEvent(ctx context.Context, eventType, aggregateID string, payload []byte) error {
	m.outboxEvents = append(m.outboxEvents, eventType+":"+aggregateID)
	return nil
}

// --- Test Cases ---

// Test 1: Skenario Sukses Transfer P2P (Balanced Double-Entry & SHA-256 Chain)
func TestTransferUsecase_Success(t *testing.T) {
	// 1. ARRANGE
	accRepo := newMockAccountRepo()
	ledgerRepo := newMockLedgerRepo()
	txMgr := &mockTxManager{}
	uc := NewTransferUsecase(accRepo, ledgerRepo, txMgr)

	sourceAcc := &domain.Account{
		ID:            "acc-001-source",
		AccountNumber: "1001000001",
		Balance:       1000000, // Rp 1.000.000
		HoldBalance:   0,
		Status:        domain.StatusActive,
		CreatedAt:     time.Now().UTC(),
	}
	destAcc := &domain.Account{
		ID:            "acc-002-dest",
		AccountNumber: "1001000002",
		Balance:       500000, // Rp 500.000
		HoldBalance:   0,
		Status:        domain.StatusActive,
		CreatedAt:     time.Now().UTC(),
	}
	accRepo.accounts[sourceAcc.ID] = sourceAcc
	accRepo.accounts[destAcc.ID] = destAcc

	cmd := TransferCommand{
		ReferenceID:          "ref-tx-001",
		SourceAccountID:      sourceAcc.ID,
		DestinationAccountID: destAcc.ID,
		Amount:               350000, // Transfer Rp 350.000
		Description:          "Pembayaran Invoice #01",
		MakerID:              "customer_user_01",
	}

	// 2. ACT
	result, err := uc.Execute(context.Background(), cmd)

	// 3. ASSERT
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result == nil {
		t.Fatal("expected result not to be nil")
	}

	// Cek Saldo Akhir
	if sourceAcc.Balance != 650000 {
		t.Errorf("expected source balance 650.000, got: %d", sourceAcc.Balance)
	}
	if destAcc.Balance != 850000 {
		t.Errorf("expected dest balance 850.000, got: %d", destAcc.Balance)
	}

	// Cek Invarian Double-Entry: Harus ada tepat 2 entri jurnal (1 Debit, 1 Kredit)
	if len(ledgerRepo.entries) != 2 {
		t.Fatalf("expected 2 ledger entries, got: %d", len(ledgerRepo.entries))
	}
	entryDebit := ledgerRepo.entries[0]
	entryCredit := ledgerRepo.entries[1]

	if entryDebit.EntryType != domain.EntryTypeDebit || entryDebit.Amount != 350000 {
		t.Errorf("invalid debit entry: %+v", entryDebit)
	}
	if entryCredit.EntryType != domain.EntryTypeCredit || entryCredit.Amount != 350000 {
		t.Errorf("invalid credit entry: %+v", entryCredit)
	}

	// Verifikasi Validitas Kriptografis Rantai Hash SHA-256
	isValidHash := hashchain.VerifyHash(entryDebit.PreviousHash, entryDebit.AccountID, entryDebit.Amount, entryDebit.BalanceAfter, entryDebit.CreatedAt, entryDebit.CurrentHash)
	if !isValidHash {
		t.Error("SHA-256 audit hash verification failed for debit entry!")
	}
}

// Test 2: Skenario Penolakan Saldo Tidak Mencukupi (Insufficient Funds)
func TestTransferUsecase_InsufficientFunds(t *testing.T) {
	accRepo := newMockAccountRepo()
	ledgerRepo := newMockLedgerRepo()
	txMgr := &mockTxManager{}
	uc := NewTransferUsecase(accRepo, ledgerRepo, txMgr)

	sourceAcc := &domain.Account{
		ID:            "acc-001",
		AccountNumber: "1001",
		Balance:       100000, // Hanya Rp 100.000
		Status:        domain.StatusActive,
	}
	destAcc := &domain.Account{
		ID:            "acc-002",
		AccountNumber: "1002",
		Balance:       50000,
		Status:        domain.StatusActive,
	}
	accRepo.accounts[sourceAcc.ID] = sourceAcc
	accRepo.accounts[destAcc.ID] = destAcc

	cmd := TransferCommand{
		ReferenceID:          "ref-tx-002",
		SourceAccountID:      sourceAcc.ID,
		DestinationAccountID: destAcc.ID,
		Amount:               500000, // Mencoba transfer Rp 500.000
		MakerID:              "customer_01",
	}

	_, err := uc.Execute(context.Background(), cmd)

	if !errors.Is(err, domain.ErrInsufficientFunds) {
		t.Fatalf("expected ErrInsufficientFunds, got: %v", err)
	}

	// Pastikan saldo tetap utuh (tidak berubah)
	if sourceAcc.Balance != 100000 {
		t.Errorf("source balance should remain 100.000, got: %d", sourceAcc.Balance)
	}
	if destAcc.Balance != 50000 {
		t.Errorf("dest balance should remain 50.000, got: %d", destAcc.Balance)
	}
}

// Test 3: Skenario Penolakan Rekening Berstatus FROZEN
func TestTransferUsecase_FrozenAccount(t *testing.T) {
	accRepo := newMockAccountRepo()
	ledgerRepo := newMockLedgerRepo()
	txMgr := &mockTxManager{}
	uc := NewTransferUsecase(accRepo, ledgerRepo, txMgr)

	sourceAcc := &domain.Account{
		ID:            "acc-001",
		AccountNumber: "1001",
		Balance:       1000000,
		Status:        domain.StatusFrozen, // Rekening Dibekukan!
	}
	destAcc := &domain.Account{
		ID:            "acc-002",
		AccountNumber: "1002",
		Balance:       50000,
		Status:        domain.StatusActive,
	}
	accRepo.accounts[sourceAcc.ID] = sourceAcc
	accRepo.accounts[destAcc.ID] = destAcc

	cmd := TransferCommand{
		ReferenceID:          "ref-tx-003",
		SourceAccountID:      sourceAcc.ID,
		DestinationAccountID: destAcc.ID,
		Amount:               100000,
		MakerID:              "customer_01",
	}

	_, err := uc.Execute(context.Background(), cmd)

	if !errors.Is(err, domain.ErrAccountNotActive) {
		t.Fatalf("expected ErrAccountNotActive, got: %v", err)
	}
}

// Test 4: Verifikasi Algoritma Determinisme Pengurutan Leksikografis
func TestTransferUsecase_SortIDs(t *testing.T) {
	first, second := sortIDs("uuid-zebra", "uuid-apple")
	if first != "uuid-apple" || second != "uuid-zebra" {
		t.Errorf("expected (uuid-apple, uuid-zebra), got (%s, %s)", first, second)
	}

	first2, second2 := sortIDs("uuid-apple", "uuid-zebra")
	if first2 != "uuid-apple" || second2 != "uuid-zebra" {
		t.Errorf("expected (uuid-apple, uuid-zebra), got (%s, %s)", first2, second2)
	}
}
