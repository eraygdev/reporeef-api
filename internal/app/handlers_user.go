package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

// handlePing — liveness probe. DB'ye vurmaz, ucuz.
// Rate limit dışı (skipRateLimitPaths). Docker healthcheck + uptime
// monitor buraya bakar. DB kontrolü istiyorsan /ready kullan.
func handlePing(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// handleReady — readiness probe. DB bağlantısını doğrular.
// Rate limit altında — DB'ye vuruyor, flood'a kapalı olmalı.
func handleReady(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	if err := db.Ping(ctx); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "unhealthy",
			"db":     "down",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "healthy",
		"db":     "up",
	})
}

// handleMe token sahibinin bilgilerini ve proje sayısını döner.
func handleMe(c *gin.Context) {
	userID := c.GetInt("user_id")

	var (
		email          string
		username       string
		githubUsername string
		avatarURL      string
		bio            string
		isPremium      bool
		onboarded      bool
	)

	err := db.QueryRow(c.Request.Context(), `
		SELECT email, COALESCE(username, ''), COALESCE(github_username, ''),
		       COALESCE(avatar_url, ''), COALESCE(bio, ''), is_premium, onboarded
		FROM users WHERE id = $1
	`, userID).Scan(&email, &username, &githubUsername, &avatarURL, &bio, &isPremium, &onboarded)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user_not_found"})
		return
	}

	var projectCount int
	_ = db.QueryRow(c.Request.Context(),
		`SELECT COUNT(*) FROM projects WHERE author_id = $1`, userID,
	).Scan(&projectCount)

	c.JSON(http.StatusOK, gin.H{
		"user_id":         userID,
		"email":           email,
		"username":        username,
		"github_username": githubUsername,
		"avatar_url":      avatarURL,
		"bio":             bio,
		"is_premium":      isPremium,
		"onboarded":       onboarded,
		"projectCount":    projectCount,
		"maxProjects":     MaxProjectsPerUser,
	})
}

