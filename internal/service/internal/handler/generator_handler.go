package handler

import (
	"context"
	"log/slog"
	"net/http"
)

type iGeneratorS interface {
	// * declare methods as the service grows

	NextID(ctx context.Context) (string, error)
}

type Generator struct {
	genS iGeneratorS
}

func NewGenerator(genS iGeneratorS) *Generator {
	return &Generator{genS: genS}
}

func (g *Generator) Generate(w http.ResponseWriter, r *http.Request) {
	id, err := g.genS.NextID(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	slog.Info("generated", "next-id", id)
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte(id)); err != nil {
		slog.Error("failed to write generate response", "error", err)
	}
}
