package app

import (
	"bytes"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
)

func decodeJSON(r *http.Request, value any) error {
	decoder := json.NewDecoder(r.Body)
	var raw json.RawMessage
	if err := decoder.Decode(&raw); err != nil {
		return err
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		if err == nil {
			return errors.New("multiple JSON values")
		}
		return err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil || fields == nil {
		return errors.New("JSON object required")
	}
	for _, field := range fields {
		if bytes.Equal(bytes.TrimSpace(field), []byte("null")) {
			return errors.New("null is not allowed")
		}
	}
	strict := json.NewDecoder(bytes.NewReader(raw))
	strict.DisallowUnknownFields()
	return strict.Decode(value)
}

type apiError struct {
	Code          string `json:"code"`
	Error         string `json:"error"`
	ExistingEntry *Entry `json:"existingEntry,omitempty"`
	ExistingGroup *Group `json:"existingGroup,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	if problem, ok := value.(apiError); ok && problem.Code == "" {
		switch {
		case problem.ExistingEntry != nil:
			problem.Code = "duplicate_entry"
		case problem.ExistingGroup != nil:
			problem.Code = "duplicate_group"
		case status == http.StatusBadRequest:
			problem.Code = "invalid_request"
		case status == http.StatusNotFound:
			problem.Code = "not_found"
		case status == http.StatusPreconditionFailed:
			problem.Code = "version_conflict"
		default:
			problem.Code = "internal_error"
		}
		value = problem
	}
	if status == http.StatusOK || status == http.StatusCreated {
		switch value.(type) {
		case Entry, Group:
			w.Header().Set("ETag", entityTag(value))
		}
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("write response: %v", err)
	}
}

func entityTag(value any) string {
	data, _ := json.Marshal(value)
	return fmt.Sprintf("\"%x\"", sha256.Sum256(data))
}

func internalError(w http.ResponseWriter, err error) {
	log.Printf("request failed: %v", err)
	writeJSON(w, http.StatusInternalServerError, apiError{Error: "暂时无法处理请求，请重试。"})
}

func (a *App) showStats(w http.ResponseWriter, r *http.Request) {
	var totals struct {
		Entries int `json:"entries"`
		Groups  int `json:"groups"`
	}
	err := a.db.QueryRowContext(r.Context(), `SELECT
		(SELECT COUNT(*) FROM entries),
		(SELECT COUNT(*) FROM relation_groups)`).Scan(&totals.Entries, &totals.Groups)
	if err != nil {
		internalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, totals)
}

func (a *App) createEntry(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Spelling   string `json:"spelling"`
		Definition string `json:"definition"`
	}
	if err := decodeJSON(r, &input); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError{Error: "请输入有效的 JSON 词条。"})
		return
	}
	entry, err := a.addEntry(r.Context(), input.Spelling, input.Definition)
	if errors.Is(err, errDuplicateEntry) {
		writeJSON(w, http.StatusConflict, apiError{Error: "这个词条已收录。", ExistingEntry: &entry})
		return
	}
	if errors.Is(err, errEmptySpelling) {
		writeJSON(w, http.StatusBadRequest, apiError{Error: "请输入词条拼写。"})
		return
	}
	if err != nil {
		internalError(w, err)
		return
	}
	w.Header().Set("Location", "/api/entries/"+strconv.FormatInt(entry.ID, 10))
	writeJSON(w, http.StatusCreated, entry)
}

func (a *App) showEntries(w http.ResponseWriter, r *http.Request) {
	sortOrder := r.URL.Query().Get("sort")
	entries, err := a.listEntries(r.Context(), sortOrder, r.URL.Query().Get("q"))
	if err != nil {
		internalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, entries)
}

func (a *App) findEntry(w http.ResponseWriter, r *http.Request) {
	key := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("spelling")))
	if key == "" {
		writeJSON(w, http.StatusBadRequest, apiError{Error: "请输入词条拼写。"})
		return
	}
	var entry Entry
	err := a.db.QueryRowContext(r.Context(), `SELECT id, spelling, definition, created_at, updated_at FROM entries WHERE spelling_key = ?`, key).
		Scan(&entry.ID, &entry.Spelling, &entry.Definition, &entry.CreatedAt, &entry.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		writeJSON(w, http.StatusNotFound, apiError{Error: "找不到这个词条。"})
		return
	}
	if err != nil {
		internalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, entry)
}

func (a *App) showEntry(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		writeJSON(w, http.StatusBadRequest, apiError{Error: "无效的词条编号。"})
		return
	}
	entry, err := a.getEntry(r.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		writeJSON(w, http.StatusNotFound, apiError{Error: "找不到这个词条。"})
		return
	}
	if err != nil {
		internalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, entry)
}

func (a *App) editEntry(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		writeJSON(w, http.StatusBadRequest, apiError{Error: "无效的词条编号。"})
		return
	}
	var input struct {
		Spelling   string `json:"spelling"`
		Definition string `json:"definition"`
	}
	if err := decodeJSON(r, &input); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError{Error: "请输入有效的 JSON 词条。"})
		return
	}
	entry, err := a.updateEntry(r.Context(), id, input.Spelling, input.Definition)
	if errors.Is(err, errDuplicateEntry) {
		writeJSON(w, http.StatusConflict, apiError{Error: "这个词条已收录。", ExistingEntry: &entry})
		return
	}
	if errors.Is(err, errEmptySpelling) {
		writeJSON(w, http.StatusBadRequest, apiError{Error: "请输入词条拼写。"})
		return
	}
	if errors.Is(err, sql.ErrNoRows) {
		writeJSON(w, http.StatusNotFound, apiError{Error: "找不到这个词条。"})
		return
	}
	if err != nil {
		internalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, entry)
}

func (a *App) patchEntry(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		writeJSON(w, http.StatusBadRequest, apiError{Error: "无效的词条编号。"})
		return
	}
	var input struct {
		Spelling   json.RawMessage `json:"spelling"`
		Definition json.RawMessage `json:"definition"`
	}
	if decodeJSON(r, &input) != nil {
		writeJSON(w, http.StatusBadRequest, apiError{Error: "请输入有效的 JSON 词条。"})
		return
	}
	entry, err := a.getEntry(r.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		writeJSON(w, http.StatusNotFound, apiError{Error: "找不到这个词条。"})
		return
	}
	if err != nil {
		internalError(w, err)
		return
	}
	for _, field := range []struct {
		raw    json.RawMessage
		target *string
	}{{input.Spelling, &entry.Spelling}, {input.Definition, &entry.Definition}} {
		if field.raw != nil && (string(field.raw) == "null" || json.Unmarshal(field.raw, field.target) != nil) {
			writeJSON(w, http.StatusBadRequest, apiError{Error: "词条字段必须是字符串。"})
			return
		}
	}
	entry, err = a.updateEntry(r.Context(), id, entry.Spelling, entry.Definition)
	switch {
	case errors.Is(err, errDuplicateEntry):
		writeJSON(w, http.StatusConflict, apiError{Error: "这个词条已收录。", ExistingEntry: &entry})
	case errors.Is(err, errEmptySpelling):
		writeJSON(w, http.StatusBadRequest, apiError{Error: "请输入词条拼写。"})
	case err != nil:
		internalError(w, err)
	default:
		writeJSON(w, http.StatusOK, entry)
	}
}

func (a *App) deleteEntry(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		writeJSON(w, http.StatusBadRequest, apiError{Error: "无效的词条编号。"})
		return
	}
	updated, deleted := []int64{}, []int64{}
	group, err := a.removeEntry(r.Context(), id, &updated, &deleted)
	switch {
	case errors.Is(err, errDuplicateGroup):
		writeJSON(w, http.StatusConflict, apiError{Error: "删除会产生重复的关系组，请先整理冲突组的说明和成员。", ExistingGroup: &group})
	case errors.Is(err, sql.ErrNoRows):
		writeJSON(w, http.StatusNotFound, apiError{Error: "找不到这个词条。"})
	case err != nil:
		internalError(w, err)
	default:
		writeJSON(w, http.StatusOK, struct {
			Deleted         bool    `json:"deleted"`
			UpdatedGroupIDs []int64 `json:"updatedGroupIds"`
			DeletedGroupIDs []int64 `json:"deletedGroupIds"`
		}{true, updated, deleted})
	}
}
