package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

var githubClientID = os.Getenv("GITHUB_CLIENT_ID")
var githubClientSecret = os.Getenv("GITHUB_CLIENT_SECRET")


// Step 1: Redirect to GitHub
func githubLogin(w http.ResponseWriter, r *http.Request) {
	url := fmt.Sprintf(
		"https://github.com/login/oauth/authorize?client_id=%s&scope=repo",
		githubClientID,
	)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

// Step 2: Callback from GitHub
func githubCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "Missing code", 400)
		return
	}

	// Exchange code for token
	req, _ := http.NewRequest("POST",
		"https://github.com/login/oauth/access_token",
		nil,
	)

	q := req.URL.Query()
	q.Add("client_id", githubClientID)
	q.Add("client_secret", githubClientSecret)
	q.Add("code", code)
	req.URL.RawQuery = q.Encode()
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		http.Error(w, "OAuth failed", 500)
		return
	}
	defer resp.Body.Close()

	var result map[string]string
	json.NewDecoder(resp.Body).Decode(&result)

	// 🔍 DEBUG: Print what GitHub returned
	fmt.Println("GitHub OAuth Response:")
	fmt.Printf("  Client ID: %s\n", githubClientID)
	fmt.Printf("  Response: %+v\n", result)

	token := result["access_token"]
	if token == "" {
		fmt.Println("❌ No access_token in response!")
		http.Error(w, "No token received", 500)
		return
	}

	// ✅ CALL HELPER FROM github.go
	username := getGitHubUsername(token)

	// ✅ SAVE USER
	_, err = DB.Exec(`
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
	http.Redirect(
		w,
		r,
		// "https://gitsense-ooly.onrender.com/token?value=" + token,
		"http://localhost:8080/token?value=" + token,
		http.StatusTemporaryRedirect,
	)
}
