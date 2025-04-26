package proxy

import (
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/JaLe29/ratelimit-simple-proxy/internal/config"
)

type Proxy struct {
	config *config.Config
}

func NewProxy(cfg *config.Config) *Proxy {
	return &Proxy{
		config: cfg,
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

	return clientIp
}

func (p *Proxy) ProxyHandler(w http.ResponseWriter, r *http.Request) {

	clientIp := p.getClientIp(r)

	fmt.Println("Client IP:", clientIp)

	target, ok := p.config.RateLimits[r.Host]
	if !ok {
		http.Error(w, "Host '"+r.Host+"' not found", http.StatusBadGateway)
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

	client := &http.Client{}
	resp, err := client.Do(proxyReq)

	if err != nil {
		http.Error(w, "Request failed", http.StatusBadGateway)
		return
	}

	defer resp.Body.Close()

	for k, vv := range resp.Header {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}

	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}
