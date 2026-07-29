// Package db opens the SQLite database and runs embedded migrations.
//
// Currently dormant: the site is pure content and nothing reads the
// database. To revive it, call Open + Migrate in cmd/server/main.go
// and hand the *sql.DB to whatever needs it.
package db

import (
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"sort"

	_ "modernc.org/sqlite" // pure-Go SQLite driver, no CGO needed
)

//go:embed migrations/*.sql
var migrations embed.FS

// Open creates the database file (and its directory) if needed and
// applies sane defaults: WAL journaling, foreign keys, busy timeout.
func Open(path string) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("db: create data dir: %w", err)
	}

	dsn := fmt.Sprintf("file:%s?_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)", path)
	database, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("db: open: %w", err)
	}
	// SQLite handles one writer at a time; a single connection avoids
	// SQLITE_BUSY errors under concurrent writes.
	database.SetMaxOpenConns(1)

	if err := database.Ping(); err != nil {
		return nil, fmt.Errorf("db: ping: %w", err)
	}
	return database, nil
}

// Migrate applies all migrations/*.sql files in lexical order that have
// not been applied yet, tracked in the schema_migrations table.
// Name migrations 0001_description.sql, 0002_description.sql, ...
func Migrate(database *sql.DB) error {
	if _, err := database.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (version TEXT PRIMARY KEY)`); err != nil {
		return fmt.Errorf("db: create schema_migrations: %w", err)
	}

	entries, err := fs.ReadDir(migrations, "migrations")
	if err != nil {
		return fmt.Errorf("db: read migrations: %w", err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	sort.Strings(names)

	for _, name := range names {
		var applied int
		if err := database.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE version = ?`, name).Scan(&applied); err != nil {
			return err
		}
		if applied > 0 {
			continue
		}

		script, err := fs.ReadFile(migrations, "migrations/"+name)
		if err != nil {
			return err
		}
		tx, err := database.Begin()
		if err != nil {
			return err
		}
		if _, err := tx.Exec(string(script)); err != nil {
			tx.Rollback()
			return fmt.Errorf("db: migration %s: %w", name, err)
		}
		if _, err := tx.Exec(`INSERT INTO schema_migrations (version) VALUES (?)`, name); err != nil {
			tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
		slog.Info("applied migration", "version", name)
	}
	return nil
}
