package http

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/elif-bank/apps/api-gateway/internal/auth"
	"github.com/gin-gonic/gin"
)

// ==============================================================================
// Delivery HTTP: Handlers & Router Setup (Gin)
// Dokumen Terkait: docs/INTEGRATION.md (Seksi 1 & 2) & docs/ARCHITECTURE.md
// Catatan Arsitektur: API Gateway tidak mengimpor internal core-ledger secara langsung
//                     melainkan menggunakan DTO & Service Interface mandiri (Loose Coupling).
// ==============================================================================

var (
	ErrInsufficientFunds   = errors.New("insufficient available balance")
	ErrAccountNotActive    = errors.New("account is not active")
	ErrInvalidAmount       = errors.New("transaction amount must be greater than zero")
	ErrSameAccountTransfer = errors.New("source and destination account cannot be the same")
	ErrAccountNotFound     = errors.New("account not found")
)

// TransferCommand DTO untuk pengiriman transaksi ke service layer / gRPC.
type TransferCommand struct {
	ReferenceID          string
	SourceAccountID      string
	DestinationAccountID string
	Amount               int64
	Description          string
	MakerID              string
}

// TransferResult DTO balasan dari eksekusi transfer.
type TransferResult struct {
	TransactionID string    `json:"transaction_id"`
	ReferenceID   string    `json:"reference_id"`
	Amount        int64     `json:"amount"`
	SourceBalance int64     `json:"source_balance_after"`
	DestBalance   int64     `json:"dest_balance_after"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
}

// AccountInfo DTO informasi rekening.
type AccountInfo struct {
	ID               string `json:"id"`
	AccountNumber    string `json:"account_number"`
	Currency         string `json:"currency"`
	Balance          int64  `json:"total_balance"`
	AvailableBalance int64  `json:"available_balance"`
	HoldBalance      int64  `json:"hold_balance"`
	Status           string `json:"status"`
}

// TransferService mendefinisikan kontrak operasi transfer ke Core Ledger.
type TransferService interface {
	Execute(ctx context.Context, cmd TransferCommand) (*TransferResult, error)
}

// AccountService mendefinisikan kontrak pembacaan data akun.
type AccountService interface {
	GetAccount(ctx context.Context, id string) (*AccountInfo, error)
}

// Handler membundel seluruh dependensi router HTTP.
type Handler struct {
	transferService TransferService
	accountService  AccountService
	tokenManager    *auth.TokenManager
}

func NewHandler(
	transferService TransferService,
	accountService AccountService,
	tokenManager *auth.TokenManager,
) *Handler {
	return &Handler{
		transferService: transferService,
		accountService:  accountService,
		tokenManager:    tokenManager,
	}
}

// TransferRequestPayload merepresentasikan DTO body request transfer JSON.
type TransferRequestPayload struct {
	DestinationAccountID string `json:"destination_account_id" binding:"required"`
	SourceAccountID      string `json:"source_account_id" binding:"required"`
	Amount               int64  `json:"amount" binding:"required,gt=0"`
	Description          string `json:"description"`
}

// HandleTransfer memproses endpoint POST /api/v1/transfers
func (h *Handler) HandleTransfer(c *gin.Context) {
	var req TransferRequestPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":        "INVALID_PAYLOAD",
				"message":     "Format request tidak valid: " + err.Error(),
				"status_code": http.StatusBadRequest,
			},
		})
		return
	}

	idempotencyKey := c.GetHeader("X-Idempotency-Key")
	userIDVal, _ := c.Get("user_id")
	makerID, _ := userIDVal.(string)
	if makerID == "" {
		makerID = "unknown_client"
	}

	cmd := TransferCommand{
		ReferenceID:          idempotencyKey,
		SourceAccountID:      req.SourceAccountID,
		DestinationAccountID: req.DestinationAccountID,
		Amount:               req.Amount,
		Description:          req.Description,
		MakerID:              makerID,
	}

	result, err := h.transferService.Execute(c.Request.Context(), cmd)
	if err != nil {
		statusCode := http.StatusInternalServerError
		errorCode := "INTERNAL_LEDGER_ERROR"

		switch {
		case errors.Is(err, ErrInsufficientFunds):
			statusCode = http.StatusUnprocessableEntity
			errorCode = "INSUFFICIENT_FUNDS"
		case errors.Is(err, ErrAccountNotActive):
			statusCode = http.StatusUnprocessableEntity
			errorCode = "ACCOUNT_NOT_ACTIVE"
		case errors.Is(err, ErrInvalidAmount):
			statusCode = http.StatusBadRequest
			errorCode = "INVALID_AMOUNT"
		case errors.Is(err, ErrSameAccountTransfer):
			statusCode = http.StatusBadRequest
			errorCode = "SAME_ACCOUNT_TRANSFER"
		case errors.Is(err, ErrAccountNotFound):
			statusCode = http.StatusNotFound
			errorCode = "ACCOUNT_NOT_FOUND"
		}

		c.JSON(statusCode, gin.H{
			"success": false,
			"error": gin.H{
				"code":        errorCode,
				"message":     err.Error(),
				"status_code": statusCode,
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
		"meta": gin.H{
			"server_time": time.Now().UTC().Format(time.RFC3339),
		},
	})
}

// HandleGetBalance memproses endpoint GET /api/v1/accounts/:id/balance
func (h *Handler) HandleGetBalance(c *gin.Context) {
	accountID := c.Param("id")
	if accountID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Account ID is required"})
		return
	}

	acc, err := h.accountService.GetAccount(c.Request.Context(), accountID)
	if err != nil {
		if errors.Is(err, ErrAccountNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error": gin.H{
					"code":        "ACCOUNT_NOT_FOUND",
					"message":     "Rekening tidak ditemukan di sistem buku besar",
					"status_code": http.StatusNotFound,
				},
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    acc,
	})
}
