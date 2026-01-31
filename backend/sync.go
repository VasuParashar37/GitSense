package main

import (
	"bufio"
	"fmt"
	"os/exec"
	"strings"
	"strconv"
)

/**
	"os/exec"
	1. Used to run OS commands
	2. Lets Go execute shell commands like:
	3. git
	4. ls
	5. pwd
**/

// SyncCommits reads git history and stores it

/**
	What this function does?
		Run git log --oneline
		Read each commit
		Split hash and message
		Store commits in SQLite
		Avoid duplicates
		Report success or failure
**/

func gitPull(repoPath string) error {
	cmd := exec.Command("git", "pull")
	cmd.Dir = repoPath
	return cmd.Run()
}

func SyncCommits(repoPath string) error {

	// First, do a git pull to get latest changes
	err := gitPull(repoPath)
	if err != nil {
		fmt.Println("⚠️ Git pull failed:", err)
	} else {
		fmt.Println("✅ Git repository updated")
	}

	// Run git log command
	cmd := exec.Command("git", "log", "--pretty=format:%H|%ct|%s")
	cmd.Dir = repoPath

	/**
		cmd.Dir = repoPath
		1. Sets the working directory for the command
		2. So that git commands run in the context of the specified repo
		Equivalent to:
		-> cd repoPath
		-> git log --oneline
	**/

	out, err := cmd.Output()
	if err != nil {
		return err
	}

	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	// means -> “Take git output and prepare it to be read line by line.”

	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, "|", 3)

		if len(parts) == 3 {
			hash := parts[0]
			unixTime := parts[1]
			message := parts[2]

			// Insert safely (no duplicates)
			DB.Exec(
				`INSERT OR IGNORE INTO commits
				(hash, message, commit_time)
				VALUES (?, ?, datetime(?, 'unixepoch'))`,
				hash, message, unixTime,
			)
		}
	}


	// Now, track file activity
	cmd = exec.Command(
		"git",
		"log",
		"--name-only",
		"--pretty=format:%ct",
	)
	cmd.Dir = repoPath
	
	out, _ = cmd.Output()
	scanner = bufio.NewScanner(strings.NewReader(string(out)))
	
	var currentTime string
	
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
	
		if line == "" {
			continue
		}
	
		// If line is timestamp
		if _, err := strconv.ParseInt(line, 10, 64); err == nil {
			currentTime = line
			continue
		}
	
		// Otherwise it's a filename
		DB.Exec(`
			INSERT INTO file_activity
			(file_name, commit_count, last_modified)
			VALUES (?, 1, datetime(?, 'unixepoch'))
			ON CONFLICT(file_name)
			DO UPDATE SET
				commit_count = commit_count + 1,
				last_modified = datetime(?, 'unixepoch')
		`, line, currentTime, currentTime)
	}
	

	return nil
}
