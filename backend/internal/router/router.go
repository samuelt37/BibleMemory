package router

import (
	"net/http"
	"github.com/go-chi/chi/v5"
	"github.com/samuelt37/GoStarter/internal/handlers"
)

func NewRouter() http.Handler {
	r := chi.NewRouter()

	r.Get("/health", handlers.Health)
	
	return r
}
