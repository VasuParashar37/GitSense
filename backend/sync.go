// package main

// import (
// 	"fmt"
// 	"net/http"
// 	"time"
// )

// func syncHandler(w http.ResponseWriter, r *http.Request) {
// 	owner := r.URL.Query().Get("owner")
// 	repo := r.URL.Query().Get("repo")
// 	token := r.Header.Get("Authorization")

// 	fmt.Printf("🔄 Sync request: owner=%s, repo=%s, token=%s\n", owner, repo, token[:20]+"...")

// 	if owner == "" || repo == "" || token == "" {
// 		fmt.Println("❌ Missing parameters!")
// 		http.Error(w, "Missing owner / repo / token", http.StatusBadRequest)
// 		return
// 	}

// 	fmt.Println("📡 Fetching commits from GitHub...")
// 	err := SyncFromGitHub(owner, repo, token)
// 	if err != nil {
// 		fmt.Printf("❌ Sync failed: %v\n", err)
// 		http.Error(w, "Sync failed", http.StatusInternalServerError)
// 		return
// 	}
// 	fmt.Println("✅ Commits fetched successfully")

// 	// Save repo under user
// 	fmt.Println("💾 Saving repo under user...")
// 	DB.Exec(`
// 	INSERT INTO user_repos (user_id, repo_name, last_synced)
// 	SELECT id, ?, CURRENT_TIMESTAMP
// 	FROM users
// 	WHERE access_token = ?
// 	`, repo, token)

// 	fmt.Println("📊 Creating snapshot...")

// 	// Check if this is the first sync for this repo
// 	var existingSnapshots int
// 	DB.QueryRow(`SELECT COUNT(*) FROM repo_snapshots WHERE repo_name = ?`, repo).Scan(&existingSnapshots)

// 	if existingSnapshots == 0 {
// 		fmt.Println("🎯 First sync detected! Generating 30 days of historical snapshots...")
// 		generateHistoricalSnapshots(repo, 30)
// 	} else {
// 		fmt.Println("📊 Creating today's snapshot...")
// 		saveSnapshot(repo)
// 	}

// 	if shouldNotify(repo) {
// 		fmt.Println("🔔 Significant activity detected!")
// 		w.Write([]byte("🔔 Significant activity detected"))
// 		return
// 	}

// 	fmt.Println("✅ Sync completed successfully")
// 	w.Write([]byte("Synced successfully"))
// }

// func saveSnapshot(repo string) {
// 	saveSnapshotForDate(repo, "")
// }

// func saveSnapshotForDate(repo string, referenceDate string) {
// 	// If no date provided, use current time
// 	dateClause := "julianday('now')"
// 	if referenceDate != "" {
// 		dateClause = fmt.Sprintf("julianday('%s')", referenceDate)
// 	}

// 	// First check how many rows exist
// 	var count int
// 	DB.QueryRow(`SELECT COUNT(*) FROM file_activity`).Scan(&count)

// 	if referenceDate == "" {
// 		fmt.Printf("  📊 Total file_activity records: %d\n", count)
// 	}

// 	query := fmt.Sprintf(`
// 		SELECT
// 			SUM(CASE WHEN %s - julianday(last_modified) <= 7 THEN 1 ELSE 0 END),
// 			SUM(CASE WHEN %s - julianday(last_modified) BETWEEN 7 AND 30 THEN 1 ELSE 0 END),
// 			SUM(CASE WHEN %s - julianday(last_modified) > 30 THEN 1 ELSE 0 END)
// 		FROM file_activity
// 	`, dateClause, dateClause, dateClause)

// 	row := DB.QueryRow(query)

// 	var active, stable, inactive int
// 	row.Scan(&active, &stable, &inactive)

// 	if referenceDate == "" {
// 		fmt.Printf("  📊 Snapshot: active=%d, stable=%d, inactive=%d\n", active, stable, inactive)
// 	}

