// Package opentelemetry provides OpenTelemetry integration for the monitoring system
package opentelemetry

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	"github.com/lookatitude/beluga-ai/pkg/monitoring/iface"
)

// noOpSpan is a no-operation span for our internal use.
type noOpSpan struct{}

func (s *noOpSpan) End() {}

func (s *noOpSpan) SetStatus(code codes.Code, description string) {}

func (s *noOpSpan) SetAttributes(kv ...attribute.KeyValue) {}

// Provider implements OpenTelemetry integration.
type Provider struct {
	config Config
}

// Config configures the OpenTelemetry provider.
type Config struct {
	ResourceAttrs  map[string]string
	Endpoint       string
	ServiceName    string
	ServiceVersion string
	Environment    string
	ExportTimeout  string
	SampleRate     float64
}

// NewProvider creates a new OpenTelemetry provider.
func NewProvider(config Config) (*Provider, error) {
	provider := &Provider{
		config: config,
	}

	return provider, nil
}

// Tracer returns the OpenTelemetry tracer.
func (p *Provider) Tracer() iface.Tracer {
	return &otelTracer{}
}

// Shutdown shuts down the provider.
func (p *Provider) Shutdown(ctx context.Context) error {
	return nil
}

// otelTracer implements the iface.Tracer interface using OpenTelemetry.
type otelTracer struct{}

func (t *otelTracer) StartSpan(ctx context.Context, name string, opts ...iface.SpanOption) (context.Context, iface.Span) {
	// Create a no-op span
	span := &otelSpan{span: &noOpSpan{}}

	// Apply options to the span
	for _, opt := range opts {
		opt(span)
	}

	return ctx, span
}

func (t *otelTracer) FinishSpan(span iface.Span) {
	if otelSpan, ok := span.(*otelSpan); ok {
		otelSpan.span.End()
	}
}

func (t *otelTracer) GetSpan(spanID string) (iface.Span, bool) {
	// OpenTelemetry doesn't provide span lookup by ID in the same way
	// This would require custom span storage
	return nil, false
}

func (t *otelTracer) GetTraceSpans(traceID string) []iface.Span {
	// Similar to GetSpan, this would require custom trace storage
	return nil
}

// otelSpan implements the iface.Span interface using OpenTelemetry.
type otelSpan struct {
	span *noOpSpan
}

func (s *otelSpan) Log(message string, fields ...map[string]any) {
	// Convert fields to attributes (no-op for now)
	_ = message
	_ = fields
}

func (s *otelSpan) SetError(err error) {
	if err != nil {
		s.span.SetStatus(codes.Error, err.Error())
	}
}

func (s *otelSpan) SetStatus(status string) {
	// Map string status to OpenTelemetry status codes
	var code codes.Code
	switch status {
	case "error":
		code = codes.Error
	case "ok":
		code = codes.Ok
	default:
		code = codes.Unset
	}
	s.span.SetStatus(code, status)
}

func (s *otelSpan) GetDuration() time.Duration {
	// OpenTelemetry spans don't expose duration directly
	// This would require tracking start time separately
	return 0
}

func (s *otelSpan) IsFinished() bool {
	// OpenTelemetry spans don't have a direct "finished" check
	// In practice, we'd need to track this separately
	return false
}

func (s *otelSpan) SetTag(key string, value any) {
	// No-op for now
	_ = key
	_ = value
}

// WithTag creates a span option for adding tags.
func WithTag(key string, value any) iface.SpanOption {
	return func(span iface.Span) {
		span.SetTag(key, value)
	}
}

// WithTags creates a span option for adding multiple tags.
func WithTags(tags map[string]any) iface.SpanOption {
	return func(span iface.Span) {
		for k, v := range tags {
			span.SetTag(k, v)
		}
	}
}
