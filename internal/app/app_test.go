package app_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
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
