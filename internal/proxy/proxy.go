package proxy

import (
	"bufio"
	"context"
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"time"

	"github.com/JaLe29/ratelimit-simple-proxy/internal/auth"
	"github.com/JaLe29/ratelimit-simple-proxy/internal/config"
	"github.com/JaLe29/ratelimit-simple-proxy/internal/metric"
	"github.com/JaLe29/ratelimit-simple-proxy/internal/middleware"
	"github.com/JaLe29/ratelimit-simple-proxy/internal/storage"
	"github.com/JaLe29/ratelimit-simple-proxy/internal/templates"
)

// Proxy represents the reverse proxy
type Proxy struct {
	config        *config.Config
	limiters      map[string]storage.Storage
	metric        *metric.Metric
	auth          *auth.GoogleAuthenticator
	loginTemplate *template.Template
	proxyCache    map[string]*httputil.ReverseProxy // Cache for proxy instances
	proxyMutex    sync.RWMutex
	handlerCache  map[string]http.Handler // Cache for pre-built middleware chains
	handlerMutex  sync.RWMutex
}

// responseTimeWriter wraps http.ResponseWriter to track response time
type responseTimeWriter struct {
	http.ResponseWriter
	startTime time.Time
	metric    *metric.Metric
	origin    string
	recorded  bool
}

func (w *responseTimeWriter) WriteHeader(statusCode int) {
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *responseTimeWriter) Write(data []byte) (int, error) {
	return w.ResponseWriter.Write(data)
}

func (w *responseTimeWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hijacker, ok := w.ResponseWriter.(http.Hijacker); ok {
		return hijacker.Hijack()
	}
	return nil, nil, fmt.Errorf("underlying ResponseWriter does not implement http.Hijacker")
}

func (w *responseTimeWriter) recordResponseTime() {
	if !w.recorded {
		duration := time.Since(w.startTime).Seconds()
		w.metric.ResponseTime.WithLabelValues(w.origin).Observe(duration)
		w.recorded = true
	}
}

// NewProxy creates a new proxy instance
func NewProxy(cfg *config.Config, metric *metric.Metric) (*Proxy, error) {
	limiters := make(map[string]storage.Storage)
	var authenticator *auth.GoogleAuthenticator

	// Initialize limiters for all configured hosts
	for host, target := range cfg.RateLimits {
		if target.PerSecond == -1 && target.Requests == -1 {
			store := storage.NewFakeStorage()
			limiters[host] = store
			fmt.Println("Host:", host, "is using fake storage")
		} else {
			var store storage.Storage = storage.NewIPRateLimiter(target.PerSecond, target.Requests)
			fmt.Println("Host:", host, "is using ip rate limiter")
			limiters[host] = store
		}
	}

	// Initialize Google authenticator if enabled globally
	if cfg.GoogleAuth != nil && cfg.GoogleAuth.Enabled {
		authenticator = auth.NewGoogleAuthenticator(
			cfg.GoogleAuth.ClientID,
			cfg.GoogleAuth.ClientSecret,
			cfg.GoogleAuth.RedirectURL,
			cfg,
		)
		fmt.Println("Google authentication is enabled globally")
	}

	// Load login template once at startup
	var loginTemplate *template.Template
	if cfg.GoogleAuth != nil && cfg.GoogleAuth.Enabled {
		var err error
		loginTemplate, err = templates.LoadLoginTemplate()
		if err != nil {
			return nil, fmt.Errorf("failed to load login template: %w", err)
		}
	}

	return &Proxy{
		config:        cfg,
		limiters:      limiters,
		metric:        metric,
		auth:          authenticator,
		loginTemplate: loginTemplate,
		proxyCache:    make(map[string]*httputil.ReverseProxy),
		proxyMutex:    sync.RWMutex{},
		handlerCache:  make(map[string]http.Handler),
		handlerMutex:  sync.RWMutex{},
	}, nil
}

func (p *Proxy) getClientIp(r *http.Request) string {
	var clientIp string

	for _, header := range p.config.IPHeader.Headers {
		if ip := r.Header.Get(header); ip != "" {
			clientIp = ip
			break
		}
	}

	if clientIp == "" {
		return "empty"
	}

	return clientIp
}

// normalizeDomain removes www prefix from domain names for consistent metric labeling
func (p *Proxy) normalizeDomain(host string) string {
	if len(host) > 4 && host[:4] == "www." {
		return host[4:]
	}
	return host
}

