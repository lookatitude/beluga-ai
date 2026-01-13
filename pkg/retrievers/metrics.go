// Package retrievers provides metrics collection for retriever operations.
package retrievers

import (
	"context"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// Metrics holds the metrics for retriever operations.
type Metrics struct {
	// Retrieval metrics
	retrievalRequestsTotal  metric.Int64Counter
	retrievalDuration       metric.Float64Histogram
	retrievalErrorsTotal    metric.Int64Counter
	documentsRetrievedTotal metric.Int64Counter
	retrievalScoreAvg       metric.Float64Histogram

	// Vector store metrics
	vectorStoreRequestsTotal metric.Int64Counter
	vectorStoreDuration      metric.Float64Histogram
	vectorStoreErrorsTotal   metric.Int64Counter
	documentsStoredTotal     metric.Int64Counter
	documentsDeletedTotal    metric.Int64Counter

	// Performance metrics
	batchSizeAvg       metric.Float64Histogram
	embeddingDimension metric.Int64ObservableGauge

	// Tracer for span creation
	tracer trace.Tracer
}

// NewMetrics creates a new Metrics instance with registered metrics.
func NewMetrics(meter metric.Meter, tracer trace.Tracer) (*Metrics, error) {
	m := &Metrics{}

	var err error

	// Retrieval metrics
	m.retrievalRequestsTotal, err = meter.Int64Counter(
		"retrievers_retrieval_requests_total",
		metric.WithDescription("Total number of retrieval requests"),
	)
	if err != nil {
		return nil, err
	}

	m.retrievalDuration, err = meter.Float64Histogram(
		"retrievers_retrieval_duration_seconds",
		metric.WithDescription("Duration of retrieval operations"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}

	m.retrievalErrorsTotal, err = meter.Int64Counter(
		"retrievers_retrieval_errors_total",
		metric.WithDescription("Total number of retrieval errors"),
	)
	if err != nil {
		return nil, err
	}

	m.documentsRetrievedTotal, err = meter.Int64Counter(
		"retrievers_documents_retrieved_total",
		metric.WithDescription("Total number of documents retrieved"),
	)
	if err != nil {
		return nil, err
	}

	m.retrievalScoreAvg, err = meter.Float64Histogram(
		"retrievers_retrieval_score_avg",
		metric.WithDescription("Average similarity scores of retrieved documents"),
	)
	if err != nil {
		return nil, err
	}

	// Vector store metrics
	m.vectorStoreRequestsTotal, err = meter.Int64Counter(
		"retrievers_vector_store_requests_total",
		metric.WithDescription("Total number of vector store requests"),
	)
	if err != nil {
		return nil, err
	}

	m.vectorStoreDuration, err = meter.Float64Histogram(
		"retrievers_vector_store_duration_seconds",
		metric.WithDescription("Duration of vector store operations"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}

	m.vectorStoreErrorsTotal, err = meter.Int64Counter(
		"retrievers_vector_store_errors_total",
		metric.WithDescription("Total number of vector store errors"),
	)
	if err != nil {
		return nil, err
	}

	m.documentsStoredTotal, err = meter.Int64Counter(
		"retrievers_documents_stored_total",
		metric.WithDescription("Total number of documents stored"),
	)
	if err != nil {
		return nil, err
	}

	m.documentsDeletedTotal, err = meter.Int64Counter(
		"retrievers_documents_deleted_total",
		metric.WithDescription("Total number of documents deleted"),
	)
	if err != nil {
		return nil, err
	}

	// Performance metrics
	m.batchSizeAvg, err = meter.Float64Histogram(
		"retrievers_batch_size_avg",
		metric.WithDescription("Average batch size for operations"),
	)
	if err != nil {
		return nil, err
	}

	m.embeddingDimension, err = meter.Int64ObservableGauge(
		"retrievers_embedding_dimension",
		metric.WithDescription("Embedding dimension"),
	)
	if err != nil {
		return nil, err
	}

	if tracer == nil {
		tracer = otel.Tracer("github.com/lookatitude/beluga-ai/pkg/retrievers")
	}
	m.tracer = tracer

	return m, nil
}

// RecordRetrieval records metrics for a retrieval operation.
func (m *Metrics) RecordRetrieval(ctx context.Context, retrieverType string, duration time.Duration, documentCount int, avgScore float64, err error) {
	attrs := []attribute.KeyValue{
		attribute.String("retriever_type", retrieverType),
	}

	m.retrievalRequestsTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
	m.retrievalDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))

	if err != nil {
		m.retrievalErrorsTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
	} else {
		m.documentsRetrievedTotal.Add(ctx, int64(documentCount), metric.WithAttributes(attrs...))
		if avgScore > 0 {
			m.retrievalScoreAvg.Record(ctx, avgScore, metric.WithAttributes(attrs...))
		}
	}
}

// RecordVectorStoreOperation records metrics for a vector store operation.
func (m *Metrics) RecordVectorStoreOperation(ctx context.Context, operation string, duration time.Duration, documentCount int, err error) {
	attrs := []attribute.KeyValue{
		attribute.String("operation", operation),
	}

	m.vectorStoreRequestsTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
	m.vectorStoreDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))

	if err != nil {
		m.vectorStoreErrorsTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
	} else {
		switch operation {
		case "add_documents":
			m.documentsStoredTotal.Add(ctx, int64(documentCount), metric.WithAttributes(attrs...))
		case "delete_documents":
			m.documentsDeletedTotal.Add(ctx, int64(documentCount), metric.WithAttributes(attrs...))
		}
	}
}

// RecordBatchOperation records metrics for batch operations.
func (m *Metrics) RecordBatchOperation(ctx context.Context, operation string, batchSize int, duration time.Duration) {
	attrs := []attribute.KeyValue{
		attribute.String("operation", operation),
	}

	m.batchSizeAvg.Record(ctx, float64(batchSize), metric.WithAttributes(attrs...))
}

// RecordEmbeddingDimension records the embedding dimension.
func (m *Metrics) RecordEmbeddingDimension(ctx context.Context, dimension int) {
	// This would be used with observable gauges, but for now we'll use a simple approach
	_ = ctx
	_ = dimension
	// Implementation depends on how observable gauges are set up
}

// Global metrics instance - initialized once.
var (
	globalMetrics *Metrics
	metricsOnce   sync.Once
)

// InitMetrics initializes the global metrics instance.
// This follows the standard pattern used across all Beluga AI packages.
func InitMetrics(meter metric.Meter, tracer trace.Tracer) {
	metricsOnce.Do(func() {
		if tracer == nil {
			tracer = otel.Tracer("github.com/lookatitude/beluga-ai/pkg/retrievers")
		}
		metrics, err := NewMetrics(meter, tracer)
		if err != nil {
			// If metrics creation fails, use nil (callers should check)
			globalMetrics = nil
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
