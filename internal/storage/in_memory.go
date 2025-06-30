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
	cleanupDone chan struct{}            // Channel pro graceful shutdown cleanup
}

// accessWindow represents a sliding window of access times using circular buffer
type accessWindow struct {
	accesses    []time.Time
	head        int // Index of the oldest element
	tail        int // Index where next element will be inserted
	count       int // Number of valid elements
	capacity    int // Buffer capacity
	lastCleanup time.Time
}

// newAccessWindow creates a new access window with circular buffer
func newAccessWindow(capacity int) *accessWindow {
	return &accessWindow{
		accesses:    make([]time.Time, capacity),
		head:        0,
		tail:        0,
		count:       0,
		capacity:    capacity,
		lastCleanup: time.Now(),
	}
}

// add adds a new access time to the circular buffer
func (w *accessWindow) add(t time.Time) {
	w.accesses[w.tail] = t
	w.tail = (w.tail + 1) % w.capacity
	if w.count < w.capacity {
		w.count++
	} else {
		// Buffer is full, move head forward
		w.head = (w.head + 1) % w.capacity
	}
}

// countValid counts valid accesses within the time window
func (w *accessWindow) countValid(cutoffTime time.Time) int {
	validCount := 0
	for i := 0; i < w.count; i++ {
		idx := (w.head + i) % w.capacity
		if w.accesses[idx].After(cutoffTime) {
			validCount++
		}
	}
	return validCount
}

// cleanup removes old entries from the window
func (w *accessWindow) cleanup(cutoffTime time.Time) {
	newHead := w.head
	newCount := w.count

	// Remove old entries from the beginning
	for newCount > 0 {
		if w.accesses[newHead].After(cutoffTime) {
			break
		}
		newHead = (newHead + 1) % w.capacity
		newCount--
	}

	w.head = newHead
	w.count = newCount
	w.lastCleanup = time.Now()
}

// NewIPRateLimiter vytvoří novou instanci rate limiteru
func NewIPRateLimiter(windowSeconds, maxRequests int) *IPRateLimiter {
	limiter := &IPRateLimiter{
		accessMap:   make(map[string]*accessWindow),
		windowSecs:  windowSeconds,
		maxRequests: maxRequests,
		cleanupDone: make(chan struct{}),
	}

	// Spustit goroutinu pro pravidelné čištění s optimalizovaným intervalem
	go limiter.cleanupRoutine()

	return limiter
}

// cleanupRoutine spouští pravidelné čištění s adaptive interval
func (r *IPRateLimiter) cleanupRoutine() {
	// Adaptive cleanup interval based on window size
	interval := time.Duration(r.windowSecs/2) * time.Second
	if interval < 30*time.Second {
		interval = 30 * time.Second
	}
	if interval > 5*time.Minute {
		interval = 5 * time.Minute
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			r.cleanup()
		case <-r.cleanupDone:
			return
		}
	}
}

// cleanup odstraňuje staré záznamy optimalizovaným způsobem
func (r *IPRateLimiter) cleanup() {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	cutoffTime := now.Add(-time.Duration(r.windowSecs) * time.Second)

	// Optimized cleanup - only process IPs that haven't been cleaned recently
	for ip, window := range r.accessMap {
		// Skip if recently cleaned (within last cleanup interval)
		if window.lastCleanup.Add(30 * time.Second).After(now) {
			continue
		}

		window.cleanup(cutoffTime)

		// Remove empty windows
		if window.count == 0 {
			delete(r.accessMap, ip)
		}
	}
}

// CheckLimit zkontroluje, zda IP adresa překročila limit požadavků - optimalizováno
func (r *IPRateLimiter) CheckLimit(ipAddress string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	cutoffTime := now.Add(-time.Duration(r.windowSecs) * time.Second)

	// Získat historii přístupů pro tuto IP
	window, exists := r.accessMap[ipAddress]

	// Pokud není historie, vytvořit nový záznam
	if !exists {
		window = newAccessWindow(r.maxRequests + 10) // Buffer for better performance
		r.accessMap[ipAddress] = window
	}

	// Fast cleanup for this specific window if needed
	if window.lastCleanup.Add(time.Duration(r.windowSecs/4) * time.Second).Before(now) {
		window.cleanup(cutoffTime)
	}

	// Count valid requests
	validCount := window.countValid(cutoffTime)

	// Add current access
	window.add(now)

	// Check if limit exceeded
	return validCount >= r.maxRequests
}

// Close gracefully shuts down the rate limiter
func (r *IPRateLimiter) Close() error {
	close(r.cleanupDone)
	return nil
}
