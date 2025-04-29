// internal/cache/cache.go
package cache

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// CacheItem představuje položku v cache
type CacheItem struct {
	Response     []byte
	ResponseCode int
	Headers      http.Header
	Timestamp    time.Time
	Expiry       time.Duration
}

// IsExpired kontroluje, zda položka v cache vypršela
func (item *CacheItem) IsExpired() bool {
	if item.Expiry == 0 {
		return false
	}
	return time.Since(item.Timestamp) > item.Expiry
}

// MemoryCache je implementace in-memory cache
type MemoryCache struct {
	items   map[string]CacheItem
	mutex   sync.RWMutex
	maxSize int // maximální počet položek v cache
}

// NewMemoryCache vytvoří novou instanci MemoryCache
func NewMemoryCache(maxSize int) *MemoryCache {
	cache := &MemoryCache{
		items:   make(map[string]CacheItem),
		maxSize: maxSize,
	}

	// Spustíme pravidelné čištění vypršených položek
	go cache.periodicCleanup()

	return cache
}

// Get získá položku z cache podle klíče
func (c *MemoryCache) Get(key string) (*CacheItem, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	item, found := c.items[key]
	if !found {
		return nil, false
	}

	// Pokud položka vypršela, ignorujeme ji
	if item.IsExpired() {
		fmt.Println("EXPIRED - cache key " + key)
		return nil, false
	}

	return &item, true
}

// Set uloží položku do cache
func (c *MemoryCache) Set(key string, value []byte, code int, headers http.Header, expiry time.Duration) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Pokud jsme dosáhli maximální velikosti a přidáváme novou položku, odstraníme nějakou starou
	if len(c.items) >= c.maxSize && c.items[key].Timestamp.IsZero() {
		c.evictOldest()
	}

	c.items[key] = CacheItem{
		Response:     value,
		ResponseCode: code,
		Headers:      headers,
		Timestamp:    time.Now(),
		Expiry:       expiry,
	}
}

// Delete odstraní položku z cache
func (c *MemoryCache) Delete(key string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	delete(c.items, key)
}

// Clear vymaže celou cache
func (c *MemoryCache) Clear() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.items = make(map[string]CacheItem)
}

// evictOldest odstraní nejstarší položku z cache
func (c *MemoryCache) evictOldest() {
	var oldestKey string
	var oldestTime time.Time

	// První položka bude inicializovat oldestTime
	first := true

	for key, item := range c.items {
		if first || item.Timestamp.Before(oldestTime) {
			oldestKey = key
			oldestTime = item.Timestamp
			first = false
		}
	}

	if oldestKey != "" {
		delete(c.items, oldestKey)
	}
}

// periodicCleanup periodicky čistí vypršené položky
func (c *MemoryCache) periodicCleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.mutex.Lock()
		for key, item := range c.items {
			if item.IsExpired() {
				delete(c.items, key)
			}
		}
		c.mutex.Unlock()
	}
}

// GetCacheKey generuje klíč pro cache na základě požadavku
func GetCacheKey(r *http.Request) string {
	return r.Method + ":" + r.Host + r.URL.String()
}

// ShouldCache zjistí, zda by měl být požadavek/odpověď cachován
func ShouldCache(r *http.Request, statusCode int) bool {
	// Cachujeme pouze GET požadavky
	if r.Method != http.MethodGet {
		return false
	}

	// Nekachujeme, pokud má požadavek Authorization hlavičku
	if r.Header.Get("Authorization") != "" {
		return false
	}

	// Cachujeme pouze úspěšné odpovědi
	if statusCode < 200 || statusCode >= 300 {
		return false
	}

	return true
}

// GetCacheDuration určí, jak dlouho má být odpověď cachována
func GetCacheDuration(headers http.Header, defaultTtlSeconds int) time.Duration {
	// Pokud máme Cache-Control, použijeme jeho hodnoty
	if cacheControl := headers.Get("Cache-Control"); cacheControl != "" {
		if strings.Contains(cacheControl, "no-store") || strings.Contains(cacheControl, "no-cache") {
			return 0 // Nekachujeme
		}

		if maxAge := strings.Split(cacheControl, "max-age="); len(maxAge) > 1 {
			parts := strings.Split(maxAge[1], ",")
			seconds := 0
			if _, err := strconv.Atoi(parts[0]); err == nil {
				seconds, _ = strconv.Atoi(parts[0])
				if seconds > 0 {
					return time.Duration(seconds) * time.Second
				}
			}
		}
	}

	// Defaultní doba cachování
	// return 60 * time.Second
	return time.Duration(defaultTtlSeconds) * time.Second
}
