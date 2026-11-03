package app

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
)

type apiError struct {
	Error         string `json:"error"`
	ExistingEntry *Entry `json:"existingEntry,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("write response: %v", err)
	}
}

func internalError(w http.ResponseWriter, err error) {
	log.Printf("request failed: %v", err)
	writeJSON(w, http.StatusInternalServerError, apiError{Error: "暂时无法处理请求，请重试。"})
}

func (a *App) createEntry(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Spelling   string `json:"spelling"`
		Definition string `json:"definition"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
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
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
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
