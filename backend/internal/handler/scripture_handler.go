package handler

import (
	"encoding/json"
	"net/http"

	"github.com/samuelt37/BibleMemory/internal/service"
	"github.com/samuelt37/BibleMemory/internal/dto"
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

func (h *ScriptureHandler) GetScripture(
	w http.ResponseWriter,
	r *http.Request,
) {
	// later get params here
	chapter := 1
	verseStart := 5
	verseEnd := 10
	query := dto.ScriptureQuery{
 		Translation: "KJV",
	    Book:        "Genesis",
		Chapter: 	 &chapter,
		VerseStart:  &verseStart,
		VerseEnd: 	 &verseEnd,
	}
	verses, err := h.service.GetScripture(query)

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
