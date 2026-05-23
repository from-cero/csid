package config

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
)

// Config acts as root configuration that holds all settings.
type Config struct {
	App     AppConfig     `koanf:"app"`
	Server  ServerConfig  `koanf:"server"`
	Logging LoggingConfig `koanf:"logging"`
}

// AppConfig defines the application metadata.
type AppConfig struct {
	Name    string `koanf:"name"    validate:"required"`
	Version string `koanf:"version" validate:"required"`
	Env     string `koanf:"env"     validate:"oneof=development staging production"`
}

// ServerConfig defines the server-related settings.
type ServerConfig struct {
	HTTPPort        int           `koanf:"http_port"        validate:"min=1,max=65535"`
	GRPCPort        int           `koanf:"grpc_port"        validate:"min=1,max=65535"`
	ShutdownTimeout time.Duration `koanf:"shutdown_timeout" validate:"required,gte=0"`
}

// LoggingConfig defines the logging level and format for the application.
type LoggingConfig struct {
	Level  string `koanf:"level"  validate:"oneof=debug info warn error"`
	Format string `koanf:"format" validate:"oneof=json console"`
}

// ErrInvalidConfig ...
var ErrInvalidConfig = errors.New("invalid config")

func (c *Config) validate() error {
	var errs []string

	v := validator.New()
	if err := v.Struct(c); err != nil {
		if ve, ok := errors.AsType[validator.ValidationErrors](err); ok {
			for _, fe := range ve {
				errs = append(errs, fmt.Sprintf("%s:%s", fe.Field(), fe.Tag()))
			}
		} else {
			errs = append(errs, err.Error())
		}
	}

	if c.Server.HTTPPort == c.Server.GRPCPort {
		errs = append(errs, "http and grpc ports cannot be the same")
	}

	if len(errs) > 0 {
		return fmt.Errorf("%w: %s", ErrInvalidConfig, strings.Join(errs, ", "))
	}
	return nil
}
