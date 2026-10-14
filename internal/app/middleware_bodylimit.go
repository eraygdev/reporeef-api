package app

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// maxBodySizeMiddleware — POST/PUT/PATCH request body'lerine üst sınır koyar.
//
// Neden: c.ShouldBindJSON() gövdenin tamamını okur. Saldırgan çok büyük
// Content-Length + yavaş gövde gönderirse memory şişer (slowloris-benzeri).
//
// http.MaxBytesReader, gövdeyi okurken limit aşılırsa bağlantıyı kapatır.
// Handler'da ShouldBindJSON hata döner → 400. Zararsız.
//
// GET/DELETE/PATCH/OPTIONS — body yok veya okunmuyor, sarmaya gerek yok.
func maxBodySizeMiddleware(limit int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Body == nil {
			c.Next()
			return
		}

		switch c.Request.Method {
		case http.MethodPost, http.MethodPut, http.MethodPatch:
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limit)
		}

		c.Next()
	}
}
