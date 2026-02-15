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

	// Get optional limit parameter (default: all commits)
	limit := r.URL.Query().Get("limit")

	query := `
		SELECT commit_sha, author, message, commit_date
		FROM commits
		WHERE repo_name = ?
		ORDER BY commit_date DESC
	`

	if limit != "" {
		query += " LIMIT " + limit
	}

	rows, err := DB.Query(query, repo)

	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type CommitResponse struct {
		SHA     string `json:"sha"`
		Author  string `json:"author"`
		Message string `json:"message"`
		Date    string `json:"date"`
	}

	var commits []CommitResponse

	for rows.Next() {
		var c CommitResponse
		rows.Scan(&c.SHA, &c.Author, &c.Message, &c.Date)
		commits = append(commits, c)
	}

	json.NewEncoder(w).Encode(commits)
}
