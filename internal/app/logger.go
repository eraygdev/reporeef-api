package app

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// logger — paket genelinde kullanılan slog instance'ı.
// main.go'da InitLogger ile set edilir.
var logger *slog.Logger

// skipPaths — log ve rate limit için atlanacak path'ler.
// Healthcheck gürültüsünü engeller.
var skipPaths = map[string]bool{
	"/ping": true,
}

// InitLogger — slog'u başlatır.
//   - production: JSON (Loki/Datadog/CloudWatch uyumlu)
//   - development: text (terminal'de okunabilir)
func InitLogger() {
	// Her zaman JSON handler kullan — dev ve prod format aynı olsun.
	// Terminal'de insan-okunur görünüm için `make logs` jq kullanır.
	// Dosyaya (docker logs) JSON olarak yazılır → Loki/Datadog'a hazır.
	//
	// Seviye: prod=Info, dev=Debug (repo cache hit/miss gibi satırlar görünür).
	level := slog.LevelInfo
	if os.Getenv("ENV") != "production" {
		level = slog.LevelDebug
	}

	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	})

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

// clientIP — gerçek kullanıcı IP'sini döner.
//
// Topoloji: Cloudflare Edge → cloudflared (Docker) → backend
// c.ClientIP() bu topolojide cloudflared container IP'sini (172.x.x.x)
// döner, çünkü backend'in doğrudan peer'ı odur. Cloudflare edge,
// gerçek kullanıcı IP'sini CF-Connecting-IP header'ına set eder;
// cloudflared bunu backend'e forward eder.
//
// Güvenlik: Prod'da backend'e sadece cloudflared erişebilir (docker-compose'da
// 127.0.0.1:8080:8080), bu yüzden header spoof edilemez. Dev'de backend
// doğrudan localhost'tan erişilebilir, spoof'a açık — bu yüzden
// dev'de header yok sayılır, c.ClientIP() kullanılır.
func clientIP(c *gin.Context) string {
	if os.Getenv("ENV") == "production" {
		if ip := strings.TrimSpace(c.GetHeader("CF-Connecting-IP")); ip != "" {
			return ip
		}
	}
	return c.ClientIP()
}

// RequestLogger — her isteği baştan sona loglar.
// Request'e request_id atar, X-Request-ID header'ı olarak geri döner.
// Seviye: 5xx → error, 4xx → warn, diğer → info.
func RequestLogger() gin.HandlerFunc {
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
			"ip", clientIP(c),
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
