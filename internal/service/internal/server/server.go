package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"time"

	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"

	"github.com/from-cero/csid/service/internal/config"
)

// Server encapsulates the HTTP, gRPC, and other servers along with their configuration and dependencies.
type Server struct {
	cfg     *config.ServerConfig
	http    *http.Server
	grpc    *grpc.Server
	grpcLis net.Listener
}

// New creates a new Server instance with the provided configuration, handlers, and services.
func New(cfg *config.ServerConfig, h *Handlers, s *Services) *Server {
	return &Server{
		cfg:  cfg,
		http: newHTTPServer(&cfg.HTTP, h),
		grpc: newGRPCServer(&cfg.GRPC, s),
	}
}

// Start starts the HTTP, gRPC, and other servers and waits for them to exit.
func (s *Server) Start(ctx context.Context) error {
	// bind the gRPC listener before launching goroutines
	// so a port conflict is a hard error here rather than a mid-flight goroutine failure
	grpcLis, err := net.Listen("tcp", net.JoinHostPort("", strconv.Itoa(s.cfg.GRPC.Port)))
	if err != nil {
		return fmt.Errorf("listen grpc server: %w", err)
	}
	s.grpcLis = grpcLis

	g, ctx := errgroup.WithContext(ctx)
	g.Go(
		func() error {
			<-ctx.Done()
			s.shutdown(s.cfg.ShutdownTimeout)
			return nil
		},
	)

	// start http server
	g.Go(
		func() error {
			slog.Info("starting http server", "port", s.cfg.HTTP.Port)
			if err := s.http.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				return fmt.Errorf("listen and serve http server: %w", err)
			}
			return nil
		},
	)

	// start grpc server
	g.Go(
		func() error {
			slog.Info("starting grpc server", "port", s.cfg.GRPC.Port)
			if err := s.grpc.Serve(s.grpcLis); err != nil {
				return fmt.Errorf("serve grpc server: %w", err)
			}
			return nil
		},
	)

	return g.Wait()
}

func (s *Server) shutdown(timeout time.Duration) {
	// each server gets its own full timeout budget
	// so HTTP drain time does not consume the gRPC grace period
	newCtx := func() (context.Context, context.CancelFunc) {
		return context.WithTimeout(context.Background(), timeout)
	}

	slog.Info("shutting down http server", "timeout", timeout.String())
	httpCtx, httpCancel := newCtx()
	defer httpCancel()
	if err := s.http.Shutdown(httpCtx); err != nil {
		slog.Error("failed to shutdown http server", "error", err)
		if err := s.http.Close(); err != nil {
			slog.Error("failed to close http server", "error", err)
		}
		slog.Warn("http server force closed")
	} else {
		slog.Info("http server stopped gracefully")
	}

	slog.Info("shutting down grpc server", "timeout", timeout.String())
	grpcCtx, grpcCancel := newCtx()
	defer grpcCancel()
	done := make(chan struct{})
	go func() {
		s.grpc.GracefulStop()
		close(done)
	}()

	select {
	case <-done:
		slog.Info("grpc server stopped gracefully")
	case <-grpcCtx.Done():
		slog.Warn("grpc server shutdown timed out, force stopping")
		s.grpc.Stop()
	}
}
