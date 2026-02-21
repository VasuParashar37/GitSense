package commits

import (
	"encoding/json"
	"net/http"

	"gitsense"
	"gitsense/internal/db"
)

func GetCommits(w http.ResponseWriter, r *http.Request) {
	gitsense.SetCORSHeaders(w)

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Validate repo parameter
	repo, err := gitsense.ValidateRepoParam(r)
	if err != nil {
		gitsense.SendJSONError(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validate limit parameter
	limit, err := gitsense.ValidateLimitParam(r)
	if err != nil {
		gitsense.SendJSONError(w, err.Error(), http.StatusBadRequest)
		return
	}

	query := `
		SELECT commit_sha, author, message, commit_date
		FROM commits
		WHERE repo_name = ?
		ORDER BY commit_date DESC
		LIMIT ?
	`

	rows, err := db.DB.Query(query, repo, limit)

	if err != nil {
		gitsense.SendJSONError(w, "Database error", http.StatusInternalServerError)
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
		if err := rows.Scan(&c.SHA, &c.Author, &c.Message, &c.Date); err != nil {
			gitsense.SendJSONError(w, "Failed to scan commit data", http.StatusInternalServerError)
			return
		}
		commits = append(commits, c)
	}

	if err := rows.Err(); err != nil {
		gitsense.SendJSONError(w, "Database iteration error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(commits); err != nil {
		gitsense.SendJSONError(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}
