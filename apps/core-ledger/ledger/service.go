package ledger

import (
	"context"
	"database/sql"
	"time"

	"github.com/elif-bank/apps/core-ledger/internal/domain"
	"github.com/elif-bank/apps/core-ledger/internal/repository"
	"github.com/elif-bank/apps/core-ledger/internal/usecase"
)

// ==============================================================================
// Public Facade Package: ledger
// Purpose: Exposes Core Ledger usecases & TiDB queries to API Gateway without
//          breaching Go's internal/ module visibility boundaries.
// ==============================================================================

const DefaultVaultAccountID = "acc-vault-jkt01"

var (
	ErrInsufficientFunds     = domain.ErrInsufficientFunds
	ErrAccountNotActive      = domain.ErrAccountNotActive
	ErrInvalidAmount         = domain.ErrInvalidAmount
	ErrAccountNotFound       = domain.ErrAccountNotFound
	ErrTxNotFound            = domain.ErrTxNotFound
	ErrTxNotPendingApproval  = domain.ErrTxNotPendingApproval
	ErrSelfApprovalForbidden = domain.ErrSelfApprovalForbidden
	ErrInvalidBranchTxType   = usecase.ErrInvalidBranchTxType
	ErrInvalidApprovalAction = usecase.ErrInvalidApprovalAction
)

type BranchCashInput struct {
	ReferenceID     string
	AccountNumber   string
	TransactionType string
	Amount          int64
	Description     string
	MakerID         string
}

type BranchCashOutput struct {
	TransactionID    string    `json:"transaction_id"`
	ReferenceID      string    `json:"reference_id"`
	TransactionType  string    `json:"transaction_type"`
	Amount           int64     `json:"amount"`
	BalanceAfter     int64     `json:"balance_after"`
	HoldBalanceAfter int64     `json:"hold_balance_after"`
	Status           string    `json:"status"`
	CreatedAt        time.Time `json:"created_at"`
}

type BranchApprovalInput struct {
	TransactionID string
	CheckerID     string
	Action        string
	Notes         string
}

type BranchApprovalOutput struct {
	TransactionID    string    `json:"transaction_id"`
	Status           string    `json:"status"`
	CheckerID        string    `json:"checker_id"`
	BalanceAfter     int64     `json:"balance_after"`
	HoldBalanceAfter int64     `json:"hold_balance_after"`
	ApprovedAt       time.Time `json:"approved_at"`
}

type PendingBranchTx = repository.PendingBranchTx
type AuditLedgerRow = repository.AuditLedgerRow

type Service struct {
	cashUC     *usecase.BranchCashUsecase
	approvalUC *usecase.BranchApprovalUsecase
	ledgerRepo *repository.MySQLLedgerRepository
}

func NewService(db *sql.DB) *Service {
	accRepo := repository.NewMySQLAccountRepository(db)
	ledgerRepo := repository.NewMySQLLedgerRepository(db)
	txMgr := repository.NewMySQLTransactionManager(db)

	return &Service{
		cashUC:     usecase.NewBranchCashUsecase(accRepo, ledgerRepo, txMgr),
		approvalUC: usecase.NewBranchApprovalUsecase(accRepo, ledgerRepo, txMgr),
		ledgerRepo: ledgerRepo,
	}
}

func SeedDemoAccountsIfEmpty(ctx context.Context, db *sql.DB) error {
	return repository.SeedDemoAccountsIfEmpty(ctx, db)
}

func (s *Service) ExecuteBranchCash(ctx context.Context, in BranchCashInput) (*BranchCashOutput, error) {
	res, err := s.cashUC.Execute(ctx, usecase.BranchCashCommand{
		ReferenceID:     in.ReferenceID,
		AccountNumber:   in.AccountNumber,
		VaultAccountID:  DefaultVaultAccountID,
		TransactionType: domain.TransactionType(in.TransactionType),
		Amount:          in.Amount,
		Description:     in.Description,
		MakerID:         in.MakerID,
	})
	if err != nil {
		return nil, err
	}
	return &BranchCashOutput{
		TransactionID:    res.TransactionID,
		ReferenceID:      res.ReferenceID,
		TransactionType:  res.TransactionType,
		Amount:           res.Amount,
		BalanceAfter:     res.BalanceAfter,
		HoldBalanceAfter: res.HoldBalanceAfter,
		Status:           res.Status,
		CreatedAt:        res.CreatedAt,
	}, nil
}

func (s *Service) ExecuteBranchApproval(ctx context.Context, in BranchApprovalInput) (*BranchApprovalOutput, error) {
	res, err := s.approvalUC.Execute(ctx, usecase.BranchApprovalCommand{
		TransactionID: in.TransactionID,
		CheckerID:     in.CheckerID,
		Action:        usecase.ApprovalAction(in.Action),
		Notes:         in.Notes,
	})
	if err != nil {
		return nil, err
	}
	return &BranchApprovalOutput{
		TransactionID:    res.TransactionID,
		Status:           res.Status,
		CheckerID:        res.CheckerID,
		BalanceAfter:     res.BalanceAfter,
		HoldBalanceAfter: res.HoldBalanceAfter,
		ApprovedAt:       res.ApprovedAt,
	}, nil
}

func (s *Service) ListPendingApprovals(ctx context.Context) ([]PendingBranchTx, error) {
	return s.ledgerRepo.ListPendingBranchTransactions(ctx)
}

func (s *Service) ListRecentLedgerEntries(ctx context.Context, limit int) ([]AuditLedgerRow, error) {
	return s.ledgerRepo.ListRecentLedgerEntries(ctx, limit)
}
