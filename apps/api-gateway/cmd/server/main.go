package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/elif-bank/apps/api-gateway/internal/auth"
	deliveryHttp "github.com/elif-bank/apps/api-gateway/internal/delivery/http"
	"github.com/gin-gonic/gin"
)

// Mock implementation of TransferService for standalone Gateway execution
type dummyTransferService struct{}

func (s *dummyTransferService) Execute(ctx context.Context, cmd deliveryHttp.TransferCommand) (*deliveryHttp.TransferResult, error) {
	return &deliveryHttp.TransferResult{
		TransactionID: fmt.Sprintf("tx_%d", time.Now().UnixNano()),
		ReferenceID:   cmd.ReferenceID,
		Amount:        cmd.Amount,
		SourceBalance: 9500000,
		DestBalance:   5500000,
		Status:        "POSTED",
		CreatedAt:     time.Now().UTC(),
	}, nil
}

// Mock implementation of AccountService for standalone Gateway execution
type dummyAccountService struct{}

func (s *dummyAccountService) GetAccount(ctx context.Context, id string) (*deliveryHttp.AccountInfo, error) {
	return &deliveryHttp.AccountInfo{
		ID:               id,
		AccountNumber:    "1001000012",
		Currency:         "IDR",
		Balance:          10000000,
		AvailableBalance: 10000000,
		HoldBalance:      0,
		Status:           "ACTIVE",
	}, nil
}

func main() {
	port := os.Getenv("GATEWAY_HTTP_PORT")
	if port == "" {
		port = "8080"
	}

	secret := os.Getenv("JWT_ACCESS_SECRET")
	if secret == "" {
		secret = "aethelcore-default-super-secret-key-32b"
	}

	tokenManager := auth.NewTokenManager(secret, 15*time.Minute)
	transferSvc := &dummyTransferService{}
	accountSvc := &dummyAccountService{}

	handler := deliveryHttp.NewHandler(transferSvc, accountSvc, tokenManager)

	router := gin.Default()

	// CORS Middleware for UI development
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Idempotency-Key")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	})

	// Health Check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":      "HEALTHY",
			"service":     "elif-core-bank-api-gateway",
			"server_time": time.Now().UTC().Format(time.RFC3339),
		})
	})

	// Auth Mock Endpoint
	router.POST("/api/v1/auth/login", func(c *gin.Context) {
		var req struct {
			Username string        `json:"username" binding:"required"`
			Role     auth.UserRole `json:"role"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if req.Role == "" {
			req.Role = auth.RoleTeller
		}

		token, err := tokenManager.GenerateToken("usr_"+req.Username, req.Username, req.Role, "branch-001")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": gin.H{
				"access_token": token,
				"user": gin.H{
					"username":  req.Username,
					"role":      req.Role,
					"branch_id": "branch-001",
				},
			},
		})
	})

	// Transaction & Account API Endpoints
	api := router.Group("/api/v1")
	{
		api.POST("/transfers", handler.HandleTransfer)
		api.GET("/accounts/:id/balance", handler.HandleGetBalance)
	}

	addr := ":" + port
	log.Printf("======================================================")
	log.Printf("🚀 AethelCore API Gateway is running on http://localhost%s", addr)
	log.Printf("📌 Health Check: http://localhost%s/health", addr)
	log.Printf("======================================================")

	if err := router.Run(addr); err != nil {
		log.Fatalf("Failed to run HTTP server: %v", err)
	}
}
