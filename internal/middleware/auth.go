package middleware

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"

	"github.com/JaLe29/ratelimit-simple-proxy/internal/auth"
	"github.com/JaLe29/ratelimit-simple-proxy/internal/config"
)

// AuthMiddleware handles authentication for the proxy
type AuthMiddleware struct {
	config        *config.Config
	authenticator *auth.GoogleAuthenticator
	host          string
}

// NewAuthMiddleware creates a new authentication middleware
func NewAuthMiddleware(cfg *config.Config, authenticator *auth.GoogleAuthenticator, host string) *AuthMiddleware {
	return &AuthMiddleware{
		config:        cfg,
		authenticator: authenticator,
		host:          host,
	}
}

// Handle processes the authentication middleware
func (m *AuthMiddleware) Handle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Pokud jsme na auth doméně, zpracujeme callback
		if r.Host == "auth.mojedomena.com" {
			if r.URL.Path == "/auth/callback" {
				m.handleCallback(w, r)
				return
			}
			// Jiné cesty na auth doméně nejsou povoleny
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}

		// Kontrola, zda je doména chráněná
		isProtected := false
		for _, domain := range m.config.GoogleAuth.ProtectedDomains {
			if domain == r.Host {
				isProtected = true
				break
			}
		}

		if !isProtected {
			next.ServeHTTP(w, r)
			return
		}

		target, ok := m.config.RateLimits[m.host]
		if !ok {
			http.Error(w, fmt.Sprintf("Host (%s) not found", m.host), http.StatusBadGateway)
			return
		}

		// Skip auth if no allowed emails for this domain
		if len(target.AllowedEmails) == 0 {
			next.ServeHTTP(w, r)
			return
		}

		// Handle logout
		if r.URL.Path == "/auth/logout" {
			m.authenticator.Logout(w)
			http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
			return
		}

		// Check if user is authenticated
		if !m.authenticator.IsAuthenticated(r) {
			// Vytvoříme state parametr s informací o cílové doméně
			state := base64.URLEncoding.EncodeToString([]byte(r.Host))
			// Redirect to Google login
			authURL := m.authenticator.GetAuthURL(state)
			http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (m *AuthMiddleware) handleCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "No code provided", http.StatusBadRequest)
		return
	}

	// Získáme cílovou doménu ze state parametru
	state := r.URL.Query().Get("state")
	if state == "" {
		http.Error(w, "No state provided", http.StatusBadRequest)
		return
	}

	targetDomain, err := base64.URLEncoding.DecodeString(state)
	if err != nil {
		http.Error(w, "Invalid state parameter", http.StatusBadRequest)
		return
	}

	userInfo, err := m.authenticator.GetUserInfo(code)
	if err != nil {
		http.Error(w, "Failed to get user info", http.StatusInternalServerError)
		return
	}

	// Kontrola, zda je doména chráněná
	isProtected := false
	for _, domain := range m.config.GoogleAuth.ProtectedDomains {
		if domain == string(targetDomain) {
			isProtected = true
			break
		}
	}

	if !isProtected {
		http.Error(w, "Invalid target domain", http.StatusBadRequest)
		return
	}

	target, ok := m.config.RateLimits[string(targetDomain)]
	if !ok {
		http.Error(w, fmt.Sprintf("Host (%s) not found", targetDomain), http.StatusBadGateway)
		return
	}

	// Check if email is allowed
	emailAllowed := false
	for _, allowedEmail := range target.AllowedEmails {
		if allowedEmail == userInfo.Email {
			emailAllowed = true
			break
		}
	}
	if !emailAllowed {
		http.Error(w, fmt.Sprintf("Access denied. Email %s is not authorized to access this resource.", userInfo.Email), http.StatusForbidden)
		return
	}

	// Nastavíme cookie pro cílovou doménu
	m.authenticator.SetAuthCookie(w, userInfo)

	// Přesměrujeme na cílovou doménu
	targetURL := url.URL{
		Scheme: "https",
		Host:   string(targetDomain),
		Path:   "/",
	}
	http.Redirect(w, r, targetURL.String(), http.StatusTemporaryRedirect)
}
