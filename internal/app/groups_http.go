package app

import (
	"cmp"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"slices"
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
		groups, err = a.queryGroups(r.Context(), "instr(lower(g.note), lower(?)) > 0", query)
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
	where := "1 = 1"
	var args []any
	if kind := r.URL.Query().Get("type"); kind != "" {
		where = "g.type = ?"
		args = append(args, kind)
	}
	groups, err := a.queryGroups(r.Context(), where, args...)
	if err != nil {
		internalError(w, err)
		return
	}
	sortOrder := r.URL.Query().Get("sort")
	byCreated := sortOrder == "created_asc" || sortOrder == "created_desc"
	ascending := sortOrder == "created_asc" || sortOrder == "updated_asc"
	groupTime := func(group Group) string {
		stamp := group.CreatedAt
		if !byCreated && group.UpdatedAt != nil {
			stamp = *group.UpdatedAt
		}
		// UTC RFC3339Nano strings sort chronologically without their variable-position Z suffix.
		return strings.TrimSuffix(stamp, "Z")
	}
	slices.SortFunc(groups, func(a, b Group) int {
		order := cmp.Compare(groupTime(a), groupTime(b))
		if order == 0 {
			order = cmp.Compare(a.ID, b.ID)
		}
		if !ascending {
			order = -order
		}
		return order
	})
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
	if json.NewDecoder(r.Body).Decode(&input) != nil {
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

func (a *App) createGroup(w http.ResponseWriter, r *http.Request) {
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
	var input struct {
		Type      string  `json:"type"`
		Note      string  `json:"note"`
		MemberIDs []int64 `json:"memberIds"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError{Error: "请输入有效的 JSON 关系组。"})
		return
	}
	group, err := a.addGroup(r.Context(), input.Type, input.Note, append(input.MemberIDs, id))
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
