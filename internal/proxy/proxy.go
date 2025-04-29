package proxy

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/JaLe29/ratelimit-simple-proxy/internal/config"
	"github.com/JaLe29/ratelimit-simple-proxy/internal/storage"
)

type Proxy struct {
	config   *config.Config
	limiters map[string]storage.Storage
}

func NewProxy(cfg *config.Config) *Proxy {
	limiters := make(map[string]storage.Storage)

	// Initialize limiters for all configured hosts
	for host, target := range cfg.RateLimits {
		var store storage.Storage = storage.NewIPRateLimiter(target.PerSecond, target.Requests)
		limiters[host] = store
	}

	return &Proxy{
		config:   cfg,
		limiters: limiters,
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

	// Get limiter for this host
	limiter := p.limiters[r.Host]

	// Check rate limit
	if limiter.CheckLimit(clientIp) {
		http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
		return
	}

	targetURL, err := url.Parse(target.Destination)
	if err != nil {
		http.Error(w, "Invalid target URL", http.StatusInternalServerError)
		return
	}

	// Vytvoření reverse proxy
	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	// Přepsání původních hlaviček
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)

		// Přidání důležitých hlaviček pro CORS
		if origin := r.Header.Get("Origin"); origin != "" {
			req.Header.Set("Origin", origin)
		}

		// Další potřebné hlavičky pro proxy
		req.Header.Set("X-Forwarded-Host", r.Host)
		req.Header.Set("X-Forwarded-Proto", r.URL.Scheme)
		req.Header.Add("X-Forwarded-For", clientIp)
	}

	// Přesměrování požadavku
	proxy.ServeHTTP(w, r)
}
