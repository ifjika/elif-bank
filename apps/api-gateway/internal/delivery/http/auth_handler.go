package http

import (
	"errors"
	"net/http"
	"time"

	"github.com/elif-bank/apps/api-gateway/internal/auth"
	"github.com/gin-gonic/gin"
)

// ==============================================================================
// Delivery HTTP: Banking-Grade Auth Handler (Login, Silent Refresh, Logout)
// Related Documents: docs/IMPLEMENTATION_PLAN.md (4.1.4) & docs/INTEGRATION.md
// ==============================================================================

const RefreshCookieName = "refresh_token"

type AuthHandler struct {
	authService  *auth.AuthService
	cookieSecure bool
}

func NewAuthHandler(authService *auth.AuthService, cookieSecure bool) *AuthHandler {
	return &AuthHandler{
		authService:  authService,
		cookieSecure: cookieSecure,
	}
}

type LoginRequestPayload struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// HandleLogin processes POST /api/v1/auth/login
// Issues Access Token in JSON body and Refresh Token in an HttpOnly SameSite=Strict cookie.
func (h *AuthHandler) HandleLogin(c *gin.Context) {
	var req LoginRequestPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":        "INVALID_PAYLOAD",
				"message":     "Username and password are required",
				"status_code": http.StatusBadRequest,
			},
		})
		return
	}

	res, err := h.authService.Login(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrInvalidCredentials), errors.Is(err, auth.ErrEmptyCredentials):
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error": gin.H{
					"code":        "INVALID_CREDENTIALS",
					"message":     "Invalid username or password",
					"status_code": http.StatusUnauthorized,
				},
			})
		case errors.Is(err, auth.ErrUserSuspended):
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"error": gin.H{
					"code":        "USER_SUSPENDED",
					"message":     "Account is suspended. Contact branch administrator.",
					"status_code": http.StatusForbidden,
				},
			})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error": gin.H{
					"code":        "INTERNAL_AUTH_ERROR",
					"message":     "Authentication service error",
					"status_code": http.StatusInternalServerError,
				},
			})
		}
		return
	}

	// Banking standard: store Refresh Token strictly in HttpOnly, SameSite=Strict cookie
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     RefreshCookieName,
		Value:    res.RefreshToken,
		Path:     "/api/v1/auth",
		MaxAge:   int(auth.RefreshTokenTTL.Seconds()),
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteStrictMode,
	})

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"access_token": res.AccessToken,
			"user":         res.User,
		},
		"meta": gin.H{
			"server_time": time.Now().UTC().Format(time.RFC3339),
		},
	})
}

// HandleRefresh processes POST /api/v1/auth/refresh
// Reads HttpOnly refresh cookie to silently issue a new in-memory Access Token.
func (h *AuthHandler) HandleRefresh(c *gin.Context) {
	refreshToken, err := c.Cookie(RefreshCookieName)
	if err != nil || refreshToken == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error": gin.H{
				"code":        "MISSING_REFRESH_TOKEN",
				"message":     "No active session refresh cookie found",
				"status_code": http.StatusUnauthorized,
			},
		})
		return
	}

	res, err := h.authService.Refresh(c.Request.Context(), refreshToken)
	if err != nil {
		if errors.Is(err, auth.ErrUserSuspended) {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"error": gin.H{
					"code":        "USER_SUSPENDED",
					"message":     "Account is suspended",
					"status_code": http.StatusForbidden,
				},
			})
			return
		}
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error": gin.H{
				"code":        "INVALID_REFRESH_TOKEN",
				"message":     "Session expired or revoked. Please sign in again.",
				"status_code": http.StatusUnauthorized,
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"access_token": res.AccessToken,
			"user":         res.User,
		},
	})
}

// HandleLogout processes POST /api/v1/auth/logout
// Revokes the stateful refresh token and clears the HttpOnly cookie.
func (h *AuthHandler) HandleLogout(c *gin.Context) {
	if refreshToken, err := c.Cookie(RefreshCookieName); err == nil && refreshToken != "" {
		_ = h.authService.Logout(c.Request.Context(), refreshToken)
	}

	http.SetCookie(c.Writer, &http.Cookie{
		Name:     RefreshCookieName,
		Value:    "",
		Path:     "/api/v1/auth",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteStrictMode,
	})

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Signed out cleanly",
	})
}
