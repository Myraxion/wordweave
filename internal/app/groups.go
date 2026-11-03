package app

import (
	"context"
	"database/sql"
	"errors"
	"slices"
	"strconv"
	"strings"
	"time"
)

var relationTypes = []string{"形音相似", "近义", "反义", "词族", "屈折变化", "拼写变体", "语义相关"}

type Group struct {
	ID        int64   `json:"id"`
	Type      string  `json:"type"`
	Note      string  `json:"note"`
	CreatedAt string  `json:"createdAt"`
	Members   []Entry `json:"members"`
}

var errInvalidGroup = errors.New("invalid group")
var errDuplicateGroup = errors.New("duplicate group")

func (a *App) addGroup(ctx context.Context, kind, note string, memberIDs []int64) (Group, error) {
	ids := slices.Clone(memberIDs)
	slices.Sort(ids)
	ids = slices.Compact(ids)
	if !slices.Contains(relationTypes, kind) || len(ids) < 2 || ids[0] <= 0 {
		return Group{}, errInvalidGroup
	}
	keys := make([]string, len(ids))
	for i, id := range ids {
		keys[i] = strconv.FormatInt(id, 10)
	}
	key := strings.Join(keys, ",")
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return Group{}, err
	}
	defer tx.Rollback()
	var id int64
	err = tx.QueryRowContext(ctx, `INSERT INTO relation_groups (type, member_key, note, created_at)
		VALUES (?, ?, ?, ?) ON CONFLICT(type, member_key) DO NOTHING RETURNING id`,
		kind, key, note, time.Now().UTC().Format(time.RFC3339Nano)).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		if err := tx.QueryRowContext(ctx, "SELECT id FROM relation_groups WHERE type = ? AND member_key = ?", kind, key).Scan(&id); err != nil {
			return Group{}, err
		}
		tx.Rollback() // Release the single connection before reading the existing group.
		existing, err := a.getGroup(ctx, id)
		if err != nil {
			return Group{}, err
		}
		return existing, errDuplicateGroup
	}
	if err != nil {
		return Group{}, err
	}
	for _, memberID := range ids {
		result, err := tx.ExecContext(ctx, `INSERT INTO group_members (group_id, entry_id)
			SELECT ?, id FROM entries WHERE id = ?`, id, memberID)
		if err != nil {
			return Group{}, err
		}
		count, err := result.RowsAffected()
		if err != nil {
			return Group{}, err
		}
		if count != 1 {
			return Group{}, errInvalidGroup
		}
	}
	if err := tx.Commit(); err != nil {
		return Group{}, err
	}
	return a.getGroup(ctx, id)
}

func (a *App) getGroup(ctx context.Context, id int64) (Group, error) {
	groups, err := a.queryGroups(ctx, "g.id = ?", id)
	if err != nil {
		return Group{}, err
	}
	if len(groups) == 0 {
		return Group{}, sql.ErrNoRows
	}
	return groups[0], nil
}

func (a *App) entryGroups(ctx context.Context, entryID int64) ([]Group, error) {
	return a.queryGroups(ctx, "g.id IN (SELECT group_id FROM group_members WHERE entry_id = ?)", entryID)
}

// Only callers in this file supply the WHERE clause; all values are parameters.
// One joined query returns a consistent view, including current entry definitions.
// AUTOINCREMENT IDs track creation order without comparing variable-precision timestamps.
func (a *App) queryGroups(ctx context.Context, where string, id int64) ([]Group, error) {
	rows, err := a.db.QueryContext(ctx, `SELECT g.id, g.type, g.note, g.created_at,
		e.id, e.spelling, e.definition, e.created_at
		FROM relation_groups g
		JOIN group_members m ON m.group_id = g.id
		JOIN entries e ON e.id = m.entry_id
		WHERE `+where+`
		ORDER BY g.id DESC, e.spelling_key ASC, e.id ASC`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	groups := []Group{}
	for rows.Next() {
		var group Group
		var entry Entry
		if err := rows.Scan(&group.ID, &group.Type, &group.Note, &group.CreatedAt,
			&entry.ID, &entry.Spelling, &entry.Definition, &entry.CreatedAt); err != nil {
			return nil, err
		}
		if len(groups) == 0 || groups[len(groups)-1].ID != group.ID {
			groups = append(groups, group)
		}
		last := &groups[len(groups)-1]
		last.Members = append(last.Members, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	slices.SortStableFunc(groups, func(a, b Group) int {
		return slices.Index(relationTypes, a.Type) - slices.Index(relationTypes, b.Type)
	})
	return groups, nil
}
