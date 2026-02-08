package o11y

import (
	"context"
	"log/slog"
	"os"
)

// loggerKey is an unexported type for context keys to prevent collisions.
type loggerKey struct{}

// Logger wraps slog.Logger with context-aware convenience methods for
// structured logging in GenAI operations.
type Logger struct {
	inner *slog.Logger
}

// LogOption configures a Logger created by NewLogger.
type LogOption func(*loggerConfig)

type loggerConfig struct {
	level   slog.Level
	handler slog.Handler
}

// WithLogLevel sets the minimum log level. Accepted values: "debug", "info",
// "warn", "error". Defaults to "info" if the value is unrecognised.
func WithLogLevel(level string) LogOption {
	return func(cfg *loggerConfig) {
		switch level {
		case "debug":
			cfg.level = slog.LevelDebug
		case "info":
			cfg.level = slog.LevelInfo
		case "warn":
			cfg.level = slog.LevelWarn
		case "error":
			cfg.level = slog.LevelError
		}
	}
}

// WithJSON configures the logger to emit JSON-formatted output.
func WithJSON() LogOption {
	return func(cfg *loggerConfig) {
		cfg.handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: cfg.level,
		})
	}
}

// NewLogger creates a Logger with the given options. Without options it
// defaults to info-level text output on stdout.
func NewLogger(opts ...LogOption) *Logger {
	cfg := &loggerConfig{
		level: slog.LevelInfo,
	}
	for _, opt := range opts {
		opt(cfg)
	}
	if cfg.handler == nil {
		cfg.handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: cfg.level,
		})
	}
	return &Logger{inner: slog.New(cfg.handler)}
}

// Slog returns the underlying *slog.Logger for interop with libraries that
// accept a standard slog logger.
func (l *Logger) Slog() *slog.Logger {
	return l.inner
}

// Info logs a message at INFO level with optional key-value attributes.
func (l *Logger) Info(ctx context.Context, msg string, attrs ...any) {
	l.inner.InfoContext(ctx, msg, attrs...)
}

// Error logs a message at ERROR level with optional key-value attributes.
func (l *Logger) Error(ctx context.Context, msg string, attrs ...any) {
	l.inner.ErrorContext(ctx, msg, attrs...)
}

// Debug logs a message at DEBUG level with optional key-value attributes.
func (l *Logger) Debug(ctx context.Context, msg string, attrs ...any) {
	l.inner.DebugContext(ctx, msg, attrs...)
}

// Warn logs a message at WARN level with optional key-value attributes.
func (l *Logger) Warn(ctx context.Context, msg string, attrs ...any) {
	l.inner.WarnContext(ctx, msg, attrs...)
}

// With returns a new Logger carrying the given key-value attributes on every
// subsequent log entry.
func (l *Logger) With(attrs ...any) *Logger {
	return &Logger{inner: l.inner.With(attrs...)}
}

// WithLogger returns a copy of ctx carrying the given Logger.
func WithLogger(ctx context.Context, logger *Logger) context.Context {
	return context.WithValue(ctx, loggerKey{}, logger)
}

// FromContext extracts the Logger from ctx. If no Logger is present, a
// default info-level text logger is returned.
func FromContext(ctx context.Context) *Logger {
	if l, ok := ctx.Value(loggerKey{}).(*Logger); ok {
		return l
	}
	return NewLogger()
}
