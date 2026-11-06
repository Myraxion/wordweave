package app_test

import (
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"
)

func requireUpdatedSince(t *testing.T, stamp string, since time.Time) {
	t.Helper()
	got, err := time.Parse(time.RFC3339Nano, stamp)
	if err != nil || got.Before(since) || got.After(time.Now()) || got.Location() != time.UTC {
		t.Fatalf("updatedAt = %q, want UTC timestamp since %s (parse error: %v)", stamp, since, err)
	}
}

func TestEntryModificationTime(t *testing.T) {
	s := startApp(t, t.TempDir())
	before := time.Now()
	original := collect(t, s, "apple", "苹果")
	if original.UpdatedAt != original.CreatedAt {
		t.Fatalf("new entry timestamps differ: %+v", original)
	}
	requireUpdatedSince(t, original.UpdatedAt, before)
	path := "/api/entries/" + strconv.FormatInt(original.ID, 10)

	for _, body := range []string{
		`{"spelling":"Apple","definition":"苹果"}`,
		`{"spelling":"Apple","definition":"苹果树"}`,
	} {
		before = time.Now()
		var updated entry
		request(t, s, "PUT", path, body, http.StatusOK, &updated)
		requireUpdatedSince(t, updated.UpdatedAt, before)
		if updated.CreatedAt != original.CreatedAt {
			t.Fatalf("edit changed createdAt: %+v", updated)
		}
	}

	var saved, unchanged entry
	request(t, s, "GET", path, "", http.StatusOK, &saved)
	request(t, s, "PUT", path, `{"spelling":" Apple ","definition":"苹果树"}`, http.StatusOK, &unchanged)
	if unchanged != saved {
		t.Fatalf("unchanged save changed entry: got %+v, want %+v", unchanged, saved)
	}
	var list []entry
	request(t, s, "GET", "/api/entries", "", http.StatusOK, &list)
	if !reflect.DeepEqual(list, []entry{saved}) {
		t.Fatalf("list has stale timestamps: %+v", list)
	}
}

func TestGroupModificationTime(t *testing.T) {
	s := startApp(t, t.TempDir())
	a := collect(t, s, "A", "")
	b := collect(t, s, "B", "")
	c := collect(t, s, "C", "")
	before := time.Now()
	original := createGroup(t, s, a.ID, "近义", "原说明", b.ID)
	if original.UpdatedAt != original.CreatedAt {
		t.Fatalf("new group timestamps differ: %+v", original)
	}
	requireUpdatedSince(t, original.UpdatedAt, before)
	path := "/api/groups/" + strconv.FormatInt(original.ID, 10)

	var unchanged group
	request(t, s, "PUT", path, groupBody(t, "近义", "原说明", []int64{b.ID, a.ID, b.ID}), http.StatusOK, &unchanged)
	if !reflect.DeepEqual(unchanged, original) {
		t.Fatalf("member reordering changed group: %+v", unchanged)
	}
	for _, change := range []struct {
		kind, note string
		ids        []int64
	}{
		{"反义", "原说明", []int64{a.ID, b.ID}},
		{"反义", "新说明", []int64{a.ID, b.ID}},
		{"反义", "新说明", []int64{b.ID, c.ID}},
	} {
		before = time.Now()
		var updated group
		request(t, s, "PUT", path, groupBody(t, change.kind, change.note, change.ids), http.StatusOK, &updated)
		requireUpdatedSince(t, updated.UpdatedAt, before)
		if updated.CreatedAt != original.CreatedAt {
			t.Fatalf("edit changed createdAt: %+v", updated)
		}
	}
	for _, want := range []entry{a, b, c} {
		var got entry
		request(t, s, "GET", "/api/entries/"+strconv.FormatInt(want.ID, 10), "", http.StatusOK, &got)
		if got != want {
			t.Fatalf("group changes modified member entry: %+v", got)
		}
	}
}

