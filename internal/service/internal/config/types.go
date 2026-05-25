package config

import (
	"errors"
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"

	"github.com/from-cero/csid/service/internal/apperror"
)

// Config acts as root configuration that holds all settings.
type Config struct {
	App       AppConfig       `koanf:"app"`
	Server    ServerConfig    `koanf:"server"`
	Logging   LoggingConfig   `koanf:"logging"`
	Telemetry TelemetryConfig `koanf:"telemetry"`
}

// AppConfig defines the application metadata.
type AppConfig struct {
	Name    string `koanf:"name"`
	Version string `koanf:"version"`
	Env     string `koanf:"env"`
}

func (c *Config) validate() error {
	var errs []string

	// validate struct fields based on tags
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

	// extend validations
	if c.Server.HTTP.Port == c.Server.GRPC.Port {
		errs = append(errs, "http and grpc ports cannot be the same")
	}

	// combine all errors into one
	if len(errs) > 0 {
		return fmt.Errorf("%w: %s", apperror.ErrInvalidConfig, strings.Join(errs, ", "))
	}
	return nil
}
