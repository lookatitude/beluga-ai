package prompts

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// Metrics provides metrics collection for the prompts package
type Metrics struct {
	// Template metrics
	templatesCreated  metric.Int64Counter
	templatesExecuted metric.Int64Counter
	templateErrors    metric.Int64Counter
	templateDuration  metric.Float64Histogram

	// Formatting metrics
	formattingRequests metric.Int64Counter
	formattingErrors   metric.Int64Counter
	formattingDuration metric.Float64Histogram

	// Variable validation metrics
	validationRequests metric.Int64Counter
	validationErrors   metric.Int64Counter

	// Cache metrics
	cacheHits   metric.Int64Counter
	cacheMisses metric.Int64Counter
	cacheSize   metric.Int64UpDownCounter

	// Adapter metrics
	adapterRequests metric.Int64Counter
	adapterErrors   metric.Int64Counter

	// Tracer for span creation
	tracer trace.Tracer
}

// NewMetrics creates a new metrics collector with OTEL instrumentation
func NewMetrics(meter metric.Meter, tracer trace.Tracer) (*Metrics, error) {
	m := &Metrics{tracer: tracer}

	var err error

	// Initialize template metrics
	m.templatesCreated, err = meter.Int64Counter(
		"prompts_templates_created_total",
		metric.WithDescription("Total number of templates created"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	m.templatesExecuted, err = meter.Int64Counter(
		"prompts_templates_executed_total",
		metric.WithDescription("Total number of templates executed"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	m.templateErrors, err = meter.Int64Counter(
		"prompts_template_errors_total",
		metric.WithDescription("Total number of template execution errors"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	m.templateDuration, err = meter.Float64Histogram(
		"prompts_template_duration_seconds",
		metric.WithDescription("Duration of template operations"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}

	// Initialize formatting metrics
	m.formattingRequests, err = meter.Int64Counter(
		"prompts_formatting_requests_total",
		metric.WithDescription("Total number of formatting requests"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	m.formattingErrors, err = meter.Int64Counter(
		"prompts_formatting_errors_total",
		metric.WithDescription("Total number of formatting errors"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	m.formattingDuration, err = meter.Float64Histogram(
		"prompts_formatting_duration_seconds",
		metric.WithDescription("Duration of formatting operations"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}

	// Initialize validation metrics
	m.validationRequests, err = meter.Int64Counter(
		"prompts_validation_requests_total",
		metric.WithDescription("Total number of validation requests"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	m.validationErrors, err = meter.Int64Counter(
		"prompts_validation_errors_total",
		metric.WithDescription("Total number of validation errors"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	// Initialize cache metrics
	m.cacheHits, err = meter.Int64Counter(
		"prompts_cache_hits_total",
		metric.WithDescription("Total number of cache hits"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	m.cacheMisses, err = meter.Int64Counter(
		"prompts_cache_misses_total",
		metric.WithDescription("Total number of cache misses"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	m.cacheSize, err = meter.Int64UpDownCounter(
		"prompts_cache_size",
		metric.WithDescription("Current cache size"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	// Initialize adapter metrics
	m.adapterRequests, err = meter.Int64Counter(
		"prompts_adapter_requests_total",
		metric.WithDescription("Total number of adapter requests"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	m.adapterErrors, err = meter.Int64Counter(
		"prompts_adapter_errors_total",
		metric.WithDescription("Total number of adapter errors"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	return m, nil
}

// RecordTemplateCreated records a template creation
func (m *Metrics) RecordTemplateCreated(ctx context.Context, templateType string) {
	if m == nil || m.templatesCreated == nil {
		return
	}

	attrs := []attribute.KeyValue{
		attribute.String("template_type", templateType),
	}

	m.templatesCreated.Add(ctx, 1, metric.WithAttributes(attrs...))
}

// RecordTemplateExecuted records a template execution
func (m *Metrics) RecordTemplateExecuted(ctx context.Context, templateName string, duration time.Duration, success bool) {
	if m == nil {
		return
	}

	attrs := []attribute.KeyValue{
		attribute.String("template_name", templateName),
		attribute.Bool("success", success),
	}

	if m.templatesExecuted != nil {
		m.templatesExecuted.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
	if m.templateDuration != nil {
		m.templateDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))
	}

	if !success && m.templateErrors != nil {
		m.templateErrors.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
}

// RecordTemplateError records a template execution error
func (m *Metrics) RecordTemplateError(ctx context.Context, templateName string, errorType string) {
	if m == nil || m.templateErrors == nil {
		return
	}

	attrs := []attribute.KeyValue{
		attribute.String("template_name", templateName),
		attribute.String("error_type", errorType),
	}

	m.templateErrors.Add(ctx, 1, metric.WithAttributes(attrs...))
}

// RecordFormattingRequest records a formatting request
func (m *Metrics) RecordFormattingRequest(ctx context.Context, adapterType string, duration time.Duration, success bool) {
	if m == nil {
		return
	}

	attrs := []attribute.KeyValue{
		attribute.String("adapter_type", adapterType),
		attribute.Bool("success", success),
	}

	if m.formattingRequests != nil {
		m.formattingRequests.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
	if m.formattingDuration != nil {
		m.formattingDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))
	}

	if !success && m.formattingErrors != nil {
		m.formattingErrors.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
}

// RecordFormattingError records a formatting error
func (m *Metrics) RecordFormattingError(ctx context.Context, adapterType string, errorType string) {
	if m == nil || m.formattingErrors == nil {
		return
	}

	attrs := []attribute.KeyValue{
		attribute.String("adapter_type", adapterType),
		attribute.String("error_type", errorType),
	}

	m.formattingErrors.Add(ctx, 1, metric.WithAttributes(attrs...))
}

// RecordValidationRequest records a validation request
func (m *Metrics) RecordValidationRequest(ctx context.Context, validationType string, success bool) {
	if m == nil {
		return
	}

	attrs := []attribute.KeyValue{
		attribute.String("validation_type", validationType),
		attribute.Bool("success", success),
	}

	if m.validationRequests != nil {
		m.validationRequests.Add(ctx, 1, metric.WithAttributes(attrs...))
	}

	if !success && m.validationErrors != nil {
		m.validationErrors.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
}

// RecordValidationError records a validation error
func (m *Metrics) RecordValidationError(ctx context.Context, errorType string) {
	if m == nil || m.validationErrors == nil {
		return
	}

	attrs := []attribute.KeyValue{
		attribute.String("error_type", errorType),
	}

	m.validationErrors.Add(ctx, 1, metric.WithAttributes(attrs...))
}

// RecordCacheHit records a cache hit
func (m *Metrics) RecordCacheHit(ctx context.Context, cacheType string) {
	if m == nil || m.cacheHits == nil {
		return
	}

	attrs := []attribute.KeyValue{
		attribute.String("cache_type", cacheType),
	}

	m.cacheHits.Add(ctx, 1, metric.WithAttributes(attrs...))
}

// RecordCacheMiss records a cache miss
func (m *Metrics) RecordCacheMiss(ctx context.Context, cacheType string) {
	if m == nil || m.cacheMisses == nil {
		return
	}

	attrs := []attribute.KeyValue{
		attribute.String("cache_type", cacheType),
	}

	m.cacheMisses.Add(ctx, 1, metric.WithAttributes(attrs...))
}

// RecordCacheSize records the current cache size
func (m *Metrics) RecordCacheSize(ctx context.Context, size int64, cacheType string) {
	if m == nil || m.cacheSize == nil {
		return
	}

	attrs := []attribute.KeyValue{
		attribute.String("cache_type", cacheType),
	}

	m.cacheSize.Add(ctx, size, metric.WithAttributes(attrs...))
}

// RecordAdapterRequest records an adapter request
func (m *Metrics) RecordAdapterRequest(ctx context.Context, adapterType string, success bool) {
	if m == nil {
		return
	}

	attrs := []attribute.KeyValue{
		attribute.String("adapter_type", adapterType),
		attribute.Bool("success", success),
	}

	if m.adapterRequests != nil {
		m.adapterRequests.Add(ctx, 1, metric.WithAttributes(attrs...))
	}

	if !success && m.adapterErrors != nil {
		m.adapterErrors.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
}

// RecordAdapterError records an adapter error
func (m *Metrics) RecordAdapterError(ctx context.Context, adapterType string, errorType string) {
	if m == nil || m.adapterErrors == nil {
		return
	}

	attrs := []attribute.KeyValue{
		attribute.String("adapter_type", adapterType),
		attribute.String("error_type", errorType),
	}

	m.adapterErrors.Add(ctx, 1, metric.WithAttributes(attrs...))
}

// StartTemplateSpan starts a new span for template operations
func (m *Metrics) StartTemplateSpan(ctx context.Context, templateName, operation string) (context.Context, trace.Span) {
	if m.tracer == nil {
		return ctx, trace.SpanFromContext(ctx)
	}

	return m.tracer.Start(ctx, "prompts.template."+operation,
		trace.WithAttributes(
			attribute.String("template.name", templateName),
		),
	)
}

// StartFormattingSpan starts a new span for formatting operations
func (m *Metrics) StartFormattingSpan(ctx context.Context, adapterType, operation string) (context.Context, trace.Span) {
	if m.tracer == nil {
		return ctx, trace.SpanFromContext(ctx)
	}

	return m.tracer.Start(ctx, "prompts.formatting."+operation,
		trace.WithAttributes(
			attribute.String("adapter.type", adapterType),
		),
	)
}

// NoOpMetrics returns a metrics instance that does nothing.
// Useful for testing or when metrics are disabled.
func NoOpMetrics() *Metrics {
	return &Metrics{}
}
