package main

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

func syncHandler(w http.ResponseWriter, r *http.Request) {
	owner := r.URL.Query().Get("owner")
	repo := r.URL.Query().Get("repo")
	token := r.Header.Get("Authorization")

	if owner == "" || repo == "" || token == "" {
		http.Error(w, "Missing owner / repo / token", http.StatusBadRequest)
		return
	}

	fmt.Printf("🔄 Syncing %s/%s\n", owner, repo)

	// Count commits before sync
	var before int
	DB.QueryRow(
		`SELECT COUNT(*) FROM commits WHERE repo_name = ?`,
		repo,
	).Scan(&before)

	// Fetch from GitHub
	if err := SyncFromGitHub(owner, repo, token); err != nil {
		http.Error(w, "Sync failed", http.StatusInternalServerError)
		return
	}

	// Count commits after sync
	var after int
	DB.QueryRow(
		`SELECT COUNT(*) FROM commits WHERE repo_name = ?`,
		repo,
	).Scan(&after)

	newCommits := after - before

	// Save repo under user
	DB.Exec(`
		INSERT INTO user_repos (user_id, repo_name, last_synced)
		SELECT id, ?, CURRENT_TIMESTAMP
		FROM users
		WHERE access_token = ?
	`, repo, token)

	// Snapshot handling
	var snapshotCount int
	DB.QueryRow(
		`SELECT COUNT(*) FROM repo_snapshots WHERE repo_name = ?`,
		repo,
	).Scan(&snapshotCount)

	fmt.Printf("📊 Snapshot count for '%s': %d\n", repo, snapshotCount)

	if snapshotCount == 0 {
		fmt.Println("🎯 First sync detected - generating historical snapshots...")
		generateHistoricalSnapshots(repo, 30)
	} else {
		fmt.Println("📊 Creating today's snapshot...")
		saveSnapshot(repo)
	}

	// Notify if new commits exist
	if newCommits > 0 {
		msg := fmt.Sprintf("🔔 %d new commit(s) detected", newCommits)
		w.Write([]byte(msg))
		return
	}

	w.Write([]byte("Synced successfully"))
}

// ----------------------------
// SNAPSHOT HELPERS
// ----------------------------
func saveSnapshot(repo string) {
	saveSnapshotForDate(repo, "")
}

func saveSnapshotForDate(repo string, referenceDate string) {
	dateExpr := "julianday('now')"
	if referenceDate != "" {
		dateExpr = fmt.Sprintf("julianday('%s')", referenceDate)
	}

	row := DB.QueryRow(fmt.Sprintf(`
		SELECT
			SUM(CASE WHEN %s - julianday(last_modified) <= 7 THEN 1 ELSE 0 END),
			SUM(CASE WHEN %s - julianday(last_modified) BETWEEN 7 AND 30 THEN 1 ELSE 0 END),
			SUM(CASE WHEN %s - julianday(last_modified) > 30 THEN 1 ELSE 0 END)
		FROM file_activity
		WHERE repo_name = ?
	`, dateExpr, dateExpr, dateExpr), repo)

	var active, stable, inactive int
	row.Scan(&active, &stable, &inactive)

	total := active + stable + inactive
	score := 0.0
	if total > 0 {
		score = (float64(active) / float64(total)) * 100
	}

	// Retry logic for SQLITE_BUSY errors
	var err error
	maxRetries := 5
	retryDelay := 100 * time.Millisecond

	for attempt := 0; attempt < maxRetries; attempt++ {
		if referenceDate != "" {
			_, err = DB.Exec(`
				INSERT INTO repo_snapshots
				(repo_name, active_files, stable_files, inactive_files, activity_score, created_at)
				VALUES (?, ?, ?, ?, ?, ?)
			`, repo, active, stable, inactive, score, referenceDate)
		} else {
			_, err = DB.Exec(`
				INSERT INTO repo_snapshots
				(repo_name, active_files, stable_files, inactive_files, activity_score)
				VALUES (?, ?, ?, ?, ?)
			`, repo, active, stable, inactive, score)
		}

		// If success, break
		if err == nil {
			break
		}

		// Check if it's a database locked error
		errMsg := err.Error()
		if strings.Contains(errMsg, "database is locked") || strings.Contains(errMsg, "SQLITE_BUSY") {
			// Database is busy, wait and retry
			if attempt < maxRetries-1 {
				time.Sleep(retryDelay)
				retryDelay *= 2 // Exponential backoff
			}
		} else {
			break // Non-busy error, don't retry
		}
	}

	if err != nil {
		fmt.Printf("❌ Failed to save snapshot for '%s' (date: %s): %v\n", repo, referenceDate, err)
	} else {
		fmt.Printf("✅ Saved snapshot for '%s' - Score: %.1f (date: %s)\n", repo, score, referenceDate)
	}
}

// ----------------------------
// HISTORICAL SNAPSHOTS (FROM COMMITS)
// ----------------------------
func generateHistoricalSnapshots(repo string, days int) {
	fmt.Printf("📅 Generating historical snapshots for '%s' (past %d days)...\n", repo, days)
	start := time.Now().AddDate(0, 0, -days)

	rows, err := DB.Query(`
		SELECT DISTINCT DATE(commit_date)
		FROM commits
		WHERE repo_name = ?
		  AND julianday(commit_date) >= julianday(?)
		ORDER BY DATE(commit_date)
	`, repo, start.Format("2006-01-02"))

	if err != nil {
		fmt.Println("❌ History snapshot error:", err)
		return
	}
	defer rows.Close()

	var dates []string
	for rows.Next() {
		var d string
		rows.Scan(&d)
		dates = append(dates, d)
		saveSnapshotForDate(repo, d+" 23:59:59")
	}

	fmt.Printf("✅ Generated %d snapshots for dates: %v\n", len(dates), dates)
}
