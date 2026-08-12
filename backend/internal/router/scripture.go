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
		"/books",
		scriptureHandler.GetBooks,
	)

	r.Get(
		"/{book}/chapters",
		scriptureHandler.GetChapters,
	)

	r.Post(
		"/scripture",
		scriptureHandler.GetScripture,
	)
}
