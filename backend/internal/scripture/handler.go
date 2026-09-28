package scripture

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{
		service: service,
	}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Get("/books", h.GetBooks)
	r.Get("/books/{book}/chapters", h.GetChapters)
	r.Post("/scripture", h.GetScripture)
}

func (h *Handler) GetBooks(w http.ResponseWriter, r *http.Request) {
	books, err := h.service.GetBooks()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(books)
}

func (h *Handler) GetChapters(w http.ResponseWriter, r *http.Request) {
	book := chi.URLParam(r, "book")
	chapters, err := h.service.GetChapters(book)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(chapters)
}

func (h *Handler) GetScripture(w http.ResponseWriter, r *http.Request) {
	var query Query

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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(verses)
}
