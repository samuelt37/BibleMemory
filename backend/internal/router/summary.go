package router

import (
	"github.com/go-chi/chi/v5"
	"github.com/samuelt37/BibleMemory/internal/handler"
)

func RegisterSummaryRoutes(
	r chi.Router,
	summaryHandler *handler.SummaryHandler,
) {
	r.Post(
		"/check",
		summaryHandler.CheckSummary,
	)
}
