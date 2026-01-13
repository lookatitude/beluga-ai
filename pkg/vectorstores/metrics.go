package vectorstores

import (
	"context"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/trace"
)

// MetricsCollector provides metrics collection for vector store operations.
// It follows OpenTelemetry conventions for consistent observability.
type MetricsCollector struct {
	meter  metric.Meter
	tracer trace.Tracer

	// Document operations
	documentsAdded   metric.Int64Counter
	documentsDeleted metric.Int64Counter
	documentsStored  metric.Int64UpDownCounter

	// Search operations
	searchRequests     metric.Int64Counter
	searchDuration     metric.Float64Histogram
	searchResultsCount metric.Int64Histogram

	// Embedding operations
	embeddingRequests metric.Int64Counter
	embeddingDuration metric.Float64Histogram

	// Error tracking
	errorsTotal metric.Int64Counter

	// Resource usage
	memoryUsage metric.Int64UpDownCounter
	diskUsage   metric.Int64UpDownCounter
}

// NewMetricsCollector creates a new metrics collector with OpenTelemetry.
func NewMetricsCollector(meter metric.Meter, tracer trace.Tracer) (*MetricsCollector, error) {
	if meter == nil {
		meter = noop.NewMeterProvider().Meter("vectorstores")
	}

	if tracer == nil {
		tracer = trace.NewNoopTracerProvider().Tracer("vectorstores")
	}

	mc := &MetricsCollector{
		meter:  meter,
		tracer: tracer,
	}

	var err error

	// Document operations metrics
	if mc.documentsAdded, err = meter.Int64Counter(
		"vectorstore_documents_added_total",
		metric.WithDescription("Total number of documents added to vector stores"),
		metric.WithUnit("1"),
	); err != nil {
		return nil, err
	}

	if mc.documentsDeleted, err = meter.Int64Counter(
		"vectorstore_documents_deleted_total",
		metric.WithDescription("Total number of documents deleted from vector stores"),
		metric.WithUnit("1"),
	); err != nil {
		return nil, err
	}

	if mc.documentsStored, err = meter.Int64UpDownCounter(
		"vectorstore_documents_stored",
		metric.WithDescription("Current number of documents stored in vector stores"),
		metric.WithUnit("1"),
	); err != nil {
		return nil, err
	}

	// Search operations metrics
	if mc.searchRequests, err = meter.Int64Counter(
		"vectorstore_search_requests_total",
		metric.WithDescription("Total number of search requests"),
		metric.WithUnit("1"),
	); err != nil {
		return nil, err
	}

	if mc.searchDuration, err = meter.Float64Histogram(
		"vectorstore_search_duration_seconds",
		metric.WithDescription("Duration of search operations"),
		metric.WithUnit("s"),
	); err != nil {
		return nil, err
	}

	if mc.searchResultsCount, err = meter.Int64Histogram(
		"vectorstore_search_results_count",
		metric.WithDescription("Number of results returned by search operations"),
		metric.WithUnit("1"),
	); err != nil {
		return nil, err
	}

	// Embedding operations metrics
	if mc.embeddingRequests, err = meter.Int64Counter(
		"vectorstore_embedding_requests_total",
		metric.WithDescription("Total number of embedding requests"),
		metric.WithUnit("1"),
	); err != nil {
		return nil, err
	}

	if mc.embeddingDuration, err = meter.Float64Histogram(
		"vectorstore_embedding_duration_seconds",
		metric.WithDescription("Duration of embedding operations"),
		metric.WithUnit("s"),
	); err != nil {
		return nil, err
	}

	// Error tracking
	if mc.errorsTotal, err = meter.Int64Counter(
		"vectorstore_errors_total",
		metric.WithDescription("Total number of errors by type"),
		metric.WithUnit("1"),
	); err != nil {
		return nil, err
	}

	// Resource usage metrics
	if mc.memoryUsage, err = meter.Int64UpDownCounter(
		"vectorstore_memory_usage_bytes",
		metric.WithDescription("Current memory usage of vector stores"),
		metric.WithUnit("By"),
	); err != nil {
		return nil, err
	}

	if mc.diskUsage, err = meter.Int64UpDownCounter(
		"vectorstore_disk_usage_bytes",
		metric.WithDescription("Current disk usage of vector stores"),
		metric.WithUnit("By"),
	); err != nil {
		return nil, err
	}

	return mc, nil
}

// RecordDocumentsAdded records metrics when documents are added.
func (mc *MetricsCollector) RecordDocumentsAdded(ctx context.Context, count int, storeName string) {
	if mc == nil {
		return
	}

	mc.documentsAdded.Add(ctx, int64(count), metric.WithAttributes(attribute.String("store_name", storeName)))
	mc.documentsStored.Add(ctx, int64(count), metric.WithAttributes(attribute.String("store_name", storeName)))
}

// RecordDocumentsDeleted records metrics when documents are deleted.
func (mc *MetricsCollector) RecordDocumentsDeleted(ctx context.Context, count int, storeName string) {
	if mc == nil {
		return
	}

	mc.documentsDeleted.Add(ctx, int64(count), metric.WithAttributes(attribute.String("store_name", storeName)))
	mc.documentsStored.Add(ctx, -int64(count), metric.WithAttributes(attribute.String("store_name", storeName)))
}

// RecordSearch records metrics for search operations.
func (mc *MetricsCollector) RecordSearch(ctx context.Context, duration time.Duration, resultCount int, storeName string) {
	if mc == nil {
		return
	}

	attrs := metric.WithAttributes(attribute.String("store_name", storeName))
	mc.searchRequests.Add(ctx, 1, attrs)
	mc.searchDuration.Record(ctx, duration.Seconds(), attrs)
	mc.searchResultsCount.Record(ctx, int64(resultCount), attrs)
}

