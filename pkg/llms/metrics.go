package llms

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// MetricsRecorder defines the interface for recording metrics.
type MetricsRecorder interface {
	RecordRequest(ctx context.Context, provider, model string, duration time.Duration)
	RecordError(ctx context.Context, provider, model, errorCode string, duration time.Duration)
	RecordTokenUsage(ctx context.Context, provider, model string, inputTokens, outputTokens int)
	RecordToolCall(ctx context.Context, provider, model, toolName string)
	RecordBatch(ctx context.Context, provider, model string, batchSize int, duration time.Duration)
	RecordStream(ctx context.Context, provider, model string, duration time.Duration)
	IncrementActiveRequests(ctx context.Context, provider, model string)
	DecrementActiveRequests(ctx context.Context, provider, model string)
	IncrementActiveStreams(ctx context.Context, provider, model string)
	DecrementActiveStreams(ctx context.Context, provider, model string)
}

// Metrics contains all the metrics for LLM operations.
type Metrics struct {
	requests          metric.Int64Counter
	successful        metric.Int64Counter
	errors            metric.Int64Counter
	failed            metric.Int64Counter
	inputTokens       metric.Int64Counter
	outputTokens      metric.Int64Counter
	toolCalls         metric.Int64Counter
	batches           metric.Int64Counter
	streams           metric.Int64Counter
	generationLatency metric.Float64Histogram
	batchLatency      metric.Float64Histogram
	streamLatency     metric.Float64Histogram
	activeRequests    metric.Int64UpDownCounter
	activeStreams     metric.Int64UpDownCounter
	tracer            trace.Tracer
}

// NewMetrics creates a new Metrics instance.
func NewMetrics(meter metric.Meter, tracer trace.Tracer) (*Metrics, error) {
	m := &Metrics{}

	var err error

	m.requests, err = meter.Int64Counter("llm.requests.total", metric.WithDescription("Total LLM requests"))
	if err != nil {
		return nil, fmt.Errorf("failed to create requests metric: %w", err)
	}

	m.successful, err = meter.Int64Counter("llm.generations.successful", metric.WithDescription("Successful LLM generations"))
	if err != nil {
		return nil, fmt.Errorf("failed to create successful metric: %w", err)
	}

	m.errors, err = meter.Int64Counter("llm.errors.total", metric.WithDescription("Total LLM errors"))
	if err != nil {
		return nil, fmt.Errorf("failed to create errors metric: %w", err)
	}

	m.failed, err = meter.Int64Counter("llm.generations.failed", metric.WithDescription("Failed LLM generations"))
	if err != nil {
		return nil, fmt.Errorf("failed to create failed metric: %w", err)
	}

	m.inputTokens, err = meter.Int64Counter("llm.tokens.input", metric.WithDescription("Input tokens count"))
	if err != nil {
		return nil, fmt.Errorf("failed to create inputTokens metric: %w", err)
	}

	m.outputTokens, err = meter.Int64Counter("llm.tokens.output", metric.WithDescription("Output tokens count"))
	if err != nil {
		return nil, fmt.Errorf("failed to create outputTokens metric: %w", err)
	}

	m.toolCalls, err = meter.Int64Counter("llm.tool_calls.total", metric.WithDescription("Total tool calls"))
	if err != nil {
		return nil, fmt.Errorf("failed to create toolCalls metric: %w", err)
	}

	m.batches, err = meter.Int64Counter("llm.batches.total", metric.WithDescription("Total batch requests"))
	if err != nil {
		return nil, fmt.Errorf("failed to create batches metric: %w", err)
	}

	m.streams, err = meter.Int64Counter("llm.streams.total", metric.WithDescription("Total stream requests"))
	if err != nil {
		return nil, fmt.Errorf("failed to create streams metric: %w", err)
	}

	m.generationLatency, err = meter.Float64Histogram("llm.generation.latency", metric.WithDescription("Generation latency"), metric.WithUnit("s"))
	if err != nil {
		return nil, fmt.Errorf("failed to create generationLatency metric: %w", err)
	}

	m.batchLatency, err = meter.Float64Histogram("llm.batch.latency", metric.WithDescription("Batch latency"), metric.WithUnit("s"))
	if err != nil {
		return nil, fmt.Errorf("failed to create batchLatency metric: %w", err)
	}

	m.streamLatency, err = meter.Float64Histogram("llm.stream.latency", metric.WithDescription("Stream latency"), metric.WithUnit("s"))
	if err != nil {
		return nil, fmt.Errorf("failed to create streamLatency metric: %w", err)
	}

	m.activeRequests, err = meter.Int64UpDownCounter("llm.requests.active", metric.WithDescription("Active LLM requests"))
	if err != nil {
		return nil, fmt.Errorf("failed to create activeRequests metric: %w", err)
	}

	m.activeStreams, err = meter.Int64UpDownCounter("llm.streams.active", metric.WithDescription("Active LLM streams"))
	if err != nil {
		return nil, fmt.Errorf("failed to create activeStreams metric: %w", err)
	}

	m.tracer = tracer

	return m, nil
}

// NoOpMetrics returns a metrics instance that does nothing.
// Useful for testing or when metrics are disabled.
func NoOpMetrics() *Metrics {
	return &Metrics{
		tracer: trace.NewNoopTracerProvider().Tracer("llms"),
	}
}

