package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type GitHubCommit struct {
	SHA    string `json:"sha"`
	Commit struct {
		Author struct {
			Date string `json:"date"`
		} `json:"author"`
	} `json:"commit"`
}

type GitHubFile struct {
	Filename string `json:"filename"`
}

func SyncFromGitHub(owner, repo, token string) error {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/commits", owner, repo)

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var commits []GitHubCommit
	json.NewDecoder(resp.Body).Decode(&commits)

	for _, commit := range commits {
		fileURL := fmt.Sprintf(
			"https://api.github.com/repos/%s/%s/commits/%s",
			owner, repo, commit.SHA,
		)

		req, _ := http.NewRequest("GET", fileURL, nil)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			continue
		}

		var detail struct {
			Files []GitHubFile `json:"files"`
		}

		json.NewDecoder(resp.Body).Decode(&detail)
		resp.Body.Close()

		for _, f := range detail.Files {
			DB.Exec(`
				INSERT INTO file_activity
				(repo_name, file_name, commit_count, last_modified)
				VALUES (?, ?, 1, ?)
				ON CONFLICT(file_name)
				DO UPDATE SET
					commit_count = commit_count + 1,
					last_modified = ?
			`, repo, f.Filename, commit.Commit.Author.Date, commit.Commit.Author.Date)
		}
	}

	return nil
}

// 🔹 Fetch GitHub username using token
func getGitHubUsername(token string) string {
	req, _ := http.NewRequest(
		"GET",
		"https://api.github.com/user",
		nil,
	)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
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