// 	total := active + stable + inactive
// 	score := 0.0
// 	if total > 0 {
// 		score = (float64(active) / float64(total)) * 100
// 	}

// 	// Insert with custom timestamp if provided
// 	if referenceDate != "" {
// 		DB.Exec(`
// 			INSERT INTO repo_snapshots
// 			(repo_name, active_files, stable_files, inactive_files, activity_score, created_at)
// 			VALUES (?, ?, ?, ?, ?, ?)
// 		`, repo, active, stable, inactive, score, referenceDate)
// 	} else {
// 		DB.Exec(`
// 			INSERT INTO repo_snapshots
// 			(repo_name, active_files, stable_files, inactive_files, activity_score)
// 			VALUES (?, ?, ?, ?, ?)
// 		`, repo, active, stable, inactive, score)
// 	}
// }

// func generateHistoricalSnapshots(repo string, days int) {
// 	now := time.Now()
// 	thirtyDaysAgo := now.AddDate(0, 0, -days)

// 	// Get unique commit dates from the past 30 days (only days with actual commits)
// 	rows, err := DB.Query(`
// 		SELECT DISTINCT DATE(last_modified) as commit_date
// 		FROM file_activity
// 		WHERE julianday(last_modified) >= julianday(?)
// 		ORDER BY commit_date ASC
// 	`, thirtyDaysAgo.Format("2006-01-02"))

// 	if err != nil {
// 		fmt.Printf("❌ Error getting commit dates: %v\n", err)
// 		return
// 	}
// 	defer rows.Close()

// 	var commitDates []string
// 	for rows.Next() {
// 		var date string
// 		rows.Scan(&date)
// 		commitDates = append(commitDates, date)
// 	}

// 	fmt.Printf("📅 Found %d unique commit dates in past %d days\n", len(commitDates), days)

// 	if len(commitDates) == 0 {
// 		fmt.Println("⚠️  No commits found in the past 30 days")
// 		return
// 	}

// 	// Generate snapshots only for dates with commits
// 	for i, dateStr := range commitDates {
// 		// Use end of day for snapshot calculation (captures all commits from that day)
// 		snapshotDate := dateStr + " 23:59:59"
// 		saveSnapshotForDate(repo, snapshotDate)

// 		if (i+1)%10 == 0 {
// 			fmt.Printf("  ✅ Generated %d snapshots so far\n", i+1)
// 		}
// 	}

// 	fmt.Printf("  🎉 Generated %d historical snapshots (only on commit days)!\n", len(commitDates))
// }

// func shouldNotify(repo string) bool {
// 	row := DB.QueryRow(`
// 		SELECT activity_score
// 		FROM repo_snapshots
// 		WHERE repo_name = ?
// 		ORDER BY created_at DESC
// 		LIMIT 2
// 	`, repo)

// 	var latest, previous float64

// 	err := row.Scan(&latest)
// 	if err != nil {
// 		return false
// 	}

// 	row = DB.QueryRow(`
// 		SELECT activity_score
// 		FROM repo_snapshots
// 		WHERE repo_name = ?
// 		ORDER BY created_at DESC
// 		LIMIT 1 OFFSET 1
// 	`, repo)

// 	err = row.Scan(&previous)
// 	if err != nil {
// 		return false
// 	}

// 	// Notify only if change > 10%
// 	diff := latest - previous
// 	if diff < 0 {
// 		diff = -diff
// 	}

// 	return diff >= 10
// }

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

	if snapshotCount == 0 {
		generateHistoricalSnapshots(repo, 30)
	} else {
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

// ----------------------------
// HISTORICAL SNAPSHOTS (FROM COMMITS)
// ----------------------------
func generateHistoricalSnapshots(repo string, days int) {
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

	for rows.Next() {
		var d string
		rows.Scan(&d)
		saveSnapshotForDate(repo, d+" 23:59:59")
	}
}
