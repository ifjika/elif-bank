package http

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/elif-bank/apps/api-gateway/internal/auth"
	"github.com/elif-bank/apps/api-gateway/internal/repository"
	"github.com/gin-gonic/gin"
)

func setupAuthRouter(t *testing.T) *gin.Engine {
	t.Helper()
	userRepo := repository.NewInMemoryUserRepository(false)
	refreshStore := repository.NewInMemoryRefreshTokenStore()

	hash, _ := auth.HashPassword("BankElif2026!")
	userRepo.AddUser(&auth.User{
		ID:           "usr_01",
		Username:     "teller_budi",
		PasswordHash: hash,
		FullName:     "Budi Santoso",
		Role:         auth.RoleTeller,
		BranchID:     "branch-01",
		Status:       auth.UserStatusActive,
	})
	userRepo.AddUser(&auth.User{
		ID:           "usr_02",
		Username:     "suspended_op",
		PasswordHash: hash,
		FullName:     "Suspended Op",
		Role:         auth.RoleTeller,
		BranchID:     "branch-01",
		Status:       auth.UserStatusSuspended,
	})

	accessMgr := auth.NewTokenManager("access-secret-key-32-bytes-minimum!", 15*time.Minute)
	refreshMgr := auth.NewTokenManager("refresh-secret-key-32-bytes-minimum", 7*24*time.Hour)
	svc := auth.NewAuthService(userRepo, refreshStore, accessMgr, refreshMgr)

	authHandler := NewAuthHandler(svc, false)

	router := gin.New()
	router.POST("/api/v1/auth/login", authHandler.HandleLogin)
	router.POST("/api/v1/auth/refresh", authHandler.HandleRefresh)
	router.POST("/api/v1/auth/logout", authHandler.HandleLogout)
	return router
}

func TestAuthHandler_Login_SetsHttpOnlyCookie(t *testing.T) {
	// 1. ARRANGE
	router := setupAuthRouter(t)
	body := `{"username":"teller_budi","password":"BankElif2026!"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// 2. ACT
	router.ServeHTTP(w, req)

	// 3. ASSERT
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d. Body: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			AccessToken  string     `json:"access_token"`
			RefreshToken string     `json:"refresh_token"`
			User         *auth.User `json:"user"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode json: %v", err)
	}
	if !resp.Success || resp.Data.AccessToken == "" {
		t.Fatal("expected success true and non-empty access_token")
	}
	if resp.Data.RefreshToken != "" {
		t.Fatal("refresh_token must NEVER be leaked in JSON response body")
	}

	// Verify HttpOnly & SameSite=Strict cookie was set
	cookies := w.Result().Cookies()
	var refreshCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == RefreshCookieName {
			refreshCookie = c
			break
		}
	}
	if refreshCookie == nil {
		t.Fatal("expected refresh_token cookie to be set")
	}
	if !refreshCookie.HttpOnly {
		t.Error("refresh_token cookie MUST be HttpOnly")
	}
	if refreshCookie.SameSite != http.SameSiteStrictMode {
		t.Errorf("expected SameSiteStrictMode, got %v", refreshCookie.SameSite)
	}
}

func TestAuthHandler_Login_WrongPassword(t *testing.T) {
	// 1. ARRANGE
	router := setupAuthRouter(t)
	body := `{"username":"teller_budi","password":"WrongPassword123!"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// 2. ACT
	router.ServeHTTP(w, req)

	// 3. ASSERT
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized, got %d", w.Code)
	}
}

func TestAuthHandler_Login_SuspendedUser(t *testing.T) {
	// 1. ARRANGE
	router := setupAuthRouter(t)
	body := `{"username":"suspended_op","password":"BankElif2026!"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// 2. ACT
	router.ServeHTTP(w, req)

	// 3. ASSERT
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden, got %d", w.Code)
	}
}

func TestAuthHandler_SilentRefresh_And_Logout(t *testing.T) {
	// 1. ARRANGE
	router := setupAuthRouter(t)
	loginReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(`{"username":"teller_budi","password":"BankElif2026!"}`))
	loginReq.Header.Set("Content-Type", "application/json")
	loginRes := httptest.NewRecorder()
	router.ServeHTTP(loginRes, loginReq)

	refreshCookie := loginRes.Result().Cookies()[0]

	// 2. ACT - Call /refresh with cookie
	refreshReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
	refreshReq.AddCookie(refreshCookie)
	refreshW := httptest.NewRecorder()
	router.ServeHTTP(refreshW, refreshReq)

	// 3. ASSERT - Refresh succeeds
	if refreshW.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on refresh, got %d", refreshW.Code)
	}

	// 4. ACT - Call /logout with cookie
	logoutReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	logoutReq.AddCookie(refreshCookie)
	logoutW := httptest.NewRecorder()
	router.ServeHTTP(logoutW, logoutReq)

	if logoutW.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on logout, got %d", logoutW.Code)
	}

	// 5. ASSERT - Subsequent refresh with revoked cookie fails with 401
	retryReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
	retryReq.AddCookie(refreshCookie)
	retryW := httptest.NewRecorder()
	router.ServeHTTP(retryW, retryReq)

	if retryW.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized after logout revocation, got %d", retryW.Code)
	}
}
