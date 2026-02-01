package main

import (
	"fmt"
	"net/http"
	"os"
)

func main() {

	err := InitDB()
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

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Println("🚀 Backend running on port", port)

	err = http.ListenAndServe(":"+port, nil)
	if err != nil {
		panic(err)
	}
}
