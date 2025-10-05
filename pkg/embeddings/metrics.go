package embeddings

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// Metrics holds the metrics for the embeddings package
type Metrics struct {
	requestsTotal    metric.Int64Counter
	requestDuration  metric.Float64Histogram
	requestsInFlight metric.Int64UpDownCounter
	errorsTotal      metric.Int64Counter
	tokensProcessed  metric.Int64Counter
}

// NewMetrics creates a new metrics instance
func NewMetrics(meter metric.Meter) (*Metrics, error) {
	requestsTotal, err := meter.Int64Counter(
		"embeddings_requests_total",
		metric.WithDescription("Total number of embedding requests"),
	)
	if err != nil {
		return nil, err
	}

	requestDuration, err := meter.Float64Histogram(
		"embeddings_request_duration_seconds",
		metric.WithDescription("Duration of embedding requests in seconds"),
	)
	if err != nil {
		return nil, err
	}

	requestsInFlight, err := meter.Int64UpDownCounter(
		"embeddings_requests_in_flight",
		metric.WithDescription("Number of embedding requests currently in flight"),
	)
	if err != nil {
		return nil, err
	}

	errorsTotal, err := meter.Int64Counter(
		"embeddings_errors_total",
		metric.WithDescription("Total number of embedding errors"),
	)
	if err != nil {
		return nil, err
	}

	tokensProcessed, err := meter.Int64Counter(
		"embeddings_tokens_processed_total",
		metric.WithDescription("Total number of tokens processed for embeddings"),
	)
	if err != nil {
		return nil, err
	}

	return &Metrics{
		requestsTotal:    requestsTotal,
		requestDuration:  requestDuration,
		requestsInFlight: requestsInFlight,
		errorsTotal:      errorsTotal,
		tokensProcessed:  tokensProcessed,
	}, nil
}

// NoOpMetrics returns a no-op metrics implementation for testing
func NoOpMetrics() *Metrics {
	return &Metrics{}
}

// RecordRequest records a successful embedding request
func (m *Metrics) RecordRequest(ctx context.Context, provider, model string, duration time.Duration, inputCount, outputDimension int) {
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

// RecordError records an embedding error
func (m *Metrics) RecordError(ctx context.Context, provider, model, errorType string) {
	if m.errorsTotal != nil {
		m.errorsTotal.Add(ctx, 1,
			metric.WithAttributes(
				attribute.String("provider", provider),
				attribute.String("model", model),
				attribute.String("error_type", errorType),
			))
	}
}

// RecordTokensProcessed records the number of tokens processed
func (m *Metrics) RecordTokensProcessed(ctx context.Context, provider, model string, tokenCount int) {
	if m.tokensProcessed != nil {
		m.tokensProcessed.Add(ctx, int64(tokenCount),
			metric.WithAttributes(
				attribute.String("provider", provider),
				attribute.String("model", model),
			))
	}
}

// StartRequest increments the in-flight counter
func (m *Metrics) StartRequest(ctx context.Context, provider, model string) {
	if m.requestsInFlight != nil {
		m.requestsInFlight.Add(ctx, 1,
			metric.WithAttributes(
				attribute.String("provider", provider),
				attribute.String("model", model),
			))
	}
}

// EndRequest decrements the in-flight counter
func (m *Metrics) EndRequest(ctx context.Context, provider, model string) {
	if m.requestsInFlight != nil {
		m.requestsInFlight.Add(ctx, -1,
			metric.WithAttributes(
				attribute.String("provider", provider),
				attribute.String("model", model),
			))
	}
}
