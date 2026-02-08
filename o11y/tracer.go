// Package o11y provides observability primitives for the Beluga AI framework:
// OpenTelemetry-based tracing and metrics using GenAI semantic conventions,
// structured logging via slog, health checks, and LLM-specific trace exporting.
package o11y

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	otelcodes "go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.39.0"
	"go.opentelemetry.io/otel/trace"
)

// GenAI semantic convention attribute keys (OTel GenAI conventions v1.37+).
const (
	// AttrAgentName is the name of the agent performing the operation.
	AttrAgentName = "gen_ai.agent.name"

	// AttrOperationName is the type of GenAI operation (e.g. "chat", "embed").
	AttrOperationName = "gen_ai.operation.name"

	// AttrToolName is the name of the tool being invoked.
	AttrToolName = "gen_ai.tool.name"

	// AttrRequestModel is the model requested by the caller.
	AttrRequestModel = "gen_ai.request.model"

	// AttrResponseModel is the model that actually served the request.
	AttrResponseModel = "gen_ai.response.model"

	// AttrInputTokens is the number of input tokens consumed.
	AttrInputTokens = "gen_ai.usage.input_tokens"

	// AttrOutputTokens is the number of output tokens produced.
	AttrOutputTokens = "gen_ai.usage.output_tokens"

	// AttrSystem is the GenAI provider system (e.g. "openai", "anthropic").
	AttrSystem = "gen_ai.system"
)

// Attrs is a convenience alias for span attribute maps.
type Attrs map[string]any

// StatusCode represents the outcome of a traced operation.
type StatusCode int

const (
	// StatusOK indicates the operation completed successfully.
	StatusOK StatusCode = iota

	// StatusError indicates the operation failed.
	StatusError
)

// Span wraps an OpenTelemetry span with a simplified API for GenAI operations.
type Span interface {
	// End finishes the span, recording its duration.
	End()

	// SetAttributes adds key-value attributes to the span.
	SetAttributes(attrs Attrs)

	// RecordError records an error on the span without setting its status.
	RecordError(err error)

	// SetStatus sets the span's status code and descriptive message.
	SetStatus(code StatusCode, msg string)
}

// otelSpan adapts an OTel trace.Span to the Span interface.
type otelSpan struct {
	span trace.Span
}

// End finishes the underlying OTel span.
func (s *otelSpan) End() {
	s.span.End()
}

// SetAttributes converts the generic Attrs map to OTel attributes and sets
// them on the span.
func (s *otelSpan) SetAttributes(attrs Attrs) {
	s.span.SetAttributes(attrsToOTel(attrs)...)
}

// RecordError records err on the underlying OTel span.
func (s *otelSpan) RecordError(err error) {
	s.span.RecordError(err)
}

// SetStatus maps StatusCode to OTel codes and sets the span status.
func (s *otelSpan) SetStatus(code StatusCode, msg string) {
	switch code {
	case StatusOK:
		s.span.SetStatus(otelcodes.Ok, msg)
	case StatusError:
		s.span.SetStatus(otelcodes.Error, msg)
	}
}

// tracer is the package-level OTel tracer used by StartSpan.
var tracer trace.Tracer

func init() {
	tracer = otel.Tracer("github.com/lookatitude/beluga-ai/o11y")
}

// StartSpan creates a new OTel span with the given name and attributes.
// The returned context carries the span for downstream propagation.
func StartSpan(ctx context.Context, name string, attrs Attrs) (context.Context, Span) {
	ctx, span := tracer.Start(ctx, name, trace.WithAttributes(attrsToOTel(attrs)...))
	return ctx, &otelSpan{span: span}
}

// TracerOption configures the tracer provider initialised by InitTracer.
type TracerOption func(*tracerConfig)

type tracerConfig struct {
	exporter   sdktrace.SpanExporter
	sampler    sdktrace.Sampler
	syncExport bool
}

// WithSpanExporter sets a custom span exporter for the tracer provider.
func WithSpanExporter(exp sdktrace.SpanExporter) TracerOption {
	return func(cfg *tracerConfig) {
		cfg.exporter = exp
	}
}

// WithSampler sets a custom sampler for the tracer provider.
func WithSampler(s sdktrace.Sampler) TracerOption {
	return func(cfg *tracerConfig) {
		cfg.sampler = s
	}
}

// WithSyncExport configures synchronous span export instead of batched.
// This is useful in tests where spans must be available immediately after End().
func WithSyncExport() TracerOption {
	return func(cfg *tracerConfig) {
		cfg.syncExport = true
	}
}

// InitTracer initialises the global OTel tracer provider with the given service
// name and options. It returns a shutdown function that should be called on
// application exit to flush pending spans.
func InitTracer(serviceName string, opts ...TracerOption) (func(), error) {
	cfg := &tracerConfig{
		sampler: sdktrace.AlwaysSample(),
	}
	for _, opt := range opts {
		opt(cfg)
	}

	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(serviceName),
		),
	)
	if err != nil {
		return nil, err
	}

	tpOpts := []sdktrace.TracerProviderOption{
		sdktrace.WithResource(res),
		sdktrace.WithSampler(cfg.sampler),
	}
	if cfg.exporter != nil {
		if cfg.syncExport {
			tpOpts = append(tpOpts, sdktrace.WithSyncer(cfg.exporter))
		} else {
			tpOpts = append(tpOpts, sdktrace.WithBatcher(cfg.exporter))
		}
	}

	tp := sdktrace.NewTracerProvider(tpOpts...)
	otel.SetTracerProvider(tp)
	tracer = tp.Tracer("github.com/lookatitude/beluga-ai/o11y")

	shutdown := func() {
		_ = tp.Shutdown(context.Background())
	}
	return shutdown, nil
}

// attrsToOTel converts a generic Attrs map to OTel key-value attributes.
func attrsToOTel(attrs Attrs) []attribute.KeyValue {
	if len(attrs) == 0 {
		return nil
	}
	kvs := make([]attribute.KeyValue, 0, len(attrs))
	for k, v := range attrs {
		switch val := v.(type) {
		case string:
			kvs = append(kvs, attribute.String(k, val))
		case int:
			kvs = append(kvs, attribute.Int(k, val))
		case int64:
			kvs = append(kvs, attribute.Int64(k, val))
		case float64:
			kvs = append(kvs, attribute.Float64(k, val))
		case bool:
			kvs = append(kvs, attribute.Bool(k, val))
		}
	}
	return kvs
}
