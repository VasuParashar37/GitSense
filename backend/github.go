// package main

// import (
// 	"encoding/json"
// 	"fmt"
// 	"net/http"
// 	"time"
// )

// type GitHubCommit struct {
// 	SHA    string `json:"sha"`
// 	Commit struct {
// 		Author struct {
// 			Date string `json:"date"`
// 		} `json:"author"`
// 	} `json:"commit"`
// }

// type GitHubFile struct {
// 	Filename string `json:"filename"`
// }

// func SyncFromGitHub(owner, repo, token string) error {
// 	// Limit to recent 30 commits to avoid timeouts
// 	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/commits?per_page=30", owner, repo)

// 	req, _ := http.NewRequest("GET", url, nil)
// 	req.Header.Set("Authorization", "Bearer "+token)

// 	client := &http.Client{Timeout: 30 * time.Second}
// 	resp, err := client.Do(req)
// 	if err != nil {
// 		return err
// 	}
// 	defer resp.Body.Close()

// 	var commits []GitHubCommit
// 	json.NewDecoder(resp.Body).Decode(&commits)

// 	fmt.Printf("📊 Found %d commits\n", len(commits))

// 	for _, commit := range commits {
// 		fileURL := fmt.Sprintf(
// 			"https://api.github.com/repos/%s/%s/commits/%s",
// 			owner, repo, commit.SHA,
// 		)

// 		req, _ := http.NewRequest("GET", fileURL, nil)
// 		req.Header.Set("Authorization", "Bearer "+token)

// 		resp, err := client.Do(req)
// 		if err != nil {
// 			continue
// 		}

// 		var detail struct {
// 			Files []GitHubFile `json:"files"`
// 		}

// 		json.NewDecoder(resp.Body).Decode(&detail)
// 		resp.Body.Close()

// 		fmt.Printf("  Commit %s: %d files\n", commit.SHA[:7], len(detail.Files))

// 		for _, f := range detail.Files {
// 			result, err := DB.Exec(`
// 				INSERT INTO file_activity
// 				(repo_name, file_name, commit_count, last_modified)
// 				VALUES (?, ?, 1, ?)
// 				ON CONFLICT(file_name)
// 				DO UPDATE SET
// 					commit_count = commit_count + 1,
// 					last_modified = ?
// 			`, repo, f.Filename, commit.Commit.Author.Date, commit.Commit.Author.Date)

// 			if err != nil {
// 				fmt.Printf("    ❌ DB error for %s: %v\n", f.Filename, err)
// 			} else {
// 				rows, _ := result.RowsAffected()
// 				fmt.Printf("    ✅ Saved %s (%d rows affected)\n", f.Filename, rows)
// 			}
// 		}
// 	}

// 	return nil
// }

// // 🔹 Fetch GitHub username using token
// func getGitHubUsername(token string) string {
// 	req, _ := http.NewRequest(
// 		"GET",
// 		"https://api.github.com/user",
// 		nil,
// 	)
// 	req.Header.Set("Authorization", "Bearer "+token)

// 	client := &http.Client{Timeout: 10 * time.Second}
// 	resp, err := client.Do(req)
// 	if err != nil {
// 		return ""
// 	}
// 	defer resp.Body.Close()

// 	var data struct {
// 		Login string `json:"login"`
// 	}

// 	json.NewDecoder(resp.Body).Decode(&data)

// 	return data.Login
// }

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// ----------------------------
// GitHub API MODELS
// ----------------------------
type GitHubCommit struct {
	SHA string `json:"sha"`
	Commit struct {
		Message string `json:"message"`
		Author  struct {
			Name string `json:"name"`
			Date string `json:"date"`
		} `json:"author"`
	} `json:"commit"`
}

type GitHubFile struct {
	Filename string `json:"filename"`
}

// ----------------------------
// SYNC FROM GITHUB
// ----------------------------
func SyncFromGitHub(owner, repo, token string) error {

	// Limit commits to avoid timeout
	url := fmt.Sprintf(
		"https://api.github.com/repos/%s/%s/commits?per_page=30",
		owner, repo,
	)

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var commits []GitHubCommit
	if err := json.NewDecoder(resp.Body).Decode(&commits); err != nil {
		return err
	}

	fmt.Printf("📊 Found %d commits\n", len(commits))

	// ----------------------------
	// PROCESS EACH COMMIT
	// ----------------------------
	for _, c := range commits {

		// 🔹 Save commit into commits table
		_, err := DB.Exec(`
			INSERT OR IGNORE INTO commits
			(repo_name, commit_sha, author, message, commit_date)
			VALUES (?, ?, ?, ?, ?)
		`,
			repo,
			c.SHA,
			c.Commit.Author.Name,
			c.Commit.Message,
			c.Commit.Author.Date,
		)

		if err != nil {
			fmt.Printf(" ⚠️  Commit insertion error: %v\n", err)
		}

		// Fetch files changed in this commit
		fileURL := fmt.Sprintf(
			"https://api.github.com/repos/%s/%s/commits/%s",
			owner, repo, c.SHA,
		)

		req, _ := http.NewRequest("GET", fileURL, nil)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := client.Do(req)
		if err != nil {
			continue
		}

		var detail struct {
			Files []GitHubFile `json:"files"`
		}

		json.NewDecoder(resp.Body).Decode(&detail)
		resp.Body.Close()

		fmt.Printf(" Commit %s: %d files\n", c.SHA[:7], len(detail.Files))

		// ----------------------------
		// UPDATE FILE ACTIVITY
		// ----------------------------
		for _, f := range detail.Files {

			_, err := DB.Exec(`
				INSERT INTO file_activity
				(repo_name, file_name, commit_count, last_modified)
				VALUES (?, ?, 1, ?)
				ON CONFLICT(repo_name, file_name)
				DO UPDATE SET
					commit_count = commit_count + 1,
					last_modified = excluded.last_modified
			`,
				repo,
				f.Filename,
				c.Commit.Author.Date,
			)

			if err != nil {
				fmt.Printf(" ❌ DB error for %s: %v\n", f.Filename, err)
			}
		}
	}

	return nil
}

// ----------------------------
// FETCH GITHUB USERNAME
// ----------------------------
func getGitHubUsername(token string) string {
	req, _ := http.NewRequest(
		"GET",
		"https://api.github.com/user",
		nil,
	)
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	var data struct {
		Login string `json:"login"`
	}

	json.NewDecoder(resp.Body).Decode(&data)
	return data.Login
}
