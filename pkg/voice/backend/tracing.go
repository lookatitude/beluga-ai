package backend

import (
	"context"
	"fmt"
	"log/slog"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// LogWithOTELContext extracts OTEL trace/span IDs from context and logs with structured logging.
func LogWithOTELContext(ctx context.Context, level slog.Level, msg string, attrs ...any) {
	// Extract OTEL context
	spanCtx := trace.SpanContextFromContext(ctx)
	if spanCtx.IsValid() {
		otelAttrs := []any{
			"trace_id", spanCtx.TraceID().String(),
			"span_id", spanCtx.SpanID().String(),
		}
		attrs = append(otelAttrs, attrs...)
	}

	// Use slog for structured logging
	logger := slog.Default()
	logger.Log(ctx, level, msg, attrs...)
}

// StartSpan starts a new trace span for a voice backend operation.
func StartSpan(ctx context.Context, operation, provider string) (context.Context, trace.Span) {
	tracer := otel.Tracer("github.com/lookatitude/beluga-ai/pkg/voice/backend")
	ctx, span := tracer.Start(ctx, "backend."+operation,
		trace.WithAttributes(
			attribute.String("backend.provider", provider),
			attribute.String("operation", operation),
		))
	return ctx, span
}

// RecordSpanError records an error on a span.
func RecordSpanError(span trace.Span, err error) {
	if span.IsRecording() {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
}

// AddSpanAttributes adds attributes to a span.
func AddSpanAttributes(span trace.Span, attrs map[string]any) {
	if !span.IsRecording() {
		return
	}

	var otelAttrs []attribute.KeyValue
	for k, v := range attrs {
		switch val := v.(type) {
		case string:
			otelAttrs = append(otelAttrs, attribute.String(k, val))
		case int:
			otelAttrs = append(otelAttrs, attribute.Int(k, val))
		case int64:
			otelAttrs = append(otelAttrs, attribute.Int64(k, val))
		case float32:
			otelAttrs = append(otelAttrs, attribute.Float64(k, float64(val)))
		case float64:
			otelAttrs = append(otelAttrs, attribute.Float64(k, val))
		case bool:
			otelAttrs = append(otelAttrs, attribute.Bool(k, val))
		default:
			// Convert to string for unknown types
			otelAttrs = append(otelAttrs, attribute.String(k, fmt.Sprintf("%v", val)))
		}
	}
	span.SetAttributes(otelAttrs...)
}
