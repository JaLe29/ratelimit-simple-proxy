package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/JaLe29/ratelimit-simple-proxy/internal/config"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type GoogleAuthenticator struct {
	oauthConfig *oauth2.Config
	cfg         *config.Config
}

type GoogleUserInfo struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
}

func NewGoogleAuthenticator(clientID, clientSecret, redirectURL string, cfg *config.Config) *GoogleAuthenticator {
	oauthConfig := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		Endpoint: google.Endpoint,
	}

	return &GoogleAuthenticator{
		oauthConfig: oauthConfig,
		cfg:         cfg,
	}
}

// GetAuthURL generates auth URL with dynamic redirect URL based on target domain
func (ga *GoogleAuthenticator) GetAuthURL(state string, targetDomain string) string {
	// Get domain-specific auth configuration
	domainConfig, exists := ga.cfg.RateLimits[targetDomain]
	if !exists {
		// Fallback to default config
		return ga.oauthConfig.AuthCodeURL(state)
	}

	// If domain has specific auth config, use it
	if domainConfig.Auth != nil && domainConfig.Auth.RedirectURL != "" {
		// Create new OAuth config with domain-specific redirect URL
		domainOAuthConfig := &oauth2.Config{
			ClientID:     ga.oauthConfig.ClientID,
			ClientSecret: ga.oauthConfig.ClientSecret,
			RedirectURL:  domainConfig.Auth.RedirectURL,
			Scopes:       ga.oauthConfig.Scopes,
			Endpoint:     ga.oauthConfig.Endpoint,
		}
		return domainOAuthConfig.AuthCodeURL(state)
	}

	// Fallback to default config
	return ga.oauthConfig.AuthCodeURL(state)
}

// GetUserInfo gets user info using domain-specific OAuth config
func (ga *GoogleAuthenticator) GetUserInfo(code string, targetDomain string) (*GoogleUserInfo, error) {
	// Get domain-specific auth configuration
	domainConfig, exists := ga.cfg.RateLimits[targetDomain]
	if !exists {
		// Fallback to default config
		return ga.getUserInfoWithConfig(code, ga.oauthConfig)
	}

	// If domain has specific auth config, use it
	if domainConfig.Auth != nil && domainConfig.Auth.RedirectURL != "" {
		// Create new OAuth config with domain-specific redirect URL
		domainOAuthConfig := &oauth2.Config{
			ClientID:     ga.oauthConfig.ClientID,
			ClientSecret: ga.oauthConfig.ClientSecret,
			RedirectURL:  domainConfig.Auth.RedirectURL,
			Scopes:       ga.oauthConfig.Scopes,
			Endpoint:     ga.oauthConfig.Endpoint,
		}
		return ga.getUserInfoWithConfig(code, domainOAuthConfig)
	}

	// Fallback to default config
	return ga.getUserInfoWithConfig(code, ga.oauthConfig)
}

// getUserInfoWithConfig gets user info using specific OAuth config
func (ga *GoogleAuthenticator) getUserInfoWithConfig(code string, oauthConfig *oauth2.Config) (*GoogleUserInfo, error) {
	token, err := oauthConfig.Exchange(context.Background(), code)
	if err != nil {
		return nil, err
	}

	client := oauthConfig.Client(context.Background(), token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var userInfo GoogleUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, err
	}

	return &userInfo, nil
}

// GetAuthDomain returns the auth domain for a specific target domain
func (ga *GoogleAuthenticator) GetAuthDomain(targetDomain string) string {
	// Get domain-specific auth configuration
	domainConfig, exists := ga.cfg.RateLimits[targetDomain]
	if !exists {
		// Fallback to default auth domain
		return ga.cfg.GoogleAuth.AuthDomain
	}

	// If domain has specific auth config, use it
	if domainConfig.Auth != nil && domainConfig.Auth.Domain != "" {
		return domainConfig.Auth.Domain
	}

	// Fallback to default auth domain
	return ga.cfg.GoogleAuth.AuthDomain
}

func (ga *GoogleAuthenticator) SetAuthCookie(w http.ResponseWriter, userInfo *GoogleUserInfo) {
	// Set cookie for all shared domains
	for _, domain := range ga.cfg.GoogleAuth.SharedDomains {
		// Pro localhost nepoužíváme Domain parametr
		if strings.Contains(domain, "localhost") {
			http.SetCookie(w, &http.Cookie{
				Name:     "google_auth",
				Value:    userInfo.Email,
				Path:     "/",
				HttpOnly: true,
				Secure:   false, // localhost není HTTPS
				SameSite: http.SameSiteLaxMode,
				Expires:  time.Now().Add(24 * time.Hour),
			})
		} else {
			// Pro ostatní domény nastavíme Domain parametr pro lepší sdílení
			http.SetCookie(w, &http.Cookie{
				Name:     "google_auth",
				Value:    userInfo.Email,
				Path:     "/",
				Domain:   domain,
				HttpOnly: true,
				Secure:   true,
				SameSite: http.SameSiteNoneMode,
				Expires:  time.Now().Add(24 * time.Hour),
			})
		}
	}
}

func (ga *GoogleAuthenticator) IsAuthenticated(r *http.Request) bool {
	// Check cookie for current domain
	cookie, err := r.Cookie("google_auth")
	if err == nil && cookie.Value != "" {
		return true
	}

	// Check cookie for all shared domains
	for _, domain := range ga.cfg.GoogleAuth.SharedDomains {
		// Pro localhost zkontrolujeme přímo
		if strings.Contains(domain, "localhost") && r.Host == domain {
			cookie, err = r.Cookie("google_auth")
			if err == nil && cookie.Value != "" {
				return true
			}
		} else {
			// Pro ostatní domény zkontrolujeme cross-domain cookies
			if isSubdomainOrSame(r.Host, domain) {
				cookie, err = r.Cookie("google_auth")
				if err == nil && cookie.Value != "" {
					return true
				}
			}
		}
	}

	return false
}

// isSubdomainOrSame checks if host is a subdomain of domain or the same domain
func isSubdomainOrSame(host, domain string) bool {
	if host == domain {
		return true
	}

	// Remove port if present
	if idx := strings.Index(host, ":"); idx != -1 {
		host = host[:idx]
	}

	// Check if host ends with domain (for subdomains)
	return strings.HasSuffix(host, "."+domain) || host == domain
}

func (ga *GoogleAuthenticator) Logout(w http.ResponseWriter) {
	// Remove cookie for all shared domains
	for _, domain := range ga.cfg.GoogleAuth.SharedDomains {
		if strings.Contains(domain, "localhost") {
			http.SetCookie(w, &http.Cookie{
				Name:     "google_auth",
				Value:    "",
				Path:     "/",
				HttpOnly: true,
				Secure:   false,
				SameSite: http.SameSiteLaxMode,
				MaxAge:   -1,
			})
		} else {
			http.SetCookie(w, &http.Cookie{
				Name:     "google_auth",
				Value:    "",
				Path:     "/",
				Domain:   domain,
				HttpOnly: true,
				Secure:   true,
				SameSite: http.SameSiteNoneMode,
				MaxAge:   -1,
			})
		}
	}
}
