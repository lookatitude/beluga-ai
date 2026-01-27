package twilio

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// Metrics contains all the metrics for Twilio voice operations.
type Metrics struct {
	operationsTotal     metric.Int64Counter
	operationDuration   metric.Float64Histogram
	errorsTotal         metric.Int64Counter
	callsTotal          metric.Int64Counter
	callDuration        metric.Float64Histogram
	streamsTotal        metric.Int64Counter
	streamDuration      metric.Float64Histogram
	webhooksTotal       metric.Int64Counter
	webhookDuration     metric.Float64Histogram
	transcriptionsTotal metric.Int64Counter
	activeCalls         metric.Int64UpDownCounter
	activeStreams       metric.Int64UpDownCounter
	callQuality         metric.Float64Histogram // Call quality metrics (latency, jitter, etc.)
	tracer              trace.Tracer
}

// NewMetrics creates a new Metrics instance.
func NewMetrics(meter metric.Meter, tracer trace.Tracer) (*Metrics, error) {
	m := &Metrics{}

	var err error

	m.operationsTotal, err = meter.Int64Counter(
		"twilio_voice_operations_total",
		metric.WithDescription("Total Twilio voice operations"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create operations_total metric: %w", err)
	}

	m.operationDuration, err = meter.Float64Histogram(
		"twilio_voice_operation_duration_seconds",
		metric.WithDescription("Twilio voice operation duration in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create operation_duration metric: %w", err)
	}

	m.errorsTotal, err = meter.Int64Counter(
		"twilio_voice_errors_total",
		metric.WithDescription("Total Twilio voice errors"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create errors_total metric: %w", err)
	}

	m.callsTotal, err = meter.Int64Counter(
		"twilio_voice_calls_total",
		metric.WithDescription("Total Twilio calls"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create calls_total metric: %w", err)
	}

	m.callDuration, err = meter.Float64Histogram(
		"twilio_voice_call_duration_seconds",
		metric.WithDescription("Twilio call duration in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create call_duration metric: %w", err)
	}

	m.streamsTotal, err = meter.Int64Counter(
		"twilio_voice_streams_total",
		metric.WithDescription("Total Twilio media streams"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create streams_total metric: %w", err)
	}

	m.streamDuration, err = meter.Float64Histogram(
		"twilio_voice_stream_duration_seconds",
		metric.WithDescription("Twilio stream duration in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create stream_duration metric: %w", err)
	}

	m.webhooksTotal, err = meter.Int64Counter(
		"twilio_voice_webhooks_total",
		metric.WithDescription("Total Twilio webhook events processed"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create webhooks_total metric: %w", err)
	}

	m.webhookDuration, err = meter.Float64Histogram(
		"twilio_voice_webhook_duration_seconds",
		metric.WithDescription("Twilio webhook processing duration in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create webhook_duration metric: %w", err)
	}

	m.transcriptionsTotal, err = meter.Int64Counter(
		"twilio_voice_transcriptions_total",
		metric.WithDescription("Total Twilio transcriptions"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create transcriptions_total metric: %w", err)
	}

	m.activeCalls, err = meter.Int64UpDownCounter(
		"twilio_voice_active_calls",
		metric.WithDescription("Number of active Twilio calls"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create active_calls metric: %w", err)
	}

	m.activeStreams, err = meter.Int64UpDownCounter(
		"twilio_voice_active_streams",
		metric.WithDescription("Number of active Twilio streams"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create active_streams metric: %w", err)
	}

	m.callQuality, err = meter.Float64Histogram(
		"twilio_voice_call_quality",
		metric.WithDescription("Twilio call quality metrics (latency, jitter, packet loss)"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create call_quality metric: %w", err)
	}

	m.tracer = tracer

	return m, nil
}

// NoOpMetrics returns a no-op metrics instance for testing.
func NoOpMetrics() *Metrics {
	return &Metrics{}
}

// RecordOperation records a Twilio voice operation.
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

// RecordCall records a Twilio call operation.
func (m *Metrics) RecordCall(ctx context.Context, callSID string, duration time.Duration, success bool) {
	if m.callsTotal == nil {
		return
	}

	attrs := []attribute.KeyValue{
		attribute.String("call_sid", callSID),
		attribute.Bool("success", success),
	}

	m.callsTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
	m.callDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))

	if !success {
		m.errorsTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
}

// RecordStream records a Twilio stream operation.
func (m *Metrics) RecordStream(ctx context.Context, streamSID string, duration time.Duration, success bool) {
	if m.streamsTotal == nil {
		return
	}

	attrs := []attribute.KeyValue{
		attribute.String("stream_sid", streamSID),
		attribute.Bool("success", success),
	}

	m.streamsTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
	m.streamDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))

	if !success {
		m.errorsTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
}

// RecordWebhook records a Twilio webhook event.
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

// RecordTranscription records a Twilio transcription.
func (m *Metrics) RecordTranscription(ctx context.Context, transcriptionSID string, success bool) {
	if m.transcriptionsTotal == nil {
		return
	}

	attrs := []attribute.KeyValue{
		attribute.String("transcription_sid", transcriptionSID),
		attribute.Bool("success", success),
	}

	m.transcriptionsTotal.Add(ctx, 1, metric.WithAttributes(attrs...))

	if !success {
		m.errorsTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
}

// RecordCallQuality records call quality metrics (latency, jitter, packet loss).
func (m *Metrics) RecordCallQuality(ctx context.Context, callSID string, latency, jitter, packetLoss float64) {
	if m.callQuality == nil {
		return
	}

	attrs := []attribute.KeyValue{
		attribute.String("call_sid", callSID),
		attribute.String("metric_type", "latency"),
	}

	m.callQuality.Record(ctx, latency, metric.WithAttributes(attrs...))

	attrs[1] = attribute.String("metric_type", "jitter")
	m.callQuality.Record(ctx, jitter, metric.WithAttributes(attrs...))

	attrs[1] = attribute.String("metric_type", "packet_loss")
	m.callQuality.Record(ctx, packetLoss, metric.WithAttributes(attrs...))
}

// IncrementActiveCalls increments the active calls counter.
func (m *Metrics) IncrementActiveCalls(ctx context.Context) {
	if m.activeCalls == nil {
		return
	}
	m.activeCalls.Add(ctx, 1)
}

// DecrementActiveCalls decrements the active calls counter.
func (m *Metrics) DecrementActiveCalls(ctx context.Context) {
	if m.activeCalls == nil {
		return
	}
	m.activeCalls.Add(ctx, -1)
}

// IncrementActiveStreams increments the active streams counter.
func (m *Metrics) IncrementActiveStreams(ctx context.Context) {
	if m.activeStreams == nil {
		return
	}
	m.activeStreams.Add(ctx, 1)
}

// DecrementActiveStreams decrements the active streams counter.
func (m *Metrics) DecrementActiveStreams(ctx context.Context) {
	if m.activeStreams == nil {
		return
	}
	m.activeStreams.Add(ctx, -1)
}

// Tracer returns the tracer instance.
func (m *Metrics) Tracer() trace.Tracer {
	return m.tracer
}
