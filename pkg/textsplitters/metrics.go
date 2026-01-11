package textsplitters

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// Metrics contains all the metrics for text splitting operations.
type Metrics struct {
	operationsTotal      metric.Int64Counter
	chunksCreated        metric.Int64Counter
	documentsProcessed   metric.Int64Counter
	splitDuration        metric.Float64Histogram
	chunkSize            metric.Int64Histogram
	chunksPerDocument    metric.Int64Histogram
}

// NewMetrics creates a new Metrics instance.
func NewMetrics(meter metric.Meter) (*Metrics, error) {
	m := &Metrics{}

	var err error
	m.operationsTotal, err = meter.Int64Counter(
		"textsplitters_operations_total",
		metric.WithDescription("Total number of text splitting operations"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create operations_total counter: %w", err)
	}

	m.chunksCreated, err = meter.Int64Counter(
		"textsplitters_chunks_created",
		metric.WithDescription("Total number of chunks created"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create chunks_created counter: %w", err)
	}

	m.documentsProcessed, err = meter.Int64Counter(
		"textsplitters_documents_processed",
		metric.WithDescription("Total number of documents processed"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create documents_processed counter: %w", err)
	}

	m.splitDuration, err = meter.Float64Histogram(
		"textsplitters_split_duration_seconds",
		metric.WithDescription("Duration of text splitting operations"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create split_duration histogram: %w", err)
	}

	m.chunkSize, err = meter.Int64Histogram(
		"textsplitters_chunk_size",
		metric.WithDescription("Size of created chunks"),
		metric.WithUnit("By"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create chunk_size histogram: %w", err)
	}

	m.chunksPerDocument, err = meter.Int64Histogram(
		"textsplitters_chunks_per_document",
		metric.WithDescription("Number of chunks per document"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create chunks_per_document histogram: %w", err)
	}

	return m, nil
}

// RecordOperation records a splitting operation.
func (m *Metrics) RecordOperation(ctx context.Context, splitterType, status string, duration time.Duration) {
	attrs := attribute.NewSet(
		attribute.String("splitter_type", splitterType),
		attribute.String("status", status),
	)
	m.operationsTotal.Add(ctx, 1, metric.WithAttributeSet(attrs))
	m.splitDuration.Record(ctx, duration.Seconds(), metric.WithAttributeSet(attrs))
}

// RecordChunksCreated records the number of chunks created.
func (m *Metrics) RecordChunksCreated(ctx context.Context, splitterType string, count int64) {
	attrs := attribute.NewSet(attribute.String("splitter_type", splitterType))
	m.chunksCreated.Add(ctx, count, metric.WithAttributeSet(attrs))
}

// RecordDocumentsProcessed records the number of documents processed.
func (m *Metrics) RecordDocumentsProcessed(ctx context.Context, splitterType string, count int64) {
	attrs := attribute.NewSet(attribute.String("splitter_type", splitterType))
	m.documentsProcessed.Add(ctx, count, metric.WithAttributeSet(attrs))
}

// RecordChunkSize records the size of a created chunk.
func (m *Metrics) RecordChunkSize(ctx context.Context, splitterType string, size int64) {
	attrs := attribute.NewSet(attribute.String("splitter_type", splitterType))
	m.chunkSize.Record(ctx, size, metric.WithAttributeSet(attrs))
}

// RecordChunksPerDocument records the number of chunks per document.
func (m *Metrics) RecordChunksPerDocument(ctx context.Context, splitterType string, count int64) {
	attrs := attribute.NewSet(attribute.String("splitter_type", splitterType))
	m.chunksPerDocument.Record(ctx, count, metric.WithAttributeSet(attrs))
}
