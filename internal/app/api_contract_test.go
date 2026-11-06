package app_test

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestEntryCreationRejectsUnknownFieldsAndTrailingData(t *testing.T) {
	s := startApp(t, t.TempDir())
	request(t, s, "POST", "/api/entries", `{"spelling":"pear","definiton":"fruit"}`, http.StatusBadRequest, nil)
	request(t, s, "POST", "/api/entries", `{"spelling":"pear"} trailing`, http.StatusBadRequest, nil)
	request(t, s, "POST", "/api/entries", `{"spelling":"pear","definition":null}`, http.StatusBadRequest, nil)
	var entries []entry
	request(t, s, "GET", "/api/entries", "", http.StatusOK, &entries)
	if len(entries) != 0 {
		t.Fatalf("invalid requests created entries: %+v", entries)
	}
}

func TestUnsupportedMethodReturnsJSONError(t *testing.T) {
	s := startApp(t, t.TempDir())
	var problem struct {
		Code  string `json:"code"`
		Error string `json:"error"`
	}
	request(t, s, "POST", "/api/stats", "", http.StatusMethodNotAllowed, &problem)
	if problem.Code != "method_not_allowed" || problem.Error == "" {
		t.Fatalf("unexpected error: %+v", problem)
	}
}

func TestConditionalEntryUpdateRejectsStaleVersion(t *testing.T) {
	s := startApp(t, t.TempDir())
	original := collect(t, s, "apple", "old")
	path := fmt.Sprintf("/api/entries/%d", original.ID)
	headers := request(t, s, "GET", path, "", http.StatusOK, nil)
	version := headers.Get("ETag")
	if version == "" {
		t.Fatal("entry response lacks ETag")
	}
	listRequest, err := http.NewRequest("PUT", s.URL+path, strings.NewReader(`{"spelling":"apple","definition":"old"}`))
	if err != nil {
		t.Fatal(err)
	}
	listRequest.Header.Set("If-Match", `"other", `+version)
	listResponse, err := s.Client().Do(listRequest)
	if err != nil {
		t.Fatal(err)
	}
	listResponse.Body.Close()
	if listResponse.StatusCode != http.StatusOK {
		t.Fatalf("matching tag in list: status %d", listResponse.StatusCode)
	}
	request(t, s, "PUT", path, `{"spelling":"apple","definition":"new"}`, http.StatusOK, nil)
	req, err := http.NewRequest("PUT", s.URL+path, strings.NewReader(`{"spelling":"apple","definition":"stale"}`))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("If-Match", version)
	res, err := s.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusPreconditionFailed {
		body, _ := io.ReadAll(res.Body)
		t.Fatalf("status %d: %s", res.StatusCode, body)
	}
	var got entry
	request(t, s, "GET", path, "", http.StatusOK, &got)
	if got.Definition != "new" {
		t.Fatalf("stale update overwrote entry: %+v", got)
	}
}

func TestPatchGroupPreservesMembersAndRejectsInvalidChanges(t *testing.T) {
	s := startApp(t, t.TempDir())
	a := collect(t, s, "apple", "")
	b := collect(t, s, "pear", "")
	original := createGroup(t, s, a.ID, "语义相关", "fruit", b.ID)
	path := fmt.Sprintf("/api/groups/%d", original.ID)
	var got group
	request(t, s, "PATCH", path, `{"note":""}`, http.StatusOK, &got)
	if got.Note != "" || len(got.Members) != 2 || got.Type != original.Type {
		t.Fatalf("unexpected group: %+v", got)
	}
	request(t, s, "PATCH", path, `{"note":null}`, http.StatusBadRequest, nil)
	request(t, s, "PATCH", path, fmt.Sprintf(`{"memberIds":[%d]}`, a.ID), http.StatusBadRequest, nil)
}

func TestDeleteEntryReportsAffectedGroups(t *testing.T) {
	s := startApp(t, t.TempDir())
	a := collect(t, s, "a", "")
	b := collect(t, s, "b", "")
	c := collect(t, s, "c", "")
	removed := createGroup(t, s, a.ID, "反义", "", b.ID)
	updated := createGroup(t, s, a.ID, "近义", "", b.ID, c.ID)
	var result struct {
		Deleted         bool    `json:"deleted"`
		DeletedGroupIDs []int64 `json:"deletedGroupIds"`
		UpdatedGroupIDs []int64 `json:"updatedGroupIds"`
	}
	request(t, s, "DELETE", fmt.Sprintf("/api/entries/%d", a.ID), "", http.StatusOK, &result)
	if !result.Deleted || len(result.DeletedGroupIDs) != 1 || result.DeletedGroupIDs[0] != removed.ID || len(result.UpdatedGroupIDs) != 1 || result.UpdatedGroupIDs[0] != updated.ID {
		t.Fatalf("unexpected deletion result: %+v", result)
	}
}

