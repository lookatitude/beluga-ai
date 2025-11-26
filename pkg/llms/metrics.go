package llms

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
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

// NoOpMetrics provides a no-operation implementation for when metrics are disabled.
type NoOpMetrics struct{}

// NewNoOpMetrics creates a new no-operation metrics recorder.
func NewNoOpMetrics() *NoOpMetrics {
	return &NoOpMetrics{}
}

// RecordRequest is a no-op implementation.
func (n *NoOpMetrics) RecordRequest(ctx context.Context, provider, model string, duration time.Duration) {
}

// RecordError is a no-op implementation.
func (n *NoOpMetrics) RecordError(ctx context.Context, provider, model, errorCode string, duration time.Duration) {
}

// RecordTokenUsage is a no-op implementation.
func (n *NoOpMetrics) RecordTokenUsage(ctx context.Context, provider, model string, inputTokens, outputTokens int) {
}

// RecordToolCall is a no-op implementation.
func (n *NoOpMetrics) RecordToolCall(ctx context.Context, provider, model, toolName string) {}

// RecordBatch is a no-op implementation.
func (n *NoOpMetrics) RecordBatch(ctx context.Context, provider, model string, batchSize int, duration time.Duration) {
}

// RecordStream is a no-op implementation.
func (n *NoOpMetrics) RecordStream(ctx context.Context, provider, model string, duration time.Duration) {
}

// IncrementActiveRequests is a no-op implementation.
func (n *NoOpMetrics) IncrementActiveRequests(ctx context.Context, provider, model string) {}

// DecrementActiveRequests is a no-op implementation.
func (n *NoOpMetrics) DecrementActiveRequests(ctx context.Context, provider, model string) {}

// IncrementActiveStreams is a no-op implementation.
func (n *NoOpMetrics) IncrementActiveStreams(ctx context.Context, provider, model string) {}

// DecrementActiveStreams is a no-op implementation.
func (n *NoOpMetrics) DecrementActiveStreams(ctx context.Context, provider, model string) {}

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
}

// NewMetrics creates a new Metrics instance.
func NewMetrics(meter metric.Meter) *Metrics {
	m := &Metrics{}

	m.requests, _ = meter.Int64Counter("llm.requests.total", metric.WithDescription("Total LLM requests"))                   //nolint:errcheck // Metrics initialization errors are handled by returning nil or no-op metrics
	m.successful, _ = meter.Int64Counter("llm.generations.successful", metric.WithDescription("Successful LLM generations")) //nolint:errcheck // Metrics initialization errors are handled by returning nil or no-op metrics
	m.errors, _ = meter.Int64Counter("llm.errors.total", metric.WithDescription("Total LLM errors"))                         //nolint:errcheck // Metrics initialization errors are handled by returning nil or no-op metrics
	m.failed, _ = meter.Int64Counter("llm.generations.failed", metric.WithDescription("Failed LLM generations"))             //nolint:errcheck // Metrics initialization errors are handled by returning nil or no-op metrics
	m.inputTokens, _ = meter.Int64Counter("llm.tokens.input", metric.WithDescription("Input tokens count"))                  //nolint:errcheck // Metrics initialization errors are handled by returning nil or no-op metrics
	m.outputTokens, _ = meter.Int64Counter("llm.tokens.output", metric.WithDescription("Output tokens count"))               //nolint:errcheck // Metrics initialization errors are handled by returning nil or no-op metrics
	m.toolCalls, _ = meter.Int64Counter("llm.tool_calls.total", metric.WithDescription("Total tool calls"))                  //nolint:errcheck // Metrics initialization errors are handled by returning nil or no-op metrics
	m.batches, _ = meter.Int64Counter("llm.batches.total", metric.WithDescription("Total batch requests"))
	m.streams, _ = meter.Int64Counter("llm.streams.total", metric.WithDescription("Total stream requests"))
	m.generationLatency, _ = meter.Float64Histogram("llm.generation.latency", metric.WithDescription("Generation latency"), metric.WithUnit("s"))
	m.batchLatency, _ = meter.Float64Histogram("llm.batch.latency", metric.WithDescription("Batch latency"), metric.WithUnit("s"))
	m.streamLatency, _ = meter.Float64Histogram("llm.stream.latency", metric.WithDescription("Stream latency"), metric.WithUnit("s"))
	m.activeRequests, _ = meter.Int64UpDownCounter("llm.requests.active", metric.WithDescription("Active LLM requests"))
	m.activeStreams, _ = meter.Int64UpDownCounter("llm.streams.active", metric.WithDescription("Active LLM streams"))

	return m
}

