package app

import (
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	sentrygin "github.com/getsentry/sentry-go/gin"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// NewRouter tüm middleware ve route'ları bağlar, hazır *gin.Engine döner.
func NewRouter() *gin.Engine {
	// gin.Default() kullanmıyoruz — içindeki Logger middleware'i
	// bizim structured RequestLogger ile çakışıyor. Onun yerine
	// gin.New() + gin.Recovery() kullanıyoruz.
	// Log artık tamamen slog üzerinden.
	router := gin.New()
	router.Use(Recovery())

	// Sentry middleware — panikleri yakalar
	// Repanic: true → Gin'in kendi Recovery'si hâlâ çalışsın
	router.Use(sentrygin.New(sentrygin.Options{Repanic: true}))

	// Request logger — her isteği structured JSON olarak loglar,
	// her isteğe request_id atar (X-Request-ID header'ı olarak döner).
	router.Use(RequestLogger())

	// Güvenilir proxy ayarı — X-Forwarded-For spoofing'i engeller
	// Production'da sadece Cloudflare IP'lerine güven
	if os.Getenv("ENV") == "production" {
		// Topoloji: Cloudflare Edge → cloudflared (Docker) → backend
		// Backend'in doğrudan peer'ı cloudflared container'ı (RFC1918).
		// Cloudflare edge IP'leri KALDIRILDI — tunnel modunda backend'e
		// doğrudan bağlanmıyorlar. Onun yerine Docker internal network +
		// loopback güvenilir sayılır. Cloudflared, gerçek kullanıcı IP'sini
		// X-Forwarded-For header'ına set eder; Gin bu header'ı güvenilir
		// peer'dan geldiği için okur.
		_ = router.SetTrustedProxies([]string{
			"127.0.0.1/32",
			"::1/128",
			"10.0.0.0/8",
			"172.16.0.0/12",
			"192.168.0.0/16",
		})
	} else {
		// Development'ta X-Forwarded-For'a hiç güvenme
		_ = router.SetTrustedProxies(nil)
	}

	router.Use(rateLimitMiddleware())
	router.Use(maxBodySizeMiddleware(MaxRequestBodyBytes))
	router.Use(securityHeadersMiddleware())

	setupCORS(router)

	router.GET("/ping", handlePing)
	router.GET("/ready", handleReady)
	router.GET("/sitemap.xml", handleSitemap)
	router.HEAD("/sitemap.xml", handleSitemap) // Dinamik sitemap
	router.GET("/robots.txt", handleRobotsAPI)

	// API alt alan adında favicon istenirse 204 dön.
	// Tarayıcılar/botlar otomatik arar, 404 log kirliliği yaratır.
	router.GET("/favicon.ico", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	api := router.Group("/api/" + APIVersion)

	// Auth
	auth := api.Group("/auth")
	{
		auth.GET("/github/login", handleGithubLogin)
		auth.GET("/github/callback", handleGithubCallback)
		auth.GET("/me", authMiddleware(), handleMe)
	}

	// Projects
	projects := api.Group("/projects")
	{
		projects.GET("", authMiddlewareOptional(), handleListProjects)
		projects.GET("/:id", authMiddlewareOptional(), handleGetProject)
		projects.POST("", authMiddleware(), requireOnboarded(), handleCreateProject)
		projects.PUT("/:id", authMiddleware(), requireOnboarded(), handleUpdateProject)
		projects.DELETE("/:id", authMiddleware(), requireOnboarded(), handleDeleteProject)

		projects.POST("/:id/star", authMiddleware(), requireOnboarded(), handleStarProject)
		projects.DELETE("/:id/star", authMiddleware(), requireOnboarded(), handleUnstarProject)

		projects.POST("/:id/join", authMiddleware(), requireOnboarded(), handleJoinProject)
		projects.DELETE("/:id/leave", authMiddleware(), requireOnboarded(), handleLeaveProject)
		projects.GET("/:id/my-join-status", authMiddleware(), handleMyJoinStatus)
		projects.DELETE("/:id/contributors/:userId", authMiddleware(), handleRemoveContributor)
	}

	// Me — kendi kaynaklarım
	me := api.Group("/me", authMiddleware())
	{
		me.PUT("", handleUpdateMe)
		me.GET("/projects", handleMyProjects)
		me.GET("/contributions", handleMyContributions)
		me.GET("/contributor-requests", handleContributorRequests)
	}

	// Users — search önce tanımlı olmalı
	users := api.Group("/users")
	{
		users.GET("/search", handleSearchUsers)
		users.GET("/:username", handleGetUserByUsername)
	}

	api.GET("/categories", handleListCategories)
	api.GET("/config", handleConfig)

	// Contributor requests
	requests := api.Group("/contributor-requests", authMiddleware())
	{
		requests.PUT("/:id/approve", handleApproveRequest)
		requests.PUT("/:id/reject", handleRejectRequest)
	}

	return router
}

// setupCORS CORS middleware'ini yapılandırır.
func setupCORS(router *gin.Engine) {
	// Ana frontend URL'i .env'den
	frontendURL := os.Getenv("FRONTEND_URL")

	// Ek izinli origin'ler (virgülle ayrılmış)
	// Örnek: ALLOWED_ORIGINS=http://localhost:5173,https://reporeef.com,https://www.reporeef.com
	extraOrigins := os.Getenv("ALLOWED_ORIGINS")

	origins := []string{}

	// Ana frontend URL
	if frontendURL != "" {
		origins = append(origins, frontendURL)
	} else {
		slog.Warn("FRONTEND_URL is not set")
	}

	// Ek origin'ler
	if extraOrigins != "" {
		for _, o := range strings.Split(extraOrigins, ",") {
			o = strings.TrimSpace(o)
			if o != "" {
				origins = append(origins, o)
			}
		}
	}

	slog.Info("cors origins configured", "origins", origins)

	router.Use(cors.New(cors.Config{
		AllowOrigins:     origins,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: false,
		MaxAge:           10 * time.Minute, // Preflight cache — tarayıcı 10 dk boyunca tekrar sormaz
	}))
}
