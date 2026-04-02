package main

import (
	"database/sql"
	"log"
	"strings"

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
		// Add zip_code column to donors (safe to re-run: ALTER will fail silently if exists)
		`ALTER TABLE donors ADD COLUMN zip_code TEXT DEFAULT ''`,
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
			// ALTER TABLE ADD COLUMN fails if column already exists — that's fine
			if !strings.Contains(err.Error(), "duplicate column") {
				log.Fatalf("migration failed: %v\nSQL: %s", err, m)
			}
		}
	}

	return db
}
