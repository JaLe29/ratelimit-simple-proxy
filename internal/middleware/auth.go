package middleware

import (
	"encoding/base64"
	"fmt"
	"html/template"
	"net/http"
	"net/url"

	"github.com/JaLe29/ratelimit-simple-proxy/internal/auth"
	"github.com/JaLe29/ratelimit-simple-proxy/internal/config"
	"github.com/JaLe29/ratelimit-simple-proxy/internal/templates"
)

// AuthMiddleware handles authentication for the proxy
type AuthMiddleware struct {
	config        *config.Config
	authenticator *auth.GoogleAuthenticator
	host          string
	loginTemplate *template.Template
}

// NewAuthMiddleware creates a new authentication middleware
func NewAuthMiddleware(cfg *config.Config, authenticator *auth.GoogleAuthenticator, host string, loginTemplate *template.Template) *AuthMiddleware {
	return &AuthMiddleware{
		config:        cfg,
		authenticator: authenticator,
		host:          host,
		loginTemplate: loginTemplate,
	}
}

// Handle processes the authentication middleware
func (m *AuthMiddleware) Handle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// If we're on the auth domain, process the callback
		if r.Host == m.config.GoogleAuth.AuthDomain {
			if r.URL.Path == "/auth/callback" {
				m.handleCallback(w, r)
				return
			}
			// Other paths on auth domain are not allowed
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}

		// Check if the domain is protected
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
			// Serve login page instead of direct redirect
			m.serveLoginPage(w, r)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// serveLoginPage serves the login page with Google login button
func (m *AuthMiddleware) serveLoginPage(w http.ResponseWriter, r *http.Request) {
	// Create state parameter with target domain information
	state := base64.URLEncoding.EncodeToString([]byte(r.Host))
	// Generate Google auth URL
	authURL := m.authenticator.GetAuthURL(state)

	// Set content type
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// Render and serve the login template
	data := templates.TemplateData{
		AuthURL: authURL,
	}

	err := m.loginTemplate.Execute(w, data)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

func (m *AuthMiddleware) handleCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "No code provided", http.StatusBadRequest)
		return
	}

	// Get target domain from state parameter
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

	// Check if the domain is protected
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

	// Set cookie for all subdomains
	m.authenticator.SetAuthCookie(w, userInfo)

	// Redirect to target domain
	targetURL := url.URL{
		Scheme: "https",
		Host:   string(targetDomain),
		Path:   "/",
	}
	http.Redirect(w, r, targetURL.String(), http.StatusTemporaryRedirect)
}
