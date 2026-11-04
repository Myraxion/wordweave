package app_test

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

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

const legacyGroupSchema = `
INSERT INTO schema_migrations(version) VALUES (2);
CREATE TABLE relation_groups (
 id INTEGER PRIMARY KEY AUTOINCREMENT, type TEXT NOT NULL, member_key TEXT NOT NULL,
 note TEXT NOT NULL DEFAULT '', created_at TEXT NOT NULL, UNIQUE(type, member_key)
);
CREATE TABLE group_members (
 group_id INTEGER NOT NULL REFERENCES relation_groups(id),
 entry_id INTEGER NOT NULL REFERENCES entries(id), PRIMARY KEY(group_id, entry_id)
);
INSERT INTO entries VALUES (2, 'Other', 'other', '', '2026-11-02T00:00:00Z');
INSERT INTO relation_groups VALUES (1, '近义', '1,2', '旧说明', '2026-11-02T00:00:00Z');
INSERT INTO group_members VALUES (1, 1), (1, 2);
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
	want := entry{ID: 1, Spelling: "Legacy", Definition: "升级前\n原释义", CreatedAt: "2026-11-01T00:00:00Z"}
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

func TestUpgradeKeepsModificationTimesUnknownUntilContentChanges(t *testing.T) {
	dir := t.TempDir()
	db := fixtureDB(t, dir, legacySchema+legacyGroupSchema)
	s := startApp(t, dir)
	// Check actual JSON null, including nested members, rather than a missing field.
	var raw map[string]json.RawMessage
	request(t, s, "GET", "/api/groups/1", "", 200, &raw)
	if string(raw["updatedAt"]) != "null" {
		t.Fatalf("legacy group updatedAt = %s, want null", raw["updatedAt"])
	}
	var members []map[string]json.RawMessage
	if err := json.Unmarshal(raw["members"], &members); err != nil || len(members) != 2 {
		t.Fatalf("legacy members: %s, error %v", raw["members"], err)
	}
	for _, member := range members {
		if string(member["updatedAt"]) != "null" {
			t.Fatalf("legacy member updatedAt = %s, want null", member["updatedAt"])
		}
	}

	var unchanged entry
	request(t, s, "PUT", "/api/entries/1", `{"spelling":" Legacy ","definition":"升级前\n原释义"}`, 200, &unchanged)
	request(t, s, "PUT", "/api/groups/1", groupBody(t, "近义", "旧说明", []int64{2, 1}), 200, &raw)
	if unchanged.UpdatedAt != "" || string(raw["updatedAt"]) != "null" {
		t.Fatal("unchanged legacy saves must retain unknown modification times")
	}
	for _, table := range []string{"entries", "relation_groups"} {
		var stamp sql.NullString
		if err := db.QueryRow("SELECT updated_at FROM " + table + " WHERE id = 1").Scan(&stamp); err != nil || stamp.Valid {
			t.Fatalf("%s: want database NULL, got %+v, error %v", table, stamp, err)
		}
	}

	before := time.Now()
	var edited entry
	request(t, s, "PUT", "/api/entries/1", `{"spelling":"Legacy","definition":"新释义"}`, 200, &edited)
	requireUpdatedSince(t, edited.UpdatedAt, before)
	if edited.CreatedAt != unchanged.CreatedAt {
		t.Fatal("legacy edit changed entry creation time")
	}
	request(t, s, "GET", "/api/groups/1", "", 200, &raw)
	if string(raw["updatedAt"]) != "null" {
		t.Fatal("member edit changed legacy group modification time")
	}
	before = time.Now()
	var editedGroup group
	request(t, s, "PUT", "/api/groups/1", groupBody(t, "近义", "新说明", []int64{1, 2}), 200, &editedGroup)
	requireUpdatedSince(t, editedGroup.UpdatedAt, before)
	if editedGroup.CreatedAt != "2026-11-02T00:00:00Z" {
		t.Fatal("legacy edit changed group creation time")
	}
	var version int
	if err := db.QueryRow("SELECT MAX(version) FROM schema_migrations").Scan(&version); err != nil || version != 3 {
		t.Fatalf("upgrade version = %d, error %v", version, err)
	}
}

func TestModificationTimeMigrationRollsBackOnFailure(t *testing.T) {
	dir := t.TempDir()
	db := fixtureDB(t, dir, legacySchema+legacyGroupSchema+`
		CREATE TRIGGER fail_third BEFORE INSERT ON schema_migrations
		WHEN NEW.version = 3 BEGIN SELECT RAISE(ABORT, 'controlled timestamp migration failure'); END;
	`)
	a, err := app.Open(dir)
	if err == nil {
		a.Close()
		t.Fatal("failed timestamp migration must stop startup")
	}
	for _, table := range []string{"entries", "relation_groups"} {
		var count int
		if err := db.QueryRow("SELECT COUNT(*) FROM pragma_table_info(?) WHERE name = 'updated_at'", table).Scan(&count); err != nil || count != 0 {
			t.Fatalf("partial timestamp migration on %s: count %d, error %v", table, count, err)
		}
	}
	var version int
	if err := db.QueryRow("SELECT MAX(version) FROM schema_migrations").Scan(&version); err != nil || version != 2 {
		t.Fatalf("failed migration changed version: %d, error %v", version, err)
	}
	if _, err := db.Exec("DROP TRIGGER fail_third"); err != nil {
		t.Fatal(err)
	}
	s := startApp(t, dir)
	var got group
	request(t, s, "GET", "/api/groups/1", "", 200, &got)
	if got.Note != "旧说明" || len(got.Members) != 2 || got.UpdatedAt != "" {
		t.Fatalf("retry lost legacy group: %+v", got)
	}
}
