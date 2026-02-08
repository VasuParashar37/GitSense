package main

import (
	"encoding/json"
	"net/http"
)

func getCommits(w http.ResponseWriter, r *http.Request) {
	repo := r.URL.Query().Get("repo")
	if repo == "" {
		http.Error(w, "Repo is required", http.StatusBadRequest)
		return
	}

	rows, err := DB.Query(`
		SELECT author, message, commit_date
		FROM commits
		WHERE repo_name = ?
		ORDER BY commit_date DESC
		LIMIT 20
	`, repo)

	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type CommitResponse struct {
		Author string `json:"author"`
		Message string `json:"message"`
		Date string `json:"date"`
	}

	var commits []CommitResponse

	for rows.Next() {
		var c CommitResponse
		rows.Scan(&c.Author, &c.Message, &c.Date)
		commits = append(commits, c)
	}

	json.NewEncoder(w).Encode(commits)
}
