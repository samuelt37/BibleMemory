package router

import (
	"github.com/go-chi/chi/v5"
	"github.com/samuelt37/BibleMemory/internal/handler"
)

func RegisterScriptureRoutes(
	r chi.Router,
	scriptureHandler *handler.ScriptureHandler,
) {
	r.Get(
		"/scripture/{translation}/{book}",
		scriptureHandler.GetScripture,
	)
}
