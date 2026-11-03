package app_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/Myraxion/wordweave/internal/app"
)

type entry struct {
	ID         int64  `json:"id"`
	Spelling   string `json:"spelling"`
	Definition string `json:"definition"`
	CreatedAt  string `json:"createdAt"`
}

// Every test gets a real, isolated database and uses only HTTP to observe it.
func startApp(t *testing.T, dataDir string) *httptest.Server {
	t.Helper()
	a, err := app.Open(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	s := httptest.NewServer(a)
	t.Cleanup(func() {
		s.Close()
		if err := a.Close(); err != nil {
			t.Error(err)
		}
	})
	return s
}

func request(t *testing.T, s *httptest.Server, method, path, body string, wantStatus int, result any) http.Header {
	t.Helper()
	req, err := http.NewRequest(method, s.URL+path, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := s.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != wantStatus {
		t.Fatalf("%s %s: status %d, want %d; body: %s", method, path, res.StatusCode, wantStatus, data)
	}
	if result != nil {
		if err := json.Unmarshal(data, result); err != nil {
			t.Fatalf("decode %s: %v", data, err)
		}
	}
	return res.Header
}

func TestCollectEntry(t *testing.T) {
	s := startApp(t, t.TempDir())
	var got entry
	request(t, s, "POST", "/api/entries", `{"spelling":" Apple ","definition":"苹果\n苹果树"}`, http.StatusCreated, &got)
	if got.ID <= 0 || got.Spelling != "Apple" || got.Definition != "苹果\n苹果树" || got.CreatedAt == "" {
		t.Fatalf("unexpected entry: %+v", got)
	}
}

func TestCollectWordsAndPhrases(t *testing.T) {
	s := startApp(t, t.TempDir())
	request(t, s, "POST", "/api/entries", `{"spelling":" \t\n "}`, http.StatusBadRequest, nil)
	for _, spelling := range []string{"give  up", "mother-in-law", "don't"} {
		body, err := json.Marshal(map[string]string{"spelling": " " + spelling + " "})
		if err != nil {
			t.Fatal(err)
		}
		var got entry
		request(t, s, "POST", "/api/entries", string(body), http.StatusCreated, &got)
		if got.Spelling != spelling || got.Definition != "" {
			t.Fatalf("spelling or empty definition changed: %+v", got)
		}
	}
}

func TestBrowseEntries(t *testing.T) {
	s := startApp(t, t.TempDir())
	var list []entry
	request(t, s, "GET", "/api/entries", "", http.StatusOK, &list)
	if list == nil || len(list) != 0 {
		t.Fatalf("new notebook should have an empty list: %+v", list)
	}
	var first, second entry
	request(t, s, "POST", "/api/entries", `{"spelling":"Apple"}`, http.StatusCreated, &first)
	request(t, s, "POST", "/api/entries", `{"spelling":"give up","definition":"放弃\n<script>纯文本</script>"}`, http.StatusCreated, &second)
	request(t, s, "GET", "/api/entries", "", http.StatusOK, &list)
	if len(list) != 2 || list[0] != second || list[1] != first {
		t.Fatalf("want newest entry first, got %+v", list)
	}
	for _, want := range []entry{first, second} {
		var got entry
		request(t, s, "GET", "/api/entries/"+strconv.FormatInt(want.ID, 10), "", http.StatusOK, &got)
		if got != want {
			t.Fatalf("detail: got %+v, want %+v", got, want)
		}
	}
	request(t, s, "GET", "/api/entries/999999", "", http.StatusNotFound, nil)
}

func TestDuplicateEntryLinksToOriginal(t *testing.T) {
	s := startApp(t, t.TempDir())
	var original entry
	request(t, s, "POST", "/api/entries", `{"spelling":"Apple","definition":"原释义"}`, http.StatusCreated, &original)
	var conflict struct {
		Error         string `json:"error"`
		ExistingEntry entry  `json:"existingEntry"`
	}
	request(t, s, "POST", "/api/entries", `{"spelling":" aPpLe ","definition":"不应覆盖"}`, http.StatusConflict, &conflict)
	if conflict.Error == "" || conflict.ExistingEntry != original {
		t.Fatalf("want original entry in conflict response: %+v", conflict)
	}
	var list []entry
	request(t, s, "GET", "/api/entries", "", http.StatusOK, &list)
	if len(list) != 1 || list[0] != original {
		t.Fatalf("duplicate changed notebook: %+v", list)
	}
}

func TestRestartRetainsNotebook(t *testing.T) {
	dir := t.TempDir()
	a, err := app.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	s := httptest.NewServer(a)
	t.Cleanup(func() { s.Close(); a.Close() })
	var original entry
	request(t, s, "POST", "/api/entries", `{"spelling":"don't","definition":"不要\n保持原文"}`, http.StatusCreated, &original)
	s.Close()
	if err := a.Close(); err != nil {
		t.Fatal(err)
	}
	restarted := startApp(t, dir)
	var got entry
	request(t, restarted, "GET", "/api/entries/"+strconv.FormatInt(original.ID, 10), "", http.StatusOK, &got)
	if got != original {
		t.Fatalf("restart changed entry: %+v, want %+v", got, original)
	}
	request(t, restarted, "POST", "/api/entries", `{"spelling":" DON'T "}`, http.StatusConflict, nil)
	var list []entry
	request(t, restarted, "GET", "/api/entries", "", http.StatusOK, &list)
	if len(list) != 1 || list[0] != original {
		t.Fatalf("restart changed notebook: %+v", list)
	}
}

func TestUpdateEntry(t *testing.T) {
	s := startApp(t, t.TempDir())
	var original entry
	request(t, s, "POST", "/api/entries", `{"spelling":"apple","definition":"苹果"}`, http.StatusCreated, &original)

	var updated entry
	request(t, s, "PUT", "/api/entries/"+strconv.FormatInt(original.ID, 10),
		`{"spelling":" Apple Pie ","definition":"苹果派\n美味甜点"}`, http.StatusOK, &updated)

	if updated.ID != original.ID {
		t.Fatalf("id changed: got %d, want %d", updated.ID, original.ID)
	}
	if updated.Spelling != "Apple Pie" {
		t.Fatalf("spelling not trimmed/updated: got %q", updated.Spelling)
	}
	if updated.Definition != "苹果派\n美味甜点" {
		t.Fatalf("definition not updated: got %q", updated.Definition)
	}
	if updated.CreatedAt != original.CreatedAt {
		t.Fatalf("createdAt changed: got %q, want %q", updated.CreatedAt, original.CreatedAt)
	}

	// Verify detail endpoint reflects updated entry
	var detail entry
	request(t, s, "GET", "/api/entries/"+strconv.FormatInt(original.ID, 10), "", http.StatusOK, &detail)
	if detail != updated {
		t.Fatalf("detail does not match updated entry: got %+v, want %+v", detail, updated)
	}

	// Clearing definition is permitted
	var cleared entry
	request(t, s, "PUT", "/api/entries/"+strconv.FormatInt(original.ID, 10),
		`{"spelling":"Apple Pie","definition":""}`, http.StatusOK, &cleared)
	if cleared.Definition != "" {
		t.Fatalf("definition was not cleared: got %q", cleared.Definition)
	}

	// Empty spelling should be rejected
	request(t, s, "PUT", "/api/entries/"+strconv.FormatInt(original.ID, 10),
		`{"spelling":"   \t\n  ","definition":"test"}`, http.StatusBadRequest, nil)

	// Nonexistent entry should return 404
	request(t, s, "PUT", "/api/entries/999999",
		`{"spelling":"ghost","definition":"none"}`, http.StatusNotFound, nil)
}

func TestUpdateEntryConflict(t *testing.T) {
	s := startApp(t, t.TempDir())
	var first, second entry
	request(t, s, "POST", "/api/entries", `{"spelling":"Apple","definition":"苹果"}`, http.StatusCreated, &first)
	request(t, s, "POST", "/api/entries", `{"spelling":"Banana","definition":"香蕉"}`, http.StatusCreated, &second)

	// Attempt to rename second to " apple " should conflict with first
	var conflict struct {
		Error         string `json:"error"`
		ExistingEntry entry  `json:"existingEntry"`
	}
	request(t, s, "PUT", "/api/entries/"+strconv.FormatInt(second.ID, 10),
		`{"spelling":"  aPpLe  ","definition":"试图覆盖"}`, http.StatusConflict, &conflict)

	if conflict.Error == "" || conflict.ExistingEntry != first {
		t.Fatalf("expected conflict pointing to first entry: %+v", conflict)
	}

	// Verify neither entry changed after rejected rename
	var checkSecond, checkFirst entry
	request(t, s, "GET", "/api/entries/"+strconv.FormatInt(second.ID, 10), "", http.StatusOK, &checkSecond)
	if checkSecond != second {
		t.Fatalf("second entry changed after conflict: got %+v, want %+v", checkSecond, second)
	}
	request(t, s, "GET", "/api/entries/"+strconv.FormatInt(first.ID, 10), "", http.StatusOK, &checkFirst)
	if checkFirst != first {
		t.Fatalf("first entry changed after conflict: got %+v, want %+v", checkFirst, first)
	}

	// Changing case of the same entry should succeed
	var caseUpdated entry
	request(t, s, "PUT", "/api/entries/"+strconv.FormatInt(first.ID, 10),
		`{"spelling":"APPLE","definition":"苹果"}`, http.StatusOK, &caseUpdated)
	if caseUpdated.Spelling != "APPLE" {
		t.Fatalf("case update failed: got %q, want %q", caseUpdated.Spelling, "APPLE")
	}
}

func TestBrowseEntriesSorted(t *testing.T) {
	s := startApp(t, t.TempDir())
	var banana, apple, cherry entry
	request(t, s, "POST", "/api/entries", `{"spelling":"banana"}`, http.StatusCreated, &banana)
	request(t, s, "POST", "/api/entries", `{"spelling":"Apple"}`, http.StatusCreated, &apple)
	request(t, s, "POST", "/api/entries", `{"spelling":"cherry"}`, http.StatusCreated, &cherry)

	// 1. Default (no sort parameter): newest created first
	var listDefault []entry
	request(t, s, "GET", "/api/entries", "", http.StatusOK, &listDefault)
	if len(listDefault) != 3 || listDefault[0].ID != cherry.ID || listDefault[1].ID != apple.ID || listDefault[2].ID != banana.ID {
		t.Fatalf("default sort want [cherry, apple, banana], got %+v", listDefault)
	}

	// 2. Explicit created_desc: newest created first
	var listCreatedDesc []entry
	request(t, s, "GET", "/api/entries?sort=created_desc", "", http.StatusOK, &listCreatedDesc)
	if len(listCreatedDesc) != 3 || listCreatedDesc[0].ID != cherry.ID || listCreatedDesc[1].ID != apple.ID || listCreatedDesc[2].ID != banana.ID {
		t.Fatalf("created_desc want [cherry, apple, banana], got %+v", listCreatedDesc)
	}

	// 3. created_asc: oldest created first
	var listCreatedAsc []entry
	request(t, s, "GET", "/api/entries?sort=created_asc", "", http.StatusOK, &listCreatedAsc)
	if len(listCreatedAsc) != 3 || listCreatedAsc[0].ID != banana.ID || listCreatedAsc[1].ID != apple.ID || listCreatedAsc[2].ID != cherry.ID {
		t.Fatalf("created_asc want [banana, apple, cherry], got %+v", listCreatedAsc)
	}

	// 4. alpha_asc: alphabetical A-Z ignoring case
	var listAlphaAsc []entry
	request(t, s, "GET", "/api/entries?sort=alpha_asc", "", http.StatusOK, &listAlphaAsc)
	if len(listAlphaAsc) != 3 || listAlphaAsc[0].ID != apple.ID || listAlphaAsc[1].ID != banana.ID || listAlphaAsc[2].ID != cherry.ID {
		t.Fatalf("alpha_asc want [Apple, banana, cherry], got %+v", listAlphaAsc)
	}

	// 5. alpha_desc: alphabetical Z-A ignoring case
	var listAlphaDesc []entry
	request(t, s, "GET", "/api/entries?sort=alpha_desc", "", http.StatusOK, &listAlphaDesc)
	if len(listAlphaDesc) != 3 || listAlphaDesc[0].ID != cherry.ID || listAlphaDesc[1].ID != banana.ID || listAlphaDesc[2].ID != apple.ID {
		t.Fatalf("alpha_desc want [cherry, banana, Apple], got %+v", listAlphaDesc)
	}

	// 6. Editing an entry retains creation time and position in created_desc
	request(t, s, "PUT", "/api/entries/"+strconv.FormatInt(banana.ID, 10),
		`{"spelling":"banana split","definition":"美味冰淇淋"}`, http.StatusOK, nil)
	var listAfterEdit []entry
	request(t, s, "GET", "/api/entries?sort=created_desc", "", http.StatusOK, &listAfterEdit)
	if len(listAfterEdit) != 3 || listAfterEdit[2].ID != banana.ID || listAfterEdit[2].Spelling != "banana split" {
		t.Fatalf("edit should retain original creation order, got %+v", listAfterEdit)
	}
}

func TestSearchEntries(t *testing.T) {
	s := startApp(t, t.TempDir())
	for _, body := range []string{`{"spelling":"Apple Pie","definition":"苹果派 Fruit"}`, `{"spelling":"give up"}`, `{"spelling":"give it up","definition":"放弃"}`, `{"spelling":"100%_done","definition":"完成"}`, `{"spelling":"don't"}`} {
		request(t, s, "POST", "/api/entries", body, http.StatusCreated, nil)
	}
	for _, tc := range []struct {
		query string
		want  []string
	}{
		{"APPLE PIE", []string{"Apple Pie"}}, {"pPlE", []string{"Apple Pie"}}, {"fruit", []string{"Apple Pie"}},
		{"苹果", []string{"Apple Pie"}}, {"give up", []string{"give up"}}, {"give  up", []string{}},
		{"%_", []string{"100%_done"}}, {"%", []string{"100%_done"}}, {"_", []string{"100%_done"}}, {"'", []string{"don't"}},
		{"missing", []string{}}, {"  apple  ", []string{"Apple Pie"}},
		{" ", []string{"100%_done", "Apple Pie", "don't", "give it up", "give up"}},
		{"", []string{"100%_done", "Apple Pie", "don't", "give it up", "give up"}},
	} {
		t.Run(tc.query, func(t *testing.T) {
			var got []entry
			request(t, s, "GET", "/api/entries?sort=alpha_asc&q="+url.QueryEscape(tc.query), "", http.StatusOK, &got)
			names := []string{}
			for _, e := range got {
				names = append(names, e.Spelling)
			}
			if !reflect.DeepEqual(names, tc.want) {
				t.Fatalf("got %v, want %v", names, tc.want)
			}
		})
	}
}
