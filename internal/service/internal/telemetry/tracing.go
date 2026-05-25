package telemetry

import (
	"context"
	"errors"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/from-cero/csid/service/internal/config"
)

// NewTracerProvider initialises an OTel tracer provider and sets it as the global provider.
// When cfg.Tracing.Enabled is false a no-op shutdown is returned and the global provider is left
// as the default no-op. When cfg.Tracing.OTLPEndpoint is empty a no-op exporter is used (safe
// for local dev/test). The returned shutdown func must be called on application exit.
func NewTracerProvider(
	ctx context.Context,
	cfg *config.TracingConfig,
	app *config.AppConfig,
) (shutdown func(context.Context) error, err error) {
	if !cfg.Enabled {
		return func(_ context.Context) error { return nil }, nil
	}

	res, err := resource.New(
		ctx,
		resource.WithAttributes(
			semconv.ServiceName(app.Name),
			semconv.ServiceVersion(app.Version),
			semconv.DeploymentEnvironmentName(app.Env),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("create otel resource: %w", err)
	}

	var (
		exporter sdktrace.SpanExporter
		conn     *grpc.ClientConn
	)
	if cfg.OTLPEndpoint == "" {
		exporter = newNoopExporter()
	} else {
		conn, err = grpc.NewClient(
			cfg.OTLPEndpoint,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		if err != nil {
			return nil, fmt.Errorf("connect to otel collector: %w", err)
		}
		exporter, err = otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
		if err != nil {
			return nil, errors.Join(fmt.Errorf("create otlp trace exporter: %w", err), conn.Close())
		}
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)

	return func(ctx context.Context) error {
		err := tp.Shutdown(ctx)
		if conn != nil {
			if cerr := conn.Close(); cerr != nil && err == nil {
				err = cerr
			}
		}
		return err
	}, nil
}

// noopExporter discards all spans. Used when OTLP endpoint is not configured.
type noopExporter struct{}

func newNoopExporter() *noopExporter { return &noopExporter{} }

func (n *noopExporter) ExportSpans(_ context.Context, _ []sdktrace.ReadOnlySpan) error { return nil }

func (n *noopExporter) Shutdown(_ context.Context) error { return nil }
