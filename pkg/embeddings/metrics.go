package embeddings

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// Metrics holds the metrics for the embeddings package.
type Metrics struct {
	requestsTotal    metric.Int64Counter
	requestDuration  metric.Float64Histogram
	requestsInFlight metric.Int64UpDownCounter
	errorsTotal      metric.Int64Counter
	tokensProcessed  metric.Int64Counter
	tracer           trace.Tracer
}

// NewMetrics creates a new metrics instance.
func NewMetrics(meter metric.Meter, tracer trace.Tracer) (*Metrics, error) {
	var err error
	m := &Metrics{}

	m.requestsTotal, err = meter.Int64Counter(
		"embeddings_requests_total",
		metric.WithDescription("Total number of embedding requests"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create requestsTotal metric: %w", err)
	}

	m.requestDuration, err = meter.Float64Histogram(
		"embeddings_request_duration_seconds",
		metric.WithDescription("Duration of embedding requests in seconds"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create requestDuration metric: %w", err)
	}

	m.requestsInFlight, err = meter.Int64UpDownCounter(
		"embeddings_requests_in_flight",
		metric.WithDescription("Number of embedding requests currently in flight"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create requestsInFlight metric: %w", err)
	}

	m.errorsTotal, err = meter.Int64Counter(
		"embeddings_errors_total",
		metric.WithDescription("Total number of embedding errors"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create errorsTotal metric: %w", err)
	}

	m.tokensProcessed, err = meter.Int64Counter(
		"embeddings_tokens_processed_total",
		metric.WithDescription("Total number of tokens processed for embeddings"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create tokensProcessed metric: %w", err)
	}

	if tracer == nil {
		tracer = trace.NewNoopTracerProvider().Tracer("embeddings")
	}
	m.tracer = tracer

	return m, nil
}

// RecordRequest records a successful embedding request.
func (m *Metrics) RecordRequest(ctx context.Context, provider, model string, duration time.Duration, inputCount, outputDimension int) {
	if m == nil {
		return
	}
	if m.requestsTotal != nil {
		m.requestsTotal.Add(ctx, 1,
			metric.WithAttributes(
				attribute.String("provider", provider),
				attribute.String("model", model),
				attribute.Int("input_count", inputCount),
				attribute.Int("output_dimension", outputDimension),
			))
	}
	if m.requestDuration != nil {
		m.requestDuration.Record(ctx, duration.Seconds(),
			metric.WithAttributes(
				attribute.String("provider", provider),
				attribute.String("model", model),
			))
	}
}

// RecordError records an embedding error.
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

// RecordTokensProcessed records the number of tokens processed.
func (m *Metrics) RecordTokensProcessed(ctx context.Context, provider, model string, tokenCount int) {
	if m == nil || m.tokensProcessed == nil {
		return
	}
	m.tokensProcessed.Add(ctx, int64(tokenCount),
		metric.WithAttributes(
			attribute.String("provider", provider),
			attribute.String("model", model),
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

// Global metrics instance - initialized once.
var (
	globalMetrics *Metrics
	metricsOnce   sync.Once
)

// NoOpMetrics returns a metrics instance that does nothing.
// Useful for testing or when metrics are disabled.
func NoOpMetrics() *Metrics {
	return &Metrics{
		tracer: trace.NewNoopTracerProvider().Tracer("embeddings"),
	}
}

// InitMetrics initializes the global metrics instance.
// This follows the standard pattern used across all Beluga AI packages.
func InitMetrics(meter metric.Meter, tracer trace.Tracer) {
	metricsOnce.Do(func() {
		if tracer == nil {
			tracer = trace.NewNoopTracerProvider().Tracer("embeddings")
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

// GetMetrics returns the global metrics instance.
// This follows the standard pattern used across all Beluga AI packages.
func GetMetrics() *Metrics {
	return globalMetrics
}
