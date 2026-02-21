package syncer

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"gitsense"
	"gitsense/internal/db"
	githubapi "gitsense/internal/github"
)

func SyncHandler(w http.ResponseWriter, r *http.Request) {
	gitsense.SetCORSHeaders(w)

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Validate owner parameter
	owner := r.URL.Query().Get("owner")
	if owner == "" {
		gitsense.SendErrorResponse(w, "Missing required parameter: owner", http.StatusBadRequest)
		return
	}
	if len(owner) > 100 {
		gitsense.SendErrorResponse(w, "Owner parameter too long (max 100 characters)", http.StatusBadRequest)
		return
	}

	// Validate repo parameter
	repo, err := gitsense.ValidateRepoParam(r)
	if err != nil {
		gitsense.SendErrorResponse(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validate Authorization header
	token := r.Header.Get("Authorization")
	if token == "" {
		gitsense.SendErrorResponse(w, "Missing Authorization header", http.StatusUnauthorized)
		return
	}

	fmt.Printf("🔄 Syncing %s/%s\n", owner, repo)

	// Count commits before sync
	var before int
	db.DB.QueryRow(
		`SELECT COUNT(*) FROM commits WHERE repo_name = ?`,
		repo,
	).Scan(&before)

	// Fetch from GitHub
	if err := githubapi.SyncFromGitHub(owner, repo, token); err != nil {
		http.Error(w, "Sync failed", http.StatusInternalServerError)
		return
	}

	// Count commits after sync
	var after int
	db.DB.QueryRow(
		`SELECT COUNT(*) FROM commits WHERE repo_name = ?`,
		repo,
	).Scan(&after)

	newCommits := after - before

	// Save repo under user
	db.DB.Exec(`
		INSERT INTO user_repos (user_id, repo_name, last_synced)
		SELECT id, ?, CURRENT_TIMESTAMP
		FROM users
		WHERE access_token = ?
	`, repo, token)

	// Snapshot handling
	var snapshotCount int
	db.DB.QueryRow(
		`SELECT COUNT(*) FROM repo_snapshots WHERE repo_name = ?`,
		repo,
	).Scan(&snapshotCount)

	fmt.Printf("📊 Snapshot count for '%s': %d\n", repo, snapshotCount)

	if snapshotCount == 0 {
		fmt.Println("🎯 First sync detected - generating historical snapshots...")
		if err := generateHistoricalSnapshots(repo, gitsense.HistoricalSnapshotDays); err != nil {
			fmt.Printf("⚠️  Historical snapshot generation failed: %v\n", err)
			http.Error(w, "Snapshot generation failed", http.StatusInternalServerError)
			return
		}
	} else {
		// Only create snapshot if there are new commits OR no snapshot for today exists
		if newCommits > 0 {
			fmt.Println("📊 Creating snapshot for new commits...")
			if err := saveSnapshot(repo); err != nil {
				fmt.Printf("⚠️  Failed to save snapshot: %v\n", err)
				http.Error(w, "Snapshot creation failed", http.StatusInternalServerError)
				return
			}
		} else {
			// Check if snapshot for today already exists
			var todaySnapshotCount int
			db.DB.QueryRow(`
				SELECT COUNT(*) FROM repo_snapshots
				WHERE repo_name = ? AND DATE(created_at) = DATE('now')
			`, repo).Scan(&todaySnapshotCount)

			if todaySnapshotCount == 0 {
				fmt.Println("📊 Creating today's first snapshot...")
				if err := saveSnapshot(repo); err != nil {
					fmt.Printf("⚠️  Failed to save snapshot: %v\n", err)
					http.Error(w, "Snapshot creation failed", http.StatusInternalServerError)
					return
				}
			} else {
				fmt.Println("✅ Snapshot for today already exists, skipping...")
			}
		}
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
func saveSnapshot(repo string) error {
	return saveSnapshotForDate(repo, "")
}

func saveSnapshotForDate(repo string, referenceDate string) error {
	dateExpr := "julianday('now')"
	if referenceDate != "" {
		dateExpr = fmt.Sprintf("julianday('%s')", referenceDate)
	}

	row := db.DB.QueryRow(fmt.Sprintf(`
		SELECT
			SUM(CASE WHEN %s - julianday(last_modified) <= %d THEN 1 ELSE 0 END),
			SUM(CASE WHEN %s - julianday(last_modified) BETWEEN %d AND %d THEN 1 ELSE 0 END),
			SUM(CASE WHEN %s - julianday(last_modified) > %d THEN 1 ELSE 0 END)
		FROM file_activity
		WHERE repo_name = ?
	`, dateExpr, gitsense.ActiveThreshold, dateExpr, gitsense.ActiveThreshold, gitsense.StableThreshold, dateExpr, gitsense.InactiveThreshold), repo)

	var active, stable, inactive int
	row.Scan(&active, &stable, &inactive)

	total := active + stable + inactive
	score := 0.0
	if total > 0 {
		rawScore := (float64(active) / float64(total)) * 100
		score = float64(int(rawScore + 0.5)) // Round to nearest integer
	}

	// Retry logic for SQLITE_BUSY errors
	var err error
	retryDelay := gitsense.InitialRetryDelay

	for attempt := 0; attempt < gitsense.MaxDBRetries; attempt++ {
		if referenceDate != "" {
			_, err = db.DB.Exec(`
				INSERT INTO repo_snapshots
				(repo_name, active_files, stable_files, inactive_files, activity_score, created_at)
				VALUES (?, ?, ?, ?, ?, ?)
			`, repo, active, stable, inactive, score, referenceDate)
		} else {
			_, err = db.DB.Exec(`
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
			if attempt < gitsense.MaxDBRetries-1 {
				time.Sleep(retryDelay)
				retryDelay *= 2 // Exponential backoff
			}
		} else {
			break // Non-busy error, don't retry
		}
	}

	if err != nil {
		fmt.Printf("❌ Failed to save snapshot for '%s' (date: %s): %v\n", repo, referenceDate, err)
		return fmt.Errorf("failed to save snapshot: %w", err)
	}

	fmt.Printf("✅ Saved snapshot for '%s' - Score: %.1f (date: %s)\n", repo, score, referenceDate)
	return nil
}

// ----------------------------
// HISTORICAL SNAPSHOTS (FROM COMMITS)
// ----------------------------
func generateHistoricalSnapshots(repo string, days int) error {
	fmt.Printf("📅 Generating historical snapshots for '%s' (past %d days)...\n", repo, days)
	start := time.Now().AddDate(0, 0, -days)

	rows, err := db.DB.Query(`
		SELECT DISTINCT DATE(commit_date)
		FROM commits
		WHERE repo_name = ?
		  AND julianday(commit_date) >= julianday(?)
		ORDER BY DATE(commit_date)
	`, repo, start.Format("2006-01-02"))

	if err != nil {
		fmt.Println("❌ History snapshot error:", err)
		return fmt.Errorf("failed to query commit dates: %w", err)
	}
	defer rows.Close()

	var dates []string
	for rows.Next() {
		var d string
		if err := rows.Scan(&d); err != nil {
			fmt.Println("❌ Failed to scan date:", err)
			continue
		}
		dates = append(dates, d)
		if err := saveSnapshotForDate(repo, d+" 23:59:59"); err != nil {
			fmt.Printf("⚠️  Failed to save snapshot for date %s: %v\n", d, err)
			// Continue with other dates even if one fails
		}
	}

	fmt.Printf("✅ Generated %d snapshots for dates: %v\n", len(dates), dates)
	return nil
}
