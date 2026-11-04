package app_test

import (
	"net/http"
	"reflect"
	"strconv"
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
