package tests

import (
	"bufio"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

// Builds the shipping binary and runs it without a frontend server or source files.
func TestPortableExecutable(t *testing.T) {
	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	portable := filepath.Join(t.TempDir(), "portable")
	if err := os.MkdirAll(portable, 0755); err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(portable, "wordweave")
	if runtime.GOOS == "windows" {
		exe += ".exe"
	}
	build := exec.Command("go", "build", "-trimpath", "-ldflags=-s -w", "-o", exe, "./cmd/wordweave")
	build.Dir = root
	build.Env = append(os.Environ(), "CGO_ENABLED=0")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, output)
	}
	workingDir := t.TempDir()
	url, stop := launch(t, exe, workingDir, "")
	index := fetch(t, url+"/entries/new", http.StatusOK)
	if !strings.Contains(index, "<div id=\"app\"></div>") {
		t.Fatalf("missing embedded application: %s", index)
	}
	assets := regexp.MustCompile(`(?:src|href)="(/assets/[^"]+)"`).FindAllStringSubmatch(index, -1)
	if len(assets) < 2 {
		t.Fatalf("expected local JS and CSS assets: %s", index)
	}
	for _, asset := range assets {
		if body := fetch(t, url+asset[1], http.StatusOK); len(body) == 0 {
			t.Fatalf("empty asset: %s", asset[1])
		}
	}
	fetch(t, url+"/assets/missing.js", http.StatusNotFound)
	fetch(t, url+"/api/missing", http.StatusNotFound)
	res, err := http.Post(url+"/api/entries", "application/json", strings.NewReader(`{"spelling":"Portable","definition":"离线可用\n重启保留"}`))
	if err != nil {
		t.Fatal(err)
	}
	var saved struct {
		ID         int64  `json:"id"`
		Spelling   string `json:"spelling"`
		Definition string `json:"definition"`
	}
	decodeErr := json.NewDecoder(res.Body).Decode(&saved)
	res.Body.Close()
	if res.StatusCode != http.StatusCreated || decodeErr != nil || saved.ID <= 0 {
		t.Fatalf("save: status %d, entry %+v, decode %v", res.StatusCode, saved, decodeErr)
	}
	stop()
	if _, err := os.Stat(filepath.Join(portable, "data", "wordweave.db")); err != nil {
		t.Fatalf("default data must be beside executable: %v", err)
	}
	if _, err := os.Stat(filepath.Join(workingDir, "data")); !os.IsNotExist(err) {
		t.Fatalf("unexpected data in working directory: %v", err)
	}
	url, stop = launch(t, exe, workingDir, "")
	assertRetained(t, url)
	stop()
	// Move the stopped application together with its external data directory.
	moved := filepath.Join(filepath.Dir(portable), "moved")
	if err := os.Rename(portable, moved); err != nil {
		t.Fatal(err)
	}
	exe = filepath.Join(moved, filepath.Base(exe))
	url, stop = launch(t, exe, workingDir, "")
	assertRetained(t, url)
	stop()
	// An explicit flag wins over the environment; either can select another notebook.
	envDir, flagDir := t.TempDir(), t.TempDir()
	url, stop = launch(t, exe, workingDir, envDir)
	if body := fetch(t, url+"/api/entries", http.StatusOK); strings.TrimSpace(body) != "[]" {
		t.Fatalf("environment-selected notebook should be empty: %s", body)
	}
	stop()
	if _, err := os.Stat(filepath.Join(envDir, "wordweave.db")); err != nil {
		t.Fatal(err)
	}
	url, stop = launch(t, exe, workingDir, envDir, "-data-dir", flagDir)
	fetch(t, url+"/api/entries", http.StatusOK)
	stop()
	if _, err := os.Stat(filepath.Join(flagDir, "wordweave.db")); err != nil {
		t.Fatal(err)
	}
}

func launch(t *testing.T, exe, workingDir, envDir string, args ...string) (string, func()) {
	t.Helper()
	cmd := exec.Command(exe, append([]string{"-listen", "127.0.0.1:0"}, args...)...)
	cmd.Dir = workingDir
	// Do not inherit a user's data-directory setting during isolated smoke tests.
	for _, value := range os.Environ() {
		if !strings.HasPrefix(strings.ToUpper(value), "WORDWEAVE_DATA_DIR=") {
			cmd.Env = append(cmd.Env, value)
		}
	}
	cmd.Env = append(cmd.Env, "WORDWEAVE_DATA_DIR="+envDir)
	cmd.Stderr = os.Stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	stop := sync.OnceFunc(func() {
		cmd.Process.Kill()
		cmd.Wait()
	})
	t.Cleanup(stop)
	ready := make(chan string, 1)
	go func() {
		scanner := bufio.NewScanner(stdout)
		if scanner.Scan() {
			ready <- strings.TrimPrefix(scanner.Text(), "Wordweave: ")
		} else {
			ready <- ""
		}
		for scanner.Scan() {
		}
	}()
	select {
	case url := <-ready:
		if !strings.HasPrefix(url, "http://127.0.0.1:") {
			t.Fatalf("program did not start on loopback: %q", url)
		}
		return url, stop
	case <-time.After(15 * time.Second):
		t.Fatal("program startup timed out")
		return "", stop
	}
}

func fetch(t *testing.T, url string, status int) string {
	t.Helper()
	client := &http.Client{Timeout: 5 * time.Second}
	res, err := client.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	if err != nil || res.StatusCode != status {
		t.Fatalf("GET %s: status %d, read %v, body %s", url, res.StatusCode, err, data)
	}
	return string(data)
}

func assertRetained(t *testing.T, url string) {
	t.Helper()
	var entries []struct {
		Spelling   string `json:"spelling"`
		Definition string `json:"definition"`
	}
	if err := json.Unmarshal([]byte(fetch(t, url+"/api/entries", http.StatusOK)), &entries); err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Spelling != "Portable" || entries[0].Definition != "离线可用\n重启保留" {
		t.Fatalf("portable notebook lost data: %+v", entries)
	}
}
