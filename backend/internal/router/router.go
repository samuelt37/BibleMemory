package router

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/samuelt37/BibleMemory/internal/handler"
	"github.com/samuelt37/BibleMemory/internal/service"
)

func NewRouter(
	scriptureHandler *handler.ScriptureHandler,
	summaryHandler *handler.SummaryHandler,
	sessionHandler *handler.SessionHandler,
	noteHandler *handler.NoteHandler,
	userService *service.UserService,
) *chi.Mux {
	r := chi.NewRouter()

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://*", "http://*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message":"Bible Memory API is running","status":"ok"}`))
	})

	r.Get("/health", handler.Health)

	// public routes
	RegisterScriptureRoutes(r, scriptureHandler)

	// /check works for both logged-out and logged-in users;
	r.With(OptionalAuth(userService)).Post("/check", summaryHandler.CheckSummary)

	// protected routes — require login
	r.Group(func(protected chi.Router) {
		protected.Use(RequireAuth(userService))
		protected.Post("/sessions", sessionHandler.Save)
		protected.Get("/sessions/history", sessionHandler.History)
		protected.Get("/sessions/bookmarks", sessionHandler.Bookmarks)
		protected.Patch("/sessions/{id}/bookmark", sessionHandler.ToggleBookmark)
		protected.Delete("/sessions/{id}", sessionHandler.Delete)
		protected.Post("/notes", noteHandler.Upload)
		protected.Post("/notes/text", noteHandler.UploadText)
		protected.Get("/notes", noteHandler.List)
		protected.Get("/notes/{id}/download", noteHandler.DownloadURL)
		protected.Delete("/notes/{id}", noteHandler.Delete)
		protected.Get("/notes/{id}", noteHandler.Get)
	})

	return r
}