func (p *Proxy) ProxyHandler(w http.ResponseWriter, r *http.Request) {
	// Special handling for auth domain
	if r.Host == p.config.GoogleAuth.AuthDomain {
		// Create auth-only handler for auth domain
		var handler http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "Not found", http.StatusNotFound)
		})

		// Add authentication middleware for auth domain
		if p.auth != nil {
			handler = middleware.NewAuthMiddleware(p.config, p.auth, r.Host, p.loginTemplate).Handle(handler)
		}

		handler.ServeHTTP(w, r)
		return
	}

	// Use cached handler for this host
	handler := p.getOrCreateHandler(r.Host)
	handler.ServeHTTP(w, r)
}

func (p *Proxy) getOrCreateProxy(targetURL *url.URL, clientIp string) *httputil.ReverseProxy {
	p.proxyMutex.RLock()
	if proxy, exists := p.proxyCache[targetURL.String()]; exists {
		p.proxyMutex.RUnlock()
		return proxy
	}
	p.proxyMutex.RUnlock()

	p.proxyMutex.Lock()
	defer p.proxyMutex.Unlock()

	// Double-check after acquiring write lock
	if proxy, exists := p.proxyCache[targetURL.String()]; exists {
		return proxy
	}

	proxy := httputil.NewSingleHostReverseProxy(targetURL)
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		fmt.Println("Request Host Origin:", p.normalizeDomain(req.Host))
		req.Header.Set("Origin", p.normalizeDomain(req.Host))
		req.Header.Set("X-Forwarded-Host", req.Host)
		req.Header.Set("X-Forwarded-Proto", req.URL.Scheme)
		req.Header.Add("X-Forwarded-For", clientIp)
	}

	p.proxyCache[targetURL.String()] = proxy
	return proxy
}

func (p *Proxy) getOrCreateHandler(host string) http.Handler {
	p.handlerMutex.RLock()
	if handler, exists := p.handlerCache[host]; exists {
		p.handlerMutex.RUnlock()
		return handler
	}
	p.handlerMutex.RUnlock()

	p.handlerMutex.Lock()
	defer p.handlerMutex.Unlock()

	// Double-check after acquiring write lock
	if handler, exists := p.handlerCache[host]; exists {
		return handler
	}

	// Create the final handler
	finalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		target, ok := p.config.RateLimits[host]
		if !ok {
			http.Error(w, fmt.Sprintf("Host (%s) not found", host), http.StatusBadGateway)
			return
		}

		clientIp := p.getClientIp(r)
		fmt.Println("Client IP:", clientIp)
		fmt.Println("URL:", r.URL.RequestURI())

		targetURL, err := url.Parse(target.Destination)
		if err != nil {
			http.Error(w, "Invalid target URL", http.StatusInternalServerError)
			return
		}

		// Normalize domain for consistent metrics
		normalizedDomain := p.normalizeDomain(host)
		p.metric.RequestsTotal.WithLabelValues(normalizedDomain).Inc()

		// Create response time writer
		rtw := &responseTimeWriter{
			ResponseWriter: w,
			startTime:      time.Now(),
			metric:         p.metric,
			origin:         normalizedDomain,
			recorded:       false,
		}

		// Ensure response time is recorded when the handler completes
		defer rtw.recordResponseTime()

		proxy := p.getOrCreateProxy(targetURL, clientIp)
		proxy.ServeHTTP(rtw, r)
	})

	// Build middleware chain
	var handler http.Handler = finalHandler

	// Add CORS middleware (first in chain, last to execute)
	handler = middleware.NewCORSMiddleware(p.config, host).Handle(handler)

	// Add rate limiting middleware
	handler = middleware.NewRateLimitMiddleware(p.config, p.limiters[host], host, p.getClientIp).Handle(handler)

	// Add authentication middleware if enabled
	if p.auth != nil {
		handler = middleware.NewAuthMiddleware(p.config, p.auth, host, p.loginTemplate).Handle(handler)
	}

	p.handlerCache[host] = handler
	return handler
}

// Shutdown gracefully shuts down the proxy and cleans up resources
func (p *Proxy) Shutdown(ctx context.Context) error {
	log.Println("Shutting down proxy...")

	// Clean up rate limiters
	for host, limiter := range p.limiters {
		if closer, ok := limiter.(interface{ Close() error }); ok {
			if err := closer.Close(); err != nil {
				log.Printf("Error closing limiter for %s: %v", host, err)
			}
		}
	}

	log.Println("Proxy shutdown completed")
	return nil
}
