// Package llms provides tracing functionality for LLM operations using OpenTelemetry
package llms

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// TracerProvider defines the interface for tracing operations
type TracerProvider interface {
	StartSpan(ctx context.Context, operation string, provider, model string) (context.Context, trace.Span)
	RecordError(span trace.Span, err error)
	AddSpanAttributes(span trace.Span, attrs map[string]interface{})
}

// OpenTelemetryTracer implements TracerProvider using OpenTelemetry
type OpenTelemetryTracer struct {
	tracer trace.Tracer
}

// NewOpenTelemetryTracer creates a new OpenTelemetry tracer
func NewOpenTelemetryTracer(name string) *OpenTelemetryTracer {
	return &OpenTelemetryTracer{
		tracer: otel.Tracer(name),
	}
}

// StartSpan starts a new trace span for an LLM operation
func (t *OpenTelemetryTracer) StartSpan(ctx context.Context, operation string, provider, model string) (context.Context, trace.Span) {
	ctx, span := t.tracer.Start(ctx, operation,
		trace.WithAttributes(
			attribute.String("llm.provider", provider),
			attribute.String("llm.model", model),
			attribute.String("operation", operation),
		))
	return ctx, span
}

// RecordError records an error on a span
func (t *OpenTelemetryTracer) RecordError(span trace.Span, err error) {
	if span.IsRecording() {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
}

// AddSpanAttributes adds attributes to a span
func (t *OpenTelemetryTracer) AddSpanAttributes(span trace.Span, attrs map[string]interface{}) {
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

// TracingHelper provides high-level tracing utilities
type TracingHelper struct {
	tracer TracerProvider
}

// NewTracingHelper creates a new TracingHelper with OpenTelemetry
func NewTracingHelper() *TracingHelper {
	return &TracingHelper{
		tracer: NewOpenTelemetryTracer("beluga-ai-llms"),
	}
}

// StartOperation starts a new trace span for an LLM operation
func (th *TracingHelper) StartOperation(ctx context.Context, operation string, provider, model string) context.Context {
	newCtx, _ := th.tracer.StartSpan(ctx, operation, provider, model)
	return newCtx
}

// RecordError records an error on the current span
func (th *TracingHelper) RecordError(ctx context.Context, err error) {
	if span := trace.SpanFromContext(ctx); span.IsRecording() {
		th.tracer.RecordError(span, err)
	}
}

// AddSpanAttributes adds attributes to the current span
func (th *TracingHelper) AddSpanAttributes(ctx context.Context, attrs map[string]interface{}) {
	if span := trace.SpanFromContext(ctx); span.IsRecording() {
		th.tracer.AddSpanAttributes(span, attrs)
	}
}

// EndSpan ends the current span
func (th *TracingHelper) EndSpan(ctx context.Context) {
	if span := trace.SpanFromContext(ctx); span.IsRecording() {
		span.End()
	}
}

// StartSpan is a convenience function for starting spans
func StartSpan(ctx context.Context, tracer trace.Tracer, operation, provider, model string) (context.Context, trace.Span) {
	return tracer.Start(ctx, operation,
		trace.WithAttributes(
			attribute.String("llm.provider", provider),
			attribute.String("llm.model", model),
			attribute.String("operation", operation),
		))
}

// RecordSpanError records an error on a span
func RecordSpanError(span trace.Span, err error) {
	if span.IsRecording() {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
}

// AddSpanAttributesMap adds multiple attributes to a span
func AddSpanAttributesMap(span trace.Span, attrs map[string]interface{}) {
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

// LoggerAttrs returns structured logging attributes for LLM operations
func LoggerAttrs(provider, model, operation string) map[string]interface{} {
	return map[string]interface{}{
		"llm.provider": provider,
		"llm.model":    model,
		"operation":    operation,
	}
}