func TestEntriesSortedByModificationTime(t *testing.T) {
	dir := t.TempDir()
	s := startApp(t, dir)
	db := fixtureDB(t, dir, "")
	// Same-second fractions and an unknown legacy timestamp must sort as actual instants.
	for i, stamp := range []any{
		"2020-01-02T00:00:00Z",
		"2020-01-02T00:00:00.1Z",
		"2020-01-02T00:00:00.11Z",
		"2020-01-02T00:00:00.110000001Z",
		"2020-01-02T00:00:00.11Z",
		nil,
	} {
		created := "2020-01-01T00:00:00Z"
		if stamp == nil {
			created = "2020-01-02T00:00:00.105Z"
		}
		spelling := "sort-" + strconv.Itoa(i+1)
		if _, err := db.Exec(`INSERT INTO entries (id, spelling, spelling_key, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?)`, i+1, spelling, spelling, created, stamp); err != nil {
			t.Fatal(err)
		}
	}
	for _, tc := range []struct {
		name, sort string
		want       []int64
	}{
		{"default", "", []int64{4, 5, 3, 6, 2, 1}},
		{"unknown", "unknown", []int64{4, 5, 3, 6, 2, 1}},
		{"updated_desc", "updated_desc", []int64{4, 5, 3, 6, 2, 1}},
		{"updated_asc", "updated_asc", []int64{1, 2, 6, 3, 5, 4}},
		{"created_desc", "created_desc", []int64{6, 5, 4, 3, 2, 1}},
		{"created_asc", "created_asc", []int64{1, 2, 3, 4, 5, 6}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, endpoint := range []string{"/api/entries?", "/api/search?q=sort-&"} {
				var entries []entry
				if endpoint == "/api/entries?" {
					request(t, s, "GET", endpoint+"sort="+tc.sort, "", http.StatusOK, &entries)
				} else {
					var result struct {
						Entries []entry `json:"entries"`
					}
					request(t, s, "GET", endpoint+"sort="+tc.sort, "", http.StatusOK, &result)
					entries = result.Entries
				}
				var ids []int64
				for _, e := range entries {
					ids = append(ids, e.ID)
					if e.ID == 6 && e.UpdatedAt != "" {
						t.Fatal("sorting must not fill the unknown legacy modification time")
					}
				}
				if !reflect.DeepEqual(ids, tc.want) {
					t.Fatalf("%s: got IDs %v, want %v", endpoint, ids, tc.want)
				}
			}
		})
	}
}

func TestEntriesSortedByCreationTimeFractions(t *testing.T) {
	dir := t.TempDir()
	s := startApp(t, dir)
	db := fixtureDB(t, dir, "")
	for i, stamp := range []string{"2020-01-02T00:00:00.1Z", "2020-01-02T00:00:00.11Z", "2020-01-02T00:00:00.110000001Z"} {
		if _, err := db.Exec("INSERT INTO entries(spelling, spelling_key, definition, created_at) VALUES (?, ?, '', ?)", strconv.Itoa(i), strconv.Itoa(i), stamp); err != nil {
			t.Fatal(err)
		}
	}
	for _, tc := range []struct {
		path string
		want []int64
	}{{"/api/entries?sort=created_asc", []int64{1, 2, 3}}, {"/api/search?sort=created_desc", []int64{3, 2, 1}}} {
		var entries []entry
		if strings.HasPrefix(tc.path, "/api/search") {
			var result struct {
				Entries []entry `json:"entries"`
			}
			request(t, s, "GET", tc.path, "", http.StatusOK, &result)
			entries = result.Entries
		} else {
			request(t, s, "GET", tc.path, "", http.StatusOK, &entries)
		}
		var ids []int64
		for _, e := range entries {
			ids = append(ids, e.ID)
		}
		if !reflect.DeepEqual(ids, tc.want) {
			t.Fatalf("%s: got %v want %v", tc.path, ids, tc.want)
		}
	}
}
