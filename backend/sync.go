package main

import (
	"net/http"
)

func syncHandler(w http.ResponseWriter, r *http.Request) {
	owner := r.URL.Query().Get("owner")
	repo := r.URL.Query().Get("repo")
	token := r.Header.Get("Authorization")

	if owner == "" || repo == "" || token == "" {
		http.Error(w, "Missing owner / repo / token", http.StatusBadRequest)
		return
	}

	err := SyncFromGitHub(owner, repo, token)
	if err != nil {
		http.Error(w, "Sync failed", http.StatusInternalServerError)
		return
	}

	// Save repo under user
	DB.Exec(`
	INSERT INTO user_repos (user_id, repo_name, last_synced)
	SELECT id, ?, CURRENT_TIMESTAMP
	FROM users
	WHERE access_token = ?
	`, repo, token)

	saveSnapshot(repo)

	if shouldNotify(repo) {
		w.Write([]byte("🔔 Significant activity detected"))
		return
	}

	w.Write([]byte("Synced successfully"))
}

func saveSnapshot(repo string) {
	row := DB.QueryRow(`
		SELECT 
			SUM(CASE WHEN julianday('now') - julianday(last_modified) <= 7 THEN 1 ELSE 0 END),
			SUM(CASE WHEN julianday('now') - julianday(last_modified) BETWEEN 7 AND 30 THEN 1 ELSE 0 END),
			SUM(CASE WHEN julianday('now') - julianday(last_modified) > 30 THEN 1 ELSE 0 END)
		FROM file_activity
	`)

	var active, stable, inactive int
	row.Scan(&active, &stable, &inactive)

	total := active + stable + inactive
	score := 0.0
	if total > 0 {
		score = (float64(active) / float64(total)) * 100
	}

	DB.Exec(`
		INSERT INTO repo_snapshots
		(repo_name, active_files, stable_files, inactive_files, activity_score)
		VALUES (?, ?, ?, ?, ?)
	`, repo, active, stable, inactive, score)
}

func shouldNotify(repo string) bool {
	row := DB.QueryRow(`
		SELECT activity_score
		FROM repo_snapshots
		WHERE repo_name = ?
		ORDER BY created_at DESC
		LIMIT 2
	`, repo)

	var latest, previous float64

	err := row.Scan(&latest)
	if err != nil {
		return false
	}

	row = DB.QueryRow(`
		SELECT activity_score
		FROM repo_snapshots
		WHERE repo_name = ?
		ORDER BY created_at DESC
		LIMIT 1 OFFSET 1
	`, repo)

	err = row.Scan(&previous)
	if err != nil {
		return false
	}

	// Notify only if change > 10%
	diff := latest - previous
	if diff < 0 {
		diff = -diff
	}

	return diff >= 10
}
