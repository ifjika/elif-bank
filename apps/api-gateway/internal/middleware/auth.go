package middleware

import (
	"net/http"
	"strings"

	"github.com/elif-bank/apps/api-gateway/internal/auth"
	"github.com/gin-gonic/gin"
)

// ==============================================================================
// Middleware: JWT Authentication & Role-Based Access Control (RBAC)
// Dokumen Terkait: docs/SECURITY_GUIDE.md (Seksi 1) & docs/INTEGRATION.md (Seksi 2.1)
// ==============================================================================

// AuthMiddleware mengekstrak Bearer token dari header Authorization dan memverifikasinya.
func AuthMiddleware(tokenManager *auth.TokenManager, allowedRoles ...auth.UserRole) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error": gin.H{
					"code":        "MISSING_AUTH_HEADER",
					"message":     "Header 'Authorization: Bearer <token>' wajib disertakan",
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
					"message":     "Format token otentikasi wajib menggunakan skema Bearer",
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

		// Validasi Role-Based Access Control (RBAC) jika ada batasan role
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
						"message":     "Peran Anda tidak memiliki izin untuk mengakses sumber daya ini",
						"status_code": http.StatusForbidden,
					},
				})
				c.Abort()
				return
			}
		}

		// Simpan klaim ke dalam context Gin agar bisa dibaca oleh Handler berikutnya
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)
		c.Set("branch_id", claims.BranchID)

		c.Next()
	}
}
