package llms

import (
	"context"
	"time"
)

// MetricsRecorder defines the interface for recording metrics
type MetricsRecorder interface {
	RecordRequest(ctx context.Context, provider, model string, duration time.Duration)
	RecordError(ctx context.Context, provider, model, errorCode string)
	RecordTokenUsage(ctx context.Context, provider, model string, inputTokens, outputTokens int)
	RecordToolCall(ctx context.Context, provider, model, toolName string)
	RecordBatch(ctx context.Context, provider, model string, batchSize int, duration time.Duration)
	RecordStream(ctx context.Context, provider, model string, duration time.Duration)
	IncrementActiveRequests(ctx context.Context, provider, model string)
	DecrementActiveRequests(ctx context.Context, provider, model string)
	IncrementActiveStreams(ctx context.Context, provider, model string)
	DecrementActiveStreams(ctx context.Context, provider, model string)
}

// NoOpMetrics provides a no-operation implementation for when metrics are disabled
type NoOpMetrics struct{}

// NewNoOpMetrics creates a new no-operation metrics recorder
func NewNoOpMetrics() *NoOpMetrics {
	return &NoOpMetrics{}
}

// RecordRequest is a no-op implementation
func (n *NoOpMetrics) RecordRequest(ctx context.Context, provider, model string, duration time.Duration) {
}

// RecordError is a no-op implementation
func (n *NoOpMetrics) RecordError(ctx context.Context, provider, model, errorCode string) {}

// RecordTokenUsage is a no-op implementation
func (n *NoOpMetrics) RecordTokenUsage(ctx context.Context, provider, model string, inputTokens, outputTokens int) {
}

// RecordToolCall is a no-op implementation
func (n *NoOpMetrics) RecordToolCall(ctx context.Context, provider, model, toolName string) {}

// RecordBatch is a no-op implementation
func (n *NoOpMetrics) RecordBatch(ctx context.Context, provider, model string, batchSize int, duration time.Duration) {
}

// RecordStream is a no-op implementation
func (n *NoOpMetrics) RecordStream(ctx context.Context, provider, model string, duration time.Duration) {
}

// IncrementActiveRequests is a no-op implementation
func (n *NoOpMetrics) IncrementActiveRequests(ctx context.Context, provider, model string) {}

// DecrementActiveRequests is a no-op implementation
func (n *NoOpMetrics) DecrementActiveRequests(ctx context.Context, provider, model string) {}

// IncrementActiveStreams is a no-op implementation
func (n *NoOpMetrics) IncrementActiveStreams(ctx context.Context, provider, model string) {}

// DecrementActiveStreams is a no-op implementation
func (n *NoOpMetrics) DecrementActiveStreams(ctx context.Context, provider, model string) {}

// Metrics contains all the metrics for LLM operations
type Metrics struct {
	// Counter metrics
	requestsTotal      int64
	requestsByProvider map[string]int64
	requestsByModel    map[string]int64
	errorsTotal        int64
	errorsByProvider   map[string]int64
	errorsByCode       map[string]int64
	tokenUsageTotal    int64
	toolCallsTotal     int64

	// Histogram metrics (simplified as maps)
	requestDurations []time.Duration
	streamDurations  []time.Duration
	tokenUsages      []int
	batchSizes       []int

	// UpDownCounter metrics
	activeRequests int64
	activeStreams  int64
}

// NewMetrics creates a new Metrics instance
func NewMetrics() *Metrics {
	return &Metrics{
		requestsByProvider: make(map[string]int64),
		requestsByModel:    make(map[string]int64),
		errorsByProvider:   make(map[string]int64),
		errorsByCode:       make(map[string]int64),
		requestDurations:   make([]time.Duration, 0),
		streamDurations:    make([]time.Duration, 0),
		tokenUsages:        make([]int, 0),
		batchSizes:         make([]int, 0),
	}
}

// RecordRequest records a request metric
func (m *Metrics) RecordRequest(ctx context.Context, provider, model string, duration time.Duration) {
	m.requestsTotal++
	m.requestsByProvider[provider]++
	m.requestsByModel[model]++
	m.requestDurations = append(m.requestDurations, duration)
}

// RecordError records an error metric
func (m *Metrics) RecordError(ctx context.Context, provider, model, errorCode string) {
	m.errorsTotal++
	m.errorsByProvider[provider]++
	m.errorsByCode[errorCode]++
}

// RecordTokenUsage records token usage metrics
func (m *Metrics) RecordTokenUsage(ctx context.Context, provider, model string, inputTokens, outputTokens int) {
	totalTokens := inputTokens + outputTokens
	m.tokenUsageTotal += int64(totalTokens)
	m.tokenUsages = append(m.tokenUsages, totalTokens)
}

// RecordToolCall records a tool call metric
func (m *Metrics) RecordToolCall(ctx context.Context, provider, model, toolName string) {
	m.toolCallsTotal++
}

// RecordBatch records batch processing metrics
func (m *Metrics) RecordBatch(ctx context.Context, provider, model string, batchSize int, duration time.Duration) {
	m.batchSizes = append(m.batchSizes, batchSize)
	m.requestDurations = append(m.requestDurations, duration)
}

// RecordStream records streaming metrics
func (m *Metrics) RecordStream(ctx context.Context, provider, model string, duration time.Duration) {
	m.streamDurations = append(m.streamDurations, duration)
}

// IncrementActiveRequests increments the active requests counter
func (m *Metrics) IncrementActiveRequests(ctx context.Context, provider, model string) {
	m.activeRequests++
}

// DecrementActiveRequests decrements the active requests counter
func (m *Metrics) DecrementActiveRequests(ctx context.Context, provider, model string) {
	m.activeRequests--
}

// IncrementActiveStreams increments the active streams counter
func (m *Metrics) IncrementActiveStreams(ctx context.Context, provider, model string) {
	m.activeStreams++
}

// DecrementActiveStreams decrements the active streams counter
func (m *Metrics) DecrementActiveStreams(ctx context.Context, provider, model string) {
	m.activeStreams--
}

// GetRequestsTotal returns the total number of requests
func (m *Metrics) GetRequestsTotal() int64 {
	return m.requestsTotal
}

// GetErrorsTotal returns the total number of errors
func (m *Metrics) GetErrorsTotal() int64 {
	return m.errorsTotal
}

// GetTokenUsageTotal returns the total token usage
func (m *Metrics) GetTokenUsageTotal() int64 {
	return m.tokenUsageTotal
}

// GetActiveRequests returns the number of active requests
func (m *Metrics) GetActiveRequests() int64 {
	return m.activeRequests
}

// GetActiveStreams returns the number of active streams
func (m *Metrics) GetActiveStreams() int64 {
	return m.activeStreams
}

// Reset resets all metrics to zero
func (m *Metrics) Reset() {
	m.requestsTotal = 0
	m.requestsByProvider = make(map[string]int64)
	m.requestsByModel = make(map[string]int64)
	m.errorsTotal = 0
	m.errorsByProvider = make(map[string]int64)
	m.errorsByCode = make(map[string]int64)
	m.tokenUsageTotal = 0
	m.toolCallsTotal = 0
	m.requestDurations = make([]time.Duration, 0)
	m.streamDurations = make([]time.Duration, 0)
	m.tokenUsages = make([]int, 0)
	m.batchSizes = make([]int, 0)
	m.activeRequests = 0
	m.activeStreams = 0
}
