package s2s

import (
	"context"
	"log/slog"
)

// Logger is the structured logger for S2S operations.
var Logger *slog.Logger

func init() {
	// Initialize with default logger if not set
	if Logger == nil {
		Logger = slog.Default()
	}
}

// SetLogger sets the structured logger for S2S operations.
func SetLogger(logger *slog.Logger) {
	Logger = logger
}

// LogProcess logs a process operation with structured fields.
func LogProcess(ctx context.Context, level slog.Level, msg string, provider, model, sessionID string, err error) {
	attrs := []any{
		"provider", provider,
		"model", model,
		"session_id", sessionID,
	}
	if err != nil {
		attrs = append(attrs, "error", err.Error())
	}

	switch level {
	case slog.LevelDebug:
		Logger.DebugContext(ctx, msg, attrs...)
	case slog.LevelInfo:
		Logger.InfoContext(ctx, msg, attrs...)
	case slog.LevelWarn:
		Logger.WarnContext(ctx, msg, attrs...)
	case slog.LevelError:
		Logger.ErrorContext(ctx, msg, attrs...)
	default:
		Logger.InfoContext(ctx, msg, attrs...)
	}
}

// LogStreaming logs a streaming operation with structured fields.
func LogStreaming(ctx context.Context, level slog.Level, msg string, provider, model, sessionID string, err error) {
	attrs := []any{
		"provider", provider,
		"model", model,
		"session_id", sessionID,
		"operation", "streaming",
	}
	if err != nil {
		attrs = append(attrs, "error", err.Error())
	}

	switch level {
	case slog.LevelDebug:
		Logger.DebugContext(ctx, msg, attrs...)
	case slog.LevelInfo:
		Logger.InfoContext(ctx, msg, attrs...)
	case slog.LevelWarn:
		Logger.WarnContext(ctx, msg, attrs...)
	case slog.LevelError:
		Logger.ErrorContext(ctx, msg, attrs...)
	default:
		Logger.InfoContext(ctx, msg, attrs...)
	}
}

// LogError logs an error with structured fields.
func LogError(ctx context.Context, msg string, provider, model, errorCode, sessionID string, err error) {
	attrs := []any{
		"provider", provider,
		"model", model,
		"error_code", errorCode,
		"session_id", sessionID,
	}
	if err != nil {
		attrs = append(attrs, "error", err.Error())
	}
	Logger.ErrorContext(ctx, msg, attrs...)
}

// LogFallback logs a fallback event with structured fields.
func LogFallback(ctx context.Context, fromProvider, toProvider, sessionID string) {
	Logger.InfoContext(ctx, "S2S provider fallback",
		"from_provider", fromProvider,
		"to_provider", toProvider,
		"session_id", sessionID,
	)
}
