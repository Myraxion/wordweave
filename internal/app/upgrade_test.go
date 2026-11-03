package app_test

import (
	"database/sql"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/Myraxion/wordweave/internal/app"
)

// Fixtures describe databases made by earlier releases, not production helpers.
const legacySchema = `
CREATE TABLE schema_migrations (version INTEGER PRIMARY KEY, applied_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP);
INSERT INTO schema_migrations(version) VALUES (1);
CREATE TABLE entries (
 id INTEGER PRIMARY KEY AUTOINCREMENT, spelling TEXT NOT NULL, spelling_key TEXT NOT NULL UNIQUE,
 definition TEXT NOT NULL DEFAULT '', created_at TEXT NOT NULL
);
INSERT INTO entries VALUES (1, 'Legacy', 'legacy', '升级前' || char(10) || '原释义', '2026-11-01T00:00:00Z');
`

func fixtureDB(t *testing.T, dir, script string) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(dir, "wordweave.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if _, err := db.Exec(script); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestUpgradeFailureStopsStartupAndRollsBack(t *testing.T) {
	for _, stage := range []string{"script", "version", "commit"} {
		t.Run(stage, func(t *testing.T) {
			dir := t.TempDir()
			script := legacySchema
			switch stage {
			case "script":
				script += "CREATE TABLE relation_groups (id INTEGER PRIMARY KEY);"
			case "version":
				script += `CREATE TRIGGER fail_version BEFORE INSERT ON schema_migrations
					WHEN NEW.version = 2 BEGIN SELECT RAISE(ABORT, 'controlled version failure'); END;`
			case "commit":
				script += `CREATE TABLE deferred_failure (id INTEGER REFERENCES entries(id) DEFERRABLE INITIALLY DEFERRED);
					CREATE TRIGGER fail_commit AFTER INSERT ON schema_migrations
					BEGIN INSERT INTO deferred_failure VALUES (99999); END;`
			}
			db := fixtureDB(t, dir, script)
			var before string
			if err := db.QueryRow("SELECT group_concat(sql, ';') FROM sqlite_master").Scan(&before); err != nil {
				t.Fatal(err)
			}
			a, err := app.Open(dir)
			if err == nil {
				a.Close()
				t.Fatal("failed upgrade must stop application startup")
			}
			if !strings.Contains(err.Error(), "migration") {
				t.Fatalf("unhelpful error: %v", err)
			}
			var after string
			if err := db.QueryRow("SELECT group_concat(sql, ';') FROM sqlite_master").Scan(&after); err != nil {
				t.Fatal(err)
			}
			if before != after {
				t.Fatal("failed migration changed original schema")
			}
			var version int
			if err := db.QueryRow("SELECT MAX(version) FROM schema_migrations").Scan(&version); err != nil || version != 1 {
				t.Fatalf("failed migration changed version: %d, %v", version, err)
			}
			// Remove only the fixture's failure, then verify data through HTTP after retry.
			switch stage {
			case "script":
				_, err = db.Exec("DROP TABLE relation_groups")
			case "version":
				_, err = db.Exec("DROP TRIGGER fail_version")
			case "commit":
				_, err = db.Exec("DROP TRIGGER fail_commit")
			}
			if err != nil {
				t.Fatal(err)
			}
			s := startApp(t, dir)
			var got entry
			request(t, s, "GET", "/api/entries/1", "", 200, &got)
			if got.Spelling != "Legacy" || got.Definition != "升级前\n原释义" {
				t.Fatalf("original data lost: %+v", got)
			}
		})
	}
}

func TestAllPendingMigrationsRollBackTogether(t *testing.T) {
	dir := t.TempDir()
	db := fixtureDB(t, dir, `
		CREATE TABLE schema_migrations (version INTEGER PRIMARY KEY, applied_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP);
		CREATE TRIGGER fail_second BEFORE INSERT ON schema_migrations
		WHEN NEW.version = 2 BEGIN SELECT RAISE(ABORT, 'controlled second migration failure'); END;
	`)
	a, err := app.Open(dir)
	if err == nil {
		a.Close()
		t.Fatal("second migration must fail")
	}
	// Verify the legacy fixture, including the first migration's schema, is unchanged.
	var versions, entries int
	if err := db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&versions); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE name = 'entries'").Scan(&entries); err != nil {
		t.Fatal(err)
	}
	if versions != 0 || entries != 0 {
		t.Fatalf("partial migrations remain: versions=%d entries=%d", versions, entries)
	}
	if _, err := db.Exec("DROP TRIGGER fail_second"); err != nil {
		t.Fatal(err)
	}
	s := startApp(t, dir)
	var got []entry
	request(t, s, "GET", "/api/entries", "", 200, &got)
	if got == nil || len(got) != 0 {
		t.Fatalf("retry failed: %+v", got)
	}
}

