package server

import (
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"github.com/from-cero/csid/generator/internal/config"
)

func newHTTPServer(cfg config.ServerConfig, h *Handlers) *http.Server {
	r := chi.NewRouter()
	r.Use(chimw.RequestID) // injects X-Request-Id header + ctx
	r.Use(chimw.Recoverer) // catches panics, returns 500
	// more middlewares (e.g. logging, timeout, etc.)
	registerRoutes(r, h)
	return &http.Server{
		Addr:    net.JoinHostPort("", fmt.Sprint(cfg.HTTPPort)),
		Handler: r,

		// TODO: make it configurable
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
}
