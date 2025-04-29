package storage

import (
	"sync"
	"time"
)

// IPRateLimiter představuje rate limiter na základě IP adres
type IPRateLimiter struct {
	mu          sync.RWMutex
	accessMap   map[string][]time.Time // IP adresa -> seznam časů přístupu
	windowSecs  int                    // Časové okno v sekundách
	maxRequests int                    // Maximální počet požadavků v okně
}

// NewIPRateLimiter vytvoří novou instanci rate limiteru
func NewIPRateLimiter(windowSeconds, maxRequests int) *IPRateLimiter {
	limiter := &IPRateLimiter{
		accessMap:   make(map[string][]time.Time),
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

	for ip, accesses := range r.accessMap {
		var recentAccesses []time.Time
		for _, accessTime := range accesses {
			if accessTime.After(cutoffTime) {
				recentAccesses = append(recentAccesses, accessTime)
			}
		}

		if len(recentAccesses) == 0 {
			// Pokud nejsou žádné nedávné přístupy, odstraň IP z mapy
			delete(r.accessMap, ip)
		} else {
			// Aktualizuj seznam pouze na nedávné přístupy
			r.accessMap[ip] = recentAccesses
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
	accessHistory, exists := r.accessMap[ipAddress]

	// Pokud není historie, vytvořit nový záznam
	if !exists {
		r.accessMap[ipAddress] = []time.Time{now}
		return false // První přístup, určitě nepřekračuje limit
	}

	// Filtrovat pouze nedávné přístupy v rámci časového okna
	var recentAccesses []time.Time
	for _, accessTime := range accessHistory {
		if accessTime.After(cutoffTime) {
			recentAccesses = append(recentAccesses, accessTime)
		}
	}

	// Přidat aktuální přístup
	recentAccesses = append(recentAccesses, now)
	r.accessMap[ipAddress] = recentAccesses

	// Překročil limit?
	return len(recentAccesses) > r.maxRequests
}
