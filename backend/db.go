package main

import (
	"database/sql"
	_ "modernc.org/sqlite"
)

var DB *sql.DB

func InitDB() error {
	var err error

	DB, err = sql.Open("sqlite", "gitsense.db")
	if err != nil {
		return err
	}

	// ----------------------------
	// FILE ACTIVITY TABLE
	// ----------------------------
	fileActivityTable := `
	CREATE TABLE IF NOT EXISTS file_activity (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		repo_name TEXT,
		file_name TEXT UNIQUE,
		commit_count INTEGER,
		last_modified DATETIME
	);
	`

	_, err = DB.Exec(fileActivityTable)
	if err != nil {
		return err
	}

	// ----------------------------
	// REPO SNAPSHOT TABLE
	// ----------------------------
	repoSnapshotTable := `
	CREATE TABLE IF NOT EXISTS repo_snapshots (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		repo_name TEXT,
		active_files INTEGER,
		stable_files INTEGER,
		inactive_files INTEGER,
		activity_score REAL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`

	_, err = DB.Exec(repoSnapshotTable)
	if err != nil {
		return err
	}

	// ----------------------------
	// USERS TABLE
	// ----------------------------
	userTable := `
	CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    github_username TEXT UNIQUE,
    access_token TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`
	_, err = DB.Exec(userTable)
	if err != nil {
		return err
	}

	// ----------------------------
	// USER REPOS TABLE
	// ----------------------------
	repoTable := `
	CREATE TABLE IF NOT EXISTS user_repos (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER,
    repo_name TEXT,
    last_synced DATETIME,
    FOREIGN KEY(user_id) REFERENCES users(id)
	);`
	_, err = DB.Exec(repoTable)
	if err != nil {
		return err
	}

	return nil
}
