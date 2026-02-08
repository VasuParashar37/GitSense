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

	// Enable WAL mode for better concurrency
	DB.Exec("PRAGMA journal_mode=WAL;")
	DB.Exec("PRAGMA busy_timeout=5000;") // Wait up to 5 seconds if locked
	DB.Exec("PRAGMA synchronous=NORMAL;")

	// ----------------------------
	// USERS TABLE
	// ----------------------------
	userTable := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		github_username TEXT UNIQUE,
		access_token TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`
	if _, err = DB.Exec(userTable); err != nil {
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
	);
	`
	if _, err = DB.Exec(repoTable); err != nil {
		return err
	}

	// ----------------------------
	// COMMITS TABLE (NEW)
	// ----------------------------
	commitsTable := `
	CREATE TABLE IF NOT EXISTS commits (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		repo_name TEXT,
		commit_sha TEXT UNIQUE,
		author TEXT,
		message TEXT,
		commit_date DATETIME
	);
	`
	if _, err = DB.Exec(commitsTable); err != nil {
		return err
	}

	// ----------------------------
	// FILE ACTIVITY TABLE
	// ⚠️ FIX: file_name should NOT be globally UNIQUE
	// Same filename can exist in different repos
	// ----------------------------
	fileActivityTable := `
	CREATE TABLE IF NOT EXISTS file_activity (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		repo_name TEXT,
		file_name TEXT,
		commit_count INTEGER,
		last_modified DATETIME,
		UNIQUE(repo_name, file_name)
	);
	`
	if _, err = DB.Exec(fileActivityTable); err != nil {
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
	if _, err = DB.Exec(repoSnapshotTable); err != nil {
		return err
	}

	return nil
}
