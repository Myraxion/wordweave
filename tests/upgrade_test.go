package tests

import (
	"context"
	"database/sql"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

// A notebook produced by schema version 1, before relation groups existed.
const oldNotebook = `
CREATE TABLE schema_migrations (version INTEGER PRIMARY KEY, applied_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP);
INSERT INTO schema_migrations(version) VALUES (1);
CREATE TABLE entries (
 id INTEGER PRIMARY KEY AUTOINCREMENT, spelling TEXT NOT NULL, spelling_key TEXT NOT NULL UNIQUE,
 definition TEXT NOT NULL DEFAULT '', created_at TEXT NOT NULL
);
INSERT INTO entries VALUES (1, 'Legacy', 'legacy', 'original definition', '2026-11-01T00:00:00Z');
`

func legacyNotebook(t *testing.T, extra string) string {
	t.Helper()
	dir := t.TempDir()
	db, err := sql.Open("sqlite", filepath.Join(dir, "wordweave.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(oldNotebook + extra); err != nil {
		t.Fatal(err)
	}
	return dir
}

func assertLegacyEntry(t *testing.T, url string) {
	t.Helper()
	body := fetch(t, url+"/api/entries/1", 200)
	if !strings.Contains(body, `"spelling":"Legacy"`) || !strings.Contains(body, `"definition":"original definition"`) {
		t.Fatalf("legacy entry lost: %s", body)
	}
}

func TestExecutableUpgrade(t *testing.T) {
	exe := buildExecutable(t)
	workingDir := t.TempDir()
	t.Run("upgrade and restore backup", func(t *testing.T) {
		dir := legacyNotebook(t, "")
		url, stop := launch(t, exe, workingDir, dir)
		assertLegacyEntry(t, url)
		fetch(t, url+"/api/entries/1/groups", 200)
		stop()
		backups, err := filepath.Glob(filepath.Join(dir, "backups", "*.db"))
		if err != nil || len(backups) != 1 {
			t.Fatalf("missing upgrade backup: %v, %v", backups, err)
		}
		data, err := os.ReadFile(backups[0])
		if err != nil {
			t.Fatal(err)
		}
		restored := t.TempDir()
		if err := os.WriteFile(filepath.Join(restored, "wordweave.db"), data, 0600); err != nil {
			t.Fatal(err)
		}
		url, stop = launch(t, exe, workingDir, restored)
		assertLegacyEntry(t, url)
		stop()
	})
	for _, stage := range []string{"migration", "backup", "newer"} {
		t.Run(stage+" failure stops startup", func(t *testing.T) {
			extra := ""
			if stage == "migration" {
				extra = `CREATE TRIGGER fail_upgrade BEFORE INSERT ON schema_migrations
					WHEN NEW.version = 2 BEGIN SELECT RAISE(ABORT, 'controlled upgrade failure'); END;`
			}
			if stage == "newer" {
				extra = "INSERT INTO schema_migrations(version) VALUES (999);"
			}
			dir := legacyNotebook(t, extra)
			if stage == "backup" {
				if err := os.WriteFile(filepath.Join(dir, "backups"), []byte("blocked"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, exe, "-data-dir", dir, "-listen", "127.0.0.1:0")
			output, err := cmd.CombinedOutput()
			if err == nil || ctx.Err() != nil || strings.Contains(string(output), "Wordweave:") {
				t.Fatalf("program must exit before serving: err=%v, context=%v, output=%s", err, ctx.Err(), output)
			}
			if !strings.Contains(string(output), stage) {
				t.Fatalf("startup error should identify %s failure: %s", stage, output)
			}
			if stage == "migration" {
				db, err := sql.Open("sqlite", filepath.Join(dir, "wordweave.db"))
				if err != nil {
					t.Fatal(err)
				}
				var version int
				err = db.QueryRow("SELECT MAX(version) FROM schema_migrations").Scan(&version)
				if err != nil || version != 1 {
					db.Close()
					t.Fatalf("failed executable changed database version: %d, %v", version, err)
				}
				_, err = db.Exec("DROP TRIGGER fail_upgrade")
				db.Close()
				if err != nil {
					t.Fatal(err)
				}
				url, stop := launch(t, exe, workingDir, dir)
				assertLegacyEntry(t, url)
				stop()
			}
		})
	}
}
