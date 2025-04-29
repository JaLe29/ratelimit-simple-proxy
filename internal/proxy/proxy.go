package proxy

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"

	"github.com/JaLe29/ratelimit-simple-proxy/internal/cache"
	"github.com/JaLe29/ratelimit-simple-proxy/internal/config"
	"github.com/JaLe29/ratelimit-simple-proxy/internal/storage"
)

type Proxy struct {
	config   *config.Config
	limiters map[string]storage.Storage
	cache    *cache.MemoryCache
}

func NewProxy(cfg *config.Config) *Proxy {
	limiters := make(map[string]storage.Storage)

	// Initialize limiters for all configured hosts
	for host, target := range cfg.RateLimits {
		if target.PerSecond == -1 && target.Requests == -1 {
			store := storage.NewFakeStorage()
			limiters[host] = store
			fmt.Println("Host:", host, "is using fake storage")
			continue
		}

		var store storage.Storage = storage.NewIPRateLimiter(target.PerSecond, target.Requests)
		fmt.Println("Host:", host, "is using ip rate limiter")
		limiters[host] = store
	}

	// Inicializace cache s kapacitou 1000 položek
	memCache := cache.NewMemoryCache(1000)

	return &Proxy{
		config:   cfg,
		limiters: limiters,
		cache:    memCache,
	}
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
	clientIp := p.getClientIp(r)
	fmt.Println("Client IP:", clientIp)
	fmt.Println("URL:", r.URL.RequestURI())

	target, ok := p.config.RateLimits[r.Host]
	if !ok {
		http.Error(w, "Host ("+r.Host+") not found", http.StatusBadGateway)
		return
	}

	// has ip perma block
	if target.IpBlackList[clientIp] {
		http.Error(w, "Access denied. Your IP ("+clientIp+") is blocked.", http.StatusForbidden)
		return
	}

	hasCache := target.CacheMaxTtlSeconds > 0

	// Get limiter for this host
	limiter := p.limiters[r.Host]

	// Check rate limit
	if limiter.CheckLimit(clientIp) {
		http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
		return
	}

	xxx := cache.GetCacheKey(r)
	fmt.Println("has cache" + strconv.FormatBool(hasCache) + ", cache key " + xxx)

	shouldCache := hasCache && r.Method == http.MethodGet
	// Zkontrolujeme, zda můžeme použít cache
	// Cache použijeme pouze pro GET requesty
	if shouldCache {
		cacheKey := cache.GetCacheKey(r)
		if item, found := p.cache.Get(cacheKey); found {
			fmt.Println("HIT - has cache" + strconv.FormatBool(hasCache) + ", cache key " + xxx)
			// Máme cache hit, vrátíme přímo z cache
			for key, values := range item.Headers {
				for _, value := range values {
					w.Header().Add(key, value)
				}
			}
			w.Header().Set("X-RLSP-Cache", "HIT")
			w.WriteHeader(item.ResponseCode)
			w.Write(item.Response)
			return
		} else {
			fmt.Println("MISS - has cache" + strconv.FormatBool(hasCache) + ", cache key " + xxx)
		}
		w.Header().Set("X-RLSP-Cache", "MISS")
	}

	// Pokračujeme standardním zpracováním proxy
	targetURL, err := url.Parse(target.Destination)
	if err != nil {
		http.Error(w, "Invalid target URL", http.StatusInternalServerError)
		return
	}

	if shouldCache {
		// Místo přímého volání proxy vytvoříme custom transport a zachytíme odpověď
		proxy := httputil.NewSingleHostReverseProxy(targetURL)

		// Upravíme Director funkci jako předtím
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

		// Zachytíme odpověď modifikací transportu
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

				// Kopírujeme tělo odpovědi
				respBody, err := io.ReadAll(resp.Body)
				if err != nil {
					return
				}

				// Znovu nastavíme tělo pro další čtení
				resp.Body = io.NopCloser(bytes.NewBuffer(respBody))

				// Určíme dobu cachování
				expiry := cache.GetCacheDuration(resp.Header, target.CacheMaxTtlSeconds)
				if expiry > 0 {
					// Kopírujeme hlavičky
					headersCopy := http.Header{}
					for k, v := range resp.Header {
						headersCopy[k] = v
					}

					// Ukládáme do cache
					cacheKey := cache.GetCacheKey(r)
					p.cache.Set(cacheKey, respBody, resp.StatusCode, headersCopy, expiry)
				}
			},
		}

		proxy.ServeHTTP(w, r)
	} else {
		// Pro non-GET metody pokračujeme bez cache
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
}

// cachingTransport je custom HTTP transport pro zachycení odpovědi
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
