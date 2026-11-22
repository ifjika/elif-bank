package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/elif-bank/apps/api-gateway/internal/auth"
	"github.com/elif-bank/apps/api-gateway/internal/middleware"
	"github.com/elif-bank/apps/core-ledger/ledger"
	"github.com/gin-gonic/gin"
)

type mockBranchLedgerService struct {
	cashFn     func(ctx context.Context, in ledger.BranchCashInput) (*ledger.BranchCashOutput, error)
	approvalFn func(ctx context.Context, in ledger.BranchApprovalInput) (*ledger.BranchApprovalOutput, error)
	pendingFn  func(ctx context.Context) ([]ledger.PendingBranchTx, error)
	entriesFn  func(ctx context.Context, limit int) ([]ledger.AuditLedgerRow, error)
}

func (m *mockBranchLedgerService) ExecuteBranchCash(ctx context.Context, in ledger.BranchCashInput) (*ledger.BranchCashOutput, error) {
	return m.cashFn(ctx, in)
}
func (m *mockBranchLedgerService) ExecuteBranchApproval(ctx context.Context, in ledger.BranchApprovalInput) (*ledger.BranchApprovalOutput, error) {
	return m.approvalFn(ctx, in)
}
func (m *mockBranchLedgerService) ListPendingApprovals(ctx context.Context) ([]ledger.PendingBranchTx, error) {
	return m.pendingFn(ctx)
}
func (m *mockBranchLedgerService) ListRecentLedgerEntries(ctx context.Context, limit int) ([]ledger.AuditLedgerRow, error) {
	return m.entriesFn(ctx, limit)
}

func TestBranchHandler_CashTransaction_Success(t *testing.T) {
	// 1. ARRANGE
	tokenMgr := auth.NewTokenManager("test-secret-key-at-least-32-bytes-long", 1*time.Hour)
	tellerToken, _ := tokenMgr.GenerateToken("usr_teller_01", "teller_budi", auth.RoleTeller, "branch-jkt-01")

	mockSvc := &mockBranchLedgerService{
		cashFn: func(ctx context.Context, in ledger.BranchCashInput) (*ledger.BranchCashOutput, error) {
			if in.MakerID != "usr_teller_01" {
				t.Errorf("expected MakerID from JWT usr_teller_01, got %s", in.MakerID)
			}
			return &ledger.BranchCashOutput{
				TransactionID:   in.ReferenceID,
				ReferenceID:     in.ReferenceID,
				TransactionType: in.TransactionType,
				Amount:          in.Amount,
				Status:          "PENDING_APPROVAL",
			}, nil
		},
	}

	handler := NewBranchHandler(mockSvc)
	router := gin.New()
	router.POST("/api/v1/branch/cash-transactions", middleware.AuthMiddleware(tokenMgr, auth.RoleTeller), handler.HandleCashTransaction)

	reqBody := `{"account_number":"1001000012","transaction_type":"CASH_WITHDRAWAL","amount":75000000,"notes":"Corporate withdrawal"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/branch/cash-transactions", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tellerToken)
	w := httptest.NewRecorder()

	// 2. ACT
	router.ServeHTTP(w, req)

	// 3. ASSERT
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d. Body: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["success"] != true {
		t.Errorf("expected success true, got %v", resp["success"])
	}
}

func TestBranchHandler_Approval_SelfApprovalForbidden(t *testing.T) {
	// 1. ARRANGE
	tokenMgr := auth.NewTokenManager("test-secret-key-at-least-32-bytes-long", 1*time.Hour)
	spvToken, _ := tokenMgr.GenerateToken("usr_spv_01", "spv_hendra", auth.RoleSupervisor, "branch-jkt-01")

	mockSvc := &mockBranchLedgerService{
		approvalFn: func(ctx context.Context, in ledger.BranchApprovalInput) (*ledger.BranchApprovalOutput, error) {
			return nil, ledger.ErrSelfApprovalForbidden
		},
	}

	handler := NewBranchHandler(mockSvc)
	router := gin.New()
	router.POST("/api/v1/branch/approvals/:transaction_id", middleware.AuthMiddleware(tokenMgr, auth.RoleSupervisor), handler.HandleApproval)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/branch/approvals/tx-001", bytes.NewBufferString(`{"action":"APPROVE","notes":"test"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+spvToken)
	w := httptest.NewRecorder()

	// 2. ACT
	router.ServeHTTP(w, req)

	// 3. ASSERT
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden on self-approval, got %d", w.Code)
	}
}
