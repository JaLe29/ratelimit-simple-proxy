package middleware

import (
	"fmt"
	"net/http"

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
		target, ok := m.config.RateLimits[m.host]
		if !ok {
			http.Error(w, fmt.Sprintf("Host (%s) not found", m.host), http.StatusBadGateway)
			return
		}

		// Skip auth if not enabled
		if target.GoogleAuth == nil || !target.GoogleAuth.Enabled {
			next.ServeHTTP(w, r)
			return
		}

		// Handle logout
		if r.URL.Path == "/auth/logout" {
			m.authenticator.Logout(w)
			http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
			return
		}

		// Handle OAuth callback
		if r.URL.Path == "/auth/callback" {
			m.handleCallback(w, r)
			return
		}

		// Check if user is authenticated
		if !m.authenticator.IsAuthenticated(r) {
			// Redirect to Google login
			authURL := m.authenticator.GetAuthURL("state")
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

	userInfo, err := m.authenticator.GetUserInfo(code)
	if err != nil {
		http.Error(w, "Failed to get user info", http.StatusInternalServerError)
		return
	}

	target := m.config.RateLimits[m.host]
	// Check if email is allowed
	if len(target.GoogleAuth.AllowedEmails) > 0 {
		emailAllowed := false
		for _, allowedEmail := range target.GoogleAuth.AllowedEmails {
			if allowedEmail == userInfo.Email {
				emailAllowed = true
				break
			}
		}
		if !emailAllowed {
			http.Error(w, fmt.Sprintf("Access denied. Email %s is not authorized to access this resource.", userInfo.Email), http.StatusForbidden)
			return
		}
	}

	m.authenticator.SetAuthCookie(w, userInfo)
	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}
