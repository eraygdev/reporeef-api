package app

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
)

// handleListCategories tüm kategorileri alfabetik döner.
//
// 5 dk in-memory cache — kategoriler statik veri. Cache hit olduğunda
// DB'ye hiç gidilmez. Cache miss olduğunda DB'den çekilir ve cache yazılır.
func handleListCategories(c *gin.Context) {
	// Cache hit — direkt dön
	if cached, ok := getCachedCategories(); ok {
		c.JSON(http.StatusOK, cached)
		return
	}

	// Cache miss — DB'den çek (circuit breaker ile sarmalı)
	rows, err := queryCB(c.Request.Context(), `
		SELECT id, name, slug FROM categories ORDER BY name ASC
	`)
	if err != nil {
		serverError(c, err, "")
		return
	}
	defer rows.Close()

	categories := []map[string]interface{}{}
	for rows.Next() {
		var id int
		var name, slug string

		if err := rows.Scan(&id, &name, &slug); err != nil {
			serverError(c, err, "")
			return
		}

		categories = append(categories, map[string]interface{}{
			"id":   id,
			"name": name,
			"slug": slug,
		})
	}

	if err := rows.Err(); err != nil {
		serverError(c, err, "")
		return
	}

	// Cache'e yaz — sonraki 5 dk için
	setCachedCategories(categories)

	c.JSON(http.StatusOK, categories)
}

// validateCategoryIDs verilen ID'lerin hepsinin geçerli olduğunu doğrular.
func validateCategoryIDs(ctx context.Context, categoryIDs []int) bool {
	if len(categoryIDs) == 0 {
		return true
	}
	var count int
	err := db.QueryRow(ctx,
		`SELECT COUNT(*) FROM categories WHERE id = ANY($1)`,
		categoryIDs,
	).Scan(&count)
	if err != nil {
		return false
	}
	return count == len(categoryIDs)
}
