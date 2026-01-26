# Wrapper Integration Tests

## Purpose

Cross-sub-package integration tests verify that wrapper packages correctly orchestrate their sub-packages. These tests ensure the facade pattern works correctly and that span aggregation, error propagation, and config handling function as expected.

## Test Categories

1. **Pipeline Tests**: Full flow through all sub-packages
2. **Error Propagation Tests**: Error handling across sub-packages
3. **Concurrency Tests**: Thread safety in wrapper operations
4. **Observability Tests**: Span aggregation and metrics
5. **Config Tests**: Hierarchical config propagation

## Pipeline Tests

### Full Pipeline Test

```go
// pkg/voice/advanced_test.go
func TestVoiceAgentPipeline(t *testing.T) {
    tests := []struct {
        name           string
        sttResponse    string
        agentResponse  string
        expectedAudio  []byte
        wantErr        bool
    }{
        {
            name:          "full pipeline success",
            sttResponse:   "hello world",
            agentResponse: "Hi there!",
            expectedAudio: []byte("synthesized audio"),
        },
        {
            name:          "empty transcription",
            sttResponse:   "",
            agentResponse: "I didn't catch that",
            expectedAudio: []byte("fallback audio"),
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            ctx := TestContext(t)

            // Setup mocks for all sub-packages
            sttMock := stt.NewMockTranscriber(
                stt.WithMockTranscription(tt.sttResponse, 0.95),
            )
            agentMock := NewMockAgent(
                WithMockResponse(tt.agentResponse),
            )
            ttsMock := tts.NewMockSpeaker(
                tts.WithMockAudio(tt.expectedAudio),
            )

            // Create voice agent with mocked sub-packages
            agent := NewVoiceAgent(
                WithSTT(sttMock),
                WithAgent(agentMock),
                WithTTS(ttsMock),
            )

            // Test full pipeline
            result, err := agent.ProcessAudio(ctx, testAudio)

            if tt.wantErr {
                require.Error(t, err)
                return
            }

            require.NoError(t, err)
            assert.Equal(t, tt.sttResponse, result.Transcription.Text)
            assert.Equal(t, tt.expectedAudio, result.Audio)

            // Verify call sequence
            assert.Equal(t, 1, sttMock.CallCount())
            assert.Equal(t, 1, agentMock.CallCount())
            assert.Equal(t, 1, ttsMock.CallCount())
        })
    }
}
```

## Error Propagation Tests

### Sub-Package Error Bubbling

```go
func TestErrorPropagation(t *testing.T) {
    tests := []struct {
        name          string
        sttErr        error
        agentErr      error
        ttsErr        error
        expectedOp    string
        expectedCode  ErrorCode
    }{
        {
            name:         "stt error bubbles up",
            sttErr:       errors.New("transcription failed"),
            expectedOp:   "ProcessAudio",
            expectedCode: ErrSTTFailed,
        },
        {
            name:         "agent error bubbles up",
            agentErr:     errors.New("agent processing failed"),
            expectedOp:   "ProcessAudio",
            expectedCode: ErrAgentFailed,
        },
        {
            name:         "tts error bubbles up",
            ttsErr:       errors.New("synthesis failed"),
            expectedOp:   "ProcessAudio",
            expectedCode: ErrTTSFailed,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            ctx := TestContext(t)

            sttMock := stt.NewMockTranscriber()
            if tt.sttErr != nil {
                sttMock = stt.NewMockTranscriber(stt.WithMockError(tt.sttErr))
            }

            agentMock := NewMockAgent()
            if tt.agentErr != nil {
                agentMock = NewMockAgent(WithMockError(tt.agentErr))
            }

            ttsMock := tts.NewMockSpeaker()
            if tt.ttsErr != nil {
                ttsMock = tts.NewMockSpeaker(tts.WithMockError(tt.ttsErr))
            }

            agent := NewVoiceAgent(
                WithSTT(sttMock),
                WithAgent(agentMock),
                WithTTS(ttsMock),
            )

            _, err := agent.ProcessAudio(ctx, testAudio)
            require.Error(t, err)

            var voiceErr *Error
            require.ErrorAs(t, err, &voiceErr)
            assert.Equal(t, tt.expectedOp, voiceErr.Op)
            assert.Equal(t, tt.expectedCode, voiceErr.Code)
        })
    }
}
```

