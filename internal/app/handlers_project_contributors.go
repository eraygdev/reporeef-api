package app

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// handleProjectContributors — projenin tüm katkıcılarını (owner dahil) döner.
// Sadece owner veya approved contributor erişebilir.
func handleProjectContributors(c *gin.Context) {
	userID := c.GetInt("user_id")
	ctx := c.Request.Context()

	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_project_id"})
		return
	}

	// Projeyi ve owner'ı al
	var (
		authorID          int
		whatsappInviteURL *string
	)
	err = queryRowCB(ctx, `
		SELECT author_id, whatsapp_invite_url FROM projects WHERE id = $1
	`, projectID).Scan(&authorID, &whatsappInviteURL)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "project_not_found"})
		return
	}

	// Yetki: owner veya approved contributor
	if authorID != userID {
		var count int
		err = queryRowCB(ctx, `
			SELECT COUNT(*) FROM project_contributors
			WHERE project_id = $1 AND user_id = $2 AND status = 'approved'
		`, projectID, userID).Scan(&count)
		if err != nil {
			serverError(c, err, "")
			return
		}
		if count == 0 {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
	}

	// Owner + approved contributor'ları çek
	rows, err := queryCB(ctx, `
		SELECT
			u.id, u.username, COALESCE(u.avatar_url, '') AS avatar_url,
			COALESCE(u.bio, '') AS bio, u.last_seen_at,
			CASE WHEN p.author_id = u.id THEN 'owner' ELSE 'contributor' END AS role,
			pc.approved_at
		FROM users u
		JOIN projects p ON p.id = $1
		LEFT JOIN project_contributors pc
			ON pc.project_id = p.id AND pc.user_id = u.id AND pc.status = 'approved'
		WHERE u.id = p.author_id OR pc.id IS NOT NULL
		ORDER BY
			CASE WHEN u.id = p.author_id THEN 0 ELSE 1 END,
			pc.approved_at ASC
	`, projectID)
	if err != nil {
		serverError(c, err, "")
		return
	}
	defer rows.Close()

	type contrib struct {
		UserID     int        `json:"user_id"`
		Username   string     `json:"username"`
		AvatarURL  string     `json:"avatar_url"`
		Bio        string     `json:"bio"`
		LastSeenAt *time.Time `json:"last_seen_at"`
		Role       string     `json:"role"`
	}

	contributors := []contrib{}
	for rows.Next() {
		var (
			ct         contrib
			approvedAt *time.Time
		)
		if err := rows.Scan(&ct.UserID, &ct.Username, &ct.AvatarURL, &ct.Bio, &ct.LastSeenAt, &ct.Role, &approvedAt); err != nil {
			serverError(c, err, "")
			return
		}
		contributors = append(contributors, ct)
	}
	if err := rows.Err(); err != nil {
		serverError(c, err, "")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"projectId":         projectID,
		"whatsappInviteUrl": whatsappInviteURL,
		"contributors":      contributors,
	})
}

// handleUpdateProjectSettings — proje ayarlarını günceller. Şimdilik sadece WhatsApp.
// Sadece owner.
func handleUpdateProjectSettings(c *gin.Context) {
	userID := c.GetInt("user_id")
	ctx := c.Request.Context()

	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_project_id"})
		return
	}

	var authorID int
	err = queryRowCB(ctx, `SELECT author_id FROM projects WHERE id = $1`, projectID).Scan(&authorID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "project_not_found"})
		return
	}
	if authorID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	var input struct {
		WhatsappInviteURL string `json:"whatsappInviteUrl"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_data"})
		return
	}

	url := strings.TrimSpace(input.WhatsappInviteURL)
	if url != "" {
		if len(url) > 200 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "whatsapp_url_too_long"})
			return
		}
		if !strings.HasPrefix(url, "https://chat.whatsapp.com/") &&
			!strings.HasPrefix(url, "https://wa.me/") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_whatsapp_url"})
			return
		}
	}

	var val interface{} = nil
	if url != "" {
		val = url
	}

	_, err = execCB(ctx, `
		UPDATE projects SET whatsapp_invite_url = $1 WHERE id = $2
	`, val, projectID)
	if err != nil {
		serverError(c, err, "")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"whatsappInviteUrl": val,
		"message":           "settings_updated",
	})
}