// RecordRequest records a request metric.
func (m *Metrics) RecordRequest(ctx context.Context, provider, model string, duration time.Duration) {
	if m == nil {
		return
	}
	attrs := attribute.NewSet(attribute.String("provider", provider), attribute.String("model", model))
	if m.requests != nil {
		m.requests.Add(ctx, 1, metric.WithAttributeSet(attrs))
	}
	if m.successful != nil {
		m.successful.Add(ctx, 1, metric.WithAttributeSet(attrs))
	}
	if m.generationLatency != nil {
		m.generationLatency.Record(ctx, duration.Seconds(), metric.WithAttributeSet(attrs))
	}
}

// RecordError records an error metric.
func (m *Metrics) RecordError(ctx context.Context, provider, model, errorCode string, duration time.Duration) {
	if m == nil {
		return
	}
	attrs := attribute.NewSet(attribute.String("provider", provider), attribute.String("model", model), attribute.String("error_code", errorCode))
	if m.errors != nil {
		m.errors.Add(ctx, 1, metric.WithAttributeSet(attrs))
	}
	if m.failed != nil {
		m.failed.Add(ctx, 1, metric.WithAttributeSet(attrs))
	}
	if m.generationLatency != nil {
		m.generationLatency.Record(ctx, duration.Seconds(), metric.WithAttributeSet(attrs))
	}
}

// RecordTokenUsage records token usage metrics.
func (m *Metrics) RecordTokenUsage(ctx context.Context, provider, model string, inputTokens, outputTokens int) {
	if m == nil {
		return
	}
	attrs := attribute.NewSet(attribute.String("provider", provider), attribute.String("model", model))
	if m.inputTokens != nil {
		m.inputTokens.Add(ctx, int64(inputTokens), metric.WithAttributeSet(attrs))
	}
	if m.outputTokens != nil {
		m.outputTokens.Add(ctx, int64(outputTokens), metric.WithAttributeSet(attrs))
	}
}

// RecordToolCall records a tool call metric.
func (m *Metrics) RecordToolCall(ctx context.Context, provider, model, toolName string) {
	if m == nil || m.toolCalls == nil {
		return
	}
	attrs := attribute.NewSet(attribute.String("provider", provider), attribute.String("model", model), attribute.String("tool_name", toolName))
	m.toolCalls.Add(ctx, 1, metric.WithAttributeSet(attrs))
}

// RecordBatch records batch processing metrics.
func (m *Metrics) RecordBatch(ctx context.Context, provider, model string, batchSize int, duration time.Duration) {
	if m == nil {
		return
	}
	attrs := attribute.NewSet(attribute.String("provider", provider), attribute.String("model", model), attribute.Int("batch_size", batchSize))
	if m.batches != nil {
		m.batches.Add(ctx, 1, metric.WithAttributeSet(attrs))
	}
	if m.batchLatency != nil {
		m.batchLatency.Record(ctx, duration.Seconds(), metric.WithAttributeSet(attrs))
	}
}

// RecordStream records streaming metrics.
func (m *Metrics) RecordStream(ctx context.Context, provider, model string, duration time.Duration) {
	if m == nil {
		return
	}
	attrs := attribute.NewSet(attribute.String("provider", provider), attribute.String("model", model))
	if m.streams != nil {
		m.streams.Add(ctx, 1, metric.WithAttributeSet(attrs))
	}
	if m.streamLatency != nil {
		m.streamLatency.Record(ctx, duration.Seconds(), metric.WithAttributeSet(attrs))
	}
}

// IncrementActiveRequests increments the active requests counter.
func (m *Metrics) IncrementActiveRequests(ctx context.Context, provider, model string) {
	if m == nil || m.activeRequests == nil {
		return
	}
	attrs := attribute.NewSet(attribute.String("provider", provider), attribute.String("model", model))
	m.activeRequests.Add(ctx, 1, metric.WithAttributeSet(attrs))
}

// DecrementActiveRequests decrements the active requests counter.
func (m *Metrics) DecrementActiveRequests(ctx context.Context, provider, model string) {
	if m == nil || m.activeRequests == nil {
		return
	}
	attrs := attribute.NewSet(attribute.String("provider", provider), attribute.String("model", model))
	m.activeRequests.Add(ctx, -1, metric.WithAttributeSet(attrs))
}

// IncrementActiveStreams increments the active streams counter.
func (m *Metrics) IncrementActiveStreams(ctx context.Context, provider, model string) {
	if m == nil || m.activeStreams == nil {
		return
	}
	attrs := attribute.NewSet(attribute.String("provider", provider), attribute.String("model", model))
	m.activeStreams.Add(ctx, 1, metric.WithAttributeSet(attrs))
}

// DecrementActiveStreams decrements the active streams counter.
func (m *Metrics) DecrementActiveStreams(ctx context.Context, provider, model string) {
	if m == nil || m.activeStreams == nil {
		return
	}
	attrs := attribute.NewSet(attribute.String("provider", provider), attribute.String("model", model))
	m.activeStreams.Add(ctx, -1, metric.WithAttributeSet(attrs))
}
