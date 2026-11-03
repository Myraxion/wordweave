package app

import (
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

//go:embed migrations/*.sql
var migrations embed.FS

// Back up before changing an existing database; commit all pending scripts together.
func migrate(db *sql.DB, dataDir string) error {
	files, err := fs.Glob(migrations, "migrations/*.sql")
	if err != nil {
		return err
	}
	versions := make([]int, len(files))
	for i, name := range files {
		prefix := strings.Split(strings.TrimPrefix(name, "migrations/"), "_")[0]
		versions[i], err = strconv.Atoi(prefix)
		if err != nil {
			return fmt.Errorf("invalid migration %s: %w", name, err)
		}
	}
	latest := versions[len(versions)-1]
	var hasVersions, tableCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'schema_migrations'").Scan(&hasVersions); err != nil {
		return err
	}
	current := 0
	if hasVersions != 0 {
		if err := db.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_migrations").Scan(&current); err != nil {
			return err
		}
	}
	if current > latest {
		return fmt.Errorf("database version %d is newer than supported version %d", current, latest)
	}
	if current == latest {
		return nil
	}
	if err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%'").Scan(&tableCount); err != nil {
		return err
	}
	if tableCount > 0 {
		if err := backupDatabase(db, dataDir, current, latest); err != nil {
			return err
		}
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version INTEGER PRIMARY KEY,
		applied_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`); err != nil {
		return err
	}
	for i, name := range files {
		if versions[i] <= current {
			continue
		}
		script, err := migrations.ReadFile(name)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(string(script)); err != nil {
			return fmt.Errorf("migration %s: %w", name, err)
		}
		if _, err := tx.Exec("INSERT INTO schema_migrations (version) VALUES (?)", versions[i]); err != nil {
			return fmt.Errorf("record migration %s: %w", name, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit migrations: %w", err)
	}
	return nil
}

func backupDatabase(db *sql.DB, dataDir string, current, latest int) error {
	dir := filepath.Join(dataDir, "backups")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create backup directory %s: %w", dir, err)
	}
	file, err := os.CreateTemp(dir, fmt.Sprintf("wordweave-v%d-to-v%d-*.db", current, latest))
	if err != nil {
		return fmt.Errorf("create upgrade backup: %w", err)
	}
	name := file.Name()
	if err := file.Close(); err != nil {
		return err
	}
	if _, err := db.Exec("VACUUM INTO ?", name); err != nil {
		os.Remove(name)
		return fmt.Errorf("backup database to %s: %w", name, err)
	}
	file, err = os.OpenFile(name, os.O_RDWR, 0600)
	if err != nil {
		return fmt.Errorf("open backup %s: %w", name, err)
	}
	defer file.Close()
	if err := file.Sync(); err != nil {
		return fmt.Errorf("sync backup %s: %w", name, err)
	}
	return nil
}
