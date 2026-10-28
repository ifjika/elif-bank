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
	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// Mock Services
type mockTransferService struct {
	executeFn func(ctx context.Context, cmd TransferCommand) (*TransferResult, error)
}

func (m *mockTransferService) Execute(ctx context.Context, cmd TransferCommand) (*TransferResult, error) {
	return m.executeFn(ctx, cmd)
}

type mockAccountService struct {
	getAccountFn func(ctx context.Context, id string) (*AccountInfo, error)
}

func (m *mockAccountService) GetAccount(ctx context.Context, id string) (*AccountInfo, error) {
	return m.getAccountFn(ctx, id)
}

func TestHandler_HandleTransfer_Success(t *testing.T) {
	tokenManager := auth.NewTokenManager("secret-key-32-bytes-minimum-length!", 1*time.Hour)
	mockTransfer := &mockTransferService{
		executeFn: func(ctx context.Context, cmd TransferCommand) (*TransferResult, error) {
			return &TransferResult{
				TransactionID: "tx_123",
				ReferenceID:   cmd.ReferenceID,
				Amount:        cmd.Amount,
				SourceBalance: 750000,
				DestBalance:   250000,
				Status:        "POSTED",
				CreatedAt:     time.Now().UTC(),
			}, nil
		},
	}
	mockAccount := &mockAccountService{}

	handler := NewHandler(mockTransfer, mockAccount, tokenManager)
	router := gin.New()
	router.POST("/api/v1/transfers", handler.HandleTransfer)

	bodyPayload := `{"source_account_id":"acc-1","destination_account_id":"acc-2","amount":250000,"description":"Transfer Test"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/transfers", bytes.NewBufferString(bodyPayload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Idempotency-Key", "idemp-key-001")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp["success"] != true {
		t.Errorf("expected success: true, got %v", resp["success"])
	}
}

func TestHandler_HandleTransfer_InsufficientFunds(t *testing.T) {
	tokenManager := auth.NewTokenManager("secret-key-32-bytes-minimum-length!", 1*time.Hour)
	mockTransfer := &mockTransferService{
		executeFn: func(ctx context.Context, cmd TransferCommand) (*TransferResult, error) {
			return nil, ErrInsufficientFunds
		},
	}
	mockAccount := &mockAccountService{}

	handler := NewHandler(mockTransfer, mockAccount, tokenManager)
	router := gin.New()
	router.POST("/api/v1/transfers", handler.HandleTransfer)

	bodyPayload := `{"source_account_id":"acc-1","destination_account_id":"acc-2","amount":999999999,"description":"Overbalance"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/transfers", bytes.NewBufferString(bodyPayload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status 422 Unprocessable Entity, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestHandler_HandleGetBalance_Success(t *testing.T) {
	tokenManager := auth.NewTokenManager("secret-key-32-bytes-minimum-length!", 1*time.Hour)
	mockTransfer := &mockTransferService{}
	mockAccount := &mockAccountService{
		getAccountFn: func(ctx context.Context, id string) (*AccountInfo, error) {
			return &AccountInfo{
				ID:               id,
				AccountNumber:    "1001000001",
				Currency:         "IDR",
				Balance:          1000000,
				AvailableBalance: 800000,
				HoldBalance:      200000,
				Status:           "ACTIVE",
			}, nil
		},
	}

	handler := NewHandler(mockTransfer, mockAccount, tokenManager)
	router := gin.New()
	router.GET("/api/v1/accounts/:id/balance", handler.HandleGetBalance)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/accounts/acc-001/balance", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
}
