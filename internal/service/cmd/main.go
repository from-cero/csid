package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/from-cero/csid"
	"github.com/from-cero/csid/generator/internal/config"
	"github.com/from-cero/csid/generator/internal/handler"
	"github.com/from-cero/csid/generator/internal/server"
	"github.com/from-cero/csid/generator/internal/service"
	"github.com/from-cero/csid/generator/internal/telemetry"
	"github.com/from-cero/csid/registry"
)

func main() {
	if err := run(); err != nil {
		slog.Error("application error", "error", err)
		os.Exit(1)
	}
}

func run() error {
	// parse command-line flags
	cfgPath := flag.String("config", "config.yaml", "path to config file")
	flag.Parse()

	// load & validate configs
	cfg, err := config.Load(*cfgPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// init loggers
	logger := telemetry.NewLogger(cfg.Logging, cfg.App)
	slog.SetDefault(logger)

	// setup signal handling for graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// acquire node identity
	reg, err := registry.NewStaticRegistry()
	if err != nil {
		return fmt.Errorf("create static registry: %w", err)
	}

	// wire dependencies
	node, err := csid.New(ctx, reg)
	if err != nil {
		return fmt.Errorf("create csid node: %w", err)
	}
	services := &server.Services{Generator: service.NewGenerator(node)}
	handlers := &server.Handlers{Generator: handler.NewGenerator(services.Generator)}
	srv := server.New(cfg.Server, handlers, services)

	// start server
	if err := srv.Start(ctx); err != nil {
		return fmt.Errorf("start server: %w", err)
	}
	return nil
}
