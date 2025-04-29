package proxy

import (
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/JaLe29/ratelimit-simple-proxy/internal/config"
	"github.com/JaLe29/ratelimit-simple-proxy/internal/storage"
)

type Proxy struct {
	config   *config.Config
	limiters map[string]*storage.IPRateLimiter
}

func NewProxy(cfg *config.Config) *Proxy {
	limiters := make(map[string]*storage.IPRateLimiter)

	// Initialize limiters for all configured hosts
	for host, target := range cfg.RateLimits {
		limiters[host] = storage.NewIPRateLimiter(target.PerSecond, target.Requests)
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

	proxyReq, err := http.NewRequest(r.Method, targetURL.String()+r.URL.RequestURI(), r.Body)
	if err != nil {
		http.Error(w, "Failed to create request", http.StatusInternalServerError)
		return
	}

	proxyReq.Header = r.Header.Clone()
	proxyReq.Host = targetURL.Host

	client := &http.Client{}
	resp, err := client.Do(proxyReq)

	if err != nil {
		http.Error(w, "Request failed", http.StatusBadGateway)
		return
	}

	fmt.Println("Response Status:", resp.Status)

	defer resp.Body.Close()

	for k, vv := range resp.Header {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}

	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}
