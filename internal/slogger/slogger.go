// Package slogger provides structured logging for the Headjack CLI using
// Go's slog with charmbracelet/log as the handler for pleasant terminal output.
package slogger

import (
	"context"
	"io"
	"log/slog"
	"os"

	charmlog "github.com/charmbracelet/log"
)

type contextKey string

const loggerKey contextKey = "logger"

// Config holds logger configuration.
type Config struct {
	// Verbosity controls log level:
	// 0 (default) -> Error only
	// 1 (-v)      -> Info level
	// 2+ (-vv)    -> Debug level
	Verbosity int

	// Output is the writer for log output. Defaults to os.Stderr.
	Output io.Writer
}

// New creates a new slog.Logger with charmbracelet/log as the handler.
func New(cfg Config) *slog.Logger {
	output := cfg.Output
	if output == nil {
		output = os.Stderr
	}

	// Map verbosity to charm log level
	var level charmlog.Level
	switch {
	case cfg.Verbosity >= 2:
		level = charmlog.DebugLevel
	case cfg.Verbosity == 1:
		level = charmlog.InfoLevel
	default:
		level = charmlog.ErrorLevel
	}

	// Create charm log handler with slog-compatible options
	handler := charmlog.NewWithOptions(output, charmlog.Options{
		Level:           level,
		ReportTimestamp: false, // CLI doesn't need timestamps
		ReportCaller:    false, // Keep output clean
	})

	return slog.New(handler)
}

// WithLogger adds a logger to the context.
func WithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

// FromContext retrieves the logger from context.
// Returns a discarding logger if none is set (never returns nil).
func FromContext(ctx context.Context) *slog.Logger {
	if logger, ok := ctx.Value(loggerKey).(*slog.Logger); ok && logger != nil {
		return logger
	}
	// Return a discarding logger as fallback
	return slog.New(discardHandler{})
}

// L is a convenience alias for FromContext.
func L(ctx context.Context) *slog.Logger {
	return FromContext(ctx)
}

// discardHandler is a slog.Handler that discards all log records.
type discardHandler struct{}

func (discardHandler) Enabled(context.Context, slog.Level) bool  { return false }
func (discardHandler) Handle(context.Context, slog.Record) error { return nil }
func (d discardHandler) WithAttrs([]slog.Attr) slog.Handler      { return d }
func (d discardHandler) WithGroup(string) slog.Handler           { return d }
