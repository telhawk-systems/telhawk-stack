package logging

import (
	"context"
	"log/slog"
	"os"

	"github.com/telhawk-systems/telhawk-stack/common/middleware"
)

// Logger wraps slog.Logger to provide context-aware structured logging.
// It automatically extracts request IDs and other contextual information.
type Logger struct {
	*slog.Logger
}

// New creates a new Logger with the specified log level and format.
// format can be "json" or "text" (default is json).
func New(level slog.Level, format string) *Logger {
	var handler slog.Handler

	opts := &slog.HandlerOptions{
		Level: level,
		// Add source location for errors and above
		AddSource: level <= slog.LevelError,
	}

	switch format {
	case "text":
		handler = slog.NewTextHandler(os.Stdout, opts)
	default:
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}

	return &Logger{
		Logger: slog.New(handler),
	}
}

// Default returns the default logger (uses slog.Default).
func Default() *Logger {
	return &Logger{Logger: slog.Default()}
}

// WithContext returns a logger that extracts contextual information from ctx.
// It automatically includes request ID if present in the context.
func (l *Logger) WithContext(ctx context.Context) *slog.Logger {
	// Extract request ID from context
	if reqID := middleware.GetRequestID(ctx); reqID != "" {
		return l.Logger.With(slog.String("request_id", reqID))
	}
	return l.Logger
}

// InfoContext logs at Info level with context-aware fields.
func (l *Logger) InfoContext(ctx context.Context, msg string, args ...any) {
	l.WithContext(ctx).InfoContext(ctx, msg, args...)
}

// WarnContext logs at Warn level with context-aware fields.
func (l *Logger) WarnContext(ctx context.Context, msg string, args ...any) {
	l.WithContext(ctx).WarnContext(ctx, msg, args...)
}

// ErrorContext logs at Error level with context-aware fields.
func (l *Logger) ErrorContext(ctx context.Context, msg string, args ...any) {
	l.WithContext(ctx).ErrorContext(ctx, msg, args...)
}

// DebugContext logs at Debug level with context-aware fields.
func (l *Logger) DebugContext(ctx context.Context, msg string, args ...any) {
	l.WithContext(ctx).DebugContext(ctx, msg, args...)
}

// With returns a new logger with the given attributes added.
func (l *Logger) With(args ...any) *Logger {
	return &Logger{Logger: l.Logger.With(args...)}
}

// WithGroup returns a new logger with the given group name.
func (l *Logger) WithGroup(name string) *Logger {
	return &Logger{Logger: l.Logger.WithGroup(name)}
}

// ParseLevel converts a string log level to slog.Level.
// Valid values: "debug", "info", "warn", "error".
// Returns slog.LevelInfo for invalid values.
func ParseLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// SetDefault sets the default logger for the application.
// This affects both slog.Default() and log package functions.
func SetDefault(l *Logger) {
	slog.SetDefault(l.Logger)
}
