package telemetry

import (
	"log/slog"
	"os"
	"time"

	"github.com/from-cero/csid/generator/internal/config"
)

// NewLogger ...
func NewLogger(logging config.LoggingConfig, app config.AppConfig) *slog.Logger {
	var level slog.Level
	if err := level.UnmarshalText([]byte(logging.Level)); err != nil {
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: level,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.Attr{
					Key:   a.Key,
					Value: slog.StringValue(a.Value.Time().UTC().Format(time.RFC3339Nano)),
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

	// attach service-level fields to every log line
	return slog.New(handler).With(
		"service", app.Name,
		"version", app.Version,
		"env", app.Env,
	)
}
