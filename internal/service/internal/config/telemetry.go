package config

// LoggingConfig defines the logging level and format.
type LoggingConfig struct {
	Level  string `koanf:"level"  validate:"oneof=debug info warn error"` // Default is info
	Format string `koanf:"format" validate:"oneof=json console"`          // Default is json
}

// TelemetryConfig defines OpenTelemetry settings.
type TelemetryConfig struct {
	Tracing TracingConfig `koanf:"tracing"`
	Metrics MetricsConfig `koanf:"metrics"`
}

// TracingConfig defines tracing-specific settings.
type TracingConfig struct {
	Enabled bool `koanf:"enabled"` // Default is false

	// OTLPEndpoint is the gRPC endpoint of the OTel collector (e.g. "localhost:4317").
	// Leave empty to disable trace exporting (no-op exporter is used).
	OTLPEndpoint string `koanf:"otlp_endpoint"`
}

// MetricsConfig defines metrics-specific settings.
type MetricsConfig struct {
	Enabled bool `koanf:"enabled"` // Default is false
}
