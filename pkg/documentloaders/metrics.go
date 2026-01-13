package documentloaders

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// Metrics contains all the metrics for document loading operations.
type Metrics struct {
	operationsTotal metric.Int64Counter
	documentsLoaded metric.Int64Counter
	filesSkipped    metric.Int64Counter
	loadDuration    metric.Float64Histogram
	fileSize        metric.Int64Histogram
	tracer          trace.Tracer
}

// NewMetrics creates a new Metrics instance.
func NewMetrics(meter metric.Meter, tracer trace.Tracer) (*Metrics, error) {
	m := &Metrics{}

	var err error
	m.operationsTotal, err = meter.Int64Counter(
		"documentloaders_operations_total",
		metric.WithDescription("Total number of document loading operations"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create operations_total counter: %w", err)
	}

	m.documentsLoaded, err = meter.Int64Counter(
		"documentloaders_documents_loaded",
		metric.WithDescription("Total number of documents loaded"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create documents_loaded counter: %w", err)
	}

	m.filesSkipped, err = meter.Int64Counter(
		"documentloaders_files_skipped",
		metric.WithDescription("Total number of files skipped"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create files_skipped counter: %w", err)
	}

	m.loadDuration, err = meter.Float64Histogram(
		"documentloaders_load_duration_seconds",
		metric.WithDescription("Duration of document loading operations"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create load_duration histogram: %w", err)
	}

	m.fileSize, err = meter.Int64Histogram(
		"documentloaders_file_size_bytes",
		metric.WithDescription("Size of loaded files"),
		metric.WithUnit("By"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create file_size histogram: %w", err)
	}

	if tracer == nil {
		tracer = otel.Tracer("github.com/lookatitude/beluga-ai/pkg/documentloaders")
	}
	m.tracer = tracer

	return m, nil
}

// RecordOperation records a loading operation.
func (m *Metrics) RecordOperation(ctx context.Context, loaderType, status string, duration time.Duration) {
	attrs := attribute.NewSet(
		attribute.String("loader_type", loaderType),
		attribute.String("status", status),
	)
	m.operationsTotal.Add(ctx, 1, metric.WithAttributeSet(attrs))
	m.loadDuration.Record(ctx, duration.Seconds(), metric.WithAttributeSet(attrs))
}

// RecordDocumentsLoaded records the number of documents loaded.
func (m *Metrics) RecordDocumentsLoaded(ctx context.Context, loaderType string, count int64) {
	attrs := attribute.NewSet(attribute.String("loader_type", loaderType))
	m.documentsLoaded.Add(ctx, count, metric.WithAttributeSet(attrs))
}

// RecordFileSkipped records a skipped file.
func (m *Metrics) RecordFileSkipped(ctx context.Context, loaderType, reason string) {
	attrs := attribute.NewSet(
		attribute.String("loader_type", loaderType),
		attribute.String("reason", reason),
	)
	m.filesSkipped.Add(ctx, 1, metric.WithAttributeSet(attrs))
}

// RecordFileSize records the size of a loaded file.
func (m *Metrics) RecordFileSize(ctx context.Context, loaderType string, size int64) {
	attrs := attribute.NewSet(attribute.String("loader_type", loaderType))
	m.fileSize.Record(ctx, size, metric.WithAttributeSet(attrs))
}
