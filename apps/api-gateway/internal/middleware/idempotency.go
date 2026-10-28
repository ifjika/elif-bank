package middleware

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// ==============================================================================
// Middleware: Idempotency Protection via Redis Atomic SETNX
// Related Documents: docs/SECURITY_GUIDE.md (Section 2.2) & docs/INTEGRATION.md
// Purpose: Prevent double debits from network replay attacks or retry storms
// ==============================================================================

const (
	HeaderIdempotencyKey = "X-Idempotency-Key"
	IdempotencyTTL       = 24 * time.Hour

	StateProcessing = "PROCESSING"
	StateSuccess    = "SUCCESS"
)

// responseWriterInterceptor is an http.ResponseWriter wrapper that intercepts JSON responses
// prior to client delivery so a copy can be stored into Redis cache.
type responseWriterInterceptor struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *responseWriterInterceptor) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// IdempotencyMiddleware validates and locks X-Idempotency-Key using Redis.
func IdempotencyMiddleware(redisClient *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Only protect state-mutating requests (POST, PUT, PATCH, DELETE)
		if c.Request.Method == http.MethodGet || c.Request.Method == http.MethodHead {
			c.Next()
			return
		}

		idempotencyKey := strings.TrimSpace(c.GetHeader(HeaderIdempotencyKey))
		if idempotencyKey == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error": gin.H{
					"code":        "MISSING_IDEMPOTENCY_KEY",
					"message":     "Header 'X-Idempotency-Key' is required for financial transactions",
					"status_code": http.StatusBadRequest,
				},
			})
			c.Abort()
			return
		}

		redisKey := fmt.Sprintf("idempotency:%s", idempotencyKey)
		ctx := c.Request.Context()

		// 1. Atomic SETNX Operation: Set if Not eXists
		// docs/SECURITY_GUIDE.md Section 2.2
		acquired, err := redisClient.SetNX(ctx, redisKey, StateProcessing, IdempotencyTTL).Result()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "CACHE_FAILURE",
					"message": "Failed to communicate with idempotency cache service",
				},
			})
			c.Abort()
			return
		}

		// 2. If key already exists (retransmitted request)
		if !acquired {
			cachedVal, err := redisClient.Get(ctx, redisKey).Result()
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read cached idempotency state"})
				c.Abort()
				return
			}

			// Case A: Still being processed by another worker
			if cachedVal == StateProcessing {
				c.JSON(http.StatusConflict, gin.H{
					"success": false,
					"error": gin.H{
						"code":        "TRANSACTION_IN_PROGRESS",
						"message":     "Transaction with this idempotency key is currently processing. Please wait.",
						"status_code": http.StatusConflict,
					},
				})
				c.Abort()
				return
			}

			// Case B: Transaction already succeeded -> Return cached response
			// Downstream handlers are bypassed entirely; customer balances are protected from duplicate mutations.
			c.Header("X-Cache-Lookup", "HIT-IDEMPOTENT")
			c.Data(http.StatusOK, "application/json; charset=utf-8", []byte(cachedVal))
			c.Abort()
			return
		}

		// 3. Key acquired (First request)
		// Attach response interceptor to capture handler output
		buffer := bytes.NewBufferString("")
		interceptor := &responseWriterInterceptor{
			ResponseWriter: c.Writer,
			body:           buffer,
		}
		c.Writer = interceptor

		// Proceed to downstream handler (Core Ledger usecase)
		c.Next()

		// 4. Post-execution: On success (HTTP 2xx), cache response in Redis
		if c.Writer.Status() >= 200 && c.Writer.Status() < 300 {
			_ = redisClient.Set(context.Background(), redisKey, buffer.String(), IdempotencyTTL).Err()
		} else {
			// On failure (e.g., insufficient funds / 422), delete key so client can retry
			_ = redisClient.Del(context.Background(), redisKey).Err()
		}
	}
}
