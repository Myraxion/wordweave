package app

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"strings"
	"time"
)

// removeEntry checks all resulting groups and applies the deletion in one transaction.
func (a *App) removeEntry(ctx context.Context, id int64, updated, deleted *[]int64) (Group, error) {
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return Group{}, err
	}
	defer tx.Rollback()
	var found int64
	if err = tx.QueryRowContext(ctx, "SELECT id FROM entries WHERE id = ?", id).Scan(&found); err != nil {
		return Group{}, err
	}
	rows, err := tx.QueryContext(ctx, `SELECT g.id, g.type, g.member_key FROM relation_groups g
  JOIN group_members m ON m.group_id = g.id WHERE m.entry_id = ?`, id)
	if err != nil {
		return Group{}, err
	}
	type affected struct {
		id        int64
		kind, key string
		count     int
	}
	var groups []affected
	for rows.Next() {
		var g affected
		var key string
		if err = rows.Scan(&g.id, &g.kind, &key); err != nil {
			rows.Close()
			return Group{}, err
		}
		var remaining []string
		for _, member := range strings.Split(key, ",") {
			if member != strconv.FormatInt(id, 10) {
				remaining = append(remaining, member)
			}
		}
		g.key = strings.Join(remaining, ",")
		g.count = len(remaining)
		groups = append(groups, g)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return Group{}, err
	}
	for _, g := range groups {
		if g.count < 2 {
			continue
		}
		var conflictID int64
		err = tx.QueryRowContext(ctx, "SELECT id FROM relation_groups WHERE type = ? AND member_key = ? AND id != ?", g.kind, g.key, g.id).Scan(&conflictID)
		if err == nil {
			tx.Rollback()
			conflict, err := a.getGroup(ctx, conflictID)
			if err != nil {
				return Group{}, err
			}
			return conflict, errDuplicateGroup
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return Group{}, err
		}
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	for _, g := range groups {
		if g.count < 2 {
			*deleted = append(*deleted, g.id)
			if _, err = tx.ExecContext(ctx, "DELETE FROM group_members WHERE group_id = ?", g.id); err != nil {
				return Group{}, err
			}
			if _, err = tx.ExecContext(ctx, "DELETE FROM relation_groups WHERE id = ?", g.id); err != nil {
				return Group{}, err
			}
		} else {
			*updated = append(*updated, g.id)
			if _, err = tx.ExecContext(ctx, "DELETE FROM group_members WHERE group_id = ? AND entry_id = ?", g.id, id); err != nil {
				return Group{}, err
			}
			if _, err = tx.ExecContext(ctx, "UPDATE relation_groups SET member_key = ?, updated_at = ? WHERE id = ?", g.key, now, g.id); err != nil {
				return Group{}, err
			}
		}
	}
	if _, err = tx.ExecContext(ctx, "DELETE FROM entries WHERE id = ?", id); err != nil {
		return Group{}, err
	}
	return Group{}, tx.Commit()
}
