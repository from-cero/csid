package server

import (
	"github.com/go-chi/chi/v5"
)

func registerRoutes(r chi.Router, h *Handlers) {
	r.Get("/next-id", h.Generator.NextID)
}
