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
// Dokumen Terkait: docs/SECURITY_GUIDE.md (Seksi 2.2) & docs/INTEGRATION.md
// Fungsi: Mencegah debit ganda akibat network replay attack atau retry storm
// ==============================================================================

const (
	HeaderIdempotencyKey = "X-Idempotency-Key"
	IdempotencyTTL       = 24 * time.Hour

	StateProcessing = "PROCESSING"
	StateSuccess    = "SUCCESS"
)

// responseWriterInterceptor adalah wrapper http.ResponseWriter untuk menyadap respons JSON
// sebelum dikirimkan ke klien, agar salinannya bisa disimpan ke cache Redis.
type responseWriterInterceptor struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *responseWriterInterceptor) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// IdempotencyMiddleware memvalidasi dan mengunci X-Idempotency-Key menggunakan Redis.
func IdempotencyMiddleware(redisClient *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Hanya terapkan proteksi pada request yang mengubah data (POST, PUT, PATCH, DELETE)
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
					"message":     "Header 'X-Idempotency-Key' wajib disertakan untuk transaksi finansial",
					"status_code": http.StatusBadRequest,
				},
			})
			c.Abort()
			return
		}

		redisKey := fmt.Sprintf("idempotency:%s", idempotencyKey)
		ctx := c.Request.Context()

		// 1. Operasi Atomik SETNX: Hanya buat jika belum ada (Set if Not eXists)
		// docs/SECURITY_GUIDE.md Seksi 2.2
		acquired, err := redisClient.SetNX(ctx, redisKey, StateProcessing, IdempotencyTTL).Result()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "CACHE_FAILURE",
					"message": "Gagal menghubungi layanan cache idempotensi",
				},
			})
			c.Abort()
			return
		}

		// 2. Jika Kunci Sudah Pernah Ada (Bukan Request Pertama)
		if !acquired {
			cachedVal, err := redisClient.Get(ctx, redisKey).Result()
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read cached idempotency state"})
				c.Abort()
				return
			}

			// Kasus A: Sedang diproses oleh goroutine/worker lain
			if cachedVal == StateProcessing {
				c.JSON(http.StatusConflict, gin.H{
					"success": false,
					"error": gin.H{
						"code":        "TRANSACTION_IN_PROGRESS",
						"message":     "Transaksi dengan kunci idempotensi ini sedang diproses. Mohon tunggu.",
						"status_code": http.StatusConflict,
					},
				})
				c.Abort()
				return
			}

			// Kasus B: Transaksi sudah berhasil sebelumnya -> Kembalikan salinan respons ter-cache
			// Handler transfer tidak disentuh sama sekali! Saldo nasabah 100% aman dari mutasi ganda.
			c.Header("X-Cache-Lookup", "HIT-IDEMPOTENT")
			c.Data(http.StatusOK, "application/json; charset=utf-8", []byte(cachedVal))
			c.Abort()
			return
		}

		// 3. Jika Kunci Berhasil Didapat (Request Pertama)
		// Pasang penyadap respons untuk merekam output handler
		buffer := bytes.NewBufferString("")
		interceptor := &responseWriterInterceptor{
			ResponseWriter: c.Writer,
			body:           buffer,
		}
		c.Writer = interceptor

		// Lanjutkan eksekusi ke handler berikutnya (Core Ledger usecase)
		c.Next()

		// 4. Setelah Handler Selesai: Jika sukses (HTTP 2xx), simpan output respons ke Redis
		if c.Writer.Status() >= 200 && c.Writer.Status() < 300 {
			_ = redisClient.Set(context.Background(), redisKey, buffer.String(), IdempotencyTTL).Err()
		} else {
			// Jika transaksi gagal (misal: saldo tidak cukup / 422), hapus kunci agar klien bisa mencoba lagi
			_ = redisClient.Del(context.Background(), redisKey).Err()
		}
	}
}
