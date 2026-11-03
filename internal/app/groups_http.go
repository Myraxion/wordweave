package app

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
)

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
