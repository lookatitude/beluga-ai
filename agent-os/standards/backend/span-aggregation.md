# OTEL Span Aggregation Pattern

## Purpose

Wrapper packages aggregate spans from sub-packages into parent traces for unified observability. This enables end-to-end tracing across the full operation pipeline.

## Trace Hierarchy

```
voice.session (parent span)
├── stt.transcribe (child span)
│   ├── stt.preprocess (grandchild)
│   └── stt.api_call (grandchild)
├── vad.detect (child span)
├── agent.process (child span)
└── tts.synthesize (child span)
    └── tts.api_call (grandchild)
```

## Implementation

### Parent Span Creation

```go
// pkg/voice/metrics.go
package voice

import (
    "context"
    "time"

    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/metric"
    "go.opentelemetry.io/otel/trace"

    "github.com/lookatitude/beluga-ai/pkg/voice/stt"
    "github.com/lookatitude/beluga-ai/pkg/voice/tts"
    "github.com/lookatitude/beluga-ai/pkg/voice/vad"
)

type Metrics struct {
    // Root-level metrics
    sessionsTotal      metric.Int64Counter
    sessionDuration    metric.Float64Histogram
    errorsTotal        metric.Int64Counter
    audioProcessed     metric.Int64Counter

    // Sub-package metrics (aggregated)
    sttMetrics *stt.Metrics
    ttsMetrics *tts.Metrics
    vadMetrics *vad.Metrics

    tracer trace.Tracer
}

func NewMetrics(meterProvider metric.MeterProvider, tracerProvider trace.TracerProvider) (*Metrics, error) {
    meter := meterProvider.Meter("beluga.voice")
    tracer := tracerProvider.Tracer("beluga.voice")

    m := &Metrics{tracer: tracer}
    var err error

    m.sessionsTotal, err = meter.Int64Counter(
        "voice.sessions.total",
        metric.WithDescription("Total voice sessions"),
    )
    if err != nil {
        return nil, err
    }

    m.sessionDuration, err = meter.Float64Histogram(
        "voice.session.duration",
        metric.WithDescription("Voice session duration in seconds"),
        metric.WithUnit("s"),
    )
    if err != nil {
        return nil, err
    }

    // Initialize sub-package metrics
    m.sttMetrics, err = stt.NewMetrics(meterProvider, tracerProvider)
    if err != nil {
        return nil, err
    }

    m.ttsMetrics, err = tts.NewMetrics(meterProvider, tracerProvider)
    if err != nil {
        return nil, err
    }

    m.vadMetrics, err = vad.NewMetrics(meterProvider, tracerProvider)
    if err != nil {
        return nil, err
    }

    return m, nil
}
```

### Parent Span with Context Propagation

```go
// pkg/voice/session.go
func (v *voiceAgent) ProcessAudio(ctx context.Context, audio []byte) (*ProcessResult, error) {
    // Create parent span
    ctx, span := v.metrics.tracer.Start(ctx, "voice.process_audio",
        trace.WithAttributes(
            attribute.Int("audio.length", len(audio)),
            attribute.String("session.id", v.sessionID),
        ))
    defer span.End()

    start := time.Now()
    v.metrics.sessionsTotal.Add(ctx, 1)

    // Sub-package operations inherit context and create child spans
    // STT transcription
    transcription, err := v.stt.Transcribe(ctx, audio)  // Creates child span
    if err != nil {
        span.RecordError(err)
        span.SetStatus(codes.Error, "transcription failed")
        return nil, err
    }
    span.AddEvent("transcription_complete", trace.WithAttributes(
        attribute.String("text", transcription.Text),
        attribute.Float64("confidence", transcription.Confidence),
    ))

    // Agent processing
    response, err := v.agent.Process(ctx, transcription.Text)  // Creates child span
    if err != nil {
        span.RecordError(err)
        return nil, err
    }

    // TTS synthesis
    audioOutput, err := v.tts.Synthesize(ctx, response)  // Creates child span
    if err != nil {
        span.RecordError(err)
        return nil, err
    }

    // Record duration
    duration := time.Since(start)
    v.metrics.sessionDuration.Record(ctx, duration.Seconds())

    span.SetAttributes(attribute.Float64("duration_seconds", duration.Seconds()))

    return &ProcessResult{
        Transcription: transcription,
        Response:      response,
        Audio:         audioOutput,
    }, nil
}
```

