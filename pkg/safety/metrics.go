// Package safety provides OTEL metrics for safety validation.
package safety

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// Metrics holds OTEL metrics for the safety package.
type Metrics struct {
	// Counters
	checksTotal  metric.Int64Counter
	issuesTotal  metric.Int64Counter
	errorsTotal  metric.Int64Counter
	unsafeTotal  metric.Int64Counter

	// Histograms
	checkDuration metric.Float64Histogram
	riskScore     metric.Float64Histogram

	// Tracer
	tracer trace.Tracer
}

// NewMetrics creates a new Metrics instance with the given meter and tracer.
func NewMetrics(meter metric.Meter, tracer trace.Tracer) (*Metrics, error) {
	m := &Metrics{tracer: tracer}
	var err error

	// Initialize counters
	m.checksTotal, err = meter.Int64Counter(
		"safety.checks.total",
		metric.WithDescription("Total number of safety checks performed"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	m.issuesTotal, err = meter.Int64Counter(
		"safety.issues.total",
		metric.WithDescription("Total number of safety issues detected"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	m.errorsTotal, err = meter.Int64Counter(
		"safety.errors.total",
		metric.WithDescription("Total number of safety check errors"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	m.unsafeTotal, err = meter.Int64Counter(
		"safety.unsafe.total",
		metric.WithDescription("Total number of unsafe content detections"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	// Initialize histograms
	m.checkDuration, err = meter.Float64Histogram(
		"safety.check.duration",
		metric.WithDescription("Duration of safety checks in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}

	m.riskScore, err = meter.Float64Histogram(
		"safety.risk.score",
		metric.WithDescription("Distribution of risk scores"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	return m, nil
}

// NoOpMetrics returns a Metrics instance that does nothing.
// Useful for testing or when metrics are disabled.
func NoOpMetrics() *Metrics {
	return &Metrics{}
}

// RecordCheck records a safety check operation.
func (m *Metrics) RecordCheck(ctx context.Context, duration time.Duration, riskScore float64, safe bool, issueCount int) {
	if m == nil {
		return
	}

	attrs := []attribute.KeyValue{
		attribute.Bool("safe", safe),
	}

	if m.checksTotal != nil {
		m.checksTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
	}

	if m.checkDuration != nil {
		m.checkDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))
	}

	if m.riskScore != nil {
		m.riskScore.Record(ctx, riskScore, metric.WithAttributes(attrs...))
	}

	if !safe && m.unsafeTotal != nil {
		m.unsafeTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
}

// RecordIssue records a detected safety issue.
func (m *Metrics) RecordIssue(ctx context.Context, issueType, severity string) {
	if m == nil || m.issuesTotal == nil {
		return
	}

	attrs := []attribute.KeyValue{
		attribute.String("issue_type", issueType),
		attribute.String("severity", severity),
	}

	m.issuesTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
}

// RecordError records a safety check error.
func (m *Metrics) RecordError(ctx context.Context, op string, err error) {
	if m == nil || m.errorsTotal == nil {
		return
	}

	attrs := []attribute.KeyValue{
		attribute.String("operation", op),
		attribute.String("error_type", errorType(err)),
	}

	m.errorsTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
}

// StartSpan starts a new trace span for a safety operation.
func (m *Metrics) StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	if m == nil || m.tracer == nil {
		// Use OTEL's noop tracer for a proper noop span
		return otel.Tracer("safety.noop").Start(ctx, name, opts...)
	}
	return m.tracer.Start(ctx, name, opts...)
}

// GetTracer returns the tracer instance.
func (m *Metrics) GetTracer() trace.Tracer {
	if m == nil || m.tracer == nil {
		return otel.Tracer("safety")
	}
	return m.tracer
}

// errorType extracts a type string from an error.
func errorType(err error) string {
	if err == nil {
		return "none"
	}
	switch err {
	case ErrUnsafe, ErrUnsafeContent:
		return "unsafe_content"
	case ErrSafetyCheckFailed:
		return "check_failed"
	case ErrHighRiskContent:
		return "high_risk"
	default:
		return "unknown"
	}
}