## Concurrency Tests

### Thread Safety Test

```go
func TestConcurrentProcessing(t *testing.T) {
    ctx := TestContext(t)

    sttMock := stt.NewMockTranscriber(
        stt.WithMockTranscription("concurrent test", 0.95),
        stt.WithMockDelay(10 * time.Millisecond),
    )
    ttsMock := tts.NewMockSpeaker(
        tts.WithMockAudio([]byte("audio")),
        tts.WithMockDelay(10 * time.Millisecond),
    )

    agent := NewVoiceAgent(
        WithSTT(sttMock),
        WithTTS(ttsMock),
    )

    const numGoroutines = 100
    var wg sync.WaitGroup
    errors := make(chan error, numGoroutines)

    for i := 0; i < numGoroutines; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()
            audio := []byte(fmt.Sprintf("audio-%d", id))
            _, err := agent.ProcessAudio(ctx, audio)
            if err != nil {
                errors <- err
            }
        }(i)
    }

    wg.Wait()
    close(errors)

    for err := range errors {
        t.Errorf("concurrent error: %v", err)
    }

    // Verify all calls completed
    assert.Equal(t, numGoroutines, sttMock.CallCount())
    assert.Equal(t, numGoroutines, ttsMock.CallCount())
}
```

## Observability Tests

### Span Aggregation Test

```go
func TestSpanAggregation(t *testing.T) {
    // Setup test tracer with recording
    sr := tracetest.NewSpanRecorder()
    tp := trace.NewTracerProvider(trace.WithSpanProcessor(sr))

    metrics, err := NewMetrics(
        metric.NewNoopMeterProvider(),
        tp,
    )
    require.NoError(t, err)

    agent := NewVoiceAgentWithMetrics(
        stt.NewMockTranscriber(),
        tts.NewMockSpeaker(),
        metrics,
    )

    ctx := TestContext(t)
    _, err = agent.ProcessAudio(ctx, testAudio)
    require.NoError(t, err)

    // Get recorded spans
    spans := sr.Ended()

    // Find parent span
    parentSpan := findSpanByName(spans, "voice.process_audio")
    require.NotNil(t, parentSpan)

    // Find child spans
    sttSpan := findSpanByName(spans, "stt.transcribe")
    ttsSpan := findSpanByName(spans, "tts.synthesize")

    require.NotNil(t, sttSpan)
    require.NotNil(t, ttsSpan)

    // Verify parent-child relationships
    assert.Equal(t, parentSpan.SpanContext().SpanID(), sttSpan.Parent().SpanID())
    assert.Equal(t, parentSpan.SpanContext().SpanID(), ttsSpan.Parent().SpanID())

    // Verify span attributes
    assertSpanAttribute(t, parentSpan, "component", "voice")
    assertSpanAttribute(t, sttSpan, "component", "stt")
    assertSpanAttribute(t, ttsSpan, "component", "tts")
}

func findSpanByName(spans []trace.ReadOnlySpan, name string) trace.ReadOnlySpan {
    for _, s := range spans {
        if s.Name() == name {
            return s
        }
    }
    return nil
}

func assertSpanAttribute(t *testing.T, span trace.ReadOnlySpan, key, expected string) {
    for _, attr := range span.Attributes() {
        if string(attr.Key) == key {
            assert.Equal(t, expected, attr.Value.AsString())
            return
        }
    }
    t.Errorf("attribute %s not found", key)
}
```

### Metrics Recording Test

