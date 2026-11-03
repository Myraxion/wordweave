package app

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"

	"github.com/Myraxion/wordweave/web"
)

// App owns one notebook and serves its public HTTP interface.
type App struct {
	db      *sql.DB
	handler http.Handler
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
	if err := migrate(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("initialize database: %w", err)
	}
	a := &App{db: db}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/entries", a.createEntry)
	mux.HandleFunc("GET /api/entries", a.showEntries)
	mux.HandleFunc("GET /api/entries/{id}", a.showEntry)
	mux.HandleFunc("GET /api/", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusNotFound, apiError{Error: "找不到这个接口。"})
	})
	mux.Handle("GET /", web.Handler())
	a.handler = mux
	return a, nil
}

func (a *App) Close() error { return a.db.Close() }

func (a *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.handler.ServeHTTP(w, r)
}
