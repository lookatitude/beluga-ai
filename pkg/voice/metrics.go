// Package voice provides OTEL metrics for the voice package.
// This file aggregates common metrics patterns used across voice sub-packages.
package voice

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// VoiceMetrics defines comprehensive metrics for the voice package.
type VoiceMetrics struct {
	// Operation metrics
	totalOperations     metric.Int64Counter
	operationDuration   metric.Float64Histogram
	activeOperations     metric.Int64UpDownCounter

	// Error metrics
	totalErrors         metric.Int64Counter
	errorByType         metric.Int64Counter

	// Provider metrics
	providerUsage       metric.Int64Counter
	providerErrors      metric.Int64Counter

	// Audio processing metrics
	audioProcessed      metric.Int64Counter
	audioProcessingTime metric.Float64Histogram
}

// Global metrics instance
var globalMetrics *VoiceMetrics

// GetGlobalMetrics returns the global metrics instance.
func GetGlobalMetrics() *VoiceMetrics {
	return globalMetrics
}

// NewVoiceMetrics creates a new VoiceMetrics instance.
func NewVoiceMetrics(meter metric.Meter) *VoiceMetrics {
	m := &VoiceMetrics{}

	m.totalOperations, _ = meter.Int64Counter(
		"voice.operations.total",
		metric.WithDescription("Total voice operations"),
	)

	m.operationDuration, _ = meter.Float64Histogram(
		"voice.operation.duration",
		metric.WithDescription("Voice operation duration"),
		metric.WithUnit("s"),
	)

	m.activeOperations, _ = meter.Int64UpDownCounter(
		"voice.operations.active",
		metric.WithDescription("Active voice operations"),
	)

	m.totalErrors, _ = meter.Int64Counter(
		"voice.errors.total",
		metric.WithDescription("Total voice errors"),
	)

	m.errorByType, _ = meter.Int64Counter(
		"voice.errors.by_type",
		metric.WithDescription("Voice errors by type"),
	)

	m.providerUsage, _ = meter.Int64Counter(
		"voice.provider.usage",
		metric.WithDescription("Voice provider usage"),
	)

	m.providerErrors, _ = meter.Int64Counter(
		"voice.provider.errors",
		metric.WithDescription("Voice provider errors"),
	)

	m.audioProcessed, _ = meter.Int64Counter(
		"voice.audio.processed",
		metric.WithDescription("Audio data processed"),
	)

	m.audioProcessingTime, _ = meter.Float64Histogram(
		"voice.audio.processing_time",
		metric.WithDescription("Audio processing time"),
		metric.WithUnit("s"),
	)

	return m
}

// RecordOperation records a voice operation.
func (m *VoiceMetrics) RecordOperation(ctx context.Context, operation string, duration time.Duration) {
	if m == nil {
		return
	}
	m.totalOperations.Add(ctx, 1, metric.WithAttributes(
		attribute.String("operation", operation),
	))
	m.operationDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(
		attribute.String("operation", operation),
	))
}

// RecordError records a voice error.
func (m *VoiceMetrics) RecordError(ctx context.Context, errorType, errorCode string) {
	if m == nil {
		return
	}
	m.totalErrors.Add(ctx, 1, metric.WithAttributes(
		attribute.String("error_type", errorType),
		attribute.String("error_code", errorCode),
	))
	m.errorByType.Add(ctx, 1, metric.WithAttributes(
		attribute.String("error_type", errorType),
	))
}

// RecordProviderUsage records provider usage.
func (m *VoiceMetrics) RecordProviderUsage(ctx context.Context, provider string) {
	if m == nil {
		return
	}
	m.providerUsage.Add(ctx, 1, metric.WithAttributes(
		attribute.String("provider", provider),
	))
}

// RecordProviderError records a provider error.
func (m *VoiceMetrics) RecordProviderError(ctx context.Context, provider, errorCode string) {
	if m == nil {
		return
	}
	m.providerErrors.Add(ctx, 1, metric.WithAttributes(
		attribute.String("provider", provider),
		attribute.String("error_code", errorCode),
	))
}

// RecordAudioProcessing records audio processing metrics.
func (m *VoiceMetrics) RecordAudioProcessing(ctx context.Context, duration time.Duration, bytesProcessed int64) {
	if m == nil {
		return
	}
	m.audioProcessed.Add(ctx, bytesProcessed)
	m.audioProcessingTime.Record(ctx, duration.Seconds())
}

// IncrementActiveOperations increments the active operations counter.
func (m *VoiceMetrics) IncrementActiveOperations(ctx context.Context) {
	if m == nil {
		return
	}
	m.activeOperations.Add(ctx, 1)
}

// DecrementActiveOperations decrements the active operations counter.
func (m *VoiceMetrics) DecrementActiveOperations(ctx context.Context) {
	if m == nil {
		return
	}
	m.activeOperations.Add(ctx, -1)
}
