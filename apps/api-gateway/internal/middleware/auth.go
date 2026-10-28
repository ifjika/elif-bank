package middleware

import (
	"net/http"
	"strings"

	"github.com/elif-bank/apps/api-gateway/internal/auth"
	"github.com/gin-gonic/gin"
)

// ==============================================================================
// Middleware: JWT Authentication & Role-Based Access Control (RBAC)
// Related Documents: docs/SECURITY_GUIDE.md (Section 1) & docs/INTEGRATION.md (Section 2.1)
// ==============================================================================

// AuthMiddleware extracts the Bearer token from the Authorization header and validates it.
func AuthMiddleware(tokenManager *auth.TokenManager, allowedRoles ...auth.UserRole) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error": gin.H{
					"code":        "MISSING_AUTH_HEADER",
					"message":     "Header 'Authorization: Bearer <token>' is required",
					"status_code": http.StatusUnauthorized,
				},
			})
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error": gin.H{
					"code":        "INVALID_AUTH_FORMAT",
					"message":     "Authentication token must use the Bearer scheme",
					"status_code": http.StatusUnauthorized,
				},
			})
			c.Abort()
			return
		}

		tokenString := strings.TrimSpace(parts[1])
		claims, err := tokenManager.VerifyToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error": gin.H{
					"code":        "INVALID_OR_EXPIRED_TOKEN",
					"message":     err.Error(),
					"status_code": http.StatusUnauthorized,
				},
			})
			c.Abort()
			return
		}

		// Role-Based Access Control (RBAC) validation
		if len(allowedRoles) > 0 {
			roleAllowed := false
			for _, role := range allowedRoles {
				if claims.Role == role {
					roleAllowed = true
					break
				}
			}

			if !roleAllowed {
				c.JSON(http.StatusForbidden, gin.H{
					"success": false,
					"error": gin.H{
						"code":        "FORBIDDEN_ROLE",
						"message":     "Your role is not authorized to access this resource",
						"status_code": http.StatusForbidden,
					},
				})
				c.Abort()
				return
			}
		}

		// Store claims into Gin context for downstream handlers
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)
		c.Set("branch_id", claims.BranchID)

		c.Next()
	}
}
