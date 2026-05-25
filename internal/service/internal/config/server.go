package config

import "time"

// ServerConfig defines the server-related settings.
type ServerConfig struct {
	// If ShutdownTimeout is zero, the server will wait indefinitely for graceful shutdown.
	ShutdownTimeout time.Duration `koanf:"shutdown_timeout" validate:"gte=0"` // Default is 30s
	HTTP            HTTPConfig    `koanf:"http"`
	GRPC            GRPCConfig    `koanf:"grpc"`
}

// GRPCConfig defines gRPC server settings.
type GRPCConfig struct {
	// If port is zero, the os will assign an ephemeral port. Useful for testing.
	Port int `koanf:"port" validate:"min=1,max=65535"` // Default is 9090
}

// HTTPConfig defines HTTP server settings.
type HTTPConfig struct {
	// If port is zero, the os will assign an ephemeral port. Useful for testing.
	Port int `koanf:"port" validate:"min=0,max=65535"` // Default is 8080

	// Time to read the request headers.
	// Typical value is 3-5s and should not be zero to avoid slowloris attacks.
	ReadHeaderTimeout time.Duration `koanf:"read_header_timeout" validate:"gt=0"` // Default is 5s

	// Time to read the entire request, including body. Should be greater than ReadHeaderTimeout.
	// Typical value is 10-30s. If timeout is zero, the server will wait indefinitely
	// for the request body to be read and vulnerable to slow body attacks.
	ReadTimeout time.Duration `koanf:"read_timeout" validate:"gte=0"` // Default is 15s

	// Time to write the response. Streaming/SSE/WebSocket often need special handling.
	// Typical value is 15-60s. If timeout is zero, the server will wait indefinitely
	// for the response to be written. A slow client can hold the connection open by reading the response slowly.
	WriteTimeout time.Duration `koanf:"write_timeout" validate:"gte=0"` // Default is 30s

	// Maximum duration for keeping idle connections alive. Typical value is 60-120s.
	// If timeout is zero, the server will wait indefinitely for the next request and consume resources.
	IdleTimeout time.Duration `koanf:"idle_timeout" validate:"gte=0"` // Default is 120s

	// Maximum size of request headers in bytes. Should be large enough to accommodate necessary
	// headers (e.g. auth tokens, tracing info, etc.) but not too large. Typical value is 64KB-1MB
	MaxHeaderBytes int `koanf:"max_header_bytes" validate:"gt=0"` // Default is 1MB (1 << 20)

	// Per-handler deadline: how long a handler has to produce a full response.
	// Zero disables the timeout. Should be less than WriteTimeout.
	// Handlers must respect ctx cancellation or the client will receive a 503 after the deadline.
	RequestTimeout time.Duration `koanf:"request_timeout" validate:"gte=0"` // Default is 0 (disabled)

	CORS CORSConfig `koanf:"cors"`
}

// CORSConfig defines Cross-Origin Resource Sharing settings.
type CORSConfig struct {
	// AllowedOrigins is the list of origins allowed to make cross-origin requests.
	// Use ["*"] to allow all origins (not recommended for production).
	AllowedOrigins []string `koanf:"allowed_origins"`

	// AllowedMethods is the list of HTTP methods allowed in cross-origin requests.
	AllowedMethods []string `koanf:"allowed_methods"` // Default is common HTTP methods
}
