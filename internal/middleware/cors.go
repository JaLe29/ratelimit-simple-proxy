package middleware

import (
	"net/http"
	"strings"

	"github.com/JaLe29/ratelimit-simple-proxy/internal/config"
)

// CORSMiddleware handles CORS headers for the proxy
type CORSMiddleware struct {
	config *config.Config
	host   string
}

// NewCORSMiddleware creates a new CORS middleware
func NewCORSMiddleware(cfg *config.Config, host string) *CORSMiddleware {
	return &CORSMiddleware{
		config: cfg,
		host:   host,
	}
}

// Handle processes the CORS middleware
func (m *CORSMiddleware) Handle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		origin := r.Header.Get("Origin")

		// Allow requests from the same domain (with/without www)
		allowedOrigins := []string{
			"https://" + m.host,
			"http://" + m.host,
		}

		// Add www variants
		if strings.HasPrefix(m.host, "www.") {
			// If host starts with www, also allow without www
			domainWithoutWww := strings.TrimPrefix(m.host, "www.")
			allowedOrigins = append(allowedOrigins,
				"https://"+domainWithoutWww,
				"http://"+domainWithoutWww,
			)
		} else if !strings.Contains(m.host, ":") {
			// If host doesn't start with www, also allow www variant
			allowedOrigins = append(allowedOrigins,
				"https://www."+m.host,
				"http://www."+m.host,
			)
		}

		// Add shared domains from config
		if m.config.GoogleAuth != nil {
			for _, sharedDomain := range m.config.GoogleAuth.SharedDomains {
				allowedOrigins = append(allowedOrigins,
					"https://"+sharedDomain,
					"http://"+sharedDomain,
					"https://www."+sharedDomain,
					"http://www."+sharedDomain,
				)
			}
		}

		// Check if origin is allowed
		originAllowed := false
		if origin != "" {
			for _, allowedOrigin := range allowedOrigins {
				if origin == allowedOrigin {
					originAllowed = true
					w.Header().Set("Access-Control-Allow-Origin", origin)
					break
				}
			}
		}

		// If no specific origin matched, allow same host
		if !originAllowed && origin == "" {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}

		// Set other CORS headers
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With, Accept, Origin")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Max-Age", "86400") // 24 hours

		// Handle preflight OPTIONS request
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
