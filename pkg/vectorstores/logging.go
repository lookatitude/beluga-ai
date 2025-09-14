package vectorstores

import (
	"context"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel/trace"
)

// Logger provides structured logging for vector store operations.
// It integrates with OpenTelemetry tracing for consistent log correlation.
type Logger struct {
	logger *slog.Logger
}

// NewLogger creates a new structured logger for vector stores.
func NewLogger(logger *slog.Logger) *Logger {
	if logger == nil {
		logger = slog.Default()
	}
	return &Logger{
		logger: logger,
	}
}

// LogDocumentOperation logs document-related operations.
func (l *Logger) LogDocumentOperation(ctx context.Context, level slog.Level, operation string, storeName string, docCount int, duration time.Duration, err error) {
	attrs := []slog.Attr{
		slog.String("operation", operation),
		slog.String("store_name", storeName),
		slog.Int("document_count", docCount),
		slog.Duration("duration", duration),
	}

	if spanCtx := trace.SpanContextFromContext(ctx); spanCtx.IsValid() {
		attrs = append(attrs,
			slog.String("trace_id", spanCtx.TraceID().String()),
			slog.String("span_id", spanCtx.SpanID().String()),
		)
	}

	if err != nil {
		attrs = append(attrs, slog.String("error", err.Error()))
		l.logger.LogAttrs(ctx, level, "Document operation completed with error", attrs...)
	} else {
		l.logger.LogAttrs(ctx, level, "Document operation completed successfully", attrs...)
	}
}

// LogSearchOperation logs search operations.
func (l *Logger) LogSearchOperation(ctx context.Context, level slog.Level, storeName string, queryLength int, k int, resultCount int, duration time.Duration, err error) {
	attrs := []slog.Attr{
		slog.String("store_name", storeName),
		slog.Int("query_length", queryLength),
		slog.Int("search_k", k),
		slog.Int("result_count", resultCount),
		slog.Duration("duration", duration),
	}

	if spanCtx := trace.SpanContextFromContext(ctx); spanCtx.IsValid() {
		attrs = append(attrs,
			slog.String("trace_id", spanCtx.TraceID().String()),
			slog.String("span_id", spanCtx.SpanID().String()),
		)
	}

	if err != nil {
		attrs = append(attrs, slog.String("error", err.Error()))
		l.logger.LogAttrs(ctx, level, "Search operation completed with error", attrs...)
	} else {
		l.logger.LogAttrs(ctx, level, "Search operation completed successfully", attrs...)
	}
}

// LogEmbeddingOperation logs embedding operations.
func (l *Logger) LogEmbeddingOperation(ctx context.Context, level slog.Level, storeName string, textCount int, duration time.Duration, err error) {
	attrs := []slog.Attr{
		slog.String("store_name", storeName),
		slog.Int("text_count", textCount),
		slog.Duration("duration", duration),
	}

	if spanCtx := trace.SpanContextFromContext(ctx); spanCtx.IsValid() {
		attrs = append(attrs,
			slog.String("trace_id", spanCtx.TraceID().String()),
			slog.String("span_id", spanCtx.SpanID().String()),
		)
	}

	if err != nil {
		attrs = append(attrs, slog.String("error", err.Error()))
		l.logger.LogAttrs(ctx, level, "Embedding operation completed with error", attrs...)
	} else {
		l.logger.LogAttrs(ctx, level, "Embedding operation completed successfully", attrs...)
	}
}

// LogError logs errors with context.
func (l *Logger) LogError(ctx context.Context, err error, operation string, storeName string, additionalAttrs ...slog.Attr) {
	attrs := []slog.Attr{
		slog.String("operation", operation),
		slog.String("store_name", storeName),
		slog.String("error", err.Error()),
	}

	if spanCtx := trace.SpanContextFromContext(ctx); spanCtx.IsValid() {
		attrs = append(attrs,
			slog.String("trace_id", spanCtx.TraceID().String()),
			slog.String("span_id", spanCtx.SpanID().String()),
		)
	}

	// Add any additional attributes
	attrs = append(attrs, additionalAttrs...)

	l.logger.LogAttrs(ctx, slog.LevelError, "Vector store operation failed", attrs...)
}

// LogInfo logs informational messages.
func (l *Logger) LogInfo(ctx context.Context, message string, storeName string, additionalAttrs ...slog.Attr) {
	attrs := []slog.Attr{
		slog.String("store_name", storeName),
	}

	if spanCtx := trace.SpanContextFromContext(ctx); spanCtx.IsValid() {
		attrs = append(attrs,
			slog.String("trace_id", spanCtx.TraceID().String()),
			slog.String("span_id", spanCtx.SpanID().String()),
		)
	}

	// Add any additional attributes
	attrs = append(attrs, additionalAttrs...)

	l.logger.LogAttrs(ctx, slog.LevelInfo, message, attrs...)
}

