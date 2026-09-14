// Package log provides a thin slog wrapper with context propagation and
// structured field helpers used across parameters services.
package log

import (
	"context"
	"log/slog"
	"os"
)

type contextKey struct{}

// Logger wraps slog.Logger with service-level convenience methods.
type Logger struct {
	*slog.Logger
}

var defaultLogger = New(LevelInfo, FormatJSON)

// Level is a re-export of slog.Level for caller convenience.
type Level = slog.Level

const (
	LevelDebug = slog.LevelDebug
	LevelInfo  = slog.LevelInfo
	LevelWarn  = slog.LevelWarn
	LevelError = slog.LevelError
)

type Format int

const (
	FormatJSON Format = iota
	FormatText
)

// New creates a Logger with the given level and output format.
func New(level Level, format Format) *Logger {
	opts := &slog.HandlerOptions{Level: level}
	var handler slog.Handler
	if format == FormatText {
		handler = slog.NewTextHandler(os.Stdout, opts)
	} else {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}
	return &Logger{slog.New(handler)}
}

// Default returns the package-level default logger.
func Default() *Logger { return defaultLogger }

// SetDefault replaces the package-level default logger.
func SetDefault(l *Logger) {
	defaultLogger = l
	slog.SetDefault(l.Logger)
}

// WithContext stores the logger in context for retrieval.
func WithContext(ctx context.Context, l *Logger) context.Context {
	return context.WithValue(ctx, contextKey{}, l)
}

// FromContext retrieves the logger from context, falling back to the default.
func FromContext(ctx context.Context) *Logger {
	if l, ok := ctx.Value(contextKey{}).(*Logger); ok && l != nil {
		return l
	}
	return defaultLogger
}

// With returns a child logger with additional fixed fields.
func (l *Logger) With(args ...any) *Logger {
	return &Logger{l.Logger.With(args...)}
}

// Service returns a logger with "service" pre-set.
func Service(name string) *Logger {
	return defaultLogger.With("service", name)
}