// handleUpdateMe kullanıcının username ve bio'sunu günceller.
func handleUpdateMe(c *gin.Context) {
	userID := c.GetInt("user_id")
	ctx := c.Request.Context()

	var input struct {
		Username string `json:"username"`
		Bio      string `json:"bio"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_data"})
		return
	}

	cleanUsername, cleanBio, err := validateProfileInput(input.Username, input.Bio)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	input.Username = cleanUsername
	input.Bio = cleanBio

	// Case-insensitive çakışma kontrolü
	var existingID int
	err = db.QueryRow(ctx,
		`SELECT id FROM users WHERE LOWER(username) = LOWER($1) AND id != $2`,
		input.Username, userID,
	).Scan(&existingID)

	if err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "username_taken"})
		return
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		serverError(c, err, "")
		return
	}

	// Mevcut username + son değişiklik zamanını al — cooldown kontrolü için
	var (
		currentUsername string
		lastChangedAt   *time.Time
	)
	err = db.QueryRow(ctx,
		`SELECT username, username_changed_at FROM users WHERE id = $1`,
		userID,
	).Scan(&currentUsername, &lastChangedAt)
	if err != nil {
		serverError(c, err, "")
		return
	}

	// Sadece username gerçekten değişiyorsa cooldown kontrolü.
	// Bu ön kontrol hızlı fail içindir; atomik kontrol UPDATE WHERE'de.
	usernameChanged := currentUsername != input.Username
	if usernameChanged && !canChangeUsername(lastChangedAt, time.Now()) {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "username_change_too_frequent"})
		return
	}

	// onboarded=true — kullanıcı onboarding'i geçti (veya zaten geçmişti)
	// is_premium'u da döndür ki yeni JWT üretebilelim.
	// username_changed_at SADECE gerçekten değiştiyse güncellenir.
	//
	// Cooldown atomik: WHERE'de tekrar kontrol edilir. Eşzamanlı iki istek
	// gelirse biri RowsAffected=0 alır → 429 döner (TOCTOU yok).
	var isPremium bool

	if usernameChanged {
		err = db.QueryRow(ctx, `
			UPDATE users
			SET username = $1,
			    bio = $2,
			    onboarded = true,
			    username_changed_at = NOW()
			WHERE id = $3
			  AND (username_changed_at IS NULL
			       OR username_changed_at < NOW() - INTERVAL '48 hours')
			RETURNING is_premium
		`, input.Username, input.Bio, userID).Scan(&isPremium)

		if errors.Is(err, pgx.ErrNoRows) {
			// Kullanıcı var ama cooldown aktif → 429.
			// (Kullanıcı yok senaryosu yukarıdaki SELECT'te yakalandı.)
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "username_change_too_frequent"})
			return
		}
	} else {
		err = db.QueryRow(ctx, `
			UPDATE users
			SET username = $1,
			    bio = $2,
			    onboarded = true
			WHERE id = $3
			RETURNING is_premium
		`, input.Username, input.Bio, userID).Scan(&isPremium)
	}

	if err != nil {
		if isUniqueViolation(err) {
			c.JSON(http.StatusConflict, gin.H{"error": "username_taken"})
			return
		}
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "user_not_found"})
			return
		}
		serverError(c, err, "")
		return
	}

	auditLog(c, userID, "update", "user", userID)

	// Yeni JWT — onboarded=true claim'i ile. Frontend bunu localStorage'a yazmalı.
	newToken, err := generateJWT(userID, isPremium, true)
	if err != nil {
		serverError(c, err, "")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "profile_updated",
		"username":  input.Username,
		"bio":       input.Bio,
		"onboarded": true,
		"token":     newToken,
	})
}

// handleGetUserByUsername belirli bir kullanıcının profilini ve projelerini döner.
func handleGetUserByUsername(c *gin.Context) {
	username := c.Param("username")
	ctx := c.Request.Context()

	if err := validateUsernameParam(username); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	limit := MaxProjectsPerPage
	if l := c.Query("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= MaxProjectsPerPage {
			limit = n
		}
	}
	offset := 0
	if o := c.Query("offset"); o != "" {
		if n, err := strconv.Atoi(o); err == nil && n >= 0 && n <= 10000 {
			offset = n
		}
	}

	sortParam := c.DefaultQuery("sort", "newest")

	var (
		userID     int
		dbUsername string
		avatarURL  string
		bio        string
		createdAt  time.Time
	)

	err := db.QueryRow(ctx, `
    SELECT id, username, COALESCE(avatar_url, ''), COALESCE(bio, ''), created_at
    FROM users WHERE LOWER(username) = LOWER($1)
`, username).Scan(&userID, &dbUsername, &avatarURL, &bio, &createdAt)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user_not_found"})
		return
	}

	var totalProjects, totalStars, totalContributors int
	_ = db.QueryRow(ctx, `
		SELECT 
			(SELECT COUNT(*) FROM projects WHERE author_id = $1),
			COALESCE((SELECT SUM((SELECT COUNT(*) FROM project_stars WHERE project_id = p.id)) FROM projects p WHERE p.author_id = $1), 0),
			COALESCE((SELECT COUNT(*) FROM project_contributors pc JOIN projects p ON p.id = pc.project_id WHERE p.author_id = $1 AND pc.status = 'approved'), 0)
	`, userID).Scan(&totalProjects, &totalStars, &totalContributors)

	orderClause := "p.created_at DESC, p.id DESC"
	if sortParam == "popular" {
		orderClause = "stars DESC, p.created_at DESC, p.id DESC"
	}

	query := fmt.Sprintf(`
		SELECT 
			p.id, p.title, p.description,
			COALESCE(p.image_url, '') AS image_url,
			(SELECT COUNT(*) FROM project_stars WHERE project_id = p.id) AS stars,
			(SELECT COUNT(*) FROM project_contributors WHERE project_id = p.id AND status = 'approved') AS contributors,
			p.created_at,
			COALESCE((SELECT ARRAY_AGG(c.name ORDER BY c.name) FROM project_categories pc JOIN categories c ON c.id = pc.category_id WHERE pc.project_id = p.id), ARRAY[]::varchar[]) AS categories,
			COALESCE((SELECT ARRAY_AGG(pc.category_id ORDER BY pc.category_id) FROM project_categories pc WHERE pc.project_id = p.id), ARRAY[]::int[]) AS category_ids
		FROM projects p
		WHERE p.author_id = $1
		ORDER BY %s
		LIMIT $2 OFFSET $3
	`, orderClause)

	rows, err := db.Query(ctx, query, userID, limit+1, offset)
	if err != nil {
		serverError(c, err, "")
		return
	}
	defer rows.Close()

	projects := []map[string]interface{}{}
	count := 0

	for rows.Next() {
		var id, stars, contributors int
		var title, description, imageURL string
		var projCreatedAt time.Time
		var categories []string
		var categoryIDs []int

		if err := rows.Scan(&id, &title, &description, &imageURL, &stars, &contributors, &projCreatedAt, &categories, &categoryIDs); err != nil {
			serverError(c, err, "")
			return
		}

		count++
		if count > limit {
			break
		}

		if categories == nil {
			categories = []string{}
		}
		if categoryIDs == nil {
			categoryIDs = []int{}
		}

		projects = append(projects, map[string]interface{}{
			"id":           id,
			"title":        title,
			"description":  description,
			"imageUrl":     imageURL,
			"stars":        stars,
			"contributors": contributors,
			"createdAt":    projCreatedAt,
			"categories":   categories,
			"categoryIds":  categoryIDs,
			"author":       dbUsername,
			"authorAvatar": avatarURL,
		})
	}

	if err := rows.Err(); err != nil {
		serverError(c, err, "")
		return
	}

	hasMore := count > limit

	c.JSON(http.StatusOK, gin.H{
		"user_id":    userID,
		"username":   dbUsername,
		"avatar_url": avatarURL,
		"bio":        bio,
		"created_at": createdAt,
		"stats": gin.H{
			"totalProjects":     totalProjects,
			"totalStars":        totalStars,
			"totalContributors": totalContributors,
		},
		"projects": projects,
		"hasMore":  hasMore,
	})
}

// handleSearchUsers kullanıcı adına göre arama yapar (case-insensitive).
func handleSearchUsers(c *gin.Context) {
	query, err := validateSearchQuery(c.Query("q"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if query == "" {
		c.JSON(http.StatusOK, []map[string]interface{}{})
		return
	}

	limit := MaxUsersPerSearch
	if l := c.Query("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= MaxUsersPerSearch {
			limit = n
		}
	}

	rows, err := db.Query(c.Request.Context(), `
		SELECT id, username, COALESCE(avatar_url, ''), COALESCE(bio, '')
		FROM users
		WHERE LOWER(username) LIKE LOWER($1) ESCAPE '\'
		ORDER BY username ASC
		LIMIT $2
	`, "%"+query+"%", limit)
	if err != nil {
		serverError(c, err, "")
		return
	}
	defer rows.Close()

	users := []map[string]interface{}{}
	for rows.Next() {
		var id int
		var username, avatarURL, bio string

		if err := rows.Scan(&id, &username, &avatarURL, &bio); err != nil {
			serverError(c, err, "")
			return
		}

		if utf8.RuneCountInString(bio) > 60 {
			runes := []rune(bio)
			bio = string(runes[:60]) + "..."
		}

		users = append(users, map[string]interface{}{
			"user_id":    id,
			"username":   username,
			"avatar_url": avatarURL,
			"bio":        bio,
		})
	}

	if err := rows.Err(); err != nil {
		serverError(c, err, "")
		return
	}

	c.JSON(http.StatusOK, users)
}
