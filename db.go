package main

import (
	"database/sql"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

func initDB(path string) *sql.DB {
	db, err := sql.Open("sqlite3", path+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		log.Fatal(err)
	}

	migrations := []string{
		`CREATE TABLE IF NOT EXISTS donors (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			phone TEXT UNIQUE NOT NULL,
			name TEXT DEFAULT '',
			email TEXT DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS sessions (
			token TEXT PRIMARY KEY,
			donor_id INTEGER NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			expires_at DATETIME NOT NULL,
			FOREIGN KEY (donor_id) REFERENCES donors(id)
		)`,
		`CREATE TABLE IF NOT EXISTS donations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			donor_id INTEGER NOT NULL,
			receipt_number TEXT UNIQUE NOT NULL,
			donation_date DATE NOT NULL,
			location TEXT DEFAULT '',
			items_description TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (donor_id) REFERENCES donors(id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_token ON sessions(token)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_expires ON sessions(expires_at)`,
		`CREATE INDEX IF NOT EXISTS idx_donations_donor ON donations(donor_id)`,
		`CREATE TABLE IF NOT EXISTS feedback_requests (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			donor_id INTEGER NOT NULL,
			body TEXT NOT NULL,
			type TEXT NOT NULL DEFAULT 'feature',
			urgency TEXT NOT NULL DEFAULT 'normal',
			status TEXT NOT NULL DEFAULT 'new',
			admin_notes TEXT DEFAULT '',
			github_issue_url TEXT DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME,
			FOREIGN KEY (donor_id) REFERENCES donors(id)
		)`,
	}

	for _, m := range migrations {
		if _, err := db.Exec(m); err != nil {
			log.Fatalf("migration failed: %v\nSQL: %s", err, m)
		}
	}

	return db
}
