package handler

import (
	"encoding/json"
	"net/http"

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

func (h *ScriptureHandler) GetBook(
	w http.ResponseWriter,
	r *http.Request,
) {
	// later get params here

	verses, err := h.service.GetBook(
		"KJV",
		"Genesis",
	)

	if err != nil {
		http.Error(
			w,
			err.Error(),
			http.StatusInternalServerError,
		)
		return
	}

	json.NewEncoder(w).Encode(verses)
}
