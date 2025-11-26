package embeddings

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// Metrics holds the metrics for the embeddings package.
type Metrics struct {
	requestsTotal    metric.Int64Counter
	requestDuration  metric.Float64Histogram
	requestsInFlight metric.Int64UpDownCounter
	errorsTotal      metric.Int64Counter
	tokensProcessed  metric.Int64Counter
}

// NewMetrics creates a new metrics instance.
func NewMetrics(meter metric.Meter) *Metrics {
	// OpenTelemetry metric initialization errors are ignored as they're rare
	// and the metrics will just be nil, causing no-op behavior
	requestsTotal, _ := meter.Int64Counter( //nolint:errcheck // Metric initialization errors are rare and handled gracefully
		"embeddings_requests_total",
		metric.WithDescription("Total number of embedding requests"),
	)
	requestDuration, _ := meter.Float64Histogram( //nolint:errcheck // Metric initialization errors are rare and handled gracefully
		"embeddings_request_duration_seconds",
		metric.WithDescription("Duration of embedding requests in seconds"),
	)
	requestsInFlight, _ := meter.Int64UpDownCounter( //nolint:errcheck // Metric initialization errors are rare and handled gracefully
		"embeddings_requests_in_flight",
		metric.WithDescription("Number of embedding requests currently in flight"),
	)
	errorsTotal, _ := meter.Int64Counter( //nolint:errcheck // Metric initialization errors are rare and handled gracefully
		"embeddings_errors_total",
		metric.WithDescription("Total number of embedding errors"),
	)
	tokensProcessed, _ := meter.Int64Counter( //nolint:errcheck // Metric initialization errors are rare and handled gracefully
		"embeddings_tokens_processed_total",
		metric.WithDescription("Total number of tokens processed for embeddings"),
	)

	return &Metrics{
		requestsTotal:    requestsTotal,
		requestDuration:  requestDuration,
		requestsInFlight: requestsInFlight,
		errorsTotal:      errorsTotal,
		tokensProcessed:  tokensProcessed,
	}
}

// RecordRequest records a successful embedding request.
func (m *Metrics) RecordRequest(ctx context.Context, provider, model string, duration time.Duration, inputCount, outputDimension int) {
	m.requestsTotal.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("provider", provider),
			attribute.String("model", model),
			attribute.Int("input_count", inputCount),
			attribute.Int("output_dimension", outputDimension),
		))
	m.requestDuration.Record(ctx, duration.Seconds(),
		metric.WithAttributes(
			attribute.String("provider", provider),
			attribute.String("model", model),
		))
}

// RecordError records an embedding error.
func (m *Metrics) RecordError(ctx context.Context, provider, model, errorType string) {
	m.errorsTotal.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("provider", provider),
			attribute.String("model", model),
			attribute.String("error_type", errorType),
		))
}

// RecordTokensProcessed records the number of tokens processed.
func (m *Metrics) RecordTokensProcessed(ctx context.Context, provider, model string, tokenCount int) {
	m.tokensProcessed.Add(ctx, int64(tokenCount),
		metric.WithAttributes(
			attribute.String("provider", provider),
			attribute.String("model", model),
		))
}

// StartRequest increments the in-flight counter.
func (m *Metrics) StartRequest(ctx context.Context, provider, model string) {
	m.requestsInFlight.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("provider", provider),
			attribute.String("model", model),
		))
}

// EndRequest decrements the in-flight counter.
func (m *Metrics) EndRequest(ctx context.Context, provider, model string) {
	m.requestsInFlight.Add(ctx, -1,
		metric.WithAttributes(
			attribute.String("provider", provider),
			attribute.String("model", model),
		))
}
