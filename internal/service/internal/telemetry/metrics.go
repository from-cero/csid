package telemetry

import (
	"context"
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	otelprom "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"

	"github.com/from-cero/csid/service/internal/config"
)

// NewMeterProvider initialises a Prometheus-backed OTel meter provider and sets it as the global
// provider. The returned http.Handler serves the /metrics endpoint for Prometheus scraping.
// When cfg.MetricsEnabled is false a nil handler and no-op shutdown are returned and the global
// provider is left as the default no-op.
// The returned shutdown func must be called on application exit to release internal resources.
// Note: Prometheus is pull-based so shutdown does not flush buffered data, unlike the tracer.
func NewMeterProvider(cfg *config.MetricsConfig) (http.Handler, func(context.Context) error, error) {
	if !cfg.Enabled {
		return nil, func(_ context.Context) error { return nil }, nil
	}

	exporter, err := otelprom.New(otelprom.WithoutScopeInfo())
	if err != nil {
		return nil, nil, fmt.Errorf("create prometheus exporter: %w", err)
	}

	res := resource.Default()
	mp := metric.NewMeterProvider(metric.WithReader(exporter), metric.WithResource(res))
	otel.SetMeterProvider(mp)

	return promhttp.Handler(), mp.Shutdown, nil
}
