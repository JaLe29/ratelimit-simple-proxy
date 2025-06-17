package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/JaLe29/ratelimit-simple-proxy/internal/config"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type GoogleAuthenticator struct {
	config *oauth2.Config
	cfg    *config.Config
}

type GoogleUserInfo struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
}

func NewGoogleAuthenticator(clientID, clientSecret, redirectURL string, cfg *config.Config) *GoogleAuthenticator {
	config := &oauth2.Config{
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
		config: config,
		cfg:    cfg,
	}
}

func (ga *GoogleAuthenticator) GetAuthURL(state string) string {
	return ga.config.AuthCodeURL(state)
}

func (ga *GoogleAuthenticator) GetUserInfo(code string) (*GoogleUserInfo, error) {
	token, err := ga.config.Exchange(context.Background(), code)
	if err != nil {
		return nil, err
	}

	client := ga.config.Client(context.Background(), token)
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

func (ga *GoogleAuthenticator) SetAuthCookie(w http.ResponseWriter, userInfo *GoogleUserInfo) {
	// Set cookie for all shared domains
	for _, domain := range ga.cfg.GoogleAuth.SharedDomains {
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

func (ga *GoogleAuthenticator) IsAuthenticated(r *http.Request) bool {
	// Check cookie for current domain
	cookie, err := r.Cookie("google_auth")
	if err == nil && cookie.Value != "" {
		return true
	}

	// Check cookie for all shared domains
	for _, domain := range ga.cfg.GoogleAuth.SharedDomains {
		// Create new request with modified host for cookie check
		req := r.Clone(r.Context())
		req.Host = domain
		cookie, err = req.Cookie("google_auth")
		if err == nil && cookie.Value != "" {
			return true
		}
	}

	return false
}

func (ga *GoogleAuthenticator) Logout(w http.ResponseWriter) {
	// Remove cookie for all shared domains
	for _, domain := range ga.cfg.GoogleAuth.SharedDomains {
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
