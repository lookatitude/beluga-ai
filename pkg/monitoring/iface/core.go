// Package iface provides interfaces for the monitoring system.
// All interfaces follow the Beluga AI framework's design patterns:
// - Interface segregation principle
// - Single responsibility principle
// - Dependency inversion principle
package iface

import (
	"context"
	"time"
)

// Logger provides structured logging with context support.
type Logger interface {
	Debug(ctx context.Context, message string, fields ...map[string]any)
	Info(ctx context.Context, message string, fields ...map[string]any)
	Warning(ctx context.Context, message string, fields ...map[string]any)
	Error(ctx context.Context, message string, fields ...map[string]any)
	Fatal(ctx context.Context, message string, fields ...map[string]any)
	WithFields(fields map[string]any) ContextLogger
}

// ContextLogger provides logging with persistent context fields.
type ContextLogger interface {
	Debug(ctx context.Context, message string, fields ...map[string]any)
	Info(ctx context.Context, message string, fields ...map[string]any)
	Error(ctx context.Context, message string, fields ...map[string]any)
}

// Tracer provides distributed tracing functionality.
type Tracer interface {
	StartSpan(ctx context.Context, name string, opts ...SpanOption) (context.Context, Span)
	FinishSpan(span Span)
	GetSpan(spanID string) (Span, bool)
	GetTraceSpans(traceID string) []Span
}

// Span represents a trace span.
type Span interface {
	Log(message string, fields ...map[string]any)
	SetError(err error)
	SetStatus(status string)
	GetDuration() time.Duration
	IsFinished() bool
	SetTag(key string, value any)
}

// SpanOption represents functional options for span configuration.
type SpanOption func(Span)

// MetricsCollector provides metrics collection and reporting.
type MetricsCollector interface {
	Counter(ctx context.Context, name, description string, value float64, labels map[string]string)
	Gauge(ctx context.Context, name, description string, value float64, labels map[string]string)
	Histogram(ctx context.Context, name, description string, value float64, labels map[string]string)
	Timing(ctx context.Context, name, description string, duration time.Duration, labels map[string]string)
	Increment(ctx context.Context, name, description string, labels map[string]string)
	StartTimer(ctx context.Context, name string, labels map[string]string) Timer
}

// Timer provides convenient operation timing.
type Timer interface {
	Stop(ctx context.Context, description string)
}

// HealthChecker provides health monitoring capabilities.
type HealthChecker interface {
	RegisterCheck(name string, check HealthCheckFunc) error
	RunChecks(ctx context.Context) map[string]HealthCheckResult
	IsHealthy(ctx context.Context) bool
}

// HealthCheckFunc represents a health check function.
type HealthCheckFunc func(ctx context.Context) HealthCheckResult

// HealthCheckResult represents the result of a health check.
type HealthCheckResult struct {
	Timestamp time.Time
	Details   map[string]any
	Status    HealthStatus
	Message   string
	CheckName string
}

// HealthStatus represents health check status.
type HealthStatus string

const (
	StatusHealthy   HealthStatus = "healthy"
	StatusDegraded  HealthStatus = "degraded"
	StatusUnhealthy HealthStatus = "unhealthy"
)

// Monitor provides the main interface for comprehensive monitoring.
type Monitor interface {
	// Core monitoring components
	Logger() Logger
	Tracer() Tracer
	Metrics() MetricsCollector
	HealthChecker() HealthChecker

	// Safety and ethics
	SafetyChecker() SafetyChecker
	EthicalChecker() EthicalChecker
	BestPracticesChecker() BestPracticesChecker

	// Lifecycle management
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	IsHealthy(ctx context.Context) bool
}
