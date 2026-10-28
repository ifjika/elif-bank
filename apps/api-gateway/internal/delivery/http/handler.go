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
// Related Documents: docs/INTEGRATION.md (Section 1 & 2) & docs/ARCHITECTURE.md
// Architectural Note: API Gateway does not import core-ledger internals directly;
//                      it relies on standalone DTOs & Service Interfaces (Loose Coupling).
// ==============================================================================

var (
	ErrInsufficientFunds   = errors.New("insufficient available balance")
	ErrAccountNotActive    = errors.New("account is not active")
	ErrInvalidAmount       = errors.New("transaction amount must be greater than zero")
	ErrSameAccountTransfer = errors.New("source and destination account cannot be the same")
	ErrAccountNotFound     = errors.New("account not found")
)

// TransferCommand DTO for dispatching transactions to service layer / gRPC.
type TransferCommand struct {
	ReferenceID          string
	SourceAccountID      string
	DestinationAccountID string
	Amount               int64
	Description          string
	MakerID              string
}

// TransferResult DTO response from transfer execution.
type TransferResult struct {
	TransactionID string    `json:"transaction_id"`
	ReferenceID   string    `json:"reference_id"`
	Amount        int64     `json:"amount"`
	SourceBalance int64     `json:"source_balance_after"`
	DestBalance   int64     `json:"dest_balance_after"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
}

// AccountInfo DTO for account details.
type AccountInfo struct {
	ID               string `json:"id"`
	AccountNumber    string `json:"account_number"`
	Currency         string `json:"currency"`
	Balance          int64  `json:"total_balance"`
	AvailableBalance int64  `json:"available_balance"`
	HoldBalance      int64  `json:"hold_balance"`
	Status           string `json:"status"`
}

// TransferService defines the contract for transfer operations to the Core Ledger.
type TransferService interface {
	Execute(ctx context.Context, cmd TransferCommand) (*TransferResult, error)
}

// AccountService defines the contract for reading account data.
type AccountService interface {
	GetAccount(ctx context.Context, id string) (*AccountInfo, error)
}

// Handler bundles all HTTP router dependencies.
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

// TransferRequestPayload represents JSON request body for transfers.
type TransferRequestPayload struct {
	DestinationAccountID string `json:"destination_account_id" binding:"required"`
	SourceAccountID      string `json:"source_account_id" binding:"required"`
	Amount               int64  `json:"amount" binding:"required,gt=0"`
	Description          string `json:"description"`
}

// HandleTransfer processes POST /api/v1/transfers endpoint
func (h *Handler) HandleTransfer(c *gin.Context) {
	var req TransferRequestPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":        "INVALID_PAYLOAD",
				"message":     "Invalid request payload: " + err.Error(),
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

// HandleGetBalance processes GET /api/v1/accounts/:id/balance endpoint
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
					"message":     "Account not found in ledger system",
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
