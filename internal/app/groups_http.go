package app

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
)

func (a *App) search(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	entries, err := a.listEntries(r.Context(), r.URL.Query().Get("sort"), query)
	if err != nil {
		internalError(w, err)
		return
	}
	groups := []Group{}
	if query != "" {
		groups, err = a.searchGroups(r.Context(), query)
		if err != nil {
			internalError(w, err)
			return
		}
	}
	writeJSON(w, http.StatusOK, struct {
		Entries []Entry `json:"entries"`
		Groups  []Group `json:"groups"`
	}{entries, groups})
}

func (a *App) browseGroups(w http.ResponseWriter, r *http.Request) {
	options := groupBrowseOptions{kind: r.URL.Query().Get("type"), sort: r.URL.Query().Get("sort")}
	if raw := r.URL.Query().Get("limit"); raw != "" {
		limit, err := strconv.Atoi(raw)
		if err != nil || limit < 1 || limit > 500 {
			writeJSON(w, http.StatusBadRequest, apiError{Error: "无效的分页参数。"})
			return
		}
		options.limit = limit
		if value := r.URL.Query().Get("offset"); value != "" {
			offset, err := strconv.Atoi(value)
			if err != nil || offset < 0 {
				writeJSON(w, http.StatusBadRequest, apiError{Error: "无效的分页参数。"})
				return
			}
			options.offset = offset
		}
	}
	groups, err := a.browseGroupRecords(r.Context(), options)
	if err != nil {
		internalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, groups)
}

func (a *App) editGroup(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		writeJSON(w, http.StatusBadRequest, apiError{Error: "无效的关系组编号。"})
		return
	}
	var input struct {
		Type      string  `json:"type"`
		Note      string  `json:"note"`
		MemberIDs []int64 `json:"memberIds"`
	}
	if decodeJSON(r, &input) != nil {
		writeJSON(w, http.StatusBadRequest, apiError{Error: "请输入有效的 JSON 关系组。"})
		return
	}
	group, err := a.updateGroup(r.Context(), id, input.Type, input.Note, input.MemberIDs)
	switch {
	case errors.Is(err, errInvalidGroup):
		writeJSON(w, http.StatusBadRequest, apiError{Error: "请选择一种关系类型和至少两个不同的已有词条。"})
	case errors.Is(err, errDuplicateGroup):
		writeJSON(w, http.StatusConflict, apiError{Error: "这个关系组已存在。", ExistingGroup: &group})
	case errors.Is(err, sql.ErrNoRows):
		writeJSON(w, http.StatusNotFound, apiError{Error: "找不到这个关系组。"})
	case err != nil:
		internalError(w, err)
	default:
		writeJSON(w, http.StatusOK, group)
	}
}

func (a *App) deleteGroup(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		writeJSON(w, http.StatusBadRequest, apiError{Error: "无效的关系组编号。"})
		return
	}
	err = a.removeGroup(r.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		writeJSON(w, http.StatusNotFound, apiError{Error: "找不到这个关系组。"})
		return
	}
	if err != nil {
		internalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"deleted": true})
}

func (a *App) patchGroup(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		writeJSON(w, http.StatusBadRequest, apiError{Error: "无效的关系组编号。"})
		return
	}
	var input struct {
		Type      json.RawMessage `json:"type"`
		Note      json.RawMessage `json:"note"`
		MemberIDs json.RawMessage `json:"memberIds"`
	}
	if decodeJSON(r, &input) != nil {
		writeJSON(w, http.StatusBadRequest, apiError{Error: "请输入有效的 JSON 关系组。"})
		return
	}
	group, err := a.getGroup(r.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		writeJSON(w, http.StatusNotFound, apiError{Error: "找不到这个关系组。"})
		return
	}
	if err != nil {
		internalError(w, err)
		return
	}
	ids := make([]int64, 0, len(group.Members))
	for _, member := range group.Members {
		ids = append(ids, member.ID)
	}
	for _, field := range []struct {
		raw    json.RawMessage
		target any
	}{{input.Type, &group.Type}, {input.Note, &group.Note}, {input.MemberIDs, &ids}} {
		if field.raw != nil && (string(field.raw) == "null" || json.Unmarshal(field.raw, field.target) != nil) {
			writeJSON(w, http.StatusBadRequest, apiError{Error: "关系组字段无效。"})
			return
		}
	}
	group, err = a.updateGroup(r.Context(), id, group.Type, group.Note, ids)
	switch {
	case errors.Is(err, errInvalidGroup):
		writeJSON(w, http.StatusBadRequest, apiError{Error: "请选择一种关系类型和至少两个不同的已有词条。"})
	case errors.Is(err, errDuplicateGroup):
		writeJSON(w, http.StatusConflict, apiError{Error: "这个关系组已存在。", ExistingGroup: &group})
	case err != nil:
		internalError(w, err)
	default:
		writeJSON(w, http.StatusOK, group)
	}
}

func (a *App) createGroup(w http.ResponseWriter, r *http.Request) {
	var id int64
	if r.PathValue("id") != "" {
		var err error
		id, err = strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil || id <= 0 {
			writeJSON(w, http.StatusBadRequest, apiError{Error: "无效的词条编号。"})
			return
		}
		if _, err := a.getEntry(r.Context(), id); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				writeJSON(w, http.StatusNotFound, apiError{Error: "找不到这个词条。"})
			} else {
				internalError(w, err)
			}
			return
		}
	}
	var input struct {
		Type      string  `json:"type"`
		Note      string  `json:"note"`
		MemberIDs []int64 `json:"memberIds"`
	}
	if err := decodeJSON(r, &input); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError{Error: "请输入有效的 JSON 关系组。"})
		return
	}
	if id != 0 {
		input.MemberIDs = append(input.MemberIDs, id)
	}
	group, err := a.addGroup(r.Context(), input.Type, input.Note, input.MemberIDs)
	if errors.Is(err, errInvalidGroup) {
		writeJSON(w, http.StatusBadRequest, apiError{Error: "请选择一种关系类型和至少两个不同的已有词条。"})
		return
	}
	if errors.Is(err, errDuplicateGroup) {
		writeJSON(w, http.StatusConflict, apiError{Error: "这个关系组已存在。", ExistingGroup: &group})
		return
	}
	if err != nil {
		internalError(w, err)
		return
	}
	w.Header().Set("Location", "/api/groups/"+strconv.FormatInt(group.ID, 10))
	writeJSON(w, http.StatusCreated, group)
}

func (a *App) showEntryGroups(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		writeJSON(w, http.StatusBadRequest, apiError{Error: "无效的词条编号。"})
		return
	}
	if _, err := a.getEntry(r.Context(), id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, apiError{Error: "找不到这个词条。"})
		} else {
			internalError(w, err)
		}
		return
	}
	groups, err := a.entryGroups(r.Context(), id)
	if err != nil {
		internalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, groups)
}

func (a *App) showGroup(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		writeJSON(w, http.StatusBadRequest, apiError{Error: "无效的关系组编号。"})
		return
	}
	group, err := a.getGroup(r.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		writeJSON(w, http.StatusNotFound, apiError{Error: "找不到这个关系组。"})
		return
	}
	if err != nil {
		internalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, group)
}
