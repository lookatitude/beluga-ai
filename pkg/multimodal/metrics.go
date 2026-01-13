// Package multimodal provides OTEL metrics and tracing for multimodal operations.
package multimodal

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// Metrics holds the metrics for the multimodal package.
type Metrics struct {
	processTotal          metric.Int64Counter
	processDuration       metric.Float64Histogram
	processStreamTotal    metric.Int64Counter
	processStreamDuration metric.Float64Histogram
	capabilityCheckTotal  metric.Int64Counter
	errorsTotal           metric.Int64Counter
	requestsInFlight      metric.Int64UpDownCounter
	reasoningTotal        metric.Int64Counter
	reasoningDuration     metric.Float64Histogram
	generationTotal       metric.Int64Counter
	generationDuration    metric.Float64Histogram
	chainTotal            metric.Int64Counter
	chainDuration         metric.Float64Histogram
	streamLatency         metric.Float64Histogram
	streamChunksTotal     metric.Int64Counter
	tracer                trace.Tracer
}

var (
	globalMetrics *Metrics
	metricsOnce   sync.Once
	meter         metric.Meter
)

// init initializes the global meter.
func init() {
	meter = otel.Meter("github.com/lookatitude/beluga-ai/pkg/multimodal")
}

// NewMetrics creates a new metrics instance.
func NewMetrics(m metric.Meter, tracer trace.Tracer) (*Metrics, error) {
	var err error
	metrics := &Metrics{}

	metrics.processTotal, err = m.Int64Counter(
		"multimodal_process_total",
		metric.WithDescription("Total number of multimodal process requests"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create processTotal metric: %w", err)
	}

	metrics.processDuration, err = m.Float64Histogram(
		"multimodal_process_duration_seconds",
		metric.WithDescription("Duration of multimodal process requests in seconds"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create processDuration metric: %w", err)
	}

	metrics.processStreamTotal, err = m.Int64Counter(
		"multimodal_process_stream_total",
		metric.WithDescription("Total number of multimodal stream process requests"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create processStreamTotal metric: %w", err)
	}

	metrics.processStreamDuration, err = m.Float64Histogram(
		"multimodal_process_stream_duration_seconds",
		metric.WithDescription("Duration of multimodal stream process requests in seconds"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create processStreamDuration metric: %w", err)
	}

	metrics.capabilityCheckTotal, err = m.Int64Counter(
		"multimodal_capability_check_total",
		metric.WithDescription("Total number of capability checks"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create capabilityCheckTotal metric: %w", err)
	}

	metrics.errorsTotal, err = m.Int64Counter(
		"multimodal_errors_total",
		metric.WithDescription("Total number of multimodal errors"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create errorsTotal metric: %w", err)
	}

	metrics.requestsInFlight, err = m.Int64UpDownCounter(
		"multimodal_requests_in_flight",
		metric.WithDescription("Number of multimodal requests currently in flight"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create requestsInFlight metric: %w", err)
	}

	metrics.reasoningTotal, err = m.Int64Counter(
		"multimodal_reasoning_total",
		metric.WithDescription("Total number of reasoning operations"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create reasoningTotal metric: %w", err)
	}

	metrics.reasoningDuration, err = m.Float64Histogram(
		"multimodal_reasoning_duration_seconds",
		metric.WithDescription("Duration of reasoning operations in seconds"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create reasoningDuration metric: %w", err)
	}

	metrics.generationTotal, err = m.Int64Counter(
		"multimodal_generation_total",
		metric.WithDescription("Total number of generation operations"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create generationTotal metric: %w", err)
	}

	metrics.generationDuration, err = m.Float64Histogram(
		"multimodal_generation_duration_seconds",
		metric.WithDescription("Duration of generation operations in seconds"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create generationDuration metric: %w", err)
	}

	metrics.chainTotal, err = m.Int64Counter(
		"multimodal_chain_total",
		metric.WithDescription("Total number of multimodal chain operations"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create chainTotal metric: %w", err)
	}

	metrics.chainDuration, err = m.Float64Histogram(
		"multimodal_chain_duration_seconds",
		metric.WithDescription("Duration of multimodal chain operations in seconds"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create chainDuration metric: %w", err)
	}

	metrics.streamLatency, err = m.Float64Histogram(
		"multimodal_stream_latency_seconds",
		metric.WithDescription("Latency of streaming operations in seconds (voice <500ms, video <1s)"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create streamLatency metric: %w", err)
	}

	metrics.streamChunksTotal, err = m.Int64Counter(
		"multimodal_stream_chunks_total",
		metric.WithDescription("Total number of chunks processed in streaming operations"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create streamChunksTotal metric: %w", err)
	}

	if tracer == nil {
		tracer = otel.Tracer("github.com/lookatitude/beluga-ai/pkg/multimodal")
	}
	metrics.tracer = tracer

	return metrics, nil
}

// NoOpMetrics returns a metrics instance that does nothing.
// Useful for testing or when metrics are disabled.
func NoOpMetrics() *Metrics {
	return &Metrics{
		tracer: trace.NewNoopTracerProvider().Tracer("multimodal"),
	}
}

// GetMetrics returns the global metrics instance.
func GetMetrics() *Metrics {
	return globalMetrics
}

// InitMetrics initializes the global metrics instance.
// This follows the standard pattern used across all Beluga AI packages.
func InitMetrics(meter metric.Meter, tracer trace.Tracer) {
	metricsOnce.Do(func() {
		if meter == nil {
			meter = otel.Meter("github.com/lookatitude/beluga-ai/pkg/multimodal")
		}
		if tracer == nil {
			tracer = otel.Tracer("github.com/lookatitude/beluga-ai/pkg/multimodal")
		}
		metrics, err := NewMetrics(meter, tracer)
		if err != nil {
			// If metrics creation fails, use no-op metrics
			globalMetrics = NoOpMetrics()
			return
		}
		globalMetrics = metrics
	})
}

// RecordProcess records a successful process request.
func (m *Metrics) RecordProcess(ctx context.Context, provider, model string, duration time.Duration) {
	if m == nil {
		return
	}
	if m.processTotal != nil {
		m.processTotal.Add(ctx, 1,
			metric.WithAttributes(
				attribute.String("provider", provider),
				attribute.String("model", model),
			))
	}
	if m.processDuration != nil {
		m.processDuration.Record(ctx, duration.Seconds(),
			metric.WithAttributes(
				attribute.String("provider", provider),
				attribute.String("model", model),
			))
	}
}

// RecordProcessStream records a successful stream process request.
func (m *Metrics) RecordProcessStream(ctx context.Context, provider, model string, duration time.Duration) {
	if m == nil {
		return
	}
	if m.processStreamTotal != nil {
		m.processStreamTotal.Add(ctx, 1,
			metric.WithAttributes(
				attribute.String("provider", provider),
				attribute.String("model", model),
			))
	}
	if m.processStreamDuration != nil {
		m.processStreamDuration.Record(ctx, duration.Seconds(),
			metric.WithAttributes(
				attribute.String("provider", provider),
				attribute.String("model", model),
			))
	}
}

// RecordCapabilityCheck records a capability check.
func (m *Metrics) RecordCapabilityCheck(ctx context.Context, provider, model, modality string, supported bool) {
	if m == nil || m.capabilityCheckTotal == nil {
		return
	}
	m.capabilityCheckTotal.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("provider", provider),
			attribute.String("model", model),
			attribute.String("modality", modality),
			attribute.Bool("supported", supported),
		))
}

// RecordError records an error.
func (m *Metrics) RecordError(ctx context.Context, provider, model, errorType string) {
	if m == nil || m.errorsTotal == nil {
		return
	}
	m.errorsTotal.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("provider", provider),
			attribute.String("model", model),
			attribute.String("error_type", errorType),
		))
}

// StartRequest increments the in-flight counter.
func (m *Metrics) StartRequest(ctx context.Context, provider, model string) {
	if m == nil || m.requestsInFlight == nil {
		return
	}
	m.requestsInFlight.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("provider", provider),
			attribute.String("model", model),
		))
}

// EndRequest decrements the in-flight counter.
func (m *Metrics) EndRequest(ctx context.Context, provider, model string) {
	if m == nil || m.requestsInFlight == nil {
		return
	}
	m.requestsInFlight.Add(ctx, -1,
		metric.WithAttributes(
			attribute.String("provider", provider),
			attribute.String("model", model),
		))
}

// RecordReasoning records a reasoning operation.
func (m *Metrics) RecordReasoning(ctx context.Context, provider, model string, duration time.Duration) {
	if m == nil {
		return
	}
	if m.reasoningTotal != nil {
		m.reasoningTotal.Add(ctx, 1,
			metric.WithAttributes(
				attribute.String("provider", provider),
				attribute.String("model", model),
			))
	}
	if m.reasoningDuration != nil {
		m.reasoningDuration.Record(ctx, duration.Seconds(),
			metric.WithAttributes(
				attribute.String("provider", provider),
				attribute.String("model", model),
			))
	}
}

// RecordGeneration records a generation operation.
func (m *Metrics) RecordGeneration(ctx context.Context, provider, model string, duration time.Duration) {
	if m == nil {
		return
	}
	if m.generationTotal != nil {
		m.generationTotal.Add(ctx, 1,
			metric.WithAttributes(
				attribute.String("provider", provider),
				attribute.String("model", model),
			))
	}
	if m.generationDuration != nil {
		m.generationDuration.Record(ctx, duration.Seconds(),
			metric.WithAttributes(
				attribute.String("provider", provider),
				attribute.String("model", model),
			))
	}
}

// RecordChain records a multimodal chain operation.
func (m *Metrics) RecordChain(ctx context.Context, provider, model string, duration time.Duration, chainLength int) {
	if m == nil {
		return
	}
	if m.chainTotal != nil {
		m.chainTotal.Add(ctx, 1,
			metric.WithAttributes(
				attribute.String("provider", provider),
				attribute.String("model", model),
				attribute.Int("chain_length", chainLength),
			))
	}
	if m.chainDuration != nil {
		m.chainDuration.Record(ctx, duration.Seconds(),
			metric.WithAttributes(
				attribute.String("provider", provider),
				attribute.String("model", model),
				attribute.Int("chain_length", chainLength),
			))
	}
}

// RecordStreamLatency records latency for streaming operations (voice <500ms, video <1s).
func (m *Metrics) RecordStreamLatency(ctx context.Context, provider, model, modality string, latency time.Duration) {
	if m == nil || m.streamLatency == nil {
		return
	}
	m.streamLatency.Record(ctx, latency.Seconds(),
		metric.WithAttributes(
			attribute.String("provider", provider),
			attribute.String("model", model),
			attribute.String("modality", modality),
		))
}

// RecordStreamChunk records a processed chunk in streaming operations.
func (m *Metrics) RecordStreamChunk(ctx context.Context, provider, model, modality string) {
	if m == nil || m.streamChunksTotal == nil {
		return
	}
	m.streamChunksTotal.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("provider", provider),
			attribute.String("model", model),
			attribute.String("modality", modality),
		))
}

// GetTracer returns the OTEL tracer for multimodal operations.
func GetTracer() trace.Tracer {
	return otel.Tracer("github.com/lookatitude/beluga-ai/pkg/multimodal")
}

// logMetrics logs metrics with OTEL context
func logMetrics(ctx context.Context, level slog.Level, msg string, attrs ...any) {
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
