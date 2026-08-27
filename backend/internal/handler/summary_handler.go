package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/samuelt37/BibleMemory/internal/dto"
	"github.com/samuelt37/BibleMemory/internal/service"
)

type SummaryHandler struct {
	service *service.SummaryService
}

func NewSummaryHandler(
	service *service.SummaryService,
) *SummaryHandler {
	return &SummaryHandler{
		service: service,
	}
}

func (h *SummaryHandler) CheckSummary(
	w http.ResponseWriter,
	r *http.Request,
) {
	var req dto.SummaryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if len(req.Answers) != len(req.Scripture.Ranges) {
		http.Error(w, "answers count does not match ranges count", http.StatusBadRequest)
		return
	}

	results, err := h.service.CheckSummary(req)
	if err != nil {
		http.Error(w, fmt.Sprintf("grading failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}
