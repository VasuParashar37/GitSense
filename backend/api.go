package main

import (
	"encoding/json"
	"net/http"
)

type Commit struct {
    ID         int    `json:"id"`
    Hash       string `json:"hash"`
    Message    string `json:"message"`
    CommitTime string `json:"commit_time"`
}
type Stats struct {
	TotalCommits int    `json:"total_commits"`
	LatestCommit string `json:"latest_commit"`
	LastUpdated  string `json:"last_updated"`
}

// Health check API
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Server is running fine 🚀"))
}

// Get commits API
func getCommitsHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := DB.Query(
		"SELECT id, hash, message, commit_time FROM commits ORDER BY datetime(commit_time) DESC LIMIT 20",
	)
	if err != nil {
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var commits []Commit

	for rows.Next() {
		var c Commit
		rows.Scan(&c.ID, &c.Hash, &c.Message, &c.CommitTime)
		commits = append(commits, c)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(commits)
}

func getStatsHandler(w http.ResponseWriter, r *http.Request) {
	var stats Stats

	// Total commits
	err := DB.QueryRow("SELECT COUNT(*) FROM commits").
		Scan(&stats.TotalCommits)
	if err != nil {
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}

	// Latest commit
	err = DB.QueryRow(
		"SELECT message, commit_time FROM commits ORDER BY datetime(commit_time) DESC LIMIT 1",
	).Scan(&stats.LatestCommit, &stats.LastUpdated)

	if err != nil {
		stats.LatestCommit = "No commits yet"
		stats.LastUpdated = "-"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}


