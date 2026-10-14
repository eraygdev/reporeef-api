package app

import (
	"fmt"
	"log/slog"
	"sync"
	"time"
)

const (
	repoCacheTTL        = 5 * time.Minute // Aynı repo 5 dk cache'te kalır
	repoCacheCleanupInt = 5 * time.Minute // Her 5 dk'da expired entry'ler silinir
	repoCacheMaxEntries = 5000            // Bellek koruması
)

type repoCacheEntry struct {
	err       error
	expiresAt time.Time
}

type repoCacheStore struct {
	mu sync.RWMutex
	m  map[string]repoCacheEntry
}

var repoPublicCache = &repoCacheStore{
	m: make(map[string]repoCacheEntry),
}

// Get — key için cache'te geçerli (expired olmayan) entry var mı?
func (c *repoCacheStore) Get(key string) (error, bool) {
	c.mu.RLock()
	entry, ok := c.m[key]
	c.mu.RUnlock()

	if !ok {
		return nil, false
	}
	if time.Now().After(entry.expiresAt) {
		return nil, false
	}
	return entry.err, true
}

// Set — sonucu (başarı veya hata) cache'e yaz.
func (c *repoCacheStore) Set(key string, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Boyut koruması: max'a ulaşıldıysa önce expired olanları temizle
	if len(c.m) >= repoCacheMaxEntries {
		now := time.Now()
		for k, v := range c.m {
			if now.After(v.expiresAt) {
				delete(c.m, k)
			}
		}
		// Hâlâ doluysa yeni entry ekleme (çok nadir, cache miss gibi davranır)
		if len(c.m) >= repoCacheMaxEntries {
			return
		}
	}

	c.m[key] = repoCacheEntry{
		err:       err,
		expiresAt: time.Now().Add(repoCacheTTL),
	}
}

// cleanup — süresi geçmiş entry'leri sil (periyodik).
func (c *repoCacheStore) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	removed := 0
	for k, v := range c.m {
		if now.After(v.expiresAt) {
			delete(c.m, k)
			removed++
		}
	}
	if removed > 0 {
		slog.Debug("repo cache cleanup", "removed", removed, "remaining", len(c.m))
	}
}

// StartRepoCacheCleanup — background goroutine, expired entry'leri temizler.
// main.go'da goroutine olarak çağrılır.
func StartRepoCacheCleanup() {
	ticker := time.NewTicker(repoCacheCleanupInt)
	defer ticker.Stop()
	for range ticker.C {
		repoPublicCache.cleanup()
	}
}

// checkGithubRepoPublicCached — checkGithubRepoPublic için cache wrapper.
// Aynı repo URL'i 5 dakika içinde tekrar sorulursa GitHub'a gitmez.
func checkGithubRepoPublicCached(repoURL string) error {
	key := normalizeGithubRepoURL(repoURL)
	if key == "" {
		return fmt.Errorf("invalid_github_url")
	}

	if err, ok := repoPublicCache.Get(key); ok {
		slog.Debug("repo cache hit", "repo", key)
		return err
	}

	slog.Debug("repo cache miss", "repo", key)
	err := checkGithubRepoPublic(key)
	repoPublicCache.Set(key, err)
	return err
}
