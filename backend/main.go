package main

import (
	"fmt"
	"time"
	"net/http"
)

func main() {

	// 1. Start database
	err := InitDB()
	if err != nil {
		fmt.Println("DB error:", err)
		return
	}
	fmt.Println("✅ Database ready")

	// 2. Set your Git repo path
	repoPath := "/Users/vasuparashar03/Desktop/LeetCode"

	// 3. First sync
	err = SyncCommits(repoPath)
	if err != nil {
		fmt.Println("Sync error:", err)
		return
	}
	fmt.Println("✅ Initial sync done")

	// Start API server
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/commits", getCommitsHandler)
	http.HandleFunc("/stats", getStatsHandler)
	http.HandleFunc("/summary", getProjectSummary)
	
	// Run server in a goroutine
	go func() {
		println("🚀 API running on http://localhost:8080")
		http.ListenAndServe(":8080", nil)
	}()

	// 4. Real-time service loop
	for {
		time.Sleep(600 * time.Second)

		fmt.Println("🔄 Syncing Git changes...", time.Now())
		err := SyncCommits(repoPath)
		if err != nil {
			fmt.Println("Sync error:", err)
		} else {
			fmt.Println("✅ Sync complete")
		}
	}
}
