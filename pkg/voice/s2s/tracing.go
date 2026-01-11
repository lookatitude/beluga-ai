package s2s

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// StartProcessSpan starts a new OTEL span for S2S processing.
func StartProcessSpan(ctx context.Context, provider, model, language string) (context.Context, trace.Span) {
	tracer := trace.SpanFromContext(ctx).TracerProvider().Tracer("beluga-ai/voice/s2s")
	ctx, span := tracer.Start(ctx, "s2s.process",
		trace.WithAttributes(
			attribute.String("s2s.provider", provider),
			attribute.String("s2s.model", model),
			attribute.String("s2s.language", language),
		),
	)
	return ctx, span
}

// StartStreamingSpan starts a new OTEL span for S2S streaming.
func StartStreamingSpan(ctx context.Context, provider, model string) (context.Context, trace.Span) {
	tracer := trace.SpanFromContext(ctx).TracerProvider().Tracer("beluga-ai/voice/s2s")
	ctx, span := tracer.Start(ctx, "s2s.stream",
		trace.WithAttributes(
			attribute.String("s2s.provider", provider),
			attribute.String("s2s.model", model),
		),
	)
	return ctx, span
}

// RecordSpanLatency records latency as a span attribute.
func RecordSpanLatency(span trace.Span, latency time.Duration) {
	span.SetAttributes(attribute.Float64("s2s.latency_ms", float64(latency.Milliseconds())))
}

// RecordSpanError records an error on the span.
func RecordSpanError(span trace.Span, err error) {
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
}

// RecordSpanAttributes records additional attributes on the span.
func RecordSpanAttributes(span trace.Span, attrs map[string]string) {
	kv := make([]attribute.KeyValue, 0, len(attrs))
	for k, v := range attrs {
		kv = append(kv, attribute.String(k, v))
	}
	span.SetAttributes(kv...)
}
