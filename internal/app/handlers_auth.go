package app

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

// ---------- GITHUB ----------
func handleGithubLogin(c *gin.Context) {
	state := generateOAuthState()
	if state == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "auth_failed"})
		return
	}
	url := githubOauthConfig.AuthCodeURL(state)
	c.Redirect(http.StatusTemporaryRedirect, url)
}

func handleGithubCallback(c *gin.Context) {
	ctx := c.Request.Context()
	code := c.Query("code")
	state := c.Query("state")
	errParam := c.Query("error")

	// Kullanıcı GitHub authorize ekranında "Cancel" derse:
	// ?error=access_denied&state=... ile döner. State'i tüketip login'e yönlendir.
	if errParam != "" {
		consumeOAuthState(state)
		frontend := os.Getenv("FRONTEND_URL")
		c.Redirect(http.StatusTemporaryRedirect, frontend+"/login?error="+errParam)
		return
	}

	if !consumeOAuthState(state) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_state"})
		return
	}

	token, err := githubOauthConfig.Exchange(ctx, code)
	if err != nil {
		serverError(c, err, "auth_failed")
		return
	}

	client := &http.Client{Timeout: 10 * time.Second}

	req, _ := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/user", nil)
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := client.Do(req)
	if err != nil {
		serverError(c, err, "auth_failed")
		return
	}
	defer func() { _ = resp.Body.Close() }()

	body, _ := io.ReadAll(resp.Body)
	var info map[string]interface{}
	if err := json.Unmarshal(body, &info); err != nil {
		serverError(c, err, "auth_failed")
		return
	}

	login, _ := info["login"].(string)
	avatar, _ := info["avatar_url"].(string)
	bio, _ := info["bio"].(string)

	var githubID int
	if v, ok := info["id"].(float64); ok {
		githubID = int(v)
	}

	if githubID == 0 {
		serverError(c, fmt.Errorf("github_id_missing"), "auth_failed")
		return
	}

	email, _ := info["email"].(string)
	if email == "" {
		emailReq, _ := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/user/emails", nil)
		emailReq.Header.Set("Authorization", "Bearer "+token.AccessToken)
		emailReq.Header.Set("Accept", "application/vnd.github+json")

		emailResp, err := client.Do(emailReq)
		if err == nil {
			defer func() { _ = emailResp.Body.Close() }()
			emailBody, _ := io.ReadAll(emailResp.Body)
			var emails []map[string]interface{}
			_ = json.Unmarshal(emailBody, &emails)
			for _, e := range emails {
				if primary, _ := e["primary"].(bool); primary {
					email, _ = e["email"].(string)
					break
				}
			}
		}
	}

	if email == "" {
		email = fmt.Sprintf("%s@users.noreply.github.com", login)
	}
	if login == "" {
		login = fmt.Sprintf("github_%d", githubID)
	}

	login = truncateRunes(login, MaxGithubUsernameLen)

	// Geçici username — çakışma imkansız çünkü github_id global unique.
	// Kullanıcı onboarding'i tamamlayınca kendi seçtiği isim yazılır.
	tempUsername := fmt.Sprintf("user_%d", githubID)

	var (
		userID    int
		isPremium bool
		onboarded bool
	)
	err = db.QueryRow(ctx, `
		INSERT INTO users (username, email, github_id, github_username, avatar_url, bio, onboarded)
		VALUES ($1, $2, $3, $4, $5, $6, false)
		ON CONFLICT (github_id) DO UPDATE
		SET github_username = EXCLUDED.github_username,
			email           = EXCLUDED.email,
			avatar_url      = COALESCE(EXCLUDED.avatar_url, users.avatar_url)
		RETURNING id, is_premium, onboarded
	`, tempUsername, email, githubID, login, avatar, bio).Scan(&userID, &isPremium, &onboarded)
	if err != nil {
		serverError(c, err, "auth_failed")
		return
	}

	auditLog(c, userID, "login", "user", userID)

	jwtToken, err := generateJWT(userID, isPremium, onboarded)
	if err != nil {
		serverError(c, err, "auth_failed")
		return
	}

	frontend := os.Getenv("FRONTEND_URL")
	c.Redirect(http.StatusTemporaryRedirect, frontend+"/auth/callback?token="+jwtToken)
}
