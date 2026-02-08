package main

import (
	"fmt"
	"net/http"
	"time"
)

func syncHandler(w http.ResponseWriter, r *http.Request) {
	owner := r.URL.Query().Get("owner")
	repo := r.URL.Query().Get("repo")
	token := r.Header.Get("Authorization")

	fmt.Printf("🔄 Sync request: owner=%s, repo=%s, token=%s\n", owner, repo, token[:20]+"...")

	if owner == "" || repo == "" || token == "" {
		fmt.Println("❌ Missing parameters!")
		http.Error(w, "Missing owner / repo / token", http.StatusBadRequest)
		return
	}

	fmt.Println("📡 Fetching commits from GitHub...")
	err := SyncFromGitHub(owner, repo, token)
	if err != nil {
		fmt.Printf("❌ Sync failed: %v\n", err)
		http.Error(w, "Sync failed", http.StatusInternalServerError)
		return
	}
	fmt.Println("✅ Commits fetched successfully")

	// Save repo under user
	fmt.Println("💾 Saving repo under user...")
	DB.Exec(`
	INSERT INTO user_repos (user_id, repo_name, last_synced)
	SELECT id, ?, CURRENT_TIMESTAMP
	FROM users
	WHERE access_token = ?
	`, repo, token)

	fmt.Println("📊 Creating snapshot...")

	// Check if this is the first sync for this repo
	var existingSnapshots int
	DB.QueryRow(`SELECT COUNT(*) FROM repo_snapshots WHERE repo_name = ?`, repo).Scan(&existingSnapshots)

	if existingSnapshots == 0 {
		fmt.Println("🎯 First sync detected! Generating 30 days of historical snapshots...")
		generateHistoricalSnapshots(repo, 30)
	} else {
		fmt.Println("📊 Creating today's snapshot...")
		saveSnapshot(repo)
	}

	if shouldNotify(repo) {
		fmt.Println("🔔 Significant activity detected!")
		w.Write([]byte("🔔 Significant activity detected"))
		return
	}

	fmt.Println("✅ Sync completed successfully")
	w.Write([]byte("Synced successfully"))
}

func saveSnapshot(repo string) {
	saveSnapshotForDate(repo, "")
}

func saveSnapshotForDate(repo string, referenceDate string) {
	// If no date provided, use current time
	dateClause := "julianday('now')"
	if referenceDate != "" {
		dateClause = fmt.Sprintf("julianday('%s')", referenceDate)
	}

	// First check how many rows exist
	var count int
	DB.QueryRow(`SELECT COUNT(*) FROM file_activity`).Scan(&count)

	if referenceDate == "" {
		fmt.Printf("  📊 Total file_activity records: %d\n", count)
	}

	query := fmt.Sprintf(`
		SELECT
			SUM(CASE WHEN %s - julianday(last_modified) <= 7 THEN 1 ELSE 0 END),
			SUM(CASE WHEN %s - julianday(last_modified) BETWEEN 7 AND 30 THEN 1 ELSE 0 END),
			SUM(CASE WHEN %s - julianday(last_modified) > 30 THEN 1 ELSE 0 END)
		FROM file_activity
	`, dateClause, dateClause, dateClause)

	row := DB.QueryRow(query)

	var active, stable, inactive int
	row.Scan(&active, &stable, &inactive)

	if referenceDate == "" {
		fmt.Printf("  📊 Snapshot: active=%d, stable=%d, inactive=%d\n", active, stable, inactive)
	}

	total := active + stable + inactive
	score := 0.0
	if total > 0 {
		score = (float64(active) / float64(total)) * 100
	}

	// Insert with custom timestamp if provided
	if referenceDate != "" {
		DB.Exec(`
			INSERT INTO repo_snapshots
			(repo_name, active_files, stable_files, inactive_files, activity_score, created_at)
			VALUES (?, ?, ?, ?, ?, ?)
		`, repo, active, stable, inactive, score, referenceDate)
	} else {
		DB.Exec(`
			INSERT INTO repo_snapshots
			(repo_name, active_files, stable_files, inactive_files, activity_score)
			VALUES (?, ?, ?, ?, ?)
		`, repo, active, stable, inactive, score)
	}
}

func generateHistoricalSnapshots(repo string, days int) {
	now := time.Now()

	// Generate snapshots for past N days
	for i := days; i >= 0; i-- {
		historicalDate := now.AddDate(0, 0, -i)
		dateStr := historicalDate.Format("2006-01-02 15:04:05")

		saveSnapshotForDate(repo, dateStr)

		if i%10 == 0 {
			fmt.Printf("  ✅ Generated snapshots up to %d days ago\n", i)
		}
	}

	fmt.Printf("  🎉 Generated %d historical snapshots!\n", days+1)
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
