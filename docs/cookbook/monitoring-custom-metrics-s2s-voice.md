---
title: "Custom Metrics for S2S Voice"
package: "monitoring"
category: "observability"
complexity: "intermediate"
---

# Custom Metrics for S2S Voice

## Problem

You need to track custom metrics specific to S2S (Speech-to-Speech) voice operations that aren't covered by standard framework metrics, such as audio buffer sizes, voice activity detection events, or speaker turn transitions.

## Solution

Use Beluga AI's monitoring package to register custom metrics and record voice-specific events. This works because the monitoring package provides `RegisterCustomCounter` and `RegisterCustomHistogram` methods that allow you to define domain-specific metrics while maintaining OTEL compatibility.

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
    
    "github.com/lookatitude/beluga-ai/pkg/monitoring"
)

var tracer = otel.Tracer("beluga.voice.s2s.metrics")

// VoiceMetrics tracks S2S voice-specific metrics
type VoiceMetrics struct {
    // Custom metrics
    audioBufferSize    metric.Int64Histogram
    vadEvents          metric.Int64Counter
    speakerTurns       metric.Int64Counter
    glassToGlassLatency metric.Float64Histogram
    audioChunks        metric.Int64Counter
    silenceDuration    metric.Float64Histogram
    
    // Standard metrics wrapper
    baseMetrics *monitoring.PackageMetrics
}

// NewVoiceMetrics creates voice-specific metrics
func NewVoiceMetrics(meter metric.Meter) (*VoiceMetrics, error) {
    baseMetrics, err := monitoring.NewPackageMetrics(meter)
    if err != nil {
        return nil, fmt.Errorf("failed to create base metrics: %w", err)
    }

    // Register custom metrics
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
        baseMetrics:         baseMetrics,
    }, nil
}

// RecordAudioBuffer records audio buffer size
func (vm *VoiceMetrics) RecordAudioBuffer(ctx context.Context, sizeBytes int64, sessionID string) {
    vm.audioBufferSize.Record(ctx, sizeBytes,
        metric.WithAttributes(
            attribute.String("session_id", sessionID),
        ))
}

// RecordVADEvent records a voice activity detection event
func (vm *VoiceMetrics) RecordVADEvent(ctx context.Context, eventType string, sessionID string) {
    vm.vadEvents.Add(ctx, 1,
        metric.WithAttributes(
            attribute.String("event_type", eventType), // "start", "end", "silence"
            attribute.String("session_id", sessionID),
        ))
}

// RecordSpeakerTurn records a speaker turn transition
func (vm *VoiceMetrics) RecordSpeakerTurn(ctx context.Context, fromSpeaker, toSpeaker, sessionID string) {
    vm.speakerTurns.Add(ctx, 1,
        metric.WithAttributes(
            attribute.String("from_speaker", fromSpeaker),
            attribute.String("to_speaker", toSpeaker),
            attribute.String("session_id", sessionID),
        ))
}

// RecordGlassToGlassLatency records end-to-end latency
func (vm *VoiceMetrics) RecordGlassToGlassLatency(ctx context.Context, latency time.Duration, sessionID string) {
    vm.glassToGlassLatency.Record(ctx, latency.Seconds(),
        metric.WithAttributes(
            attribute.String("session_id", sessionID),
        ))
}

// RecordAudioChunk records an audio chunk being processed
func (vm *VoiceMetrics) RecordAudioChunk(ctx context.Context, chunkSize int64, sessionID string) {
    vm.audioChunks.Add(ctx, 1,
        metric.WithAttributes(
            attribute.Int64("chunk_size_bytes", chunkSize),
            attribute.String("session_id", sessionID),
        ))
}

// RecordSilenceDuration records silence period duration
func (vm *VoiceMetrics) RecordSilenceDuration(ctx context.Context, duration time.Duration, sessionID string) {
    vm.silenceDuration.Record(ctx, duration.Seconds(),
        metric.WithAttributes(
            attribute.String("session_id", sessionID),
        ))
}

// Example usage in S2S handler
func handleS2SAudio(ctx context.Context, metrics *VoiceMetrics, audioData []byte, sessionID string) {
    ctx, span := tracer.Start(ctx, "s2s.handle_audio")
    defer span.End()
    
    // Record audio chunk
    metrics.RecordAudioChunk(ctx, int64(len(audioData)), sessionID)
    
    // Record buffer size
    metrics.RecordAudioBuffer(ctx, int64(len(audioData)), sessionID)
    
    // Simulate processing
    // ... process audio ...
    
    span.SetAttributes(
        attribute.String("session_id", sessionID),
        attribute.Int("audio_size", len(audioData)),
    )
}

func main() {
    ctx := context.Background()

    // Get meter from OTEL
    meter := otel.Meter("beluga.voice.s2s")
    
    // Create voice metrics
    voiceMetrics, err := NewVoiceMetrics(meter)
    if err != nil {
        log.Fatalf("Failed to create voice metrics: %v", err)
    }
    
    // Use metrics in your S2S handler
    sessionID := "session-123"
    audioData := []byte{1, 2, 3, 4, 5}
    handleS2SAudio(ctx, voiceMetrics, audioData, sessionID)
```
    
    // Record VAD event
    voiceMetrics.RecordVADEvent(ctx, "start", sessionID)
    
    // Record latency
    voiceMetrics.RecordGlassToGlassLatency(ctx, 150*time.Millisecond, sessionID)
    
    fmt.Println("Voice metrics recorded successfully")
}

## Explanation

Let's break down what's happening:

1. **Custom metric registration** - Notice how we create metrics with descriptive names following OTEL conventions (`voice.s2s.*`). Each metric has a clear description and appropriate unit. This makes metrics discoverable and understandable in observability tools.

2. **Structured attributes** - We use attributes like `session_id` and `event_type` to provide context. This allows filtering and grouping metrics by session, speaker, or event type in dashboards.

3. **Base metrics integration** - We wrap the standard `PackageMetrics` to get framework-wide metrics (requests, errors, etc.) while adding voice-specific ones. This gives you both general and domain-specific observability.

```go
**Key insight:** Use consistent attribute names across related metrics. This enables powerful queries like "show all metrics for session X" in your observability platform.

## Testing

```
Here's how to test this solution:
```go
func TestVoiceMetrics_RecordsEvents(t *testing.T) {
    meter := metric.NewNoOpMeter()
    metrics, err := NewVoiceMetrics(meter)
    require.NoError(t, err)
    
    ctx := context.Background()
    sessionID := "test-session"
    
    // Record various events
    metrics.RecordVADEvent(ctx, "start", sessionID)
    metrics.RecordAudioChunk(ctx, 1024, sessionID)
    metrics.RecordGlassToGlassLatency(ctx, 100*time.Millisecond, sessionID)
    
    // In a real test, you'd verify metrics were recorded
    // using a test meter that captures metric calls
}

## Variations

### Metric Aggregation

Aggregate metrics per time window:
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

- **[Monitoring Trace Aggregation for Multi-agents](./monitoring-trace-aggregation-multi-agents.md)** - Aggregate traces across agents
- **[Voice S2S Minimizing Glass-to-Glass Latency](./voice-s2s-minimizing-glass-to-glass-latency.md)** - Optimize latency
- **[Monitoring Package Guide](../guides/observability-tracing.md)** - For a deeper understanding of observability
