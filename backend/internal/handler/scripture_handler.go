package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

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

func (h *ScriptureHandler) GetScripture(
	w http.ResponseWriter,
	r *http.Request,
) {
	fmt.Println("handler before service")
	// later get params here
	chapter := 3
	verse := 5

	query := dto.ScriptureQuery{
		Translation: "KJV",
		Ranges: []dto.ScriptureRange{
			{
				Start: dto.Reference{
					Book:    1,
					Chapter: &chapter,
					Verse:   &verse,
				},
				// End: &dto.Reference{
				// 	Book:    1,
				// 	Chapter: 1,
				// 	Verse:   5,
				// },
			},
		},
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
