package proxy

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"

	"github.com/JaLe29/ratelimit-simple-proxy/internal/auth"
	"github.com/JaLe29/ratelimit-simple-proxy/internal/cache"
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
	cache         *cache.MemoryCache
	metric        *metric.Metric
	auth          *auth.GoogleAuthenticator
	loginTemplate *template.Template
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

	// Initialize cache with capacity of 1000 items
	memCache := cache.NewMemoryCache(1000)

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
		cache:         memCache,
		metric:        metric,
		auth:          authenticator,
		loginTemplate: loginTemplate,
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

func (p *Proxy) ProxyHandler(w http.ResponseWriter, r *http.Request) {
	// If we're on the auth domain, process the callback
	if r.Host == p.config.GoogleAuth.AuthDomain {
		if r.URL.Path == "/auth/callback" {
			// Add auth middleware for callback processing
			var handler http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Here should be the code for processing the callback
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("Processing auth callback..."))
			})
			handler = middleware.NewAuthMiddleware(p.config, p.auth, r.Host, p.loginTemplate).Handle(handler)
			handler.ServeHTTP(w, r)
			return
		}
		// Other paths on auth domain are not allowed
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	target, ok := p.config.RateLimits[r.Host]
	if !ok {
		http.Error(w, fmt.Sprintf("Host (%s) not found", r.Host), http.StatusBadGateway)
		return
	}

	clientIp := p.getClientIp(r)
	fmt.Println("Client IP:", clientIp)
	fmt.Println("URL:", r.URL.RequestURI())

	// Create middleware chain
	var handler http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// This is the final handler that will be called if all middleware passes
		hasCache := target.CacheMaxTTLSeconds > 0
		shouldCache := hasCache && r.Method == http.MethodGet

		if shouldCache {
			cacheKey := cache.GetCacheKey(r)
			if item, found := p.cache.Get(cacheKey); found {
				fmt.Println("HIT - has cache" + strconv.FormatBool(hasCache) + ", cache key " + cacheKey)
				for key, values := range item.Headers {
					for _, value := range values {
						w.Header().Add(key, value)
					}
				}
				w.Header().Set("X-RLSP-Cache", "HIT")
				w.WriteHeader(item.ResponseCode)
				w.Write(item.Response)
				return
			}
			fmt.Println("MISS - has cache" + strconv.FormatBool(hasCache) + ", cache key " + cacheKey)
			w.Header().Set("X-RLSP-Cache", "MISS")
		}

		targetURL, err := url.Parse(target.Destination)
		if err != nil {
			http.Error(w, "Invalid target URL", http.StatusInternalServerError)
			return
		}

		p.metric.RequestsTotal.WithLabelValues(r.Host).Inc()

		if shouldCache {
			proxy := httputil.NewSingleHostReverseProxy(targetURL)
			originalDirector := proxy.Director
			proxy.Director = func(req *http.Request) {
				originalDirector(req)
				if origin := r.Header.Get("Origin"); origin != "" {
					req.Header.Set("Origin", origin)
				}
				req.Header.Set("X-Forwarded-Host", r.Host)
				req.Header.Set("X-Forwarded-Proto", r.URL.Scheme)
				req.Header.Add("X-Forwarded-For", clientIp)
			}

			originalTransport := proxy.Transport
			if originalTransport == nil {
				originalTransport = http.DefaultTransport
			}

			proxy.Transport = &cachingTransport{
				transport: originalTransport,
				callback: func(resp *http.Response, err error) {
					if err != nil || !cache.ShouldCache(r, resp.StatusCode) {
						return
					}

					respBody, err := io.ReadAll(resp.Body)
					if err != nil {
						return
					}

					resp.Body = io.NopCloser(bytes.NewBuffer(respBody))

					expiry := cache.GetCacheDuration(resp.Header, target.CacheMaxTTLSeconds)
					if expiry > 0 {
						headersCopy := http.Header{}
						for k, v := range resp.Header {
							headersCopy[k] = v
						}

						cacheKey := cache.GetCacheKey(r)
						p.cache.Set(cacheKey, respBody, resp.StatusCode, headersCopy, expiry)
					}
				},
			}

			proxy.ServeHTTP(w, r)
		} else {
			proxy := httputil.NewSingleHostReverseProxy(targetURL)
			originalDirector := proxy.Director
			proxy.Director = func(req *http.Request) {
				originalDirector(req)
				if origin := r.Header.Get("Origin"); origin != "" {
					req.Header.Set("Origin", origin)
				}
				req.Header.Set("X-Forwarded-Host", r.Host)
				req.Header.Set("X-Forwarded-Proto", r.URL.Scheme)
				req.Header.Add("X-Forwarded-For", clientIp)
			}

			proxy.ServeHTTP(w, r)
		}
	})

	// Add rate limiting middleware
	handler = middleware.NewRateLimitMiddleware(p.config, p.limiters[r.Host], r.Host, p.getClientIp).Handle(handler)

	// Add authentication middleware if enabled
	if p.auth != nil {
		handler = middleware.NewAuthMiddleware(p.config, p.auth, r.Host, p.loginTemplate).Handle(handler)
	}

	// Execute the middleware chain
	handler.ServeHTTP(w, r)
}

// cachingTransport is a custom HTTP transport for caching responses
type cachingTransport struct {
	transport http.RoundTripper
	callback  func(*http.Response, error)
}

func (t *cachingTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	response, err := t.transport.RoundTrip(request)
	if response != nil && t.callback != nil {
		t.callback(response, err)
	}
	return response, err
}
