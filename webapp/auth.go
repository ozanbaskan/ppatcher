package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

var oauthConfig *oauth2.Config

func InitOAuth() {
	oauthConfig = &oauth2.Config{
		ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		RedirectURL:  os.Getenv("APP_URL") + "/auth/google/callback",
		Scopes:       []string{"openid", "email", "profile"},
		Endpoint:     google.Endpoint,
	}
}

func handleGoogleLogin(w http.ResponseWriter, r *http.Request) {
	state, err := generateState()
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	session, _ := store.Get(r, "session")
	session.Values["oauth_state"] = state
	session.Save(r, w)

	url := oauthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func handleGoogleCallback(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session")

	savedState, ok := session.Values["oauth_state"].(string)
	if !ok || savedState == "" || savedState != r.URL.Query().Get("state") {
		http.Error(w, "Invalid OAuth state", http.StatusBadRequest)
		return
	}
	delete(session.Values, "oauth_state")

	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "Missing code", http.StatusBadRequest)
		return
	}

	token, err := oauthConfig.Exchange(r.Context(), code)
	if err != nil {
		log.Printf("OAuth exchange error: %v", err)
		http.Error(w, "Authentication failed", http.StatusInternalServerError)
		return
	}

	userInfo, err := fetchGoogleUserInfo(r.Context(), token)
	if err != nil {
		log.Printf("Fetch user info error: %v", err)
		http.Error(w, "Failed to get user info", http.StatusInternalServerError)
		return
	}

	user, err := upsertUser(r.Context(), userInfo)
	if err != nil {
		log.Printf("Upsert user error: %v", err)
		http.Error(w, "Failed to save user", http.StatusInternalServerError)
		return
	}

	session.Values["user_id"] = user.ID
	session.Save(r, w)

	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

func handleLogout(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session")
	session.Options.MaxAge = -1
	session.Save(r, w)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

type googleUserInfo struct {
	Sub       string `json:"sub"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	Picture   string `json:"picture"`
	Verified  bool   `json:"email_verified"`
}

func fetchGoogleUserInfo(ctx context.Context, token *oauth2.Token) (*googleUserInfo, error) {
	client := oauthConfig.Client(ctx, token)
	resp, err := client.Get("https://openidconnect.googleapis.com/v1/userinfo")
	if err != nil {
		return nil, fmt.Errorf("get userinfo: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("userinfo returned status %d", resp.StatusCode)
	}

	var info googleUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("decode userinfo: %w", err)
	}

	if !info.Verified {
		return nil, fmt.Errorf("email not verified")
	}

	return &info, nil
}

func upsertUser(ctx context.Context, info *googleUserInfo) (*User, error) {
	var user User
	err := db.QueryRow(ctx,
		`INSERT INTO users (google_id, email, name, avatar_url)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (google_id) DO UPDATE SET
		   email = EXCLUDED.email,
		   name = EXCLUDED.name,
		   avatar_url = EXCLUDED.avatar_url,
		   updated_at = NOW()
		 RETURNING id, google_id, email, name, avatar_url, created_at, updated_at`,
		info.Sub, info.Email, info.Name, info.Picture,
	).Scan(&user.ID, &user.GoogleID, &user.Email, &user.Name, &user.AvatarURL, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func generateState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
