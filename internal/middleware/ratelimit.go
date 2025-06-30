package middleware

import (
	"fmt"
	"net/http"

	"github.com/JaLe29/ratelimit-simple-proxy/internal/config"
	"github.com/JaLe29/ratelimit-simple-proxy/internal/metric"
	"github.com/JaLe29/ratelimit-simple-proxy/internal/storage"
)

// RateLimitMiddleware handles rate limiting for the proxy
type RateLimitMiddleware struct {
	config  *config.Config
	limiter storage.Storage
	host    string
	getIP   func(*http.Request) string
	metric  *metric.Metric
}

// NewRateLimitMiddleware creates a new rate limiting middleware
func NewRateLimitMiddleware(cfg *config.Config, limiter storage.Storage, host string, getIP func(*http.Request) string, metric *metric.Metric) *RateLimitMiddleware {
	return &RateLimitMiddleware{
		config:  cfg,
		limiter: limiter,
		host:    host,
		getIP:   getIP,
		metric:  metric,
	}
}

// Handle processes the rate limiting middleware
func (m *RateLimitMiddleware) Handle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		target, ok := m.config.RateLimits[m.host]
		if !ok {
			http.Error(w, fmt.Sprintf("Host (%s) not found", m.host), http.StatusBadGateway)
			return
		}

		clientIP := m.getIP(r)

		// Check IP blacklist
		if target.IPBlackList[clientIP] {
			http.Error(w, fmt.Sprintf("Access denied. Your IP (%s) is blocked.", clientIP), http.StatusForbidden)
			return
		}

		// Check rate limit
		if m.limiter.CheckLimit(clientIP) {
			// Record rate limit hit metric
			if m.metric != nil {
				m.metric.RateLimitHits.WithLabelValues(m.host, clientIP).Inc()
			}
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}
