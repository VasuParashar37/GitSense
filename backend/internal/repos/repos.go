package repos

import (
	"encoding/json"
	"net/http"

	"gitsense"
	"gitsense/internal/auth"
)

type Repo struct {
	Name  string `json:"name"`
	Owner struct {
		Login string `json:"login"`
	} `json:"owner"`
}

func GetUserRepos(w http.ResponseWriter, r *http.Request) {
	sessionToken, err := auth.ExtractSessionToken(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	githubToken, _, err := auth.ResolveGitHubToken(sessionToken)
	if err != nil {
		http.Error(w, "Invalid session", http.StatusUnauthorized)
		return
	}

	req, err := gitsense.CreateGitHubRequest("GET", "https://api.github.com/user/repos", githubToken)
	if err != nil {
		http.Error(w, "Failed to create GitHub request", http.StatusInternalServerError)
		return
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		http.Error(w, "GitHub error", 500)
		return
	}
	defer resp.Body.Close()

	var repos []Repo
	json.NewDecoder(resp.Body).Decode(&repos)

	json.NewEncoder(w).Encode(repos)
}