```go
func TestMetricsRecording(t *testing.T) {
    // Setup test meter with recording
    reader := metric.NewManualReader()
    mp := metric.NewMeterProvider(metric.WithReader(reader))

    metrics, err := NewMetrics(mp, trace.NewNoopTracerProvider())
    require.NoError(t, err)

    agent := NewVoiceAgentWithMetrics(
        stt.NewMockTranscriber(),
        tts.NewMockSpeaker(),
        metrics,
    )

    ctx := TestContext(t)

    // Process multiple requests
    for i := 0; i < 5; i++ {
        _, err := agent.ProcessAudio(ctx, testAudio)
        require.NoError(t, err)
    }

    // Collect metrics
    var rm metricdata.ResourceMetrics
    err = reader.Collect(ctx, &rm)
    require.NoError(t, err)

    // Find sessions counter
    sessionsCounter := findMetric(rm, "voice.sessions.total")
    require.NotNil(t, sessionsCounter)

    // Verify count
    sum := sessionsCounter.Data.(metricdata.Sum[int64])
    assert.Equal(t, int64(5), sum.DataPoints[0].Value)
}
```

## Config Tests

### Hierarchical Config Propagation

```go
func TestConfigPropagation(t *testing.T) {
    // Create config with sub-package configs
    cfg := &Config{
        SessionTimeout: 5 * time.Minute,
        MaxConcurrent:  100,
        STT: stt.Config{
            Provider:   "deepgram",
            APIKey:     "test-key",
            SampleRate: 16000,
        },
        TTS: tts.Config{
            Provider: "elevenlabs",
            APIKey:   "test-key",
        },
    }

    // Verify config validation
    err := cfg.Validate()
    require.NoError(t, err)

    // Verify sub-package configs are accessible
    assert.Equal(t, "deepgram", cfg.STT.Provider)
    assert.Equal(t, "elevenlabs", cfg.TTS.Provider)
    assert.Equal(t, 16000, cfg.STT.SampleRate)
}

func TestConfigValidationErrors(t *testing.T) {
    tests := []struct {
        name      string
        cfg       *Config
        wantField string
    }{
        {
            name: "missing stt provider",
            cfg: &Config{
                SessionTimeout: 5 * time.Minute,
                STT:            stt.Config{},  // Missing provider
                TTS:            tts.Config{Provider: "elevenlabs"},
            },
            wantField: "STT.Provider",
        },
        {
            name: "missing session timeout",
            cfg: &Config{
                STT: stt.Config{Provider: "deepgram"},
                TTS: tts.Config{Provider: "elevenlabs"},
            },
            wantField: "SessionTimeout",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := tt.cfg.Validate()
            require.Error(t, err)
            assert.Contains(t, err.Error(), tt.wantField)
        })
    }
}
```

## Benchmark Tests

```go
func BenchmarkVoiceAgentPipeline(b *testing.B) {
    ctx := context.Background()

    sttMock := stt.NewMockTranscriber()
    ttsMock := tts.NewMockSpeaker()

    agent := NewVoiceAgent(
        WithSTT(sttMock),
        WithTTS(ttsMock),
    )

    testAudio := make([]byte, 16000)  // 1 second of audio

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, err := agent.ProcessAudio(ctx, testAudio)
        if err != nil {
            b.Fatal(err)
        }
    }
}

func BenchmarkConcurrentProcessing(b *testing.B) {
    ctx := context.Background()

    sttMock := stt.NewMockTranscriber()
    ttsMock := tts.NewMockSpeaker()

    agent := NewVoiceAgent(
        WithSTT(sttMock),
        WithTTS(ttsMock),
    )

    testAudio := make([]byte, 16000)

    b.ResetTimer()
    b.RunParallel(func(pb *testing.PB) {
        for pb.Next() {
            _, err := agent.ProcessAudio(ctx, testAudio)
            if err != nil {
                b.Fatal(err)
            }
        }
    })
}
```

## Related Standards

- [subpackage-mocks.md](./subpackage-mocks.md) - Sub-package mock patterns
- [advanced-test.md](./advanced-test.md) - Advanced test requirements
- [concurrency-and-errors.md](./concurrency-and-errors.md) - Concurrency testing
