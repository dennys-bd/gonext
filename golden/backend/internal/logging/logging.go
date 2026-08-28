// Package logging builds the backend's structured slog.Logger from
// runtime configuration.
package logging

import (
	"io"
	"log/slog"
	"os"

	"golden-app/backend/internal/config"
)

// New builds a slog.Logger writing to stdout, configured by cfg's
// LogLevel and LogFormat.
func New(cfg config.Config) *slog.Logger {
	return newLogger(cfg, os.Stdout)
}

func newLogger(cfg config.Config, w io.Writer) *slog.Logger {
	opts := &slog.HandlerOptions{Level: parseLevel(cfg.LogLevel)}

	var handler slog.Handler
	if cfg.LogFormat == "json" {
		handler = slog.NewJSONHandler(w, opts)
	} else {
		handler = slog.NewTextHandler(w, opts)
	}

	return slog.New(handler)
}

func parseLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
