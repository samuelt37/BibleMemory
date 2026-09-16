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
	sessionHandler *handler.SessionHandler,
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

	r.Group(func(protected chi.Router) {
		protected.Use(RequireAuth(userService))
		protected.Post("/sessions", sessionHandler.Save)
		protected.Get("/sessions/history", sessionHandler.History)
		protected.Get("/sessions/bookmarks", sessionHandler.Bookmarks)
		protected.Patch("/sessions/{id}/bookmark", sessionHandler.ToggleBookmark)
		protected.Delete("/sessions/{id}", sessionHandler.Delete)
	})

	return r
}
