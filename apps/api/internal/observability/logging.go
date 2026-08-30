package observability

import (
	"io"
	"log/slog"
	"os"
)

// NewLogger returns a JSON slog logger with a service identifier.
func NewLogger(service, level string) *slog.Logger {
	var lvl slog.Level
	switch level {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lvl})
	return slog.New(handler).With("service", service)
}

// SetDefault configures the process-wide default logger.
func SetDefault(service, level string) {
	slog.SetDefault(NewLogger(service, level))
}

// DiscardLogger returns a logger that discards output (tests).
func DiscardLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(io.Discard, nil))
}