// LogDebug logs debug messages.
func (l *Logger) LogDebug(ctx context.Context, message string, storeName string, additionalAttrs ...slog.Attr) {
	attrs := []slog.Attr{
		slog.String("store_name", storeName),
	}

	if spanCtx := trace.SpanContextFromContext(ctx); spanCtx.IsValid() {
		attrs = append(attrs,
			slog.String("trace_id", spanCtx.TraceID().String()),
			slog.String("span_id", spanCtx.SpanID().String()),
		)
	}

	// Add any additional attributes
	attrs = append(attrs, additionalAttrs...)

	l.logger.LogAttrs(ctx, slog.LevelDebug, message, attrs...)
}

// LogWarn logs warning messages.
func (l *Logger) LogWarn(ctx context.Context, message string, storeName string, additionalAttrs ...slog.Attr) {
	attrs := []slog.Attr{
		slog.String("store_name", storeName),
	}

	if spanCtx := trace.SpanContextFromContext(ctx); spanCtx.IsValid() {
		attrs = append(attrs,
			slog.String("trace_id", spanCtx.TraceID().String()),
			slog.String("span_id", spanCtx.SpanID().String()),
		)
	}

	// Add any additional attributes
	attrs = append(attrs, additionalAttrs...)

	l.logger.LogAttrs(ctx, slog.LevelWarn, message, attrs...)
}

// LogStoreLifecycle logs vector store lifecycle events.
func (l *Logger) LogStoreLifecycle(ctx context.Context, event string, storeName string, additionalAttrs ...slog.Attr) {
	attrs := []slog.Attr{
		slog.String("store_name", storeName),
		slog.String("event", event),
	}

	if spanCtx := trace.SpanContextFromContext(ctx); spanCtx.IsValid() {
		attrs = append(attrs,
			slog.String("trace_id", spanCtx.TraceID().String()),
			slog.String("span_id", spanCtx.SpanID().String()),
		)
	}

	// Add any additional attributes
	attrs = append(attrs, additionalAttrs...)

	l.logger.LogAttrs(ctx, slog.LevelInfo, "Vector store lifecycle event", attrs...)
}

// LogPerformance logs performance-related information.
func (l *Logger) LogPerformance(ctx context.Context, operation string, storeName string, duration time.Duration, additionalAttrs ...slog.Attr) {
	attrs := []slog.Attr{
		slog.String("operation", operation),
		slog.String("store_name", storeName),
		slog.Duration("duration", duration),
		slog.Float64("duration_ms", float64(duration.Nanoseconds())/1e6),
	}

	if spanCtx := trace.SpanContextFromContext(ctx); spanCtx.IsValid() {
		attrs = append(attrs,
			slog.String("trace_id", spanCtx.TraceID().String()),
			slog.String("span_id", spanCtx.SpanID().String()),
		)
	}

	// Add any additional attributes
	attrs = append(attrs, additionalAttrs...)

	l.logger.LogAttrs(ctx, slog.LevelInfo, "Performance measurement", attrs...)
}

// Global logger instance
var globalLogger *Logger

// SetGlobalLogger sets the global logger instance.
func SetGlobalLogger(logger *Logger) {
	globalLogger = logger
}

// GetGlobalLogger returns the global logger instance.
func GetGlobalLogger() *Logger {
	if globalLogger == nil {
		globalLogger = NewLogger(nil)
	}
	return globalLogger
}

// Default convenience functions that use the global logger
var (
	LogError = func(ctx context.Context, err error, operation string, storeName string, additionalAttrs ...slog.Attr) {
		GetGlobalLogger().LogError(ctx, err, operation, storeName, additionalAttrs...)
	}

	LogInfo = func(ctx context.Context, message string, storeName string, additionalAttrs ...slog.Attr) {
		GetGlobalLogger().LogInfo(ctx, message, storeName, additionalAttrs...)
	}

	LogDebug = func(ctx context.Context, message string, storeName string, additionalAttrs ...slog.Attr) {
		GetGlobalLogger().LogDebug(ctx, message, storeName, additionalAttrs...)
	}

	LogWarn = func(ctx context.Context, message string, storeName string, additionalAttrs ...slog.Attr) {
		GetGlobalLogger().LogWarn(ctx, message, storeName, additionalAttrs...)
	}

	LogDocumentOperation = func(ctx context.Context, level slog.Level, operation string, storeName string, docCount int, duration time.Duration, err error) {
		GetGlobalLogger().LogDocumentOperation(ctx, level, operation, storeName, docCount, duration, err)
	}

	LogSearchOperation = func(ctx context.Context, level slog.Level, storeName string, queryLength int, k int, resultCount int, duration time.Duration, err error) {
		GetGlobalLogger().LogSearchOperation(ctx, level, storeName, queryLength, k, resultCount, duration, err)
	}

	LogEmbeddingOperation = func(ctx context.Context, level slog.Level, storeName string, textCount int, duration time.Duration, err error) {
		GetGlobalLogger().LogEmbeddingOperation(ctx, level, storeName, textCount, duration, err)
	}
)
