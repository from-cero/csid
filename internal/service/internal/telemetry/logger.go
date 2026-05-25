package telemetry

import (
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/from-cero/csid/service/internal/config"
)

// NewLogger creates a new slog.Logger based on the provided logging configuration and application metadata.
func NewLogger(logging *config.LoggingConfig, app *config.AppConfig) *slog.Logger {
	var level slog.Level
	if err := level.UnmarshalText([]byte(logging.Level)); err != nil {
		level = slog.LevelInfo
	}

	timeFormat := "2006-01-02T15:04:05.000000000Z07:00"
	opts := &slog.HandlerOptions{
		Level: level,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.Attr{
					Key:   a.Key,
					Value: slog.StringValue(a.Value.Time().UTC().Format(timeFormat)),
				}
			}
			return a
		},
	}

	var handler slog.Handler
	switch logging.Format {
	case "console":
		opts.ReplaceAttr = func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.Attr{
					Key:   a.Key,
					Value: slog.StringValue(a.Value.Time().Format(time.TimeOnly)),
				}
			}
			return a
		}
		handler = slog.NewTextHandler(os.Stdout, opts)
	default:
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}

	// * consider adding more handler in the future
	// (e.g. log into file, send to log aggregation system, etc.)

	// attach application-metadata fields to every log line
	logger := slog.New(handler)
	if strings.TrimSpace(app.Name) != "" {
		logger = logger.With("service", app.Name)
	}
	if strings.TrimSpace(app.Version) != "" {
		logger = logger.With("version", app.Version)
	}
	if strings.TrimSpace(app.Env) != "" {
		logger = logger.With("env", app.Env)
	}
	return logger
}
