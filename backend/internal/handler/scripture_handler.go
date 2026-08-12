package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/samuelt37/BibleMemory/internal/dto"
	"github.com/samuelt37/BibleMemory/internal/service"
)

type ScriptureHandler struct {
	service *service.ScriptureService
}

func NewScriptureHandler(
	service *service.ScriptureService,
) *ScriptureHandler {
	return &ScriptureHandler{
		service: service,
	}
}

func (h *ScriptureHandler) GetBooks(
	w http.ResponseWriter,
	r *http.Request,
) {
	books, err := h.service.GetBooks()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	json.NewEncoder(w).Encode(books)
}

func (h *ScriptureHandler) GetChapters(
	w http.ResponseWriter,
	r *http.Request,
) {
	book := chi.URLParam(r, "book")
	chapters, err := h.service.GetChapters(book)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	json.NewEncoder(w).Encode(chapters)
}

func (h *ScriptureHandler) GetScripture(
	w http.ResponseWriter,
	r *http.Request,
) {
	var query dto.ScriptureQuery

	err := json.NewDecoder(r.Body).Decode(&query)
	if err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	verses, err := h.service.GetScripture(query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	json.NewEncoder(w).Encode(verses)
}
