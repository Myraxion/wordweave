package app_test

import (
	"net/http"
	"reflect"
	"strconv"
	"testing"
)

func TestDeleteEntryConflictIsAtomic(t *testing.T) {
	s := startApp(t, t.TempDir())
	a := collect(t, s, "A", "a")
	b := collect(t, s, "B", "b")
	c := collect(t, s, "C", "c")
	large := createGroup(t, s, c.ID, "近义", "large C", a.ID, b.ID)
	small := createGroup(t, s, a.ID, "近义", "small", b.ID)
	path := "/api/entries/" + strconv.FormatInt(c.ID, 10)
	var conflict struct {
		Error         string `json:"error"`
		ExistingGroup group  `json:"existingGroup"`
	}
	request(t, s, "DELETE", path, "", http.StatusConflict, &conflict)
	if conflict.Error == "" || !reflect.DeepEqual(conflict.ExistingGroup, small) {
		t.Fatalf("conflict: %+v", conflict)
	}
	var got entry
	request(t, s, "GET", path, "", http.StatusOK, &got)
	if got != c {
		t.Fatalf("entry changed: %+v", got)
	}
	for _, want := range []group{large, small} {
		var actual group
		request(t, s, "GET", "/api/groups/"+strconv.FormatInt(want.ID, 10), "", http.StatusOK, &actual)
		if !reflect.DeepEqual(actual, want) {
			t.Fatalf("group changed: %+v", actual)
		}
	}
}

func TestDeleteEntryUpdatesAllGroups(t *testing.T) {
	s := startApp(t, t.TempDir())
	a := collect(t, s, "A", "")
	b := collect(t, s, "B", "")
	c := collect(t, s, "C", "")
	d := collect(t, s, "D", "")
	kept := createGroup(t, s, c.ID, "近义", "C remains in this note", a.ID, b.ID)
	removed := createGroup(t, s, c.ID, "反义", "removed", d.ID)
	path := "/api/entries/" + strconv.FormatInt(c.ID, 10)
	var preview []group
	request(t, s, "GET", groupsPath(c.ID), "", http.StatusOK, &preview)
	if len(preview) != 2 {
		t.Fatalf("preview: %+v", preview)
	}
	request(t, s, "DELETE", path, "", http.StatusOK, nil)
	request(t, s, "GET", path, "", http.StatusNotFound, nil)
	request(t, s, "GET", "/api/groups/"+strconv.FormatInt(removed.ID, 10), "", http.StatusNotFound, nil)
	var got group
	request(t, s, "GET", "/api/groups/"+strconv.FormatInt(kept.ID, 10), "", http.StatusOK, &got)
	kept.Members = []entry{a, b}
	if !reflect.DeepEqual(got, kept) {
		t.Fatalf("remaining group: %+v", got)
	}
	request(t, s, "DELETE", "/api/entries/"+strconv.FormatInt(d.ID, 10), "", http.StatusOK, nil)
	request(t, s, "DELETE", path, "", http.StatusNotFound, nil)
}