// RecordRequest records a request metric.
func (m *Metrics) RecordRequest(ctx context.Context, provider, model string, duration time.Duration) {
	attrs := attribute.NewSet(attribute.String("provider", provider), attribute.String("model", model))
	m.requests.Add(ctx, 1, metric.WithAttributeSet(attrs))
	m.successful.Add(ctx, 1, metric.WithAttributeSet(attrs))
	m.generationLatency.Record(ctx, duration.Seconds(), metric.WithAttributeSet(attrs))
}

// RecordError records an error metric.
func (m *Metrics) RecordError(ctx context.Context, provider, model, errorCode string, duration time.Duration) {
	attrs := attribute.NewSet(attribute.String("provider", provider), attribute.String("model", model), attribute.String("error_code", errorCode))
	m.errors.Add(ctx, 1, metric.WithAttributeSet(attrs))
	m.failed.Add(ctx, 1, metric.WithAttributeSet(attrs))
	m.generationLatency.Record(ctx, duration.Seconds(), metric.WithAttributeSet(attrs))
}

// RecordTokenUsage records token usage metrics.
func (m *Metrics) RecordTokenUsage(ctx context.Context, provider, model string, inputTokens, outputTokens int) {
	attrs := attribute.NewSet(attribute.String("provider", provider), attribute.String("model", model))
	m.inputTokens.Add(ctx, int64(inputTokens), metric.WithAttributeSet(attrs))
	m.outputTokens.Add(ctx, int64(outputTokens), metric.WithAttributeSet(attrs))
}

// RecordToolCall records a tool call metric.
func (m *Metrics) RecordToolCall(ctx context.Context, provider, model, toolName string) {
	attrs := attribute.NewSet(attribute.String("provider", provider), attribute.String("model", model), attribute.String("tool_name", toolName))
	m.toolCalls.Add(ctx, 1, metric.WithAttributeSet(attrs))
}

// RecordBatch records batch processing metrics.
func (m *Metrics) RecordBatch(ctx context.Context, provider, model string, batchSize int, duration time.Duration) {
	attrs := attribute.NewSet(attribute.String("provider", provider), attribute.String("model", model), attribute.Int("batch_size", batchSize))
	m.batches.Add(ctx, 1, metric.WithAttributeSet(attrs))
	m.batchLatency.Record(ctx, duration.Seconds(), metric.WithAttributeSet(attrs))
}

// RecordStream records streaming metrics.
func (m *Metrics) RecordStream(ctx context.Context, provider, model string, duration time.Duration) {
	attrs := attribute.NewSet(attribute.String("provider", provider), attribute.String("model", model))
	m.streams.Add(ctx, 1, metric.WithAttributeSet(attrs))
	m.streamLatency.Record(ctx, duration.Seconds(), metric.WithAttributeSet(attrs))
}

// IncrementActiveRequests increments the active requests counter.
func (m *Metrics) IncrementActiveRequests(ctx context.Context, provider, model string) {
	attrs := attribute.NewSet(attribute.String("provider", provider), attribute.String("model", model))
	m.activeRequests.Add(ctx, 1, metric.WithAttributeSet(attrs))
}

// DecrementActiveRequests decrements the active requests counter.
func (m *Metrics) DecrementActiveRequests(ctx context.Context, provider, model string) {
	attrs := attribute.NewSet(attribute.String("provider", provider), attribute.String("model", model))
	m.activeRequests.Add(ctx, -1, metric.WithAttributeSet(attrs))
}

// IncrementActiveStreams increments the active streams counter.
func (m *Metrics) IncrementActiveStreams(ctx context.Context, provider, model string) {
	attrs := attribute.NewSet(attribute.String("provider", provider), attribute.String("model", model))
	m.activeStreams.Add(ctx, 1, metric.WithAttributeSet(attrs))
}

// DecrementActiveStreams decrements the active streams counter.
func (m *Metrics) DecrementActiveStreams(ctx context.Context, provider, model string) {
	attrs := attribute.NewSet(attribute.String("provider", provider), attribute.String("model", model))
	m.activeStreams.Add(ctx, -1, metric.WithAttributeSet(attrs))
}
