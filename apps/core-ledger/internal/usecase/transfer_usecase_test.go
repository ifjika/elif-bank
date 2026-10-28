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
// Related Documents: docs/TESTING_GUIDE.md (Section 2) & docs/PRD.md (Section 3.2)
// ==============================================================================

// --- Mock Implementations for Domain Interfaces (In-Memory) ---

type mockTxManager struct{}

func (m *mockTxManager) ExecuteInTransaction(ctx context.Context, fn func(txCtx context.Context) error) error {
	return fn(ctx) // Directly execute transaction logic in-memory
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
	// Find latest mutation entry for this account
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

// Test 1: Successful P2P Transfer (Balanced Double-Entry & SHA-256 Chain)
func TestTransferUsecase_Success(t *testing.T) {
	// 1. ARRANGE
	accRepo := newMockAccountRepo()
	ledgerRepo := newMockLedgerRepo()
	txMgr := &mockTxManager{}
	uc := NewTransferUsecase(accRepo, ledgerRepo, txMgr)

	sourceAcc := &domain.Account{
		ID:            "acc-001-source",
		AccountNumber: "1001000001",
		Balance:       1000000, // IDR 1,000,000
		HoldBalance:   0,
		Status:        domain.StatusActive,
		CreatedAt:     time.Now().UTC(),
	}
	destAcc := &domain.Account{
		ID:            "acc-002-dest",
		AccountNumber: "1001000002",
		Balance:       500000, // IDR 500,000
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
		Amount:               350000, // Transfer IDR 350,000
		Description:          "Invoice Payment #01",
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

	// Check Ending Balances
	if sourceAcc.Balance != 650000 {
		t.Errorf("expected source balance 650,000, got: %d", sourceAcc.Balance)
	}
	if destAcc.Balance != 850000 {
		t.Errorf("expected dest balance 850,000, got: %d", destAcc.Balance)
	}

	// Check Double-Entry Invariant: Exactly 2 journal entries (1 Debit, 1 Credit)
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

	// Verify Cryptographic Validity of SHA-256 Hash Chain
	isValidHash := hashchain.VerifyHash(entryDebit.PreviousHash, entryDebit.AccountID, entryDebit.Amount, entryDebit.BalanceAfter, entryDebit.CreatedAt, entryDebit.CurrentHash)
	if !isValidHash {
		t.Error("SHA-256 audit hash verification failed for debit entry!")
	}
}

// Test 2: Rejection Scenario for Insufficient Balance
func TestTransferUsecase_InsufficientFunds(t *testing.T) {
	accRepo := newMockAccountRepo()
	ledgerRepo := newMockLedgerRepo()
	txMgr := &mockTxManager{}
	uc := NewTransferUsecase(accRepo, ledgerRepo, txMgr)

	sourceAcc := &domain.Account{
		ID:            "acc-001",
		AccountNumber: "1001",
		Balance:       100000, // IDR 100,000 only
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
		Amount:               500000, // Attempting to transfer IDR 500,000
		MakerID:              "customer_01",
	}

	_, err := uc.Execute(context.Background(), cmd)

	if !errors.Is(err, domain.ErrInsufficientFunds) {
		t.Fatalf("expected ErrInsufficientFunds, got: %v", err)
	}

	// Ensure balances remain intact (unmodified)
	if sourceAcc.Balance != 100000 {
		t.Errorf("source balance should remain 100,000, got: %d", sourceAcc.Balance)
	}
	if destAcc.Balance != 50000 {
		t.Errorf("dest balance should remain 50,000, got: %d", destAcc.Balance)
	}
}

// Test 3: Rejection Scenario for FROZEN Account
func TestTransferUsecase_FrozenAccount(t *testing.T) {
	accRepo := newMockAccountRepo()
	ledgerRepo := newMockLedgerRepo()
	txMgr := &mockTxManager{}
	uc := NewTransferUsecase(accRepo, ledgerRepo, txMgr)

	sourceAcc := &domain.Account{
		ID:            "acc-001",
		AccountNumber: "1001",
		Balance:       1000000,
		Status:        domain.StatusFrozen, // Frozen Account
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

// Test 4: Verification of Deterministic Lexicographical Sorting Algorithm
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
