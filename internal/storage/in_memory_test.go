package storage

import (
	"sync"
	"testing"
	"time"
)

func TestIPRateLimiter_Basic(t *testing.T) {
	limiter := NewIPRateLimiter(1, 2) // 2 requests per 1 second
	defer limiter.Close()

	// First request should pass
	if limiter.CheckLimit("192.168.1.1") {
		t.Error("First request should not exceed limit")
	}

	// Second request should pass
	if limiter.CheckLimit("192.168.1.1") {
		t.Error("Second request should not exceed limit")
	}

	// Third request should exceed limit
	if !limiter.CheckLimit("192.168.1.1") {
		t.Error("Third request should exceed limit")
	}
}

func TestIPRateLimiter_TimeWindow(t *testing.T) {
	limiter := NewIPRateLimiter(1, 1) // 1 request per 1 second
	defer limiter.Close()

	// First request
	if limiter.CheckLimit("192.168.1.1") {
		t.Error("First request should not exceed limit")
	}

	// Second request should exceed limit
	if !limiter.CheckLimit("192.168.1.1") {
		t.Error("Second request should exceed limit")
	}

	// Wait for window to expire
	time.Sleep(1100 * time.Millisecond)

	// Should be able to make request again
	if limiter.CheckLimit("192.168.1.1") {
		t.Error("Request after window should not exceed limit")
	}
}

func TestIPRateLimiter_DifferentIPs(t *testing.T) {
	limiter := NewIPRateLimiter(1, 1) // 1 request per 1 second
	defer limiter.Close()

	// Different IPs should have separate limits
	if limiter.CheckLimit("192.168.1.1") {
		t.Error("First IP should not exceed limit")
	}

	if limiter.CheckLimit("192.168.1.2") {
		t.Error("Second IP should not exceed limit")
	}

	// Both IPs should now exceed limit
	if !limiter.CheckLimit("192.168.1.1") {
		t.Error("First IP should exceed limit")
	}

	if !limiter.CheckLimit("192.168.1.2") {
		t.Error("Second IP should exceed limit")
	}
}

func TestIPRateLimiter_CircularBuffer(t *testing.T) {
	limiter := NewIPRateLimiter(10, 5) // 5 requests per 10 seconds
	defer limiter.Close()

	ip := "192.168.1.1"

	// Fill the buffer
	for i := 0; i < 5; i++ {
		if limiter.CheckLimit(ip) {
			t.Errorf("Request %d should not exceed limit", i+1)
		}
	}

	// Should exceed limit now
	if !limiter.CheckLimit(ip) {
		t.Error("Request should exceed limit")
	}

	// Wait for partial window to expire
	time.Sleep(2 * time.Second)

	// Add more requests to test circular buffer
	for i := 0; i < 3; i++ {
		limiter.CheckLimit(ip) // These will exceed but we're testing buffer behavior
	}

	// Verify buffer state is consistent
	window := limiter.accessMap[ip]
	if window.count > window.capacity {
		t.Errorf("Buffer overflow: count %d > capacity %d", window.count, window.capacity)
	}
}

func TestIPRateLimiter_Concurrent(t *testing.T) {
	limiter := NewIPRateLimiter(1, 10) // 10 requests per 1 second
	defer limiter.Close()

	const numGoroutines = 20
	const requestsPerGoroutine = 5

	var wg sync.WaitGroup
	limitExceeded := make(chan bool, numGoroutines*requestsPerGoroutine)

	// Launch concurrent goroutines
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			ip := "192.168.1.1" // Same IP for all

			for j := 0; j < requestsPerGoroutine; j++ {
				exceeded := limiter.CheckLimit(ip)
				limitExceeded <- exceeded
			}
		}(i)
	}

	wg.Wait()
	close(limitExceeded)

	// Count how many exceeded
	exceededCount := 0
	totalRequests := 0
	for exceeded := range limitExceeded {
		totalRequests++
		if exceeded {
			exceededCount++
		}
	}

	// Should have exceeded after 10 requests
	if exceededCount < totalRequests-10 {
		t.Errorf("Expected at least %d exceeded, got %d", totalRequests-10, exceededCount)
	}
}

func TestIPRateLimiter_Cleanup(t *testing.T) {
	limiter := NewIPRateLimiter(1, 2) // 2 requests per 1 second
	defer limiter.Close()

	// Create entries for multiple IPs
	ips := []string{"192.168.1.1", "192.168.1.2", "192.168.1.3"}
	for _, ip := range ips {
		limiter.CheckLimit(ip)
	}

	// Verify entries exist
	if len(limiter.accessMap) != 3 {
		t.Errorf("Expected 3 entries, got %d", len(limiter.accessMap))
	}

	// Wait for cleanup
	time.Sleep(2 * time.Second)

	// Trigger cleanup manually
	limiter.cleanup()

	// Entries should be cleaned up or have reduced count
	cleanedEntries := 0
	for _, window := range limiter.accessMap {
		if window.count == 0 {
			cleanedEntries++
		}
	}

	if cleanedEntries == 0 {
		t.Error("Expected some entries to be cleaned up")
	}
}

func TestAccessWindow_CircularBuffer(t *testing.T) {
	window := newAccessWindow(3)
	now := time.Now()

	// Add items to fill buffer
	for i := 0; i < 3; i++ {
		window.add(now.Add(time.Duration(i) * time.Second))
	}

	if window.count != 3 {
		t.Errorf("Expected count 3, got %d", window.count)
	}

	// Add one more - should wrap around
	window.add(now.Add(4 * time.Second))

	if window.count != 3 {
		t.Errorf("Expected count still 3, got %d", window.count)
	}

	// Verify oldest was removed
	cutoff := now.Add(1 * time.Second)
	validCount := window.countValid(cutoff)

	if validCount != 2 { // Should have 2 valid entries (index 1 and 2 from original, plus the new one)
		t.Errorf("Expected 2 valid entries, got %d", validCount)
	}
}

func TestAccessWindow_Cleanup(t *testing.T) {
	window := newAccessWindow(5)
	now := time.Now()

	// Add old and new entries
	window.add(now.Add(-10 * time.Second)) // Old
	window.add(now.Add(-5 * time.Second))  // Old
	window.add(now.Add(-1 * time.Second))  // Recent
	window.add(now)                        // Recent

	if window.count != 4 {
		t.Errorf("Expected count 4, got %d", window.count)
	}

	// Cleanup old entries (older than 3 seconds)
	cutoff := now.Add(-3 * time.Second)
	window.cleanup(cutoff)

	if window.count != 2 {
		t.Errorf("Expected count 2 after cleanup, got %d", window.count)
	}
}

// Benchmark tests for performance verification
func BenchmarkIPRateLimiter_CheckLimit(b *testing.B) {
	limiter := NewIPRateLimiter(60, 1000) // 1000 requests per minute
	defer limiter.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		limiter.CheckLimit("192.168.1.1")
	}
}

func BenchmarkIPRateLimiter_CheckLimit_DifferentIPs(b *testing.B) {
	limiter := NewIPRateLimiter(60, 1000) // 1000 requests per minute
	defer limiter.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ip := "192.168.1." + string(rune(i%255))
		limiter.CheckLimit(ip)
	}
}

func BenchmarkIPRateLimiter_Concurrent(b *testing.B) {
	limiter := NewIPRateLimiter(60, 1000) // 1000 requests per minute
	defer limiter.Close()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			limiter.CheckLimit("192.168.1.1")
		}
	})
}
