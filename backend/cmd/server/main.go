package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"gitsense"
	"gitsense/internal/api"
	"gitsense/internal/auth"
	"gitsense/internal/commits"
	"gitsense/internal/db"
	"gitsense/internal/repos"
	syncer "gitsense/internal/sync"

	"github.com/joho/godotenv"
)

func main() {
	// Load .env file (fails silently in production, which is fine)
	err := godotenv.Load("../.env")
	if err != nil {
		fmt.Println("⚠️  No .env file found (this is normal in production)")
	} else {
		fmt.Println("✅ Loaded .env file")
	}

	// Verify required environment variables are set
	if os.Getenv("GITHUB_CLIENT_ID") == "" {
		fmt.Println("⚠️  Warning: GITHUB_CLIENT_ID not set")
	}
	if os.Getenv("GITHUB_CLIENT_SECRET") == "" {
		fmt.Println("⚠️  Warning: GITHUB_CLIENT_SECRET not set")
	}

	err = db.InitDB()
	if err != nil {
		panic(err)
	}

	// Health check
	http.HandleFunc("/health", api.HealthHandler)

	// Project data
	http.HandleFunc("/project/summary", api.GetProjectSummary)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// 🔐 OAuth routes
	http.HandleFunc("/auth/github", auth.GithubLogin)
	http.HandleFunc("/auth/callback", auth.GithubCallback)

	// Sync repo
	http.HandleFunc("/sync", syncer.SyncHandler)

	// Token bridge to extension
	http.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		token := r.URL.Query().Get("value")
		allowedOrigin := r.URL.Query().Get("origin")
		if allowedOrigin == "" {
			allowedOrigin = os.Getenv("EXTENSION_ORIGIN")
		}
		if allowedOrigin == "" || (!strings.HasPrefix(allowedOrigin, "chrome-extension://") && !strings.HasPrefix(allowedOrigin, "http://localhost")) {
			http.Error(w, "Invalid extension origin", http.StatusBadRequest)
			return
		}
		fmt.Fprintf(w, `
		<script>
		if (window.opener) {
			window.opener.postMessage({ token: %q }, %q);
		}
		window.close();
		</script>
		`, token, allowedOrigin)
	})

	http.HandleFunc("/repos", repos.GetUserRepos)
	http.HandleFunc("/history", api.GetRepoHistory)
	http.HandleFunc("/commits", commits.GetCommits)
	http.HandleFunc("/files", api.GetFileActivity)
	http.HandleFunc("/dashboard", api.DashboardHandler)

	// New analytics endpoints
	http.HandleFunc("/commits-per-day", api.GetCommitsPerDay)
	http.HandleFunc("/file-breakdown", api.GetFileBreakdown)
	http.HandleFunc("/contributor-distribution", api.GetContributorDistribution)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Println("🚀 Backend running on port", port)

	// Create server with proper timeouts to prevent slowloris attacks
	server := &http.Server{
		Addr:         ":" + port,
		Handler:      nil,
		ReadTimeout:  gitsense.ServerReadTimeout,
		WriteTimeout: gitsense.ServerWriteTimeout,
		IdleTimeout:  gitsense.ServerIdleTimeout,
	}

	err = server.ListenAndServe()
	if err != nil {
		panic(err)
	}
}
