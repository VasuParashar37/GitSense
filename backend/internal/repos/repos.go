package repos

import (
	"encoding/json"
	"net/http"

	"gitsense"
)

type Repo struct {
	Name  string `json:"name"`
	Owner struct {
		Login string `json:"login"`
	} `json:"owner"`
}

func GetUserRepos(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("Authorization")
	if token == "" {
		http.Error(w, "Unauthorized", 401)
		return
	}

	req, err := gitsense.CreateGitHubRequest("GET", "https://api.github.com/user/repos", token)
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
