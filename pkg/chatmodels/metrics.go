package chatmodels

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

// OpenTelemetry metric instruments
// These will be properly initialized when creating the Metrics instance

// Metrics holds the metrics for the chatmodels package.
type Metrics struct {
	// Message generation metrics
	messageGenerations      metric.Int64Counter
	messageGenerationTime   metric.Float64Histogram
	messageGenerationErrors metric.Int64Counter

	// Streaming metrics
	streamingSessions metric.Int64Counter
	streamingDuration metric.Float64Histogram
	streamingErrors   metric.Int64Counter
	messagesStreamed  metric.Int64Counter

	// Token metrics
	tokensGenerated metric.Int64Counter
	tokensConsumed  metric.Int64Counter

	// Provider metrics
	providerRequests metric.Int64Counter
	providerErrors   metric.Int64Counter
	providerLatency  metric.Float64Histogram

	// Model metrics
	modelRequests metric.Int64Counter
	modelErrors   metric.Int64Counter
	modelLatency  metric.Float64Histogram

	// Tracer
	tracer trace.Tracer
}

// NewMetrics creates a new Metrics instance with OpenTelemetry metrics.
func NewMetrics(meter metric.Meter, tracer trace.Tracer) (*Metrics, error) {
	// Message generation metrics
	messageGenerations, err := meter.Int64Counter(
		"chatmodels_message_generations_total",
		metric.WithDescription("Total number of message generations"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create message_generations_total counter: %w", err)
	}

	messageGenerationTime, err := meter.Float64Histogram(
		"chatmodels_message_generation_duration_seconds",
		metric.WithDescription("Duration of message generation operations"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}

	messageGenerationErrors, err := meter.Int64Counter(
		"chatmodels_message_generation_errors_total",
		metric.WithDescription("Total number of message generation errors"),
	)
	if err != nil {
		return nil, err
	}

	// Streaming metrics
	streamingSessions, err := meter.Int64Counter(
		"chatmodels_streaming_sessions_total",
		metric.WithDescription("Total number of streaming sessions"),
	)
	if err != nil {
		return nil, err
	}

	streamingDuration, err := meter.Float64Histogram(
		"chatmodels_streaming_duration_seconds",
		metric.WithDescription("Duration of streaming sessions"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}

	streamingErrors, err := meter.Int64Counter(
		"chatmodels_streaming_errors_total",
		metric.WithDescription("Total number of streaming errors"),
	)
	if err != nil {
		return nil, err
	}

	messagesStreamed, err := meter.Int64Counter(
		"chatmodels_messages_streamed_total",
		metric.WithDescription("Total number of messages streamed"),
	)
	if err != nil {
		return nil, err
	}

	// Token metrics
	tokensGenerated, err := meter.Int64Counter(
		"chatmodels_tokens_generated_total",
		metric.WithDescription("Total number of tokens generated"),
	)
	if err != nil {
		return nil, err
	}

	tokensConsumed, err := meter.Int64Counter(
		"chatmodels_tokens_consumed_total",
		metric.WithDescription("Total number of tokens consumed"),
	)
	if err != nil {
		return nil, err
	}

	// Provider metrics
	providerRequests, err := meter.Int64Counter(
		"chatmodels_provider_requests_total",
		metric.WithDescription("Total number of provider requests"),
	)
	if err != nil {
		return nil, err
	}

	providerErrors, err := meter.Int64Counter(
		"chatmodels_provider_errors_total",
		metric.WithDescription("Total number of provider errors"),
	)
	if err != nil {
		return nil, err
	}

	providerLatency, err := meter.Float64Histogram(
		"chatmodels_provider_latency_seconds",
		metric.WithDescription("Provider request latency"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}

	// Model metrics
	modelRequests, err := meter.Int64Counter(
		"chatmodels_model_requests_total",
		metric.WithDescription("Total number of model requests"),
	)
	if err != nil {
		return nil, err
	}

	modelErrors, err := meter.Int64Counter(
		"chatmodels_model_errors_total",
		metric.WithDescription("Total number of model errors"),
	)
	if err != nil {
		return nil, err
	}

	modelLatency, err := meter.Float64Histogram(
		"chatmodels_model_latency_seconds",
		metric.WithDescription("Model request latency"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}

	return &Metrics{
		messageGenerations:      messageGenerations,
		messageGenerationTime:   messageGenerationTime,
		messageGenerationErrors: messageGenerationErrors,
		streamingSessions:       streamingSessions,
		streamingDuration:       streamingDuration,
		streamingErrors:         streamingErrors,
		messagesStreamed:        messagesStreamed,
		tokensGenerated:         tokensGenerated,
		tokensConsumed:          tokensConsumed,
		providerRequests:        providerRequests,
		providerErrors:          providerErrors,
		providerLatency:         providerLatency,
		modelRequests:           modelRequests,
		modelErrors:             modelErrors,
		modelLatency:            modelLatency,
		tracer:                  tracer,
	}, nil
}

// Message generation metrics.
func (m *Metrics) RecordMessageGeneration(model, provider string, duration time.Duration, success bool, tokenCount int) {
	if m.messageGenerations == nil || m.messageGenerationTime == nil || m.messageGenerationErrors == nil || m.tokensGenerated == nil {
		return // No-op if metrics are not initialized
	}

	ctx := context.Background()
	attrs := attribute.NewSet(
		attribute.String("model", model),
		attribute.String("provider", provider),
	)

	// Record message generation count
	m.messageGenerations.Add(ctx, 1, metric.WithAttributeSet(attrs))

	// Record duration
	m.messageGenerationTime.Record(ctx, duration.Seconds(), metric.WithAttributeSet(attrs))

	// Record tokens
	if success && tokenCount > 0 {
		m.tokensGenerated.Add(ctx, int64(tokenCount), metric.WithAttributeSet(attrs))
	}

	// Record errors
	if !success {
		m.messageGenerationErrors.Add(ctx, 1, metric.WithAttributeSet(attrs))
	}
}

func (m *Metrics) RecordMessageGenerationError(model, provider, errorType string) {
	if m.messageGenerationErrors == nil {
		return // No-op if metrics are not initialized
	}

	ctx := context.Background()
	attrs := attribute.NewSet(
		attribute.String("model", model),
		attribute.String("provider", provider),
		attribute.String("error_type", errorType),
	)

	m.messageGenerationErrors.Add(ctx, 1, metric.WithAttributeSet(attrs))
}

// Streaming metrics.
func (m *Metrics) RecordStreamingSession(model, provider string, duration time.Duration, success bool, messageCount int) {
	if m.streamingSessions == nil || m.streamingDuration == nil || m.messagesStreamed == nil || m.streamingErrors == nil {
		return // No-op if metrics are not initialized
	}

	ctx := context.Background()
	attrs := attribute.NewSet(
		attribute.String("model", model),
		attribute.String("provider", provider),
	)

	// Record streaming session count
	m.streamingSessions.Add(ctx, 1, metric.WithAttributeSet(attrs))

	// Record duration
	m.streamingDuration.Record(ctx, duration.Seconds(), metric.WithAttributeSet(attrs))

	// Record messages streamed
	if success && messageCount > 0 {
		m.messagesStreamed.Add(ctx, int64(messageCount), metric.WithAttributeSet(attrs))
	}

	// Record errors
	if !success {
		m.streamingErrors.Add(ctx, 1, metric.WithAttributeSet(attrs))
	}
}

func (m *Metrics) RecordStreamingError(model, provider, errorType string) {
	if m.streamingErrors == nil {
		return // No-op if metrics are not initialized
	}

	ctx := context.Background()
	attrs := attribute.NewSet(
		attribute.String("model", model),
		attribute.String("provider", provider),
		attribute.String("error_type", errorType),
	)

	m.streamingErrors.Add(ctx, 1, metric.WithAttributeSet(attrs))
}

// Token metrics.
func (m *Metrics) RecordTokenUsage(model, provider string, tokensGenerated, tokensConsumed int) {
	if m.tokensGenerated == nil || m.tokensConsumed == nil {
		return // No-op if metrics are not initialized
	}

	ctx := context.Background()
	attrs := attribute.NewSet(
		attribute.String("model", model),
		attribute.String("provider", provider),
	)

	if tokensGenerated > 0 {
		m.tokensGenerated.Add(ctx, int64(tokensGenerated), metric.WithAttributeSet(attrs))
	}

	if tokensConsumed > 0 {
		m.tokensConsumed.Add(ctx, int64(tokensConsumed), metric.WithAttributeSet(attrs))
	}
}

// Provider metrics.
func (m *Metrics) RecordProviderRequest(provider string, duration time.Duration, success bool) {
	if m.providerRequests == nil || m.providerLatency == nil || m.providerErrors == nil {
		return // No-op if metrics are not initialized
	}

	ctx := context.Background()
	attrs := attribute.NewSet(
		attribute.String("provider", provider),
	)

	// Record request count
	m.providerRequests.Add(ctx, 1, metric.WithAttributeSet(attrs))

	// Record latency
	m.providerLatency.Record(ctx, duration.Seconds(), metric.WithAttributeSet(attrs))

	// Record errors
	if !success {
		m.providerErrors.Add(ctx, 1, metric.WithAttributeSet(attrs))
	}
}

func (m *Metrics) RecordProviderError(provider, errorType string) {
	if m.providerErrors == nil {
		return // No-op if metrics are not initialized
	}

	ctx := context.Background()
	attrs := attribute.NewSet(
		attribute.String("provider", provider),
		attribute.String("error_type", errorType),
	)

	m.providerErrors.Add(ctx, 1, metric.WithAttributeSet(attrs))
}

// Model metrics.
func (m *Metrics) RecordModelRequest(model, provider string, duration time.Duration, success bool) {
	if m.modelRequests == nil || m.modelLatency == nil || m.modelErrors == nil {
		return // No-op if metrics are not initialized
	}

	ctx := context.Background()
	attrs := attribute.NewSet(
		attribute.String("model", model),
		attribute.String("provider", provider),
	)

	// Record request count
	m.modelRequests.Add(ctx, 1, metric.WithAttributeSet(attrs))

	// Record latency
	m.modelLatency.Record(ctx, duration.Seconds(), metric.WithAttributeSet(attrs))

	// Record errors
	if !success {
		m.modelErrors.Add(ctx, 1, metric.WithAttributeSet(attrs))
	}
}

func (m *Metrics) RecordModelError(model, provider, errorType string) {
	if m.modelErrors == nil {
		return // No-op if metrics are not initialized
	}

	ctx := context.Background()
	attrs := attribute.NewSet(
		attribute.String("model", model),
		attribute.String("provider", provider),
		attribute.String("error_type", errorType),
	)

	m.modelErrors.Add(ctx, 1, metric.WithAttributeSet(attrs))
}

// Tracing helpers.
// Spans returned by these methods must be ended by the caller using span.End().
//
//nolint:spancheck // Spans are intentionally returned for caller to manage lifecycle
func (m *Metrics) StartGenerationSpan(ctx context.Context, model, provider, operation string) (context.Context, trace.Span) {
	ctx, span := m.tracer.Start(ctx, "chatmodel."+operation,
		trace.WithAttributes(
			attribute.String("chatmodel.model", model),
			attribute.String("chatmodel.provider", provider),
		),
	)
	return ctx, span
}

//nolint:spancheck // Spans are intentionally returned for caller to manage lifecycle
func (m *Metrics) StartStreamingSpan(ctx context.Context, model, provider string) (context.Context, trace.Span) {
	ctx, span := m.tracer.Start(ctx, "chatmodel.stream",
		trace.WithAttributes(
			attribute.String("chatmodel.model", model),
			attribute.String("chatmodel.provider", provider),
		),
	)
	return ctx, span
}

//nolint:spancheck // Spans are intentionally returned for caller to manage lifecycle
func (m *Metrics) StartProviderSpan(ctx context.Context, provider, operation string) (context.Context, trace.Span) {
	ctx, span := m.tracer.Start(ctx, "chatmodel.provider."+operation,
		trace.WithAttributes(
			attribute.String("chatmodel.provider", provider),
		),
	)
	return ctx, span
}

// DefaultMetrics creates a metrics instance with default meter and tracer.
func DefaultMetrics() *Metrics {
	meter := otel.Meter("beluga-chatmodels")
	tracer := otel.Tracer("beluga-chatmodels")
	metrics, err := NewMetrics(meter, tracer)
	if err != nil {
		// Fallback to no-op metrics if initialization fails
		return NoOpMetrics()
	}
	return metrics
}

// NoOpMetrics returns a metrics instance that does nothing.
// Useful for testing or when metrics are disabled.
func NoOpMetrics() *Metrics {
	// Create a metrics instance with nil values that won't panic but won't record anything
	return &Metrics{
		tracer: noop.NewTracerProvider().Tracer("noop"),
		// All metric fields will be nil, which is fine for no-op behavior
	}
}
