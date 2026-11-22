package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/elif-bank/apps/api-gateway/internal/auth"
	deliveryHttp "github.com/elif-bank/apps/api-gateway/internal/delivery/http"
	"github.com/elif-bank/apps/api-gateway/internal/middleware"
	"github.com/elif-bank/apps/api-gateway/internal/repository"
	"github.com/elif-bank/apps/core-ledger/ledger"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

// Mock implementation of TransferService for standalone P2P transfer route
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

// Mock implementation of AccountService for standalone balance query route
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
	// Load .env from monorepo root or current working directory
	_ = godotenv.Load("../../.env")
	_ = godotenv.Load(".env")

	port := os.Getenv("GATEWAY_HTTP_PORT")
	if port == "" {
		port = "8080"
	}

	accessSecret := os.Getenv("JWT_ACCESS_SECRET")
	if accessSecret == "" {
		accessSecret = "elif-core-bank-access-secret-32b!"
	}
	refreshSecret := os.Getenv("JWT_REFRESH_SECRET")
	if refreshSecret == "" {
		refreshSecret = "elif-core-bank-refresh-secret-32b"
	}

	cookieSecure := os.Getenv("COOKIE_SECURE") == "true"

	// Connect to TiDB Cloud Serverless / MySQL 8.0
	maxOpen, _ := strconv.Atoi(os.Getenv("DB_MAX_OPEN_CONNS"))
	maxIdle, _ := strconv.Atoi(os.Getenv("DB_MAX_IDLE_CONNS"))

	dbCfg := repository.DBConfig{
		Host:         os.Getenv("DB_HOST"),
		Port:         os.Getenv("DB_PORT"),
		Database:     os.Getenv("DB_NAME"),
		User:         os.Getenv("DB_USER"),
		Password:     os.Getenv("DB_PASSWORD"),
		SSLMode:      os.Getenv("DB_SSLMODE") == "true",
		MaxOpenConns: maxOpen,
		MaxIdleConns: maxIdle,
	}

	ctx := context.Background()
	db, err := repository.NewMySQLConnection(ctx, dbCfg)
	if err != nil {
		log.Fatalf("❌ TiDB / MySQL connection failed: %v\n👉 Please ensure .env exists at project root with valid DB_HOST, DB_PORT, DB_NAME, DB_USER, DB_PASSWORD, and DB_SSLMODE=true", err)
	}
	defer func() { _ = db.Close() }()
	log.Printf("✅ Connected to TiDB / MySQL (%s:%s/%s, SSL=%v)", dbCfg.Host, dbCfg.Port, dbCfg.Database, dbCfg.SSLMode)

	// Seed initial operator accounts and demo ledger accounts if empty
	if err := repository.SeedDemoUsersIfEmpty(ctx, db); err != nil {
		log.Fatalf("❌ Failed to seed initial users: %v", err)
	}
	if err := ledger.SeedDemoAccountsIfEmpty(ctx, db); err != nil {
		log.Fatalf("❌ Failed to seed initial ledger accounts: %v", err)
	}

	// Initialize dual TokenManagers (15m Access, 7d Refresh)
	accessMgr := auth.NewTokenManager(accessSecret, 15*time.Minute)
	refreshMgr := auth.NewTokenManager(refreshSecret, auth.RefreshTokenTTL)

	// Initialize Repositories & Services
	userRepo := repository.NewMySQLUserRepository(db)
	refreshStore := repository.NewInMemoryRefreshTokenStore()
	authSvc := auth.NewAuthService(userRepo, refreshStore, accessMgr, refreshMgr)
	ledgerSvc := ledger.NewService(db)

	transferSvc := &dummyTransferService{}
	accountSvc := &dummyAccountService{}

	handler := deliveryHttp.NewHandler(transferSvc, accountSvc, accessMgr)
	authHandler := deliveryHttp.NewAuthHandler(authSvc, cookieSecure)
	branchHandler := deliveryHttp.NewBranchHandler(ledgerSvc)

	router := gin.Default()

	// CORS Middleware supporting credentials (HttpOnly cookies) for local Vite dev servers
	router.Use(func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin == "http://localhost:3000" || origin == "http://localhost:5173" {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		}
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
			"database":    "tidb-mysql-connected",
			"server_time": time.Now().UTC().Format(time.RFC3339),
		})
	})

	// Public Auth Endpoints (Banking-Grade HttpOnly Cookie + In-Memory Access Token)
	authGroup := router.Group("/api/v1/auth")
	{
		authGroup.POST("/login", authHandler.HandleLogin)
		authGroup.POST("/refresh", authHandler.HandleRefresh)
		authGroup.POST("/logout", authHandler.HandleLogout)
	}

	// Protected Transaction & Account API Endpoints
	api := router.Group("/api/v1")
	api.Use(middleware.AuthMiddleware(accessMgr))
	{
		api.POST("/transfers", handler.HandleTransfer)
		api.GET("/accounts/:id/balance", handler.HandleGetBalance)

		// Branch Operations (Maker-Checker RBAC Enforced)
		branch := api.Group("/branch")
		{
			branch.POST("/cash-transactions", middleware.AuthMiddleware(accessMgr, auth.RoleTeller, auth.RoleAdmin), branchHandler.HandleCashTransaction)
			branch.GET("/approvals/pending", middleware.AuthMiddleware(accessMgr, auth.RoleSupervisor, auth.RoleAdmin), branchHandler.HandleListPending)
			branch.POST("/approvals/:transaction_id", middleware.AuthMiddleware(accessMgr, auth.RoleSupervisor, auth.RoleAdmin), branchHandler.HandleApproval)
			branch.GET("/ledger-entries", middleware.AuthMiddleware(accessMgr, auth.RoleTeller, auth.RoleSupervisor, auth.RoleAdmin), branchHandler.HandleListLedgerEntries)
		}
	}

	addr := ":" + port
	log.Printf("======================================================")
	log.Printf("🚀 Elif Core Bank API Gateway running on http://localhost%s", addr)
	log.Printf("📌 Health Check: http://localhost%s/health", addr)
	log.Printf("======================================================")

	if err := router.Run(addr); err != nil {
		log.Fatalf("Failed to run HTTP server: %v", err)
	}
}
