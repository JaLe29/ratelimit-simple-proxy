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
	return &IPRateLimiter{
		accessMap:   make(map[string][]time.Time),
		windowSecs:  windowSeconds,
		maxRequests: maxRequests,
	}
}

// CheckLimit kontroluje, zda IP adresa nepřekročila limit
// Vrací true, pokud je překročený limit, a false pokud je přístup povolen
func (r *IPRateLimiter) CheckLimit(ipAddress string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	cutoffTime := now.Add(-time.Duration(r.windowSecs) * time.Second)

	// Získat historii přístupů pro tuto IP a vyčistit staré záznamy
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

// Nepovinná metoda pro ruční vyčištění všech záznamů (může být užitečná pro testování nebo reset)
func (r *IPRateLimiter) ClearAll() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.accessMap = make(map[string][]time.Time)
}
