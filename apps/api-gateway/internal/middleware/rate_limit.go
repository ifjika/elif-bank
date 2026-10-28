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
// Related Documents: docs/SECURITY_GUIDE.md & docs/INTEGRATION.md (Section 1.3)
// Purpose: Throttle API call volume per IP/User to mitigate DDoS & Brute Force
// ==============================================================================

// RateLimitMiddleware restricts the number of requests within a given duration window.
func RateLimitMiddleware(redisClient *redis.Client, maxRequests int64, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Default to client IP (or UserID if authenticated)
		clientIdentifier := c.ClientIP()
		if userID, exists := c.Get("user_id"); exists {
			clientIdentifier = fmt.Sprintf("user:%v", userID)
		}

		key := fmt.Sprintf("rate_limit:%s", clientIdentifier)
		ctx := c.Request.Context()

		// 1. Atomic INCR Operation in Redis (docs/SECURITY_GUIDE.md)
		count, err := redisClient.Incr(ctx, key).Result()
		if err != nil {
			// Fail-open strategy: if Redis is unavailable, do not block legitimate customer traffic
			c.Next()
			return
		}

		// 2. Set window TTL on first hit
		if count == 1 {
			_ = redisClient.Expire(ctx, key, window).Err()
		}

		// RFC standard Rate Limiting headers
		c.Header("X-RateLimit-Limit", strconv.FormatInt(maxRequests, 10))
		remaining := maxRequests - count
		if remaining < 0 {
			remaining = 0
		}
		c.Header("X-RateLimit-Remaining", strconv.FormatInt(remaining, 10))

		// 3. Evaluate threshold
		if count > maxRequests {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"success": false,
				"error": gin.H{
					"code":        "RATE_LIMIT_EXCEEDED",
					"message":     "API request rate limit exceeded. Please try again shortly.",
					"status_code": http.StatusTooManyRequests,
				},
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
