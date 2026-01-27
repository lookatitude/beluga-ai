// Package voiceutils provides OTEL metrics for voice processing packages.
package voiceutils

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// Metrics holds the metrics for the voiceutils package.
type Metrics struct {
	// Audio processing metrics
	audioProcessed     metric.Int64Counter
	audioDuration      metric.Float64Histogram
	audioBuffersInUse  metric.Int64UpDownCounter
	audioSamplesTotal  metric.Int64Counter
	audioConversions   metric.Int64Counter
	conversionDuration metric.Float64Histogram

	// Buffer pool metrics
	bufferPoolHits   metric.Int64Counter
	bufferPoolMisses metric.Int64Counter
	bufferPoolSize   metric.Int64UpDownCounter

	// Retry and resilience metrics
	retryAttempts metric.Int64Counter
	circuitState  metric.Int64Counter
	rateLimitHits metric.Int64Counter

	// Error metrics
	errorsTotal metric.Int64Counter

	// Tracer
	tracer trace.Tracer
}

// NewMetrics creates a new metrics instance.
func NewMetrics(meter metric.Meter, tracer trace.Tracer) (*Metrics, error) {
	var err error
	m := &Metrics{}

	// Audio processing metrics
	m.audioProcessed, err = meter.Int64Counter(
		"voiceutils_audio_processed_total",
		metric.WithDescription("Total number of audio chunks processed"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create audioProcessed metric: %w", err)
	}

	m.audioDuration, err = meter.Float64Histogram(
		"voiceutils_audio_processing_duration_seconds",
		metric.WithDescription("Duration of audio processing operations in seconds"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create audioDuration metric: %w", err)
	}

	m.audioBuffersInUse, err = meter.Int64UpDownCounter(
		"voiceutils_audio_buffers_in_use",
		metric.WithDescription("Number of audio buffers currently in use"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create audioBuffersInUse metric: %w", err)
	}

	m.audioSamplesTotal, err = meter.Int64Counter(
		"voiceutils_audio_samples_total",
		metric.WithDescription("Total number of audio samples processed"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create audioSamplesTotal metric: %w", err)
	}

	m.audioConversions, err = meter.Int64Counter(
		"voiceutils_audio_conversions_total",
		metric.WithDescription("Total number of audio format conversions"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create audioConversions metric: %w", err)
	}

	m.conversionDuration, err = meter.Float64Histogram(
		"voiceutils_conversion_duration_seconds",
		metric.WithDescription("Duration of audio format conversions in seconds"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create conversionDuration metric: %w", err)
	}

	// Buffer pool metrics
	m.bufferPoolHits, err = meter.Int64Counter(
		"voiceutils_buffer_pool_hits_total",
		metric.WithDescription("Total number of buffer pool hits"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create bufferPoolHits metric: %w", err)
	}

	m.bufferPoolMisses, err = meter.Int64Counter(
		"voiceutils_buffer_pool_misses_total",
		metric.WithDescription("Total number of buffer pool misses"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create bufferPoolMisses metric: %w", err)
	}

	m.bufferPoolSize, err = meter.Int64UpDownCounter(
		"voiceutils_buffer_pool_size",
		metric.WithDescription("Current size of the buffer pool"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create bufferPoolSize metric: %w", err)
	}

	// Retry and resilience metrics
	m.retryAttempts, err = meter.Int64Counter(
		"voiceutils_retry_attempts_total",
		metric.WithDescription("Total number of retry attempts"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create retryAttempts metric: %w", err)
	}

	m.circuitState, err = meter.Int64Counter(
		"voiceutils_circuit_state_changes_total",
		metric.WithDescription("Total number of circuit breaker state changes"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create circuitState metric: %w", err)
	}

	m.rateLimitHits, err = meter.Int64Counter(
		"voiceutils_rate_limit_hits_total",
		metric.WithDescription("Total number of rate limit hits"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create rateLimitHits metric: %w", err)
	}

	// Error metrics
	m.errorsTotal, err = meter.Int64Counter(
		"voiceutils_errors_total",
		metric.WithDescription("Total number of errors"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create errorsTotal metric: %w", err)
	}

	if tracer == nil {
		tracer = trace.NewNoopTracerProvider().Tracer("voiceutils")
	}
	m.tracer = tracer

	return m, nil
}

// RecordAudioProcessed records a successful audio chunk processing.
func (m *Metrics) RecordAudioProcessed(ctx context.Context, operation string, duration time.Duration, sampleCount int64) {
	if m == nil {
		return
	}
	if m.audioProcessed != nil {
		m.audioProcessed.Add(ctx, 1,
			metric.WithAttributes(
				attribute.String("operation", operation),
			))
	}
	if m.audioDuration != nil {
		m.audioDuration.Record(ctx, duration.Seconds(),
			metric.WithAttributes(
				attribute.String("operation", operation),
			))
	}
	if m.audioSamplesTotal != nil && sampleCount > 0 {
		m.audioSamplesTotal.Add(ctx, sampleCount,
			metric.WithAttributes(
				attribute.String("operation", operation),
			))
	}
}

// RecordAudioConversion records an audio format conversion.
func (m *Metrics) RecordAudioConversion(ctx context.Context, fromFormat, toFormat string, duration time.Duration) {
	if m == nil {
		return
	}
	if m.audioConversions != nil {
		m.audioConversions.Add(ctx, 1,
			metric.WithAttributes(
				attribute.String("from_format", fromFormat),
				attribute.String("to_format", toFormat),
			))
	}
	if m.conversionDuration != nil {
		m.conversionDuration.Record(ctx, duration.Seconds(),
			metric.WithAttributes(
				attribute.String("from_format", fromFormat),
				attribute.String("to_format", toFormat),
			))
	}
}

// RecordBufferAcquired records a buffer being acquired from the pool.
func (m *Metrics) RecordBufferAcquired(ctx context.Context, size int, poolHit bool) {
	if m == nil {
		return
	}
	if poolHit {
		if m.bufferPoolHits != nil {
			m.bufferPoolHits.Add(ctx, 1,
				metric.WithAttributes(
					attribute.Int("size", size),
				))
		}
	} else {
		if m.bufferPoolMisses != nil {
			m.bufferPoolMisses.Add(ctx, 1,
				metric.WithAttributes(
					attribute.Int("size", size),
				))
		}
	}
	if m.audioBuffersInUse != nil {
		m.audioBuffersInUse.Add(ctx, 1)
	}
}

// RecordBufferReleased records a buffer being released back to the pool.
func (m *Metrics) RecordBufferReleased(ctx context.Context) {
	if m == nil || m.audioBuffersInUse == nil {
		return
	}
	m.audioBuffersInUse.Add(ctx, -1)
}

// RecordRetryAttempt records a retry attempt.
func (m *Metrics) RecordRetryAttempt(ctx context.Context, operation string, attempt int) {
	if m == nil || m.retryAttempts == nil {
		return
	}
	m.retryAttempts.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("operation", operation),
			attribute.Int("attempt", attempt),
		))
}

// RecordCircuitStateChange records a circuit breaker state change.
func (m *Metrics) RecordCircuitStateChange(ctx context.Context, name, fromState, toState string) {
	if m == nil || m.circuitState == nil {
		return
	}
	m.circuitState.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("name", name),
			attribute.String("from_state", fromState),
			attribute.String("to_state", toState),
		))
}

// RecordRateLimitHit records a rate limit hit.
func (m *Metrics) RecordRateLimitHit(ctx context.Context, limiterName string) {
	if m == nil || m.rateLimitHits == nil {
		return
	}
	m.rateLimitHits.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("limiter", limiterName),
		))
}

// RecordError records an error occurrence.
func (m *Metrics) RecordError(ctx context.Context, operation, errorCode string) {
	if m == nil || m.errorsTotal == nil {
		return
	}
	m.errorsTotal.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("operation", operation),
			attribute.String("error_code", errorCode),
		))
}

// Global metrics instance - initialized once.
var (
	globalMetrics *Metrics
	metricsOnce   sync.Once
)

// NoOpMetrics returns a metrics instance that does nothing.
// Useful for testing or when metrics are disabled.
func NoOpMetrics() *Metrics {
	return &Metrics{
		tracer: trace.NewNoopTracerProvider().Tracer("voiceutils"),
	}
}

// InitMetrics initializes the global metrics instance.
// This follows the standard pattern used across all Beluga AI packages.
func InitMetrics(meter metric.Meter, tracer trace.Tracer) {
	metricsOnce.Do(func() {
		if tracer == nil {
			tracer = trace.NewNoopTracerProvider().Tracer("voiceutils")
		}
		metrics, err := NewMetrics(meter, tracer)
		if err != nil {
			// If metrics creation fails, use no-op metrics
			globalMetrics = NoOpMetrics()
			return
		}
		globalMetrics = metrics
	})
}

// GetMetrics returns the global metrics instance.
// This follows the standard pattern used across all Beluga AI packages.
func GetMetrics() *Metrics {
	return globalMetrics
}
