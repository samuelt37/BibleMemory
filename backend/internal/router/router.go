package router

import (
	"github.com/go-chi/chi/v5"
	"github.com/samuelt37/BibleMemory/internal/handler"
)

func NewRouter() *chi.Mux {
	r := chi.NewRouter()

	r.Get("/health", handler.Health)
	
	return r
}
