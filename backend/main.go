package main

import (
	"fmt"
	"net/http"
	"os"

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

	// Debug: Check if env vars are loaded
	clientID := os.Getenv("GITHUB_CLIENT_ID")
	clientSecret := os.Getenv("GITHUB_CLIENT_SECRET")
	fmt.Printf("🔑 GITHUB_CLIENT_ID: %s\n", clientID)
	fmt.Printf("🔑 GITHUB_CLIENT_SECRET: %s...\n", clientSecret[:10])

	err = InitDB()
	if err != nil {
		panic(err)
	}

	// Health check
	http.HandleFunc("/health", healthHandler)

	// Project data
	http.HandleFunc("/project/summary", getProjectSummary)

	// 🔐 OAuth routes
	http.HandleFunc("/auth/github", githubLogin)
	http.HandleFunc("/auth/callback", githubCallback)

	// Sync repo
	http.HandleFunc("/sync", syncHandler)

	// Token bridge to extension
	http.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		token := r.URL.Query().Get("value")
		fmt.Fprintf(w, `
		<script>
		window.opener.postMessage({ token: "%s" }, "*");
		window.close();
		</script>
		`, token)
	})

	http.HandleFunc("/repos", getUserRepos)
	http.HandleFunc("/history", getRepoHistory)
	http.HandleFunc("/commits", getCommits)


	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Println("🚀 Backend running on port", port)

	err = http.ListenAndServe(":"+port, nil)
	if err != nil {
		panic(err)
	}
	// this is dummy code to test the deployment of the backend on render.com. It will be removed later.
}