package session

import (
	"encoding/json"
	"net/http"
	"strconv"

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
	r.Post("/sessions", h.Save)
	r.Get("/sessions/history", h.History)
	r.Get("/sessions/bookmarks", h.Bookmarks)
	r.Patch("/sessions/{id}/bookmark", h.ToggleBookmark)
	r.Delete("/sessions/{id}", h.Delete)
}

func (h *Handler) Save(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var body struct {
		Ranges []ScriptureRange `json:"ranges"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	saved, err := h.service.SaveSession(userID, body.Ranges)
	if err != nil {
		http.Error(w, "failed to save session", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(saved)
}

func (h *Handler) History(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	sessions, err := h.service.GetHistory(userID)
	if err != nil {
		http.Error(w, "failed to fetch history", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sessions)
}

func (h *Handler) Bookmarks(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	sessions, err := h.service.GetBookmarks(userID)
	if err != nil {
		http.Error(w, "failed to fetch bookmarks", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sessions)
}

func (h *Handler) ToggleBookmark(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	sessionID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var body struct {
		Bookmarked bool `json:"bookmarked"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	if err := h.service.ToggleBookmark(userID, sessionID, body.Bookmarked); err != nil {
		http.Error(w, "failed to update bookmark", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	sessionID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	if err := h.service.DeleteSession(userID, sessionID); err != nil {
		http.Error(w, "failed to delete session", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
