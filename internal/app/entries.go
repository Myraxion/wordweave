package app

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
)

type Entry struct {
	ID         int64   `json:"id"`
	Spelling   string  `json:"spelling"`
	Definition string  `json:"definition"`
	CreatedAt  string  `json:"createdAt"`
	UpdatedAt  *string `json:"updatedAt"`
}

var errEmptySpelling = errors.New("empty spelling")
var errDuplicateEntry = errors.New("duplicate entry")

func (a *App) addEntry(ctx context.Context, spelling, definition string) (Entry, error) {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	entry := Entry{Spelling: strings.TrimSpace(spelling), Definition: definition, CreatedAt: now, UpdatedAt: &now}
	if entry.Spelling == "" {
		return Entry{}, errEmptySpelling
	}
	err := a.db.QueryRowContext(ctx,
		`INSERT INTO entries (spelling, spelling_key, definition, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?) ON CONFLICT(spelling_key) DO NOTHING RETURNING id`,
		entry.Spelling, strings.ToLower(entry.Spelling), entry.Definition, entry.CreatedAt, entry.UpdatedAt,
	).Scan(&entry.ID)
	if errors.Is(err, sql.ErrNoRows) {
		var existing Entry
		err = a.db.QueryRowContext(ctx,
			`SELECT id, spelling, definition, created_at, updated_at FROM entries WHERE spelling_key = ?`, strings.ToLower(entry.Spelling),
		).Scan(&existing.ID, &existing.Spelling, &existing.Definition, &existing.CreatedAt, &existing.UpdatedAt)
		if err != nil {
			return Entry{}, err
		}
		return existing, errDuplicateEntry
	}
	return entry, err
}

func (a *App) listEntries(ctx context.Context, sortOrder, query string) ([]Entry, error) {
	var orderBy string
	switch sortOrder {
	case "created_asc":
		orderBy = "created_at ASC, id ASC"
	case "created_desc":
		orderBy = "created_at DESC, id DESC"
	case "updated_asc":
		orderBy = "rtrim(COALESCE(updated_at, created_at), 'Z') ASC, id ASC"
	case "alpha_asc":
		orderBy = "spelling_key ASC, id ASC"
	case "alpha_desc":
		orderBy = "spelling_key DESC, id DESC"
	default:
		// Strip the UTC suffix so variable-length RFC3339Nano fractions sort chronologically.
		orderBy = "rtrim(COALESCE(updated_at, created_at), 'Z') DESC, id DESC"
	}
	query = strings.TrimSpace(query)
	rows, err := a.db.QueryContext(ctx, `SELECT id, spelling, definition, created_at, updated_at FROM entries
  WHERE instr(lower(spelling), lower(?)) > 0 OR instr(lower(definition), lower(?)) > 0 ORDER BY `+orderBy, query, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	entries := []Entry{}
	for rows.Next() {
		var entry Entry
		if err := rows.Scan(&entry.ID, &entry.Spelling, &entry.Definition, &entry.CreatedAt, &entry.UpdatedAt); err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, rows.Err()
}

func (a *App) getEntry(ctx context.Context, id int64) (Entry, error) {
	var entry Entry
	err := a.db.QueryRowContext(ctx, `SELECT id, spelling, definition, created_at, updated_at FROM entries WHERE id = ?`, id).
		Scan(&entry.ID, &entry.Spelling, &entry.Definition, &entry.CreatedAt, &entry.UpdatedAt)
	return entry, err
}

func (a *App) updateEntry(ctx context.Context, id int64, spelling, definition string) (Entry, error) {
	trimmed := strings.TrimSpace(spelling)
	if trimmed == "" {
		return Entry{}, errEmptySpelling
	}
	spellingKey := strings.ToLower(trimmed)

	var conflict Entry
	err := a.db.QueryRowContext(ctx,
		`SELECT id, spelling, definition, created_at, updated_at FROM entries WHERE spelling_key = ? AND id != ?`,
		spellingKey, id,
	).Scan(&conflict.ID, &conflict.Spelling, &conflict.Definition, &conflict.CreatedAt, &conflict.UpdatedAt)
	if err == nil {
		return conflict, errDuplicateEntry
	} else if !errors.Is(err, sql.ErrNoRows) {
		return Entry{}, err
	}

	var updated Entry
	err = a.db.QueryRowContext(ctx,
		`UPDATE entries SET
		 updated_at = CASE WHEN spelling != ? OR definition != ? THEN ? ELSE updated_at END,
		 spelling = ?, spelling_key = ?, definition = ? WHERE id = ?
		 RETURNING id, spelling, definition, created_at, updated_at`,
		trimmed, definition, time.Now().UTC().Format(time.RFC3339Nano), trimmed, spellingKey, definition, id,
	).Scan(&updated.ID, &updated.Spelling, &updated.Definition, &updated.CreatedAt, &updated.UpdatedAt)
	return updated, err
}
