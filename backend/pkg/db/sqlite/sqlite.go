package sqlite

import (
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

func New(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}

	if _, err := db.Exec(`PRAGMA foreign_keys = ON;`); err != nil {
		_ = db.Close()
		return nil, err
	}

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}

	return db, nil
}

func RunMigrations(db *sql.DB, migrationsDir string) error {
	files, err := migrationFiles(migrationsDir)
	if err != nil {
		return err
	}

	if err := ensureSchemaMigrationsTable(db); err != nil {
		return err
	}

	for _, path := range files {
		name := filepath.Base(path)
		applied, err := isMigrationApplied(db, name)
		if err != nil {
			return err
		}
		if applied {
			continue
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}

		tx, err := db.Begin()
		if err != nil {
			return err
		}

		if _, err := tx.Exec(string(content)); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("apply migration %s: %w", name, err)
		}

		if _, err := tx.Exec(`INSERT INTO schema_migrations (name) VALUES (?);`, name); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("record migration %s: %w", name, err)
		}

		if err := tx.Commit(); err != nil {
			return err
		}
	}

	return nil
}

func migrationFiles(dir string) ([]string, error) {
	var files []string

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(d.Name(), ".up.sql") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("migrations directory %q does not exist", dir)
		}
		return nil, err
	}

	sort.Strings(files)
	return files, nil
}

func ensureSchemaMigrationsTable(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE,
			applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
	`)
	return err
}

func isMigrationApplied(db *sql.DB, name string) (bool, error) {
	var count int
	err := db.QueryRow(`SELECT COUNT(1) FROM schema_migrations WHERE name = ?;`, name).Scan(&count)
	return count > 0, err
}
