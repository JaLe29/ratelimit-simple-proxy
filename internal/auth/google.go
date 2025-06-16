package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type GoogleAuthenticator struct {
	config *oauth2.Config
}

type GoogleUserInfo struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
}

func NewGoogleAuthenticator(clientID, clientSecret, redirectURL string) *GoogleAuthenticator {
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
	// Create a session cookie with user info
	http.SetCookie(w, &http.Cookie{
		Name:     "google_auth",
		Value:    userInfo.Email,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteNoneMode,
		Expires:  time.Now().Add(24 * time.Hour),
	})
}

func (ga *GoogleAuthenticator) IsAuthenticated(r *http.Request) bool {
	cookie, err := r.Cookie("google_auth")
	return err == nil && cookie.Value != ""
}

func (ga *GoogleAuthenticator) Logout(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     "google_auth",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1, // Okamžité vypršení cookie
	})
}
