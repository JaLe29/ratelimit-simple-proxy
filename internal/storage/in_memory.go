package storage

import (
	"sync"
	"time"
)

// IPRateLimiter představuje rate limiter na základě IP adres
type IPRateLimiter struct {
	mu          sync.RWMutex
	accessMap   map[string]*accessWindow // IP adresa -> access window
	windowSecs  int                      // Časové okno v sekundách
	maxRequests int                      // Maximální počet požadavků v okně
}

// accessWindow represents a sliding window of access times
type accessWindow struct {
	accesses []time.Time
	lastIdx  int // Circular buffer index
}

// NewIPRateLimiter vytvoří novou instanci rate limiteru
func NewIPRateLimiter(windowSeconds, maxRequests int) *IPRateLimiter {
	limiter := &IPRateLimiter{
		accessMap:   make(map[string]*accessWindow),
		windowSecs:  windowSeconds,
		maxRequests: maxRequests,
	}

	// Spustit goroutinu pro pravidelné čištění jednou za minutu
	go limiter.cleanupRoutine()

	return limiter
}

// cleanupRoutine spouští pravidelné čištění staré historie v intervalu 1 minuty
func (r *IPRateLimiter) cleanupRoutine() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		r.cleanup()
	}
}

// cleanup odstraňuje staré záznamy a prázdné IP adresy z mapy
func (r *IPRateLimiter) cleanup() {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	cutoffTime := now.Add(-time.Duration(r.windowSecs) * time.Second)

	for ip, window := range r.accessMap {
		var recentAccesses []time.Time
		for _, accessTime := range window.accesses {
			if accessTime.After(cutoffTime) {
				recentAccesses = append(recentAccesses, accessTime)
			}
		}

		if len(recentAccesses) == 0 {
			// Pokud nejsou žádné nedávné přístupy, odstraň IP z mapy
			delete(r.accessMap, ip)
		} else {
			// Aktualizuj seznam pouze na nedávné přístupy
			window.accesses = recentAccesses
		}
	}
}

// CheckLimit zkontroluje, zda IP adresa překročila limit požadavků
func (r *IPRateLimiter) CheckLimit(ipAddress string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	cutoffTime := now.Add(-time.Duration(r.windowSecs) * time.Second)

	// Získat historii přístupů pro tuto IP
	window, exists := r.accessMap[ipAddress]

	// Pokud není historie, vytvořit nový záznam
	if !exists {
		window = &accessWindow{
			accesses: make([]time.Time, 0, r.maxRequests+1), // Pre-allocate with capacity
		}
		r.accessMap[ipAddress] = window
	}

	// Filtrovat pouze nedávné přístupy v rámci časového okna
	validCount := 0
	for _, accessTime := range window.accesses {
		if accessTime.After(cutoffTime) {
			window.accesses[validCount] = accessTime
			validCount++
		}
	}

	// Resize slice to valid count
	window.accesses = window.accesses[:validCount]

	// Přidat aktuální přístup
	window.accesses = append(window.accesses, now)

	// Překročil limit?
	return len(window.accesses) > r.maxRequests
}
