package summary

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/samuelt37/BibleMemory/internal/auth"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{
		service: service,
	}
}

func (h *Handler) CheckSummary(w http.ResponseWriter, r *http.Request) {
	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if len(req.Answers) != len(req.Scripture.Ranges) {
		http.Error(w, "answers count does not match ranges count", http.StatusBadRequest)
		return
	}

	userID, _ := auth.UserIDFromContext(r.Context())

	results, err := h.service.CheckSummary(req, userID)
	if err != nil {
		http.Error(w, fmt.Sprintf("grading failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}
