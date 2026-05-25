package server

import (
	"fmt"
	"net"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"github.com/from-cero/csid/service/internal/config"
	mw "github.com/from-cero/csid/service/internal/server/middleware"
)

func newHTTPServer(cfg *config.HTTPConfig, h *Handlers) *http.Server {
	r := chi.NewRouter()
	r.Use(chimw.RequestID) // injects X-Request-Id header + ctx
	r.Use(chimw.Recoverer) // catches panics, returns 500

	// internal routes: no logger, no CORS, no request timeout
	r.Get("/healthz", handleLiveness)
	r.Get("/readyz", handleReadiness)
	if h.metrics != nil {
		r.Handle("/metrics", h.metrics)
	}

	// api routes: full middleware stack
	r.Group(
		func(r chi.Router) {
			r.Use(mw.Logger)          // logs method, path, status, latency, request ID
			r.Use(mw.CORS(&cfg.CORS)) // CORS headers + preflight
			if cfg.RequestTimeout > 0 {
				r.Use(chimw.Timeout(cfg.RequestTimeout))
			}
			// * more middlewares (e.g. auth, etc.)
			registerAPIRoutes(r, h)
		},
	)

	return &http.Server{
		Addr:              net.JoinHostPort("", fmt.Sprint(cfg.Port)),
		Handler:           r,
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		ReadTimeout:       cfg.ReadTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
		MaxHeaderBytes:    cfg.MaxHeaderBytes,
	}
}

func registerAPIRoutes(r chi.Router, h *Handlers) {
	// * register your routes here
	r.Get("/next-id", h.genH.Generate)
}
