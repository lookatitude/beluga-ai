---
title: "Custom S2S Voice Metrics"
description: "Recipe for tracking Speech-to-Speech voice metrics in Go with OpenTelemetry — buffer sizes, VAD events, turn transitions, and glass-to-glass latency."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Beluga AI, S2S metrics, Go voice observability, OpenTelemetry, voice latency, turn transitions, custom metrics recipe"
---

## Problem

Standard observability metrics cover general system health (CPU, memory, request latency) but miss domain-specific concerns critical to voice applications. Audio buffer sizes determine memory usage and latency characteristics. Voice activity detection events indicate how often users speak versus system downtime. Speaker turn transitions measure conversation flow and detect issues like premature interruptions or delayed responses. Glass-to-glass latency spans multiple services and requires end-to-end timing that standard HTTP metrics cannot capture. Without these voice-specific metrics, you cannot diagnose problems like "VAD triggers too frequently in noisy environments" or "latency increases after 10 concurrent sessions." Custom metrics provide visibility into voice system behavior that enables performance tuning, capacity planning, and anomaly detection.

## Solution

Use OpenTelemetry's metric API to define domain-specific counters and histograms that track voice pipeline behavior. Counters accumulate totals for discrete events (VAD triggers, speaker turns, audio chunks processed). Histograms capture distributions for continuous measurements (buffer sizes, latency, silence duration). Consistent attribute naming (session_id, event_type, speaker) enables filtering and aggregation in observability tools. Metrics follow OTel conventions (voice.s2s.* namespace, SI units, descriptive names) for compatibility with Grafana, Datadog, and other platforms.

This approach extends Beluga's OTel GenAI conventions to voice-specific concerns. Just as LLM operations emit gen_ai.* metrics for token counts and model calls, voice operations emit voice.* metrics for audio processing. Recording metrics in hot paths (per audio chunk, per VAD event) provides granular visibility without requiring sampling or approximation. OpenTelemetry's efficient metric collection ensures low overhead even at high cardinality.

## Code Example

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/metric"
    "go.opentelemetry.io/otel/trace"
)

var tracer = otel.Tracer("beluga.voice.s2s.metrics")

// VoiceMetrics tracks S2S voice-specific metrics.
type VoiceMetrics struct {
    audioBufferSize     metric.Int64Histogram
    vadEvents           metric.Int64Counter
    speakerTurns        metric.Int64Counter
    glassToGlassLatency metric.Float64Histogram
    audioChunks         metric.Int64Counter
    silenceDuration     metric.Float64Histogram
}

// NewVoiceMetrics creates voice-specific metrics.
func NewVoiceMetrics(meter metric.Meter) (*VoiceMetrics, error) {
    audioBufferSize, err := meter.Int64Histogram(
        "voice.s2s.audio_buffer_size_bytes",
        metric.WithDescription("Size of audio buffers in S2S operations"),
        metric.WithUnit("By"),
    )
    if err != nil {
        return nil, fmt.Errorf("failed to create audio buffer size metric: %w", err)
    }

    vadEvents, err := meter.Int64Counter(
        "voice.s2s.vad_events_total",
        metric.WithDescription("Total voice activity detection events"),
    )
    if err != nil {
        return nil, fmt.Errorf("failed to create VAD events metric: %w", err)
    }

    speakerTurns, err := meter.Int64Counter(
        "voice.s2s.speaker_turns_total",
        metric.WithDescription("Total speaker turn transitions"),
    )
    if err != nil {
        return nil, fmt.Errorf("failed to create speaker turns metric: %w", err)
    }

    glassToGlassLatency, err := meter.Float64Histogram(
        "voice.s2s.glass_to_glass_latency_seconds",
        metric.WithDescription("End-to-end latency from user speech to AI response"),
        metric.WithUnit("s"),
    )
    if err != nil {
        return nil, fmt.Errorf("failed to create latency metric: %w", err)
    }

    audioChunks, err := meter.Int64Counter(
        "voice.s2s.audio_chunks_total",
        metric.WithDescription("Total audio chunks processed"),
    )
    if err != nil {
        return nil, fmt.Errorf("failed to create audio chunks metric: %w", err)
    }

    silenceDuration, err := meter.Float64Histogram(
        "voice.s2s.silence_duration_seconds",
        metric.WithDescription("Duration of silence periods between speech"),
        metric.WithUnit("s"),
    )
    if err != nil {
        return nil, fmt.Errorf("failed to create silence duration metric: %w", err)
    }

    return &VoiceMetrics{
        audioBufferSize:     audioBufferSize,
        vadEvents:           vadEvents,
        speakerTurns:        speakerTurns,
        glassToGlassLatency: glassToGlassLatency,
        audioChunks:         audioChunks,
        silenceDuration:     silenceDuration,
    }, nil
}

// RecordAudioBuffer records audio buffer size.
func (vm *VoiceMetrics) RecordAudioBuffer(ctx context.Context, sizeBytes int64, sessionID string) {
    vm.audioBufferSize.Record(ctx, sizeBytes,
        metric.WithAttributes(
            attribute.String("session_id", sessionID),
        ))
}

// RecordVADEvent records a voice activity detection event.
func (vm *VoiceMetrics) RecordVADEvent(ctx context.Context, eventType string, sessionID string) {
    vm.vadEvents.Add(ctx, 1,
        metric.WithAttributes(
            attribute.String("event_type", eventType), // "start", "end", "silence"
            attribute.String("session_id", sessionID),
        ))
}

