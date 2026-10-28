package middleware

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// ==============================================================================
// Middleware: Token Bucket Rate Limiter via Redis
// Dokumen Terkait: docs/SECURITY_GUIDE.md & docs/INTEGRATION.md (Seksi 1.3)
// Fungsi: Membatasi kuota panggilan API per IP/User untuk mitigasi DDoS & Brute Force
// ==============================================================================

// RateLimitMiddleware membatasi jumlah request per interval waktu tertentu.
func RateLimitMiddleware(redisClient *redis.Client, maxRequests int64, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Gunakan IP klien sebagai identifier default (atau UserID jika sudah terotentikasi)
		clientIdentifier := c.ClientIP()
		if userID, exists := c.Get("user_id"); exists {
			clientIdentifier = fmt.Sprintf("user:%v", userID)
		}

		key := fmt.Sprintf("rate_limit:%s", clientIdentifier)
		ctx := c.Request.Context()

		// 1. Operasi Atomik INCR di Redis (docs/SECURITY_GUIDE.md)
		count, err := redisClient.Incr(ctx, key).Result()
		if err != nil {
			// Fail-open strategy: jika cache Redis down, jangan blokir transaksi nasabah
			c.Next()
			return
		}

		// 2. Jika ini adalah hitungan pertama, pasang TTL jendela waktu
		if count == 1 {
			_ = redisClient.Expire(ctx, key, window).Err()
		}

		// Header standar RFC Rate Limiting
		c.Header("X-RateLimit-Limit", strconv.FormatInt(maxRequests, 10))
		remaining := maxRequests - count
		if remaining < 0 {
			remaining = 0
		}
		c.Header("X-RateLimit-Remaining", strconv.FormatInt(remaining, 10))

		// 3. Evaluasi Ambang Batas (Threshold)
		if count > maxRequests {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"success": false,
				"error": gin.H{
					"code":        "RATE_LIMIT_EXCEEDED",
					"message":     "Ambang batas panggilan API terlampaui. Silakan tunggu beberapa saat.",
					"status_code": http.StatusTooManyRequests,
				},
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
