package app

import (
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"runtime/debug"
	"strings"

	"github.com/gin-gonic/gin"
)

// isBrokenPipe — client bağlantıyı kestiğinde oluşan hataları tespit eder.
// Bu durumda log basmak gereksiz gürültü, 500 dönmek de anlamsız
// (zaten yazacak bir bağlantı yok).
func isBrokenPipe(err interface{}) bool {
	ne, ok := err.(*net.OpError)
	if !ok {
		return false
	}
	se, ok := ne.Err.(*os.SyscallError)
	if !ok {
		return false
	}
	msg := strings.ToLower(se.Error())
	return strings.Contains(msg, "broken pipe") ||
		strings.Contains(msg, "connection reset by peer")
}

// Recovery — gin.Recovery()'nin slog uyumlu versiyonu.
//
// Neden: gin.Recovery() panik durumunda düz metin stack trace basar,
// JSON log formatı bozulur (Loki/Datadog parse edemez). Bu middleware
// aynı işi structured olarak yapar.
//
// Sentry entegrasyonu: sentrygin.New(Repanic:true) middleware'i bu
// middleware'den ÖNCE çalışır, panic'i Sentry'e gönderir ve tekrar
// panic eder. Bu middleware onu yakalar ve 500 döner. Yani Sentry
// event'i bizim log'umuzdan bağımsız gider.
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Client disconnect — sessizce geç
				if isBrokenPipe(err) {
					c.Abort()
					return
				}

				slog.Error("panic recovered",
					"request_id", c.GetString("request_id"),
					"method", c.Request.Method,
					"path", c.Request.URL.Path,
					"err", fmt.Sprintf("%v", err),
					"stack", string(debug.Stack()),
				)

				c.AbortWithStatus(http.StatusInternalServerError)
			}
		}()
		c.Next()
	}
}
