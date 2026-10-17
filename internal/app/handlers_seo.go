package app

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type SitemapURL struct {
	Loc        string `xml:"loc"`
	LastMod    string `xml:"lastmod,omitempty"`
	ChangeFreq string `xml:"changefreq,omitempty"`
	Priority   string `xml:"priority,omitempty"`
}

type SitemapUrlSet struct {
	XMLName xml.Name     `xml:"urlset"`
	Xmlns   string       `xml:"xmlns,attr"`
	URLs    []SitemapURL `xml:"url"`
}

// Sitemap Cache değişkenleri
var (
	sitemapCache   []byte
	sitemapCacheAt time.Time
	sitemapCacheMu sync.RWMutex
)

const sitemapCacheTTL = 24 * time.Hour // Sitemap 24 saat boyunca cache'te kalır

func handleSitemap(c *gin.Context) {
	// 1. Cache kontrolü (Okuma kilidi)
	sitemapCacheMu.RLock()
	if sitemapCache != nil && time.Since(sitemapCacheAt) < sitemapCacheTTL {
		defer sitemapCacheMu.RUnlock()
		// Tarayıcılara ve Cloudflare'e "bunu 1 saat cache'le" der.
		c.Header("Cache-Control", "public, max-age=86400") // 24 saat (60*60*24)
		c.Data(http.StatusOK, "application/xml; charset=utf-8", sitemapCache)
		return
	}
	sitemapCacheMu.RUnlock()

	// 2. Cache yoksa veya süresi geçmişse (Yazma kilidi)
	sitemapCacheMu.Lock()
	defer sitemapCacheMu.Unlock()

	// Kilit için beklerken başka bir istek ürettiyse tekrar kontrol et
	if sitemapCache != nil && time.Since(sitemapCacheAt) < sitemapCacheTTL {
		c.Header("Cache-Control", "public, max-age=86400") // 24 saat (60*60*24)
		c.Data(http.StatusOK, "application/xml; charset=utf-8", sitemapCache)
		return
	}

	ctx := c.Request.Context()
	urls := []SitemapURL{
		{Loc: "https://reporeef.com/", ChangeFreq: "daily", Priority: "1.0"},
		{Loc: "https://reporeef.com/about", ChangeFreq: "monthly", Priority: "0.5"},
		{Loc: "https://reporeef.com/privacy", ChangeFreq: "monthly", Priority: "0.3"},
		{Loc: "https://reporeef.com/terms", ChangeFreq: "monthly", Priority: "0.3"},
		{Loc: "https://reporeef.com/cookies", ChangeFreq: "monthly", Priority: "0.3"},
	}

	// Projeleri çek
	rows, err := queryCB(ctx, `SELECT id, updated_at FROM projects ORDER BY id DESC`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var id int
			var updatedAt time.Time
			if err := rows.Scan(&id, &updatedAt); err == nil {
				urls = append(urls, SitemapURL{
					Loc:        fmt.Sprintf("https://reporeef.com/project/%d", id),
					LastMod:    updatedAt.Format("2006-01-02"),
					ChangeFreq: "weekly",
					Priority:   "0.8",
				})
			}
		}
	}

	// Kullanıcıları çek — sadece onboarding'i tamamlamış olanlar.
	// onboarded=false olanların username'i hâlâ user_{id} formatında,
	// sitemap'e eklemek gürültü yaratır.
	rowsUsers, err := queryCB(ctx, `
		SELECT username, created_at FROM users
		WHERE onboarded = true
		ORDER BY id DESC
	`)
	if err == nil {
		defer rowsUsers.Close()
		for rowsUsers.Next() {
			var username string
			var createdAt time.Time
			if err := rowsUsers.Scan(&username, &createdAt); err == nil {
				urls = append(urls, SitemapURL{
					Loc:        fmt.Sprintf("https://reporeef.com/profile/%s", username),
					LastMod:    createdAt.Format("2006-01-02"),
					ChangeFreq: "weekly",
					Priority:   "0.6",
				})
			}
		}
	}

	urlset := SitemapUrlSet{
		Xmlns: "http://www.sitemaps.org/schemas/sitemap/0.9",
		URLs:  urls,
	}

	// XML'i byte dizisine çevir (Cache'lemek için)
	xmlData, err := xml.MarshalIndent(urlset, "", "  ")
	if err != nil {
		c.String(http.StatusInternalServerError, "Error generating sitemap")
		return
	}
	xmlData = append([]byte(xml.Header), xmlData...)

	// Cache'e yaz
	sitemapCache = xmlData
	sitemapCacheAt = time.Now()

	c.Header("Cache-Control", "public, max-age=86400")
	c.Data(http.StatusOK, "application/xml; charset=utf-8", sitemapCache)
}

func handleRobotsAPI(c *gin.Context) {
	// Googlebot'un API'ye erişmesine izin ver
	c.String(http.StatusOK, `User-agent: *
Allow: /`)
}
