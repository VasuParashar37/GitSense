package main

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
)

var DB *sql.DB

/** What InitDB() does:
	Open (or create) SQLite database
	Ensure commits table exists
	Return error if anything fails
**/

func InitDB() error {
	var err error

	DB, err = sql.Open("sqlite3", "gitdata.db")
	if err != nil {
		return err
	}

	query := `
	CREATE TABLE IF NOT EXISTS commits (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		hash TEXT UNIQUE,
		message TEXT,
		commit_time DATETIME,
		synced_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	_, err = DB.Exec(query)
	if err!=nil{
		return err
	}


	fileActivityQuery := `
	CREATE TABLE IF NOT EXISTS file_activity (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	file_name TEXT UNIQUE,
	commit_count INTEGER,
	last_modified DATETIME
	);`

	_, err = DB.Exec(fileActivityQuery)
	if err != nil {
		return err
	}
	return nil
}
