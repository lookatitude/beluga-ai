package rag

import (
	"context"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

// Global metrics instance - initialized once.
var (
	globalMetrics *Metrics
	metricsOnce   sync.Once
)

// Metrics holds the metrics for the convenience RAG package.
type Metrics struct {
	// Pipeline metrics
	pipelineBuilds      metric.Int64Counter
	pipelineBuildErrors metric.Int64Counter

	// Query metrics
	queries        metric.Int64Counter
	queryErrors    metric.Int64Counter
	queryDuration  metric.Float64Histogram
	documentsFound metric.Int64Counter

	// Ingestion metrics
	documentsIngested metric.Int64Counter
	ingestionErrors   metric.Int64Counter
	ingestionDuration metric.Float64Histogram

	// Search metrics
	searches       metric.Int64Counter
	searchDuration metric.Float64Histogram

	// Tracer
	tracer trace.Tracer
}

// NewMetrics creates a new Metrics instance with OpenTelemetry metrics.
func NewMetrics(meter metric.Meter, tracer trace.Tracer) *Metrics {
	m := &Metrics{tracer: tracer}

	var err error

	// Pipeline build metrics
	m.pipelineBuilds, err = meter.Int64Counter(
		"convenience_rag_builds_total",
		metric.WithDescription("Total number of RAG pipelines built"),
	)
	if err != nil {
		m.pipelineBuilds = nil
	}

	m.pipelineBuildErrors, err = meter.Int64Counter(
		"convenience_rag_build_errors_total",
		metric.WithDescription("Total number of RAG pipeline build errors"),
	)
	if err != nil {
		m.pipelineBuildErrors = nil
	}

	// Query metrics
	m.queries, err = meter.Int64Counter(
		"convenience_rag_queries_total",
		metric.WithDescription("Total number of RAG queries"),
	)
	if err != nil {
		m.queries = nil
	}

	m.queryErrors, err = meter.Int64Counter(
		"convenience_rag_query_errors_total",
		metric.WithDescription("Total number of RAG query errors"),
	)
	if err != nil {
		m.queryErrors = nil
	}

	m.queryDuration, err = meter.Float64Histogram(
		"convenience_rag_query_duration_seconds",
		metric.WithDescription("Duration of RAG queries"),
		metric.WithUnit("s"),
	)
	if err != nil {
		m.queryDuration = nil
	}

	m.documentsFound, err = meter.Int64Counter(
		"convenience_rag_documents_found_total",
		metric.WithDescription("Total number of documents found in queries"),
	)
	if err != nil {
		m.documentsFound = nil
	}

	// Ingestion metrics
	m.documentsIngested, err = meter.Int64Counter(
		"convenience_rag_documents_ingested_total",
		metric.WithDescription("Total number of documents ingested"),
	)
	if err != nil {
		m.documentsIngested = nil
	}

	m.ingestionErrors, err = meter.Int64Counter(
		"convenience_rag_ingestion_errors_total",
		metric.WithDescription("Total number of ingestion errors"),
	)
	if err != nil {
		m.ingestionErrors = nil
	}

	m.ingestionDuration, err = meter.Float64Histogram(
		"convenience_rag_ingestion_duration_seconds",
		metric.WithDescription("Duration of document ingestion"),
		metric.WithUnit("s"),
	)
	if err != nil {
		m.ingestionDuration = nil
	}

	// Search metrics
	m.searches, err = meter.Int64Counter(
		"convenience_rag_searches_total",
		metric.WithDescription("Total number of similarity searches"),
	)
	if err != nil {
		m.searches = nil
	}

	m.searchDuration, err = meter.Float64Histogram(
		"convenience_rag_search_duration_seconds",
		metric.WithDescription("Duration of similarity searches"),
		metric.WithUnit("s"),
	)
	if err != nil {
		m.searchDuration = nil
	}

	return m
}

// RecordBuild records a build operation.
func (m *Metrics) RecordBuild(ctx context.Context, success bool) {
	if m == nil {
		return
	}
	attrs := metric.WithAttributes(
		attribute.Bool("success", success),
	)

	if m.pipelineBuilds != nil {
		m.pipelineBuilds.Add(ctx, 1, attrs)
	}
	if !success && m.pipelineBuildErrors != nil {
		m.pipelineBuildErrors.Add(ctx, 1, attrs)
	}
}

// RecordQuery records a query operation.
func (m *Metrics) RecordQuery(ctx context.Context, duration time.Duration, documentsFound int, success bool) {
	if m == nil {
		return
	}
	attrs := metric.WithAttributes(
		attribute.Bool("success", success),
	)

	if m.queries != nil {
		m.queries.Add(ctx, 1, attrs)
	}
	if m.queryDuration != nil {
		m.queryDuration.Record(ctx, duration.Seconds(), attrs)
	}
	if success && m.documentsFound != nil {
		m.documentsFound.Add(ctx, int64(documentsFound), attrs)
	}
	if !success && m.queryErrors != nil {
		m.queryErrors.Add(ctx, 1, attrs)
	}
}

// RecordIngestion records an ingestion operation.
func (m *Metrics) RecordIngestion(ctx context.Context, duration time.Duration, documentCount int, success bool) {
	if m == nil {
		return
	}
	attrs := metric.WithAttributes(
		attribute.Bool("success", success),
	)

	if success && m.documentsIngested != nil {
		m.documentsIngested.Add(ctx, int64(documentCount), attrs)
	}
	if m.ingestionDuration != nil {
		m.ingestionDuration.Record(ctx, duration.Seconds(), attrs)
	}
	if !success && m.ingestionErrors != nil {
		m.ingestionErrors.Add(ctx, 1, attrs)
	}
}

// RecordSearch records a search operation.
func (m *Metrics) RecordSearch(ctx context.Context, duration time.Duration, success bool) {
	if m == nil {
		return
	}
	attrs := metric.WithAttributes(
		attribute.Bool("success", success),
	)

	if m.searches != nil {
		m.searches.Add(ctx, 1, attrs)
	}
	if m.searchDuration != nil {
		m.searchDuration.Record(ctx, duration.Seconds(), attrs)
	}
}

// StartBuildSpan starts a tracing span for build operations.
func (m *Metrics) StartBuildSpan(ctx context.Context) (context.Context, trace.Span) {
	if m == nil || m.tracer == nil {
		return ctx, nil
	}
	return m.tracer.Start(ctx, "convenience.rag.build")
}

// StartQuerySpan starts a tracing span for query operations.
func (m *Metrics) StartQuerySpan(ctx context.Context, operation string) (context.Context, trace.Span) {
	if m == nil || m.tracer == nil {
		return ctx, nil
	}
	return m.tracer.Start(ctx, "convenience.rag."+operation)
}

// InitMetrics initializes the global metrics instance.
func InitMetrics(meter metric.Meter, tracer trace.Tracer) {
	metricsOnce.Do(func() {
		if tracer == nil {
			tracer = otel.Tracer("beluga-convenience-rag")
		}
		globalMetrics = NewMetrics(meter, tracer)
	})
}

// GetMetrics returns the global metrics instance.
func GetMetrics() *Metrics {
	return globalMetrics
}

// DefaultMetrics creates a metrics instance with default meter and tracer.
func DefaultMetrics() *Metrics {
	meter := otel.Meter("beluga-convenience-rag")
	tracer := otel.Tracer("beluga-convenience-rag")
	return NewMetrics(meter, tracer)
}

// NoOpMetrics returns a metrics instance that does nothing.
func NoOpMetrics() *Metrics {
	return &Metrics{
		tracer: noop.NewTracerProvider().Tracer("noop"),
	}
}