func TestBackupFailurePreventsUpgrade(t *testing.T) {
	dir := t.TempDir()
	db := fixtureDB(t, dir, legacySchema)
	if err := os.WriteFile(filepath.Join(dir, "backups"), []byte("blocked"), 0600); err != nil {
		t.Fatal(err)
	}
	a, err := app.Open(dir)
	if err == nil {
		a.Close()
		t.Fatal("backup failure must stop startup")
	}
	if !strings.Contains(err.Error(), "backup") {
		t.Fatalf("unhelpful error: %v", err)
	}
	var version int
	if err := db.QueryRow("SELECT MAX(version) FROM schema_migrations").Scan(&version); err != nil || version != 1 {
		t.Fatalf("backup failure changed version: %d, %v", version, err)
	}
}

func TestCurrentAndNewerDatabasesAreNotMigrated(t *testing.T) {
	dir := t.TempDir()
	a, err := app.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	a.Close()
	if _, err := os.Stat(filepath.Join(dir, "backups")); !os.IsNotExist(err) {
		t.Fatalf("new notebook unexpectedly backed up: %v", err)
	}
	// A current-version restart must work even when the backup location is unwritable.
	if err := os.WriteFile(filepath.Join(dir, "backups"), []byte("blocked"), 0600); err != nil {
		t.Fatal(err)
	}
	s := startApp(t, dir)
	var got []entry
	request(t, s, "GET", "/api/entries", "", 200, &got)
	if !reflect.DeepEqual(got, []entry{}) {
		t.Fatalf("want empty notebook: %+v", got)
	}
	newerDir := t.TempDir()
	db := fixtureDB(t, newerDir, legacySchema+"INSERT INTO schema_migrations(version) VALUES (999);")
	a, err = app.Open(newerDir)
	if err == nil {
		a.Close()
		t.Fatal("newer database must be rejected")
	}
	if !strings.Contains(err.Error(), "newer") {
		t.Fatalf("unhelpful error: %v", err)
	}
	var version int
	if err := db.QueryRow("SELECT MAX(version) FROM schema_migrations").Scan(&version); err != nil || version != 999 {
		t.Fatalf("database downgraded: %d, %v", version, err)
	}
}

func TestUpgradePreservesEntryAndBackup(t *testing.T) {
	dir := t.TempDir()
	// Keep committed data in WAL while upgrading: copying only the main file loses it.
	db := fixtureDB(t, dir, `PRAGMA journal_mode=WAL;`+legacySchema)
	s := startApp(t, dir)
	var got entry
	request(t, s, "GET", "/api/entries/1", "", 200, &got)
	want := entry{1, "Legacy", "升级前\n原释义", "2026-11-01T00:00:00Z"}
	if got != want {
		t.Fatalf("upgrade changed entry: %+v", got)
	}
	backups, err := filepath.Glob(filepath.Join(dir, "backups", "*.db"))
	if err != nil || len(backups) != 1 {
		t.Fatalf("want one upgrade backup, got %v: %v", backups, err)
	}
	data, err := os.ReadFile(backups[0])
	if err != nil {
		t.Fatal(err)
	}
	restoredDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(restoredDir, "wordweave.db"), data, 0600); err != nil {
		t.Fatal(err)
	}
	// The restored snapshot must still be at the pre-upgrade version.
	restored := fixtureDB(t, restoredDir, "")
	var version int
	if err := restored.QueryRow("SELECT MAX(version) FROM schema_migrations").Scan(&version); err != nil || version != 1 {
		t.Fatalf("backup version = %d, error %v", version, err)
	}
	restored.Close()
	restarted := startApp(t, restoredDir)
	request(t, restarted, "GET", "/api/entries/1", "", 200, &got)
	if got != want {
		t.Fatalf("backup did not restore original entry: %+v", got)
	}
	db.Close()
}
