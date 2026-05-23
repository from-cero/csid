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

func Load(configPath string) (*Config, error) {
	k := koanf.New(".") // use '.' as the key path delimiter

	// layer 1: hard-coded defaults (always loaded)
	if err := k.Load(confmap.Provider(defaults(), "."), nil); err != nil {
		return nil, fmt.Errorf("config defaults: %w", err)
	}

	// layer 2: config file (optional, missing is ok when configPath is empty)
	if err := k.Load(file.Provider(configPath), yaml.Parser()); err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("config file: %w", err)
		}
	}

	// layer 3: environment variables (overrides, highest precedence)
	if err := k.Load(env.Provider("APP_", ".", envKeyMapper), nil); err != nil {
		return nil, fmt.Errorf("config env: %w", err)
	}

	var cfg Config
	if err := k.UnmarshalWithConf("", &cfg, koanf.UnmarshalConf{Tag: "koanf"}); err != nil {
		return nil, fmt.Errorf("config unmarshal: %w", err)
	}
	return &cfg, cfg.validate()
}

func envKeyMapper(s string) string {
	s = strings.TrimPrefix(s, "APP_")
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, "_", ".")
	s = strings.ReplaceAll(s, "-", "_")
	return s
}

func defaults() map[string]any {
	return map[string]any{
		"server.http_port":        8080,
		"server.grpc_port":        9090,
		"server.shutdown_timeout": "30s",

		"logging.level":  "info",
		"logging.format": "json",

		// add more defaults as needed
	}
}
