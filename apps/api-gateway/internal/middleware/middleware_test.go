package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/elif-bank/apps/api-gateway/internal/auth"
	"github.com/gin-gonic/gin"
)

// ==============================================================================
// Unit Test: Auth & RBAC Middleware
// Dokumen Terkait: docs/TESTING_GUIDE.md & docs/SECURITY_GUIDE.md
// ==============================================================================

func init() {
	// Set Gin mode ke TestMode agar tidak mencetak log berisik saat unit test
	gin.SetMode(gin.TestMode)
}

func TestAuthMiddleware_ValidToken(t *testing.T) {
	tokenManager := auth.NewTokenManager("test-secret-key-at-least-32-bytes-long", 1*time.Hour)
	validToken, err := tokenManager.GenerateToken("usr_123", "nasabah_ani", auth.RoleCustomer, "")
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	router := gin.New()
	router.Use(AuthMiddleware(tokenManager, auth.RoleCustomer))
	router.GET("/protected", func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		c.JSON(http.StatusOK, gin.H{"user_id": userID})
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+validToken)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestAuthMiddleware_MissingHeader(t *testing.T) {
	tokenManager := auth.NewTokenManager("test-secret-key-at-least-32-bytes-long", 1*time.Hour)

	router := gin.New()
	router.Use(AuthMiddleware(tokenManager))
	router.GET("/protected", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401 Unauthorized, got %d", w.Code)
	}
}

func TestAuthMiddleware_ForbiddenRole(t *testing.T) {
	tokenManager := auth.NewTokenManager("test-secret-key-at-least-32-bytes-long", 1*time.Hour)
	// Nasabah biasa mencoba mengakses endpoint khusus SUPERVISOR
	customerToken, _ := tokenManager.GenerateToken("usr_456", "nasabah_budi", auth.RoleCustomer, "")

	router := gin.New()
	router.Use(AuthMiddleware(tokenManager, auth.RoleSupervisor)) // Hanya izinkan SUPERVISOR
	router.GET("/supervisor-only", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/supervisor-only", nil)
	req.Header.Set("Authorization", "Bearer "+customerToken)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status 403 Forbidden for customer accessing supervisor route, got %d", w.Code)
	}
}
