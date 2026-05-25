package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

// Load loads configuration from multiple sources with layered precedence:
// 1. Hard-coded defaults (lowest precedence)
// 2. Config file (optional, overrides defaults)
// 3. Environment variables (highest precedence, overrides both)
//
// The config file is expected to be in YAML format.
// Environment variables use '_' instead of '.' and '-' instead of '_' to override a setting.
// (e.g. "SERVER_HTTP-PORT" -> "server.http_port")
//
// Returns a fully populated Config struct or an error if loading or validation fails.
func Load(configPath string) (*Config, error) {
	k := koanf.New(".") // use '.' as the key path delimiter

	// layer 1: hard-coded defaults (always loaded)
	if err := k.Load(confmap.Provider(defaults(), "."), nil); err != nil {
		return nil, fmt.Errorf("config defaults: %w", err)
	}

	// layer 2: config file (optional, missing is ok when configPath is empty)
	if err := k.Load(file.Provider(configPath), yaml.Parser()); err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("config from file: %w", err)
		}
	}

	// layer 3: environment variables (overrides, highest precedence)
	if err := k.Load(env.Provider("", ".", envKeyMapper), nil); err != nil {
		return nil, fmt.Errorf("config from environment: %w", err)
	}

	var cfg Config
	if err := k.UnmarshalWithConf("", &cfg, koanf.UnmarshalConf{Tag: "koanf"}); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}
	return &cfg, cfg.validate()
}

func envKeyMapper(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, "_", ".")
	s = strings.ReplaceAll(s, "-", "_")
	return s
}

func defaults() map[string]any {
	return map[string]any{
		"server.shutdown_timeout": "30s",

		"server.http.port":                 8080,
		"server.http.read_header_timeout":  "5s",
		"server.http.read_timeout":         "15s",
		"server.http.write_timeout":        "30s",
		"server.http.idle_timeout":         "120s",
		"server.http.max_header_bytes":     1 << 20,
		"server.http.request_timeout":      "0s",
		"server.http.cors.allowed_methods": []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},

		"server.grpc.port": 9090,

		"logging.level":  "info",
		"logging.format": "json",

		"telemetry.tracing.enabled": false,
		"telemetry.metrics.enabled": false,

		// * add more defaults as needed
	}
}
