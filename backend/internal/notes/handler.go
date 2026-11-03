package notes

import (
	"encoding/json"
	"errors"
	"io"
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
	r.Post("/notes/{id}/chunks", h.CreateChunk)
	r.Delete("/notes/{id}/refs", h.DeleteRef)
	r.Get("/notes/counts-by-book", h.CountsByBook)
}

const (
	maxDocBytes   = 20 << 20  // images, PDFs, text
	maxVideoBytes = 300 << 20 // videos
)

var allowedTypes = map[string]bool{
	"application/pdf": true,
	"text/plain":      true, // also what Go sniffs .md and .csv as
	"image/jpeg":      true,
	"image/png":       true,
	"image/webp":      true,
	"image/heic":      true,
	"image/heif":      true,
}

func isAllowedType(mimeType string) bool {
	return strings.HasPrefix(mimeType, "video/") || allowedTypes[mimeType]
}

func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// Hard cap on the whole request body (video limit + a little overhead)
	r.Body = http.MaxBytesReader(w, r.Body, maxVideoBytes+(1<<20))

	// 32MB stays in memory; anything bigger spills to temp files on disk
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		var tooBig *http.MaxBytesError
		if errors.As(err, &tooBig) {
			http.Error(w, "file too large", http.StatusRequestEntityTooLarge)
		} else {
			http.Error(w, "invalid upload", http.StatusBadRequest)
		}
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "missing file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Sniff the real type from the first bytes instead of trusting the client
	buf := make([]byte, 512)
	n, _ := file.Read(buf)
	mimeType, _, _ := strings.Cut(http.DetectContentType(buf[:n]), ";") // drops "; charset=..."
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		http.Error(w, "upload failed", http.StatusInternalServerError)
		return
	}

	// The sniffer misses some formats (HEIC, .mov), so fall back to the client's header
	if mimeType == "application/octet-stream" {
		ct, _, _ := strings.Cut(header.Header.Get("Content-Type"), ";")
		if strings.HasPrefix(ct, "video/") || strings.HasPrefix(ct, "image/") {
			mimeType = ct
		}
	}

	isVideo := strings.HasPrefix(mimeType, "video/")
	if !isAllowedType(mimeType) {
		http.Error(w, "unsupported file type", http.StatusUnsupportedMediaType)
		return
	}
	if !isVideo && header.Size > maxDocBytes {
		http.Error(w, "file too large", http.StatusRequestEntityTooLarge)
		return
	}

	note, err := h.service.UploadNote(r.Context(), userID, file, header, mimeType)
	if err != nil {
		log.Println("Upload: UploadNote failed:", err)
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

func (h *Handler) CreateChunk(w http.ResponseWriter, r *http.Request) {
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

	var in CreateChunkInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil ||
		strings.TrimSpace(in.Content) == "" || len(in.Refs) == 0 {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	for _, ref := range in.Refs {
		if ref.BookID == nil {
			http.Error(w, "invalid ref", http.StatusBadRequest)
			return
		}
	}

	chunk, err := h.service.CreateChunk(r.Context(), userID, noteID, in)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			http.Error(w, "note not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to create chunk", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(chunk)
}

func (h *Handler) DeleteRef(w http.ResponseWriter, r *http.Request) {
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

	var ref verseRef
	if err := json.NewDecoder(r.Body).Decode(&ref); err != nil || ref.BookID == nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	deleted, err := h.service.DeleteReference(r.Context(), userID, noteID, ref)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			http.Error(w, "note not found", http.StatusNotFound)
			return
		}
		log.Printf("delete ref note=%d: %v", noteID, err)
		http.Error(w, "failed to delete reference", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"deleted": deleted})
}

func (h *Handler) CountsByBook(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	counts, err := h.service.NoteCountsByBook(r.Context(), userID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(counts) // {"1": 4, "43": 12}
}
