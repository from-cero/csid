package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/from-cero/csid/service/internal/config"
	"github.com/from-cero/csid/service/internal/server"
	"github.com/from-cero/csid/service/internal/telemetry"
)

func main() {
	if err := run(); err != nil {
		slog.Error("application error", "error", err)
		os.Exit(1)
	}
}

func run() error {
	// parse command-line flags
	cfgPath := flag.String("config", "config.yml", "path to config file")
	flag.Parse()

	// load & validate configs
	cfg, err := config.Load(*cfgPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// init loggers
	logger := telemetry.NewLogger(&cfg.Logging, &cfg.App)
	slog.SetDefault(logger)

	// setup signal handling for graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// init tracing
	shutdownTracer, err := telemetry.NewTracerProvider(ctx, &cfg.Telemetry.Tracing, &cfg.App)
	if err != nil {
		return fmt.Errorf("init tracer: %w", err)
	}
	defer func() {
		shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := shutdownTracer(shutCtx); err != nil {
			slog.Error("failed to shutdown tracer", "error", err)
		}
	}()

	// init metrics
	metricsHandler, shutdownMeter, err := telemetry.NewMeterProvider(&cfg.Telemetry.Metrics)
	if err != nil {
		return fmt.Errorf("init metrics: %w", err)
	}
	defer func() {
		shutCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := shutdownMeter(shutCtx); err != nil {
			slog.Error("failed to shutdown meter provider", "error", err)
		}
	}()

	// * acquire database, init cache client, etc.

	// wire dependencies
	services := server.NewServices()
	handlers := server.NewHandlers(metricsHandler, services)
	srv := server.New(&cfg.Server, handlers, services)

	// start server
	if err := srv.Start(ctx); err != nil {
		return fmt.Errorf("start server: %w", err)
	}
	return nil
}
