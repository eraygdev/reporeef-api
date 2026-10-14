package app

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

// logger — paket genelinde kullanılan slog instance'ı.
// main.go'da InitLogger ile set edilir.
var logger *slog.Logger

// InitLogger — slog'u başlatır.
//   - production: JSON (Loki/Datadog/CloudWatch uyumlu)
//   - development: text (terminal'de okunabilir)
func InitLogger() {
	var handler slog.Handler

	if os.Getenv("ENV") == "production" {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})
	} else {
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})
	}

	logger = slog.New(handler)
	slog.SetDefault(logger)
}

// generateRequestID — 16 karakterlik hex (8 byte random).
// crypto/rand kullanır → tahmin edilemez.
func generateRequestID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "unknown"
	}
	return hex.EncodeToString(b)
}

// RequestLogger — her isteği baştan sona loglar.
// Request'e request_id atar, X-Request-ID header'ı olarak geri döner.
// Seviye: 5xx → error, 4xx → warn, diğer → info.
func RequestLogger() gin.HandlerFunc {
	// Healthcheck ve benzeri gürültülü path'ler log'lanmaz.
	skipPaths := map[string]bool{
		"/ping": true,
	}

	return func(c *gin.Context) {
		if skipPaths[c.Request.URL.Path] {
			c.Next()
			return
		}

		start := time.Now()
		reqID := generateRequestID()

		c.Set("request_id", reqID)
		c.Header("X-Request-ID", reqID)

		c.Next()

		duration := time.Since(start)
		status := c.Writer.Status()

		level := slog.LevelInfo
		switch {
		case status >= 500:
			level = slog.LevelError
		case status >= 400:
			level = slog.LevelWarn
		}

		attrs := []any{
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", status,
			"duration_ms", duration.Milliseconds(),
			"request_id", reqID,
			"ip", c.ClientIP(),
		}

		if uid := c.GetInt("user_id"); uid > 0 {
			attrs = append(attrs, "user_id", uid)
		}

		logger.Log(c.Request.Context(), level, "request", attrs...)
	}
}

// LogError — handler içinden structured error loglamak için kısayol.
// request_id, method, path otomatik eklenir.
func LogError(c *gin.Context, msg string, err error) {
	logger.Error(msg,
		"request_id", c.GetString("request_id"),
		"method", c.Request.Method,
		"path", c.Request.URL.Path,
		"err", err,
	)
}
