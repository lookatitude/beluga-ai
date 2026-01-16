package messaging

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// Metrics contains all the metrics for messaging operations.
type Metrics struct {
	operationsTotal     metric.Int64Counter
	operationDuration   metric.Float64Histogram
	errorsTotal         metric.Int64Counter
	conversationsTotal  metric.Int64Counter
	messagesTotal       metric.Int64Counter
	messageDuration     metric.Float64Histogram
	participantsTotal   metric.Int64Counter
	webhooksTotal       metric.Int64Counter
	webhookDuration     metric.Float64Histogram
	activeConversations metric.Int64UpDownCounter
	activeParticipants  metric.Int64UpDownCounter
	tracer              trace.Tracer
}

// NewMetrics creates a new Metrics instance.
func NewMetrics(meter metric.Meter, tracer trace.Tracer) (*Metrics, error) {
	m := &Metrics{}

	var err error

	m.operationsTotal, err = meter.Int64Counter(
		"messaging_operations_total",
		metric.WithDescription("Total messaging operations"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create operations_total metric: %w", err)
	}

	m.operationDuration, err = meter.Float64Histogram(
		"messaging_operation_duration_seconds",
		metric.WithDescription("Messaging operation duration in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create operation_duration metric: %w", err)
	}

	m.errorsTotal, err = meter.Int64Counter(
		"messaging_errors_total",
		metric.WithDescription("Total messaging errors"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create errors_total metric: %w", err)
	}

	m.conversationsTotal, err = meter.Int64Counter(
		"messaging_conversations_total",
		metric.WithDescription("Total conversations created"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create conversations_total metric: %w", err)
	}

	m.messagesTotal, err = meter.Int64Counter(
		"messaging_messages_total",
		metric.WithDescription("Total messages sent"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create messages_total metric: %w", err)
	}

	m.messageDuration, err = meter.Float64Histogram(
		"messaging_message_duration_seconds",
		metric.WithDescription("Message send duration in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create message_duration metric: %w", err)
	}

	m.participantsTotal, err = meter.Int64Counter(
		"messaging_participants_total",
		metric.WithDescription("Total participants added"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create participants_total metric: %w", err)
	}

	m.webhooksTotal, err = meter.Int64Counter(
		"messaging_webhooks_total",
		metric.WithDescription("Total webhook events processed"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create webhooks_total metric: %w", err)
	}

	m.webhookDuration, err = meter.Float64Histogram(
		"messaging_webhook_duration_seconds",
		metric.WithDescription("Webhook processing duration in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create webhook_duration metric: %w", err)
	}

	m.activeConversations, err = meter.Int64UpDownCounter(
		"messaging_active_conversations",
		metric.WithDescription("Number of active conversations"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create active_conversations metric: %w", err)
	}

	m.activeParticipants, err = meter.Int64UpDownCounter(
		"messaging_active_participants",
		metric.WithDescription("Number of active participants"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create active_participants metric: %w", err)
	}

	m.tracer = tracer

	return m, nil
}

// NoOpMetrics returns a no-op metrics instance for testing.
func NoOpMetrics() *Metrics {
	return &Metrics{}
}

// RecordOperation records a messaging operation.
func (m *Metrics) RecordOperation(ctx context.Context, operation string, duration time.Duration, success bool) {
	if m.operationsTotal == nil {
		return
	}

	attrs := []attribute.KeyValue{
		attribute.String("operation", operation),
		attribute.Bool("success", success),
	}

	m.operationsTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
	m.operationDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))

	if !success {
		m.errorsTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
}

// RecordConversation records a conversation operation.
func (m *Metrics) RecordConversation(ctx context.Context, operation string, duration time.Duration, success bool) {
	if m.conversationsTotal == nil {
		return
	}

	attrs := []attribute.KeyValue{
		attribute.String("operation", operation),
		attribute.Bool("success", success),
	}

	m.conversationsTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
	m.operationDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))

	if !success {
		m.errorsTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
}

// RecordMessage records a message operation.
func (m *Metrics) RecordMessage(ctx context.Context, channel string, duration time.Duration, success bool) {
	if m.messagesTotal == nil {
		return
	}

	attrs := []attribute.KeyValue{
		attribute.String("channel", channel),
		attribute.Bool("success", success),
	}

	m.messagesTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
	m.messageDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))

	if !success {
		m.errorsTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
}

// RecordParticipant records a participant operation.
func (m *Metrics) RecordParticipant(ctx context.Context, operation string, success bool) {
	if m.participantsTotal == nil {
		return
	}

	attrs := []attribute.KeyValue{
		attribute.String("operation", operation),
		attribute.Bool("success", success),
	}

	m.participantsTotal.Add(ctx, 1, metric.WithAttributes(attrs...))

	if !success {
		m.errorsTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
}

// RecordWebhook records a webhook event.
func (m *Metrics) RecordWebhook(ctx context.Context, eventType string, duration time.Duration, success bool) {
	if m.webhooksTotal == nil {
		return
	}

	attrs := []attribute.KeyValue{
		attribute.String("event_type", eventType),
		attribute.Bool("success", success),
	}

	m.webhooksTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
	m.webhookDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))

	if !success {
		m.errorsTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
}

// IncrementActiveConversations increments the active conversations counter.
func (m *Metrics) IncrementActiveConversations(ctx context.Context) {
	if m.activeConversations == nil {
		return
	}
	m.activeConversations.Add(ctx, 1)
}

// DecrementActiveConversations decrements the active conversations counter.
func (m *Metrics) DecrementActiveConversations(ctx context.Context) {
	if m.activeConversations == nil {
		return
	}
	m.activeConversations.Add(ctx, -1)
}

// IncrementActiveParticipants increments the active participants counter.
func (m *Metrics) IncrementActiveParticipants(ctx context.Context) {
	if m.activeParticipants == nil {
		return
	}
	m.activeParticipants.Add(ctx, 1)
}

// DecrementActiveParticipants decrements the active participants counter.
func (m *Metrics) DecrementActiveParticipants(ctx context.Context) {
	if m.activeParticipants == nil {
		return
	}
	m.activeParticipants.Add(ctx, -1)
}

// Tracer returns the tracer instance.
func (m *Metrics) Tracer() trace.Tracer {
	return m.tracer
}