// RecordEmbedding records metrics for embedding operations.
func (mc *MetricsCollector) RecordEmbedding(ctx context.Context, duration time.Duration, textCount int, storeName string) {
	if mc == nil {
		return
	}

	attrs := metric.WithAttributes(
		attribute.String("store_name", storeName),
		attribute.Int("text_count", textCount),
	)
	mc.embeddingRequests.Add(ctx, 1, attrs)
	mc.embeddingDuration.Record(ctx, duration.Seconds(), attrs)
}

// RecordError records error metrics.
func (mc *MetricsCollector) RecordError(ctx context.Context, errorCode, storeName string) {
	if mc == nil {
		return
	}

	attrs := metric.WithAttributes(
		attribute.String("store_name", storeName),
		attribute.String("error_code", errorCode),
	)
	mc.errorsTotal.Add(ctx, 1, attrs)
}

// RecordMemoryUsage records current memory usage.
func (mc *MetricsCollector) RecordMemoryUsage(ctx context.Context, bytes int64, storeName string) {
	if mc == nil {
		return
	}

	mc.memoryUsage.Add(ctx, bytes, metric.WithAttributes(attribute.String("store_name", storeName)))
}

// RecordDiskUsage records current disk usage.
func (mc *MetricsCollector) RecordDiskUsage(ctx context.Context, bytes int64, storeName string) {
	if mc == nil {
		return
	}

	mc.diskUsage.Add(ctx, bytes, metric.WithAttributes(attribute.String("store_name", storeName)))
}

// TracerProvider provides tracing functionality for vector store operations.
type TracerProvider struct {
	tracerName string
}

// NewTracerProvider creates a new tracer provider for vector stores.
func NewTracerProvider(tracerName string) *TracerProvider {
	if tracerName == "" {
		tracerName = "github.com/lookatitude/beluga-ai/pkg/vectorstores"
	}
	return &TracerProvider{
		tracerName: tracerName,
	}
}

// StartSpan starts a new trace span for an operation.
func (tp *TracerProvider) StartSpan(ctx context.Context, operation string, opts ...attribute.KeyValue) (context.Context, func()) {
	tracer := otel.Tracer(tp.tracerName)
	ctx, span := tracer.Start(ctx, operation)

	// Add attributes to span
	for _, opt := range opts {
		span.SetAttributes(opt)
	}

	return ctx, func() { span.End() }
}

// StartAddDocumentsSpan starts a span for document addition operations.
func (tp *TracerProvider) StartAddDocumentsSpan(ctx context.Context, storeName string, docCount int) (context.Context, func()) {
	return tp.StartSpan(ctx, "vectorstore.add_documents",
		attribute.String("store_name", storeName),
		attribute.Int("document_count", docCount),
	)
}

// StartDeleteDocumentsSpan starts a span for document deletion operations.
func (tp *TracerProvider) StartDeleteDocumentsSpan(ctx context.Context, storeName string, docCount int) (context.Context, func()) {
	return tp.StartSpan(ctx, "vectorstore.delete_documents",
		attribute.String("store_name", storeName),
		attribute.Int("document_count", docCount),
	)
}

// StartSearchSpan starts a span for search operations.
func (tp *TracerProvider) StartSearchSpan(ctx context.Context, storeName string, queryLength, k int) (context.Context, func()) {
	return tp.StartSpan(ctx, "vectorstore.search",
		attribute.String("store_name", storeName),
		attribute.Int("query_length", queryLength),
		attribute.Int("search_k", k),
	)
}

// StartEmbeddingSpan starts a span for embedding operations.
func (tp *TracerProvider) StartEmbeddingSpan(ctx context.Context, storeName string, textCount int) (context.Context, func()) {
	return tp.StartSpan(ctx, "vectorstore.embedding",
		attribute.String("store_name", storeName),
		attribute.Int("text_count", textCount),
	)
}

// Global instances for convenience.
var (
	globalMetrics *MetricsCollector
	globalTracer  *TracerProvider
	metricsOnce   sync.Once
)

// InitMetrics initializes the global metrics instance.
// This follows the standard pattern used across all Beluga AI packages.
func InitMetrics(meter metric.Meter, tracer trace.Tracer) {
	metricsOnce.Do(func() {
		if meter == nil {
			meter = noop.NewMeterProvider().Meter("vectorstores")
		}
		if tracer == nil {
			tracer = trace.NewNoopTracerProvider().Tracer("vectorstores")
		}
		mc, err := NewMetricsCollector(meter, tracer)
		if err != nil {
			// If metrics creation fails, use noop meter
			meter = noop.NewMeterProvider().Meter("vectorstores")
			tracer = trace.NewNoopTracerProvider().Tracer("vectorstores")
			mc, _ = NewMetricsCollector(meter, tracer)
		}
		globalMetrics = mc
	})
}

// GetMetrics returns the global metrics instance.
// This follows the standard pattern used across all Beluga AI packages.
func GetMetrics() *MetricsCollector {
	return globalMetrics
}

// SetGlobalMetrics is deprecated. Use InitMetrics instead.
// Deprecated: Use InitMetrics(meter) instead.
func SetGlobalMetrics(mc *MetricsCollector) {
	globalMetrics = mc
}

// GetGlobalMetrics is deprecated. Use GetMetrics instead.
// Deprecated: Use GetMetrics() instead.
func GetGlobalMetrics() *MetricsCollector {
	return globalMetrics
}

// SetGlobalTracer sets the global tracer provider.
func SetGlobalTracer(tp *TracerProvider) {
	globalTracer = tp
}

// GetGlobalTracer returns the global tracer provider.
func GetGlobalTracer() *TracerProvider {
	return globalTracer
}
