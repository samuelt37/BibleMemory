package notes

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/samuelt37/BibleMemory/internal/auth"
)

type Handler struct {
	service *Service
}

func NewHandler(s *Service) *Handler {
	return &Handler{service: s}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Post("/notes", h.Upload)
	r.Post("/notes/text", h.UploadText)
	r.Get("/notes", h.List)
	r.Get("/notes/{id}/download", h.DownloadURL)
	r.Get("/notes/{id}/chunks", h.ListChunks)
	r.Delete("/notes/{id}", h.Delete)
	r.Get("/notes/{id}", h.Get)
	r.Post("/notes/{id}/rereference", h.ReRefNotes)
}

func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if err := r.ParseMultipartForm(20 << 20); err != nil {
		http.Error(w, "file too large", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "missing file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	note, err := h.service.UploadNote(r.Context(), userID, file, header)
	if err != nil {
		http.Error(w, "upload failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(note)
}

func (h *Handler) UploadText(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var body struct {
		Title string `json:"title"`
		Text  string `json:"text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	note, err := h.service.CreateTextNote(userID, body.Title, body.Text)
	if err != nil {
		http.Error(w, "failed to save note", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(note)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	q := strings.TrimSpace(r.URL.Query().Get("q"))

	parseIntPtr := func(key string) *int {
		v := r.URL.Query().Get(key)
		if v == "" {
			return nil
		}
		n, err := strconv.Atoi(v)
		if err != nil {
			return nil
		}
		return &n
	}

	notes, err := h.service.ListNotes(r.Context(), userID, NoteFilter{
		Query:   q,
		BookID:  parseIntPtr("book"),
		Chapter: parseIntPtr("chapter"),
	})

	if err != nil {
		log.Printf("List notes failed for user %d: %v", userID, err)
		http.Error(w, "failed to list notes", http.StatusInternalServerError)
		return
	}
	if notes == nil {
		notes = []Note{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(notes)
}

func (h *Handler) DownloadURL(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	noteID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	url, err := h.service.GetDownloadURL(r.Context(), userID, noteID)
	if err != nil {
		http.Error(w, "failed to generate download link", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"url": url})
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	noteID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	if err := h.service.DeleteNote(r.Context(), userID, noteID); err != nil {
		http.Error(w, "failed to delete note", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	noteID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	note, err := h.service.GetNote(userID, noteID)
	if err != nil || note == nil {
		http.Error(w, "note not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"id":         note.ID,
		"filename":   note.Filename,
		"sourceType": note.SourceType,
		"rawText":    note.RawText,
		"status":     note.Status,
		"createdAt":  note.CreatedAt,
	})
}

func (h *Handler) ListChunks(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	noteID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid note id", http.StatusBadRequest)
		return
	}

	// ownership check, since ListByNote only filters on note id
	note, err := h.service.repo.GetByID(userID, noteID)
	if err != nil {
		http.Error(w, "failed to load note", http.StatusInternalServerError)
		return
	}
	if note == nil {
		http.Error(w, "note not found", http.StatusNotFound)
		return
	}

	chunks, err := h.service.chunkRepo.ListByNote(noteID)
	if err != nil {
		log.Printf("list chunks note=%d: %v", noteID, err)
		http.Error(w, "failed to load chunks", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(chunks)
}

func (h *Handler) ReRefNotes(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	noteID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid note id", http.StatusBadRequest)
		return
	}

	if err := h.service.ReRefNotes(r.Context(), userID, noteID); err != nil {
		if errors.Is(err, ErrNotFound) {
			http.Error(w, "note not found", http.StatusNotFound)
			return
		}
		if errors.Is(err, ErrRateLimited) {
			http.Error(w, "Too many requests to the AI service. Try again in a minute.", http.StatusTooManyRequests)
			return
		}
		log.Printf("rereference note=%d user=%d: %v", noteID, userID, err)
		http.Error(w, "rereference failed", http.StatusBadGateway)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
