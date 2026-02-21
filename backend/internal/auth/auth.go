package auth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"

	"gitsense/internal/db"
	githubapi "gitsense/internal/github"
)

// Step 1: Redirect to GitHub
func GithubLogin(w http.ResponseWriter, r *http.Request) {
	githubClientID := os.Getenv("GITHUB_CLIENT_ID")
	origin := r.URL.Query().Get("origin")

	params := url.Values{}
	params.Set("client_id", githubClientID)
	params.Set("scope", "repo")
	if origin != "" {
		// Reuse GitHub OAuth state to carry extension origin through callback.
		params.Set("state", origin)
	}

	url := fmt.Sprintf(
		"https://github.com/login/oauth/authorize?%s",
		params.Encode(),
	)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

// Step 2: Callback from GitHub
func GithubCallback(w http.ResponseWriter, r *http.Request) {
	githubClientID := os.Getenv("GITHUB_CLIENT_ID")
	githubClientSecret := os.Getenv("GITHUB_CLIENT_SECRET")

	code := r.URL.Query().Get("code")
	origin := r.URL.Query().Get("state")
	if code == "" {
		http.Error(w, "Missing code", 400)
		return
	}

	// Exchange code for token
	req, err := http.NewRequest("POST",
		"https://github.com/login/oauth/access_token",
		nil,
	)
	if err != nil {
		fmt.Println("❌ Failed to create OAuth request:", err)
		http.Error(w, "Failed to create request", 500)
		return
	}

	q := req.URL.Query()
	q.Add("client_id", githubClientID)
	q.Add("client_secret", githubClientSecret)
	q.Add("code", code)
	req.URL.RawQuery = q.Encode()
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("❌ OAuth request failed:", err)
		http.Error(w, "OAuth failed", 500)
		return
	}
	defer resp.Body.Close()

	var result map[string]string
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		fmt.Println("❌ Failed to decode OAuth response:", err)
		http.Error(w, "Invalid OAuth response", 500)
		return
	}

	token := result["access_token"]
	if token == "" {
		fmt.Println("❌ No access_token in response")
		http.Error(w, "No token received", 500)
		return
	}

	// ✅ CALL HELPER FROM github.go
	username := githubapi.GetGitHubUsername(token)

	// ✅ SAVE USER
	_, err = db.DB.Exec(`
		INSERT INTO users (github_username, access_token)
		VALUES (?, ?)
		ON CONFLICT(github_username)
		DO UPDATE SET access_token = ?
	`, username, token, token)

	if err != nil {
		http.Error(w, "DB error", 500)
		return
	}

	// Send token back to extension
	backendURL := os.Getenv("BACKEND_URL")
	if backendURL == "" {
		// Default to localhost for development
		backendURL = "http://localhost:8080"
	}

	redirectURL := fmt.Sprintf("%s/token?value=%s", backendURL, url.QueryEscape(token))
	if origin != "" {
		redirectURL += "&origin=" + url.QueryEscape(origin)
	}
	http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
}
