package app

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
)

type Entry struct {
	ID         int64  `json:"id"`
	Spelling   string `json:"spelling"`
	Definition string `json:"definition"`
	CreatedAt  string `json:"createdAt"`
}

var errEmptySpelling = errors.New("empty spelling")
var errDuplicateEntry = errors.New("duplicate entry")

func (a *App) addEntry(ctx context.Context, spelling, definition string) (Entry, error) {
	entry := Entry{Spelling: strings.TrimSpace(spelling), Definition: definition, CreatedAt: time.Now().UTC().Format(time.RFC3339Nano)}
	if entry.Spelling == "" {
		return Entry{}, errEmptySpelling
	}
	err := a.db.QueryRowContext(ctx,
		`INSERT INTO entries (spelling, spelling_key, definition, created_at)
		 VALUES (?, ?, ?, ?) ON CONFLICT(spelling_key) DO NOTHING RETURNING id`,
		entry.Spelling, strings.ToLower(entry.Spelling), entry.Definition, entry.CreatedAt,
	).Scan(&entry.ID)
	if errors.Is(err, sql.ErrNoRows) {
		var existing Entry
		err = a.db.QueryRowContext(ctx,
			`SELECT id, spelling, definition, created_at FROM entries WHERE spelling_key = ?`, strings.ToLower(entry.Spelling),
		).Scan(&existing.ID, &existing.Spelling, &existing.Definition, &existing.CreatedAt)
		if err != nil {
			return Entry{}, err
		}
		return existing, errDuplicateEntry
	}
	return entry, err
}

func (a *App) listEntries(ctx context.Context) ([]Entry, error) {
	rows, err := a.db.QueryContext(ctx, `SELECT id, spelling, definition, created_at FROM entries ORDER BY id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	entries := []Entry{}
	for rows.Next() {
		var entry Entry
		if err := rows.Scan(&entry.ID, &entry.Spelling, &entry.Definition, &entry.CreatedAt); err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, rows.Err()
}

func (a *App) getEntry(ctx context.Context, id int64) (Entry, error) {
	var entry Entry
	err := a.db.QueryRowContext(ctx, `SELECT id, spelling, definition, created_at FROM entries WHERE id = ?`, id).
		Scan(&entry.ID, &entry.Spelling, &entry.Definition, &entry.CreatedAt)
	return entry, err
}
