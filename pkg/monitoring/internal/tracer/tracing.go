// Package tracer provides distributed tracing implementations
package tracer

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/monitoring/iface"
)

// Use iface.Span instead of local interface

// spanImpl implements the iface.Span interface
type spanImpl struct {
	ID           string                 `json:"id"`
	TraceID      string                 `json:"trace_id"`
	ParentID     string                 `json:"parent_id,omitempty"`
	Name         string                 `json:"name"`
	Service      string                 `json:"service"`
	StartTime    time.Time              `json:"start_time"`
	EndTime      *time.Time             `json:"end_time,omitempty"`
	SpanDuration *time.Duration         `json:"duration,omitempty"`
	Tags         map[string]interface{} `json:"tags,omitempty"`
	Logs         []SpanLog              `json:"logs,omitempty"`
	Status       string                 `json:"status"`
	Error        string                 `json:"error,omitempty"`
}

// Tracer provides distributed tracing functionality
type Tracer struct {
	mu      sync.RWMutex
	spans   map[string]iface.Span
	service string
}

// SpanLog represents a log entry within a span
type SpanLog struct {
	Timestamp time.Time              `json:"timestamp"`
	Message   string                 `json:"message"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
}

// NewTracer creates a new tracer
func NewTracer(service string) *Tracer {
	return &Tracer{
		spans:   make(map[string]iface.Span),
		service: service,
	}
}

// StartSpan starts a new span
func (t *Tracer) StartSpan(ctx context.Context, name string, opts ...iface.SpanOption) (context.Context, iface.Span) {
	spanID := generateID()
	traceID := spanID

	// Extract trace ID from context if exists
	if existingTraceID, ok := ctx.Value("trace_id").(string); ok {
		traceID = existingTraceID
	}

	// Extract parent span ID from context if exists
	var parentID string
	if existingSpanID, ok := ctx.Value("span_id").(string); ok {
		parentID = existingSpanID
	}

	span := &spanImpl{
		ID:        spanID,
		TraceID:   traceID,
		ParentID:  parentID,
		Name:      name,
		Service:   t.service,
		StartTime: time.Now(),
		Tags:      make(map[string]interface{}),
		Logs:      make([]SpanLog, 0),
		Status:    "started",
	}

	// Apply options
	for _, opt := range opts {
		opt(span)
	}

	t.mu.Lock()
	t.spans[spanID] = span
	t.mu.Unlock()

	// Add span context to context
	ctx = context.WithValue(ctx, "trace_id", traceID)
	ctx = context.WithValue(ctx, "span_id", spanID)
	ctx = context.WithValue(ctx, "current_span", span)

	return ctx, span
}

// FinishSpan finishes a span
func (t *Tracer) FinishSpan(span iface.Span) {
	if spanImpl, ok := span.(*spanImpl); ok {
		now := time.Now()
		duration := now.Sub(spanImpl.StartTime)

		t.mu.Lock()
		spanImpl.EndTime = &now
		spanImpl.SpanDuration = &duration
		spanImpl.Status = "finished"
		t.mu.Unlock()
	}
}

// GetSpan retrieves a span by ID
func (t *Tracer) GetSpan(spanID string) (iface.Span, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	span, exists := t.spans[spanID]
	return span, exists
}

// GetTraceSpans retrieves all spans for a trace
func (t *Tracer) GetTraceSpans(traceID string) []iface.Span {
	t.mu.RLock()
	defer t.mu.RUnlock()

	spans := make([]iface.Span, 0)
	for _, span := range t.spans {
		if spanImpl, ok := span.(*spanImpl); ok && spanImpl.TraceID == traceID {
			spans = append(spans, span)
		}
	}
	return spans
}

// SpanOption represents functional options for span configuration
type SpanOption func(iface.Span)

// WithTag adds a tag to the span
func WithTag(key string, value interface{}) SpanOption {
	return func(s iface.Span) {
		s.SetTag(key, value)
	}
}

// WithTags adds multiple tags to the span
func WithTags(tags map[string]interface{}) SpanOption {
	return func(s iface.Span) {
		for k, v := range tags {
			s.SetTag(k, v)
		}
	}
}

// SpanFromContext extracts the current span from context
func SpanFromContext(ctx context.Context) iface.Span {
	if span, ok := ctx.Value("current_span").(*spanImpl); ok {
		return span
	}
	return nil
}

// TraceIDFromContext extracts the trace ID from context
func TraceIDFromContext(ctx context.Context) string {
	if traceID, ok := ctx.Value("trace_id").(string); ok {
		return traceID
	}
	return ""
}

// SpanIDFromContext extracts the span ID from context
func SpanIDFromContext(ctx context.Context) string {
	if spanID, ok := ctx.Value("span_id").(string); ok {
		return spanID
	}
	return ""
}

// Log adds a log entry to a span
func (s *spanImpl) Log(message string, fields ...map[string]interface{}) {
	logEntry := SpanLog{
		Timestamp: time.Now(),
		Message:   message,
	}

	if len(fields) > 0 {
		logEntry.Fields = fields[0]
	}

	s.Logs = append(s.Logs, logEntry)
}

// SetError sets an error on the span
func (s *spanImpl) SetError(err error) {
	if err != nil {
		s.Error = err.Error()
		s.Status = "error"
	} else {
		s.Error = ""
	}
}

// SetStatus sets the status of the span
func (s *spanImpl) SetStatus(status string) {
	s.Status = status
}

// SetTag sets a tag on the span
func (s *spanImpl) SetTag(key string, value interface{}) {
	if s.Tags == nil {
		s.Tags = make(map[string]interface{})
	}
	s.Tags[key] = value
}

// GetDuration returns the duration of the span
func (s *spanImpl) GetDuration() time.Duration {
	if s.SpanDuration != nil {
		return *s.SpanDuration
	}
	if s.EndTime == nil {
		return time.Since(s.StartTime)
	}
	return s.EndTime.Sub(s.StartTime)
}

// IsFinished returns true if the span is finished
func (s *spanImpl) IsFinished() bool {
	return s.EndTime != nil
}

// generateID generates a random hex ID
func generateID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// TraceFunc is a convenience function for tracing function calls
func TraceFunc(ctx context.Context, tracer *Tracer, name string, fn func(context.Context) error) error {
	ctx, span := tracer.StartSpan(ctx, name)
	defer tracer.FinishSpan(span)

	err := fn(ctx)
	if err != nil {
		span.SetError(err)
	}
	return err
}

// TraceMethod is a convenience function for tracing method calls
func TraceMethod(ctx context.Context, tracer *Tracer, methodName string, receiver interface{}, fn func() error) error {
	spanName := fmt.Sprintf("%T.%s", receiver, methodName)
	ctx, span := tracer.StartSpan(ctx, spanName)
	defer tracer.FinishSpan(span)

	err := fn()
	if err != nil {
		span.SetError(err)
	}
	return err
}
