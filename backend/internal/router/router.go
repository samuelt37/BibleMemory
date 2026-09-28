package router

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/samuelt37/BibleMemory/internal/notes"
	"github.com/samuelt37/BibleMemory/internal/scripture"
	"github.com/samuelt37/BibleMemory/internal/session"
	"github.com/samuelt37/BibleMemory/internal/summary"
	"github.com/samuelt37/BibleMemory/internal/user"
)

func NewRouter(
	scriptureHandler *scripture.Handler,
	summaryHandler *summary.Handler,
	sessionHandler *session.Handler,
	noteHandler *notes.Handler,
	userService *user.Service,
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

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"status":"ok"}`)
	})

	// public routes
	scriptureHandler.RegisterRoutes(r)

	// /check works for both logged-out and logged-in users;
	r.With(OptionalAuth(userService)).Post("/check", summaryHandler.CheckSummary)

	// protected routes — require login
	r.Group(func(protected chi.Router) {
		protected.Use(RequireAuth(userService))
		sessionHandler.RegisterRoutes(protected)
		noteHandler.RegisterRoutes(protected)
	})

	return r
}
