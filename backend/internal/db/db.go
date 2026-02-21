package db

import (
	"database/sql"
	"fmt"
	"sync"

	_ "modernc.org/sqlite"
)

// DB is the global database instance (kept for backward compatibility)
// Prefer using GetDB() for better testability
var DB *sql.DB

var (
	db      *sql.DB
	once    sync.Once
	initErr error
)

// GetDB returns the database instance, initializing it if necessary
// This is the preferred way to access the database
func GetDB() (*sql.DB, error) {
	once.Do(func() {
		db, initErr = initializeDB()
		DB = db // Keep global DB in sync for backward compatibility
	})
	return db, initErr
}

// SetDB allows setting a custom database instance (useful for testing)
func SetDB(database *sql.DB) {
	db = database
	DB = database
}

func initializeDB() (*sql.DB, error) {
	database, err := sql.Open("sqlite", "gitsense.db")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Enable WAL mode for better concurrency
	if _, err := database.Exec("PRAGMA journal_mode=WAL;"); err != nil {
		return nil, fmt.Errorf("failed to set WAL mode: %w", err)
	}
	if _, err := database.Exec("PRAGMA busy_timeout=5000;"); err != nil {
		return nil, fmt.Errorf("failed to set busy timeout: %w", err)
	}
	if _, err := database.Exec("PRAGMA synchronous=NORMAL;"); err != nil {
		return nil, fmt.Errorf("failed to set synchronous mode: %w", err)
	}

	return database, nil
}

func InitDB() error {
	database, err := GetDB()
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
	);
	`
	if _, err = database.Exec(userTable); err != nil {
		return fmt.Errorf("failed to create users table: %w", err)
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
	if _, err = database.Exec(repoTable); err != nil {
		return fmt.Errorf("failed to create user_repos table: %w", err)
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
	if _, err = database.Exec(commitsTable); err != nil {
		return fmt.Errorf("failed to create commits table: %w", err)
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
	if _, err = database.Exec(fileActivityTable); err != nil {
		return fmt.Errorf("failed to create file_activity table: %w", err)
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
	if _, err = database.Exec(repoSnapshotTable); err != nil {
		return fmt.Errorf("failed to create repo_snapshots table: %w", err)
	}

	return nil
}
