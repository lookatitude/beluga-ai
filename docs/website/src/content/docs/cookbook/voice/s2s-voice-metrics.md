---
title: "Custom Metrics for S2S Voice"
description: "Track custom metrics specific to Speech-to-Speech voice operations using OpenTelemetry instrumentation."
---

## Problem

You need to track custom metrics specific to S2S (Speech-to-Speech) voice operations that are not covered by standard framework metrics, such as audio buffer sizes, voice activity detection events, or speaker turn transitions.

## Solution

Use OpenTelemetry's metric API to register custom metrics and record voice-specific events. Define domain-specific counters and histograms with consistent attribute names while maintaining OTel compatibility for observability tooling.

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

## Explanation

1. **Custom metric registration** — Metrics are created with descriptive names following OTel conventions (`voice.s2s.*`). Each metric has a clear description and appropriate unit. This makes metrics discoverable in observability tools like Grafana and Datadog.

2. **Structured attributes** — Attributes like `session_id` and `event_type` provide context. This allows filtering and grouping metrics by session, speaker, or event type in dashboards.

3. **Metric types** — Counters track cumulative totals (VAD events, speaker turns, audio chunks). Histograms track distributions (buffer sizes, latency, silence duration). Choose the type that matches the query pattern.

4. **Consistent naming** — All metrics use the `voice.s2s.*` prefix, and attributes use consistent names across related metrics. This enables queries like "show all metrics for session X" in observability platforms.

## Variations

### Metric Aggregation

Aggregate metrics per time window:

```go
type AggregatedVoiceMetrics struct {
    metrics *VoiceMetrics
    window  time.Duration
}
```

### Conditional Metrics

Only record metrics when enabled:

```go
func (vm *VoiceMetrics) RecordAudioChunkIfEnabled(ctx context.Context, enabled bool, chunkSize int64, sessionID string) {
    if enabled {
        vm.RecordAudioChunk(ctx, chunkSize, sessionID)
    }
}
```

## Related Recipes

- [Trace Aggregation for Multi-Agents](/cookbook/trace-aggregation) — Aggregate traces across agents
