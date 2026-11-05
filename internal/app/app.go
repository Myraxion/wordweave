package app

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	_ "modernc.org/sqlite"

	"github.com/Myraxion/wordweave/web"
)

// App owns one notebook and serves its public HTTP interface.
type App struct {
	db      *sql.DB
	handler http.Handler
	writes  sync.Mutex
}

func Open(dataDir string) (*App, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("create data directory: %w", err)
	}
	db, err := sql.Open("sqlite", filepath.Join(dataDir, "wordweave.db"))
	if err != nil {
		return nil, err
	}
	// A single connection serializes writes for this single-user application.
	db.SetMaxOpenConns(1)
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		db.Close()
		return nil, err
	}
	if err := migrate(db, dataDir); err != nil {
		db.Close()
		return nil, fmt.Errorf("initialize database: %w", err)
	}
	a := &App{db: db}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/entries", a.createEntry)
	mux.HandleFunc("GET /api/entries", a.showEntries)
	mux.HandleFunc("GET /api/entries/exact", a.findEntry)
	mux.HandleFunc("GET /api/entries/{id}", a.showEntry)
	mux.HandleFunc("PUT /api/entries/{id}", a.editEntry)
	mux.HandleFunc("PATCH /api/entries/{id}", a.patchEntry)
	mux.HandleFunc("DELETE /api/entries/{id}", a.deleteEntry)
	mux.HandleFunc("POST /api/entries/{id}/groups", a.createGroup)
	mux.HandleFunc("GET /api/entries/{id}/groups", a.showEntryGroups)
	mux.HandleFunc("GET /api/search", a.search)
	mux.HandleFunc("GET /api/stats", a.showStats)
	mux.HandleFunc("GET /api/groups", a.browseGroups)
	mux.HandleFunc("POST /api/groups", a.createGroup)
	mux.HandleFunc("GET /api/groups/{id}", a.showGroup)
	mux.HandleFunc("PUT /api/groups/{id}", a.editGroup)
	mux.HandleFunc("PATCH /api/groups/{id}", a.patchGroup)
	mux.HandleFunc("DELETE /api/groups/{id}", a.deleteGroup)
	mux.HandleFunc("GET /api/group-types", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, relationTypes)
	})
	mux.HandleFunc("GET /api/", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusNotFound, apiError{Error: "找不到这个接口。"})
	})
	mux.Handle("GET /", web.Handler())
	a.handler = mux
	return a, nil
}

func (a *App) Close() error { return a.db.Close() }

type apiResponseWriter struct {
	http.ResponseWriter
	replaced bool
}

func (w *apiResponseWriter) WriteHeader(status int) {
	if status == http.StatusMethodNotAllowed {
		w.replaced = true
		writeJSON(w.ResponseWriter, status, apiError{Code: "method_not_allowed", Error: "不支持此请求方法。"})
		return
	}
	w.ResponseWriter.WriteHeader(status)
}

func (w *apiResponseWriter) Write(p []byte) (int, error) {
	if w.replaced {
		return len(p), nil
	}
	return w.ResponseWriter.Write(p)
}

func (a *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		a.writes.Lock()
		defer a.writes.Unlock()
		if match := r.Header.Get("If-Match"); match != "" && (r.Method == http.MethodPut || r.Method == http.MethodPatch || r.Method == http.MethodDelete) {
			parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
			if len(parts) == 3 && parts[0] == "api" && (parts[1] == "entries" || parts[1] == "groups") {
				id, err := strconv.ParseInt(parts[2], 10, 64)
				if err == nil && id > 0 {
					var value any
					if parts[1] == "entries" {
						value, err = a.getEntry(r.Context(), id)
					} else {
						value, err = a.getGroup(r.Context(), id)
					}
					matched := match == "*"
					for _, candidate := range strings.Split(match, ",") {
						matched = matched || strings.TrimSpace(candidate) == entityTag(value)
					}
					if err == nil && !matched {
						writeJSON(w, http.StatusPreconditionFailed, apiError{Error: "内容已变化，请重新读取后再操作。"})
						return
					}
				}
			}
		}
	}
	if strings.HasPrefix(r.URL.Path, "/api/") {
		a.handler.ServeHTTP(&apiResponseWriter{ResponseWriter: w}, r)
	} else {
		a.handler.ServeHTTP(w, r)
	}
}
