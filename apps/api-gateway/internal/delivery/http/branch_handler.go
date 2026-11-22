package http

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/elif-bank/apps/core-ledger/ledger"
	"github.com/gin-gonic/gin"
)

// ==============================================================================
// Delivery HTTP: Branch Operations & Maker-Checker Handler (Phase 5.3)
// Related Documents: docs/INTEGRATION.md (2.3 & 2.4) & docs/SECURITY_GUIDE.md
// ==============================================================================

type BranchLedgerService interface {
	ExecuteBranchCash(ctx context.Context, in ledger.BranchCashInput) (*ledger.BranchCashOutput, error)
	ExecuteBranchApproval(ctx context.Context, in ledger.BranchApprovalInput) (*ledger.BranchApprovalOutput, error)
	ListPendingApprovals(ctx context.Context) ([]ledger.PendingBranchTx, error)
	ListRecentLedgerEntries(ctx context.Context, limit int) ([]ledger.AuditLedgerRow, error)
}

type BranchHandler struct {
	ledgerSvc BranchLedgerService
}

func NewBranchHandler(ledgerSvc BranchLedgerService) *BranchHandler {
	return &BranchHandler{ledgerSvc: ledgerSvc}
}

type BranchCashRequestPayload struct {
	AccountNumber   string `json:"account_number" binding:"required"`
	TransactionType string `json:"transaction_type" binding:"required"`
	Amount          int64  `json:"amount" binding:"required,gt=0"`
	Notes           string `json:"notes"`
}

type BranchApprovalRequestPayload struct {
	Action string `json:"action" binding:"required"`
	Notes  string `json:"notes"`
}

// HandleCashTransaction processes POST /api/v1/branch/cash-transactions (Role: TELLER / Maker).
func (h *BranchHandler) HandleCashTransaction(c *gin.Context) {
	var req BranchCashRequestPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":        "INVALID_PAYLOAD",
				"message":     "Invalid cash transaction payload: " + err.Error(),
				"status_code": http.StatusBadRequest,
			},
		})
		return
	}

	userIDVal, _ := c.Get("user_id")
	makerID, _ := userIDVal.(string)
	if makerID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error": gin.H{
				"code":        "UNAUTHORIZED",
				"message":     "Missing authenticated maker identity",
				"status_code": http.StatusUnauthorized,
			},
		})
		return
	}

	refID := c.GetHeader("X-Idempotency-Key")
	if refID == "" {
		refID = fmt.Sprintf("tx_%d", time.Now().UnixNano())
	}

	res, err := h.ledgerSvc.ExecuteBranchCash(c.Request.Context(), ledger.BranchCashInput{
		ReferenceID:     refID,
		AccountNumber:   req.AccountNumber,
		TransactionType: req.TransactionType,
		Amount:          req.Amount,
		Description:     req.Notes,
		MakerID:         makerID,
	})
	if err != nil {
		statusCode := http.StatusInternalServerError
		errorCode := "INTERNAL_LEDGER_ERROR"

		switch {
		case errors.Is(err, ledger.ErrAccountNotFound):
			statusCode = http.StatusNotFound
			errorCode = "ACCOUNT_NOT_FOUND"
		case errors.Is(err, ledger.ErrInsufficientFunds):
			statusCode = http.StatusUnprocessableEntity
			errorCode = "INSUFFICIENT_FUNDS"
		case errors.Is(err, ledger.ErrAccountNotActive):
			statusCode = http.StatusUnprocessableEntity
			errorCode = "ACCOUNT_NOT_ACTIVE"
		case errors.Is(err, ledger.ErrInvalidAmount), errors.Is(err, ledger.ErrInvalidBranchTxType):
			statusCode = http.StatusBadRequest
			errorCode = "INVALID_TRANSACTION_REQUEST"
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
		"data":    res,
		"meta": gin.H{
			"server_time": time.Now().UTC().Format(time.RFC3339),
		},
	})
}

// HandleListPending processes GET /api/v1/branch/approvals/pending (Role: SUPERVISOR / Checker).
func (h *BranchHandler) HandleListPending(c *gin.Context) {
	items, err := h.ledgerSvc.ListPendingApprovals(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":        "INTERNAL_LEDGER_ERROR",
				"message":     "Failed to query pending approvals",
				"status_code": http.StatusInternalServerError,
			},
		})
		return
	}
	if items == nil {
		items = []ledger.PendingBranchTx{}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    items,
	})
}

// HandleApproval processes POST /api/v1/branch/approvals/:transaction_id (Role: SUPERVISOR / Checker).
func (h *BranchHandler) HandleApproval(c *gin.Context) {
	txID := c.Param("transaction_id")
	if txID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":        "MISSING_TRANSACTION_ID",
				"message":     "transaction_id path parameter is required",
				"status_code": http.StatusBadRequest,
			},
		})
		return
	}

	var req BranchApprovalRequestPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":        "INVALID_PAYLOAD",
				"message":     "Invalid approval payload: " + err.Error(),
				"status_code": http.StatusBadRequest,
			},
		})
		return
	}

	userIDVal, _ := c.Get("user_id")
	checkerID, _ := userIDVal.(string)
	if checkerID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error": gin.H{
				"code":        "UNAUTHORIZED",
				"message":     "Missing authenticated checker identity",
				"status_code": http.StatusUnauthorized,
			},
		})
		return
	}

	res, err := h.ledgerSvc.ExecuteBranchApproval(c.Request.Context(), ledger.BranchApprovalInput{
		TransactionID: txID,
		CheckerID:     checkerID,
		Action:        req.Action,
		Notes:         req.Notes,
	})
	if err != nil {
		statusCode := http.StatusInternalServerError
		errorCode := "INTERNAL_LEDGER_ERROR"

		switch {
		case errors.Is(err, ledger.ErrSelfApprovalForbidden):
			statusCode = http.StatusForbidden
			errorCode = "FORBIDDEN_MAKER_CHECKER"
		case errors.Is(err, ledger.ErrTxNotFound):
			statusCode = http.StatusNotFound
			errorCode = "TRANSACTION_NOT_FOUND"
		case errors.Is(err, ledger.ErrTxNotPendingApproval):
			statusCode = http.StatusConflict
			errorCode = "TRANSACTION_ALREADY_PROCESSED"
		case errors.Is(err, ledger.ErrInvalidApprovalAction):
			statusCode = http.StatusBadRequest
			errorCode = "INVALID_APPROVAL_ACTION"
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
		"data":    res,
		"meta": gin.H{
			"server_time": time.Now().UTC().Format(time.RFC3339),
		},
	})
}

// HandleListLedgerEntries processes GET /api/v1/branch/ledger-entries.
func (h *BranchHandler) HandleListLedgerEntries(c *gin.Context) {
	entries, err := h.ledgerSvc.ListRecentLedgerEntries(c.Request.Context(), 50)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":        "INTERNAL_LEDGER_ERROR",
				"message":     "Failed to query ledger entries",
				"status_code": http.StatusInternalServerError,
			},
		})
		return
	}
	if entries == nil {
		entries = []ledger.AuditLedgerRow{}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    entries,
	})
}
