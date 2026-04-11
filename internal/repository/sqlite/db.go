package sqlite

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

const schema = `
CREATE TABLE IF NOT EXISTS users (
    id          INTEGER PRIMARY KEY,
    username    TEXT    NOT NULL DEFAULT '',
    first_name  TEXT    NOT NULL DEFAULT '',
    last_name   TEXT    NOT NULL DEFAULT '',
    admin_id    INTEGER NOT NULL DEFAULT 0,
    is_blocked  INTEGER NOT NULL DEFAULT 0,
    created_at  DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS connection_payments (
    uuid         TEXT    PRIMARY KEY,
    user_id      INTEGER NOT NULL,
    admin_id     INTEGER NOT NULL,
    status       TEXT    NOT NULL DEFAULT 'unpaid',
    created_at   DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS admin_profiles (
    admin_id     INTEGER PRIMARY KEY,
    payment_info TEXT    NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS payments (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id      INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    period_year  INTEGER NOT NULL,
    period_month INTEGER NOT NULL,
    paid_at      DATETIME,
    confirmed_by INTEGER,
    created_at   DATETIME NOT NULL DEFAULT (datetime('now')),
    UNIQUE (user_id, period_year, period_month)
);

CREATE INDEX IF NOT EXISTS idx_payments_period ON payments(period_year, period_month);
`

func Open(dbPath string) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, fmt.Errorf("create db dir: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	db.SetMaxOpenConns(1) // SQLite is not concurrent-write safe

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}

	if _, err := db.Exec("PRAGMA journal_mode=WAL; PRAGMA foreign_keys=ON;"); err != nil {
		return nil, fmt.Errorf("pragma: %w", err)
	}

	return db, nil
}

// columnMigrations are ALTER TABLE statements applied after the initial schema.
// addColumnIfMissing ignores "duplicate column name" errors so they are idempotent.
var columnMigrations = []string{
	`ALTER TABLE users ADD COLUMN is_free_friend INTEGER NOT NULL DEFAULT 0`,
	`ALTER TABLE connection_payments ADD COLUMN last_paid_at DATETIME`,
}

// dataMigrations are named one-time statements tracked in _applied_migrations.
var dataMigrations = []struct {
	name string
	sql  string
}{
	{
		"reset_free_friends_2026_04_11",
		`UPDATE users SET is_free_friend = 0 WHERE is_free_friend = 1`,
	},
}

func Migrate(db *sql.DB) error {
	if _, err := db.Exec(schema); err != nil {
		return err
	}
	for _, alter := range columnMigrations {
		if err := addColumnIfMissing(db, alter); err != nil {
			return fmt.Errorf("column migration %q: %w", alter, err)
		}
	}
	if err := runDataMigrations(db); err != nil {
		return fmt.Errorf("data migrations: %w", err)
	}
	return nil
}

func runDataMigrations(db *sql.DB) error {
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS _applied_migrations (name TEXT PRIMARY KEY)`); err != nil {
		return err
	}
	for _, m := range dataMigrations {
		var exists int
		_ = db.QueryRow(`SELECT COUNT(*) FROM _applied_migrations WHERE name = ?`, m.name).Scan(&exists)
		if exists > 0 {
			continue
		}
		if _, err := db.Exec(m.sql); err != nil {
			return fmt.Errorf("run %q: %w", m.name, err)
		}
		if _, err := db.Exec(`INSERT INTO _applied_migrations (name) VALUES (?)`, m.name); err != nil {
			return fmt.Errorf("record %q: %w", m.name, err)
		}
	}
	return nil
}

func addColumnIfMissing(db *sql.DB, alter string) error {
	_, err := db.Exec(alter)
	if err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		return err
	}
	return nil
}
