package voiceagent

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

// Metrics holds the metrics for the convenience voice agent package.
type Metrics struct {
	// Build metrics
	agentBuilds      metric.Int64Counter
	agentBuildErrors metric.Int64Counter

	// Session metrics
	sessionsStarted metric.Int64Counter
	sessionsStopped metric.Int64Counter
	sessionDuration metric.Float64Histogram

	// Transcription metrics
	transcriptions       metric.Int64Counter
	transcriptionErrors  metric.Int64Counter
	transcriptionLatency metric.Float64Histogram

	// Synthesis metrics
	syntheses        metric.Int64Counter
	synthesisErrors  metric.Int64Counter
	synthesisLatency metric.Float64Histogram

	// Audio processing metrics
	audioChunksProcessed metric.Int64Counter

	// Tracer
	tracer trace.Tracer
}

// NewMetrics creates a new Metrics instance with OpenTelemetry metrics.
func NewMetrics(meter metric.Meter, tracer trace.Tracer) *Metrics {
	m := &Metrics{tracer: tracer}

	var err error

	// Build metrics
	m.agentBuilds, err = meter.Int64Counter(
		"convenience_voiceagent_builds_total",
		metric.WithDescription("Total number of voice agents built"),
	)
	if err != nil {
		m.agentBuilds = nil
	}

	m.agentBuildErrors, err = meter.Int64Counter(
		"convenience_voiceagent_build_errors_total",
		metric.WithDescription("Total number of voice agent build errors"),
	)
	if err != nil {
		m.agentBuildErrors = nil
	}

	// Session metrics
	m.sessionsStarted, err = meter.Int64Counter(
		"convenience_voiceagent_sessions_started_total",
		metric.WithDescription("Total number of voice sessions started"),
	)
	if err != nil {
		m.sessionsStarted = nil
	}

	m.sessionsStopped, err = meter.Int64Counter(
		"convenience_voiceagent_sessions_stopped_total",
		metric.WithDescription("Total number of voice sessions stopped"),
	)
	if err != nil {
		m.sessionsStopped = nil
	}

	m.sessionDuration, err = meter.Float64Histogram(
		"convenience_voiceagent_session_duration_seconds",
		metric.WithDescription("Duration of voice sessions"),
		metric.WithUnit("s"),
	)
	if err != nil {
		m.sessionDuration = nil
	}

	// Transcription metrics
	m.transcriptions, err = meter.Int64Counter(
		"convenience_voiceagent_transcriptions_total",
		metric.WithDescription("Total number of transcriptions"),
	)
	if err != nil {
		m.transcriptions = nil
	}

	m.transcriptionErrors, err = meter.Int64Counter(
		"convenience_voiceagent_transcription_errors_total",
		metric.WithDescription("Total number of transcription errors"),
	)
	if err != nil {
		m.transcriptionErrors = nil
	}

	m.transcriptionLatency, err = meter.Float64Histogram(
		"convenience_voiceagent_transcription_latency_seconds",
		metric.WithDescription("Latency of transcriptions"),
		metric.WithUnit("s"),
	)
	if err != nil {
		m.transcriptionLatency = nil
	}

	// Synthesis metrics
	m.syntheses, err = meter.Int64Counter(
		"convenience_voiceagent_syntheses_total",
		metric.WithDescription("Total number of speech syntheses"),
	)
	if err != nil {
		m.syntheses = nil
	}

	m.synthesisErrors, err = meter.Int64Counter(
		"convenience_voiceagent_synthesis_errors_total",
		metric.WithDescription("Total number of synthesis errors"),
	)
	if err != nil {
		m.synthesisErrors = nil
	}

	m.synthesisLatency, err = meter.Float64Histogram(
		"convenience_voiceagent_synthesis_latency_seconds",
		metric.WithDescription("Latency of speech synthesis"),
		metric.WithUnit("s"),
	)
	if err != nil {
		m.synthesisLatency = nil
	}

	// Audio processing metrics
	m.audioChunksProcessed, err = meter.Int64Counter(
		"convenience_voiceagent_audio_chunks_processed_total",
		metric.WithDescription("Total number of audio chunks processed"),
	)
	if err != nil {
		m.audioChunksProcessed = nil
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

	if m.agentBuilds != nil {
		m.agentBuilds.Add(ctx, 1, attrs)
	}
	if !success && m.agentBuildErrors != nil {
		m.agentBuildErrors.Add(ctx, 1, attrs)
	}
}

// RecordSessionStart records a session start.
func (m *Metrics) RecordSessionStart(ctx context.Context, sessionID string) {
	if m == nil || m.sessionsStarted == nil {
		return
	}
	attrs := metric.WithAttributes(
		attribute.String("session_id", sessionID),
	)
	m.sessionsStarted.Add(ctx, 1, attrs)
}

// RecordSessionStop records a session stop.
func (m *Metrics) RecordSessionStop(ctx context.Context, sessionID string, duration time.Duration) {
	if m == nil {
		return
	}
	attrs := metric.WithAttributes(
		attribute.String("session_id", sessionID),
	)

	if m.sessionsStopped != nil {
		m.sessionsStopped.Add(ctx, 1, attrs)
	}
	if m.sessionDuration != nil {
		m.sessionDuration.Record(ctx, duration.Seconds(), attrs)
	}
}

// RecordTranscription records a transcription operation.
func (m *Metrics) RecordTranscription(ctx context.Context, latency time.Duration, success bool) {
	if m == nil {
		return
	}
	attrs := metric.WithAttributes(
		attribute.Bool("success", success),
	)

	if m.transcriptions != nil {
		m.transcriptions.Add(ctx, 1, attrs)
	}
	if m.transcriptionLatency != nil {
		m.transcriptionLatency.Record(ctx, latency.Seconds(), attrs)
	}
	if !success && m.transcriptionErrors != nil {
		m.transcriptionErrors.Add(ctx, 1, attrs)
	}
}

// RecordSynthesis records a synthesis operation.
func (m *Metrics) RecordSynthesis(ctx context.Context, latency time.Duration, success bool) {
	if m == nil {
		return
	}
	attrs := metric.WithAttributes(
		attribute.Bool("success", success),
	)

	if m.syntheses != nil {
		m.syntheses.Add(ctx, 1, attrs)
	}
	if m.synthesisLatency != nil {
		m.synthesisLatency.Record(ctx, latency.Seconds(), attrs)
	}
	if !success && m.synthesisErrors != nil {
		m.synthesisErrors.Add(ctx, 1, attrs)
	}
}

// RecordAudioChunk records processing of an audio chunk.
func (m *Metrics) RecordAudioChunk(ctx context.Context) {
	if m == nil || m.audioChunksProcessed == nil {
		return
	}
	m.audioChunksProcessed.Add(ctx, 1)
}

// StartBuildSpan starts a tracing span for build operations.
func (m *Metrics) StartBuildSpan(ctx context.Context) (context.Context, trace.Span) {
	if m == nil || m.tracer == nil {
		return ctx, nil
	}
	return m.tracer.Start(ctx, "convenience.voiceagent.build")
}

// StartSessionSpan starts a tracing span for session operations.
func (m *Metrics) StartSessionSpan(ctx context.Context, sessionID, operation string) (context.Context, trace.Span) {
	if m == nil || m.tracer == nil {
		return ctx, nil
	}
	return m.tracer.Start(ctx, "convenience.voiceagent."+operation,
		trace.WithAttributes(
			attribute.String("session.id", sessionID),
		),
	)
}

// InitMetrics initializes the global metrics instance.
func InitMetrics(meter metric.Meter, tracer trace.Tracer) {
	metricsOnce.Do(func() {
		if tracer == nil {
			tracer = otel.Tracer("beluga-convenience-voiceagent")
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
	meter := otel.Meter("beluga-convenience-voiceagent")
	tracer := otel.Tracer("beluga-convenience-voiceagent")
	return NewMetrics(meter, tracer)
}

// NoOpMetrics returns a metrics instance that does nothing.
func NoOpMetrics() *Metrics {
	return &Metrics{
		tracer: noop.NewTracerProvider().Tracer("noop"),
	}
}
