package app

import (
	"sync"
	"time"
)

// categoryCacheStore — /categories endpoint için in-memory cache.
//
// Kategoriler statik veri (yılda 2-3 kez değişir). Her istekte DB'ye
// gitmek israf — 5 dk TTL ile ~30-40 DB çağrısı/saat tasarruf ediyoruz.
//
// Concurrency: sync.RWMutex ile korunuyor. Get çağrıları eşzamanlı,
// Set çağrıları seri.
type categoryCacheStore struct {
	mu        sync.RWMutex
	data      []map[string]interface{}
	expiresAt time.Time
}

var catCache = &categoryCacheStore{}

const catCacheTTL = 5 * time.Minute

// getCachedCategories — cache geçerliyse data döner, ok=true.
func getCachedCategories() ([]map[string]interface{}, bool) {
	catCache.mu.RLock()
	defer catCache.mu.RUnlock()

	if catCache.data != nil && time.Now().Before(catCache.expiresAt) {
		return catCache.data, true
	}
	return nil, false
}

// setCachedCategories — yeni veriyi cache'e yazar.
func setCachedCategories(data []map[string]interface{}) {
	catCache.mu.Lock()
	defer catCache.mu.Unlock()

	catCache.data = data
	catCache.expiresAt = time.Now().Add(catCacheTTL)
}