func TestCreateGroupWithoutOriginEntry(t *testing.T) {
	s := startApp(t, t.TempDir())
	a := collect(t, s, "apple", "")
	b := collect(t, s, "pear", "")
	body := fmt.Sprintf(`{"type":"语义相关","memberIds":[%d,%d],"note":"fruit"}`, a.ID, b.ID)
	var got group
	request(t, s, "POST", "/api/groups", body, http.StatusCreated, &got)
	if len(got.Members) != 2 || got.Note != "fruit" {
		t.Fatalf("unexpected group: %+v", got)
	}
	request(t, s, "POST", "/api/groups", body, http.StatusConflict, nil)
}

func TestPatchEntryPreservesOmittedFields(t *testing.T) {
	s := startApp(t, t.TempDir())
	original := collect(t, s, "apple", "old")
	path := fmt.Sprintf("/api/entries/%d", original.ID)
	var got entry
	request(t, s, "PATCH", path, `{"definition":"new"}`, http.StatusOK, &got)
	if got.Spelling != "apple" || got.Definition != "new" {
		t.Fatalf("unexpected patched entry: %+v", got)
	}
	request(t, s, "PATCH", path, `{"definition":null}`, http.StatusBadRequest, nil)
	request(t, s, "PATCH", path, `{"definition":""}`, http.StatusOK, &got)
	if got.Definition != "" || got.Spelling != "apple" {
		t.Fatalf("unexpected cleared entry: %+v", got)
	}
}

func TestFindEntryByExactSpelling(t *testing.T) {
	s := startApp(t, t.TempDir())
	want := collect(t, s, "Apple", "fruit")
	collect(t, s, "apple pie", "apple")
	var got entry
	request(t, s, "GET", "/api/entries/exact?spelling="+url.QueryEscape(" APPLE "), "", http.StatusOK, &got)
	if got != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}
	request(t, s, "GET", "/api/entries/exact?spelling=missing", "", http.StatusNotFound, nil)
}

func TestIncompleteBodyDoesNotBlockOtherWrites(t *testing.T) {
	s := startApp(t, t.TempDir())
	reader, writer := io.Pipe()
	defer writer.Close()
	req, err := http.NewRequest("POST", s.URL+"/api/entries", reader)
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		res, err := s.Client().Do(req)
		if err == nil {
			res.Body.Close()
		}
	}()
	if _, err := writer.Write([]byte(`{"spelling":`)); err != nil {
		t.Fatal(err)
	}
	client := &http.Client{Timeout: 2 * time.Second}
	res, err := client.Post(s.URL+"/api/entries", "application/json", strings.NewReader(`{"spelling":"pear"}`))
	writer.Close()
	<-done
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("status %d", res.StatusCode)
	}
}

func TestBrowseGroupsPagination(t *testing.T) {
	s := startApp(t, t.TempDir())
	a := collect(t, s, "a", "")
	b := collect(t, s, "b", "")
	c := collect(t, s, "c", "")
	g1 := createGroup(t, s, a.ID, "近义", "", b.ID)
	g2 := createGroup(t, s, a.ID, "近义", "", c.ID)
	g3 := createGroup(t, s, b.ID, "近义", "", c.ID)
	for _, tc := range []struct {
		path string
		want []int64
	}{
		{"/api/groups?sort=created_asc&limit=2&offset=0", []int64{g1.ID, g2.ID}},
		{"/api/groups?sort=created_asc&limit=2&offset=2", []int64{g3.ID}},
		{"/api/groups?sort=created_desc", []int64{g3.ID, g2.ID, g1.ID}},
	} {
		var got []group
		request(t, s, "GET", tc.path, "", http.StatusOK, &got)
		var ids []int64
		for _, g := range got {
			ids = append(ids, g.ID)
		}
		if !reflect.DeepEqual(ids, tc.want) {
			t.Fatalf("%s got %v want %v", tc.path, ids, tc.want)
		}
	}
}

func TestOversizedRequestIsRejected(t *testing.T) {
	s := startApp(t, t.TempDir())
	request(t, s, "POST", "/api/entries", strings.Repeat("x", 16<<20+1), http.StatusRequestEntityTooLarge, nil)
	var entries []entry
	request(t, s, "GET", "/api/entries", "", http.StatusOK, &entries)
	if len(entries) != 0 {
		t.Fatal("oversized request changed data")
	}
}