### Sub-Package Child Spans

```go
// pkg/voice/stt/transcriber.go
func (t *transcriber) Transcribe(ctx context.Context, audio []byte) (*TranscriptionResult, error) {
    // Child span - automatically linked to parent via context
    ctx, span := t.metrics.tracer.Start(ctx, "stt.transcribe",
        trace.WithAttributes(
            attribute.String("provider", t.provider),
            attribute.Int("audio.length", len(audio)),
            attribute.String("language", t.config.Language),
        ))
    defer span.End()

    start := time.Now()

    // Preprocess
    processed, err := t.preprocess(ctx, audio)  // Creates grandchild span
    if err != nil {
        span.RecordError(err)
        return nil, err
    }

    // API call
    result, err := t.callAPI(ctx, processed)  // Creates grandchild span
    if err != nil {
        span.RecordError(err)
        return nil, err
    }

    // Record metrics
    t.metrics.RecordTranscription(ctx, time.Since(start), err == nil)

    span.SetAttributes(
        attribute.String("text", result.Text),
        attribute.Float64("confidence", result.Confidence),
    )

    return result, nil
}
```

## Span Attributes Convention

### Parent Span Attributes

```go
trace.WithAttributes(
    attribute.String("component", "voice"),
    attribute.String("session.id", sessionID),
    attribute.String("operation", "process_audio"),
)
```

### Sub-Package Span Attributes

```go
trace.WithAttributes(
    attribute.String("component", "stt"),
    attribute.String("provider", "deepgram"),
    attribute.String("model", "nova-2"),
    attribute.Int("audio.length", len(audio)),
)
```

## Metrics Aggregation

```go
// pkg/voice/metrics.go

// Aggregate metrics from sub-packages
func (m *Metrics) GetAggregatedStats(ctx context.Context) *AggregatedStats {
    return &AggregatedStats{
        TotalSessions:     m.sessionsTotal.Get(),
        AvgSessionDuration: m.sessionDuration.Avg(),
        STTStats:          m.sttMetrics.GetStats(),
        TTSStats:          m.ttsMetrics.GetStats(),
        VADStats:          m.vadMetrics.GetStats(),
    }
}
```

## Error Recording

```go
func (v *voiceAgent) ProcessAudio(ctx context.Context, audio []byte) (*ProcessResult, error) {
    ctx, span := v.metrics.tracer.Start(ctx, "voice.process_audio")
    defer span.End()

    result, err := v.doProcess(ctx, audio)
    if err != nil {
        // Record error at parent level
        span.RecordError(err)
        span.SetStatus(codes.Error, err.Error())

        // Increment error counter with context
        v.metrics.errorsTotal.Add(ctx, 1, metric.WithAttributes(
            attribute.String("error.type", errorType(err)),
            attribute.String("operation", "process_audio"),
        ))

        return nil, err
    }

    span.SetStatus(codes.Ok, "success")
    return result, nil
}
```

## Testing Span Aggregation

```go
func TestSpanAggregation(t *testing.T) {
    // Setup test tracer with recording
    sr := tracetest.NewSpanRecorder()
    tp := trace.NewTracerProvider(trace.WithSpanProcessor(sr))

    metrics, err := voice.NewMetrics(
        metric.NewNoopMeterProvider(),
        tp,
    )
    require.NoError(t, err)

    agent := voice.NewVoiceAgentWithMetrics(mockSTT, mockTTS, metrics)

    // Process audio
    _, err = agent.ProcessAudio(ctx, testAudio)
    require.NoError(t, err)

    // Verify span hierarchy
    spans := sr.Ended()
    require.Len(t, spans, 4)  // parent + 3 children

    parentSpan := findSpan(spans, "voice.process_audio")
    sttSpan := findSpan(spans, "stt.transcribe")
    ttsSpan := findSpan(spans, "tts.synthesize")

    // Verify parent-child relationships
    assert.Equal(t, parentSpan.SpanContext().SpanID(), sttSpan.Parent().SpanID())
    assert.Equal(t, parentSpan.SpanContext().SpanID(), ttsSpan.Parent().SpanID())
}
```

## Related Standards

- [otel-naming.md](./otel-naming.md) - OTEL naming conventions
- [metrics-go-shape.md](./metrics-go-shape.md) - Standard metrics.go structure
- [../global/wrapper-package-pattern.md](../global/wrapper-package-pattern.md) - Wrapper patterns
