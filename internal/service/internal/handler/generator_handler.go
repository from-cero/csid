package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/from-cero/csid/generator/internal/service"
)

// Generator ...
type Generator struct {
	g *service.Generator
}

// NewGenerator ...
func NewGenerator(g *service.Generator) *Generator {
	return &Generator{g: g}
}

// NextID ...
func (g *Generator) NextID(w http.ResponseWriter, _ *http.Request) {
	id, err := g.g.NextID()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	b, err := json.Marshal(id)
	if err != nil {
		slog.Error("failed to marshal response", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	slog.Info("generated", "next-id", id)
	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(b); err != nil {
		slog.Error("failed to write response", "error", err)
	}
}