// RecordSpeakerTurn records a speaker turn transition.
func (vm *VoiceMetrics) RecordSpeakerTurn(ctx context.Context, fromSpeaker, toSpeaker, sessionID string) {
    vm.speakerTurns.Add(ctx, 1,
        metric.WithAttributes(
            attribute.String("from_speaker", fromSpeaker),
            attribute.String("to_speaker", toSpeaker),
            attribute.String("session_id", sessionID),
        ))
}

// RecordGlassToGlassLatency records end-to-end latency.
func (vm *VoiceMetrics) RecordGlassToGlassLatency(ctx context.Context, latency time.Duration, sessionID string) {
    vm.glassToGlassLatency.Record(ctx, latency.Seconds(),
        metric.WithAttributes(
            attribute.String("session_id", sessionID),
        ))
}

// RecordAudioChunk records an audio chunk being processed.
func (vm *VoiceMetrics) RecordAudioChunk(ctx context.Context, chunkSize int64, sessionID string) {
    vm.audioChunks.Add(ctx, 1,
        metric.WithAttributes(
            attribute.Int64("chunk_size_bytes", chunkSize),
            attribute.String("session_id", sessionID),
        ))
}

// RecordSilenceDuration records silence period duration.
func (vm *VoiceMetrics) RecordSilenceDuration(ctx context.Context, duration time.Duration, sessionID string) {
    vm.silenceDuration.Record(ctx, duration.Seconds(),
        metric.WithAttributes(
            attribute.String("session_id", sessionID),
        ))
}

// handleS2SAudio demonstrates metrics usage in an S2S audio handler.
func handleS2SAudio(ctx context.Context, metrics *VoiceMetrics, audioData []byte, sessionID string) {
    ctx, span := tracer.Start(ctx, "s2s.handle_audio")
    defer span.End()

    metrics.RecordAudioChunk(ctx, int64(len(audioData)), sessionID)
    metrics.RecordAudioBuffer(ctx, int64(len(audioData)), sessionID)

    // ... process audio ...

    span.SetAttributes(
        attribute.String("session_id", sessionID),
        attribute.Int("audio_size", len(audioData)),
    )
}

func main() {
    ctx := context.Background()

    meter := otel.Meter("beluga.voice.s2s")

    voiceMetrics, err := NewVoiceMetrics(meter)
    if err != nil {
        log.Fatalf("Failed to create voice metrics: %v", err)
    }

    sessionID := "session-123"
    audioData := []byte{1, 2, 3, 4, 5}
    handleS2SAudio(ctx, voiceMetrics, audioData, sessionID)

    voiceMetrics.RecordVADEvent(ctx, "start", sessionID)
    voiceMetrics.RecordGlassToGlassLatency(ctx, 150*time.Millisecond, sessionID)

    fmt.Println("Voice metrics recorded successfully")
}
```

The code demonstrates metric registration at startup and recording in hot paths. Each metric includes a clear description and appropriate unit (bytes, seconds), making them self-documenting in observability dashboards. Attributes provide filtering dimensions (session_id, event_type) for drilling down into specific problems.

## Explanation

1. **Custom metric registration** -- Metrics use descriptive names following OTel conventions (voice.s2s.* namespace, _total suffix for counters, _seconds suffix for time measurements). The description field explains what the metric measures, appearing as documentation in observability tools. Unit specifications (By for bytes, s for seconds) enable automatic unit conversion and proper axis labeling in dashboards. This consistency makes metrics discoverable: engineers unfamiliar with the codebase can find relevant metrics by browsing the voice.s2s.* namespace in Grafana or Datadog.

2. **Structured attributes** -- Attributes like session_id and event_type provide query dimensions. Session IDs enable per-conversation analysis: "show glass-to-glass latency for session X" or "count VAD events per session." Event types distinguish different VAD states (start/end/silence), allowing queries like "count speech-start events per minute" to measure traffic patterns. Attributes should be low-cardinality (dozens of values, not thousands) to avoid metric explosion. Use session IDs as attributes but avoid user IDs or arbitrary strings that create unbounded cardinality.

3. **Metric types** -- Counters track cumulative totals that only increase: VAD events count, speaker turns count, audio chunks processed count. These answer "how many" questions and support rate calculations (events per second). Histograms track distributions of values: buffer sizes, latency, silence duration. These answer "what's typical" and "what's the 99th percentile" questions, critical for SLOs and capacity planning. Choose the metric type based on your query pattern: if you need totals or rates, use counters; if you need distributions or percentiles, use histograms.

4. **Consistent naming** -- All metrics share the voice.s2s.* prefix, making it easy to find related metrics in dashboards or metric browsers. Attributes use consistent names (session_id, not sessionID or SessionID) across all metrics, enabling joins and correlations. This naming discipline pays off when building dashboards: queries like "select all metrics where session_id=X" work because every metric uses the same attribute name. Follow the pattern for new metrics to maintain discoverability.

## Variations

### Metric Aggregation

Aggregate metrics per time window using a wrapper that accumulates values and flushes periodically. This reduces metric volume for high-throughput systems where per-event recording creates too many data points.

```go
type AggregatedVoiceMetrics struct {
    metrics *VoiceMetrics
    window  time.Duration
}
```

### Conditional Metrics

Only record metrics when observability is enabled, avoiding overhead in performance-critical paths during development or testing.

```go
func (vm *VoiceMetrics) RecordAudioChunkIfEnabled(ctx context.Context, enabled bool, chunkSize int64, sessionID string) {
    if enabled {
        vm.RecordAudioChunk(ctx, chunkSize, sessionID)
    }
}
```

## Related Recipes

- [Trace Aggregation for Multi-Agents](/cookbook/trace-aggregation) — Aggregate traces across agents
