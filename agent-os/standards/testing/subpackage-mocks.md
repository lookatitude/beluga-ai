# Sub-Package Mocks Pattern

## Purpose

Each sub-package maintains its own `test_utils.go` with specialized mocks. This enables independent testing of sub-packages and composed testing at the wrapper level.

## Structure

```
pkg/{wrapper}/
├── test_utils.go             # Composite mocks for wrapper
├── {subpackage1}/
│   └── test_utils.go         # Sub-package specific mocks
├── {subpackage2}/
│   └── test_utils.go         # Sub-package specific mocks
```

## Sub-Package Mock Implementation

### Standard Mock Structure

```go
// pkg/voice/stt/test_utils.go
package stt

import (
    "context"
    "sync"
    "time"

    "github.com/stretchr/testify/mock"
    "github.com/lookatitude/beluga-ai/pkg/voice/stt/iface"
)

// MockTranscriber implements iface.Transcriber for testing
type MockTranscriber struct {
    mock.Mock
    mu sync.RWMutex

    // Configurable behavior
    transcriptions []iface.TranscriptionResult
    currentIndex   int
    err            error
    delay          time.Duration

    // Call tracking
    callCount int
    lastAudio []byte
}

// MockOption for configuring mock behavior
type MockOption func(*MockTranscriber)

// Constructor with options
func NewMockTranscriber(opts ...MockOption) *MockTranscriber {
    m := &MockTranscriber{
        transcriptions: []iface.TranscriptionResult{
            {Text: "default transcription", Confidence: 0.95},
        },
    }
    for _, opt := range opts {
        opt(m)
    }
    return m
}
```

### Mock Options

```go
// WithMockTranscription sets transcription result
func WithMockTranscription(text string, confidence float64) MockOption {
    return func(m *MockTranscriber) {
        m.transcriptions = append(m.transcriptions, iface.TranscriptionResult{
            Text:       text,
            Confidence: confidence,
        })
    }
}

// WithMockTranscriptions sets multiple transcription results
func WithMockTranscriptions(results []iface.TranscriptionResult) MockOption {
    return func(m *MockTranscriber) {
        m.transcriptions = results
    }
}

// WithMockError configures error behavior
func WithMockError(err error) MockOption {
    return func(m *MockTranscriber) {
        m.err = err
    }
}

// WithMockDelay adds delay to simulate processing time
func WithMockDelay(delay time.Duration) MockOption {
    return func(m *MockTranscriber) {
        m.delay = delay
    }
}
```

### Interface Implementation

```go
// Transcribe implements iface.Transcriber
func (m *MockTranscriber) Transcribe(ctx context.Context, audio []byte) (*iface.TranscriptionResult, error) {
    m.mu.Lock()
    defer m.mu.Unlock()

    m.callCount++
    m.lastAudio = audio

    // Simulate delay
    if m.delay > 0 {
        select {
        case <-time.After(m.delay):
        case <-ctx.Done():
            return nil, ctx.Err()
        }
    }

    // Return error if configured
    if m.err != nil {
        return nil, m.err
    }

    // Return next transcription
    if m.currentIndex >= len(m.transcriptions) {
        m.currentIndex = 0
    }
    result := m.transcriptions[m.currentIndex]
    m.currentIndex++

    return &result, nil
}

// StreamTranscribe implements iface.Transcriber
func (m *MockTranscriber) StreamTranscribe(ctx context.Context, audio <-chan []byte) (<-chan *iface.TranscriptionResult, error) {
    results := make(chan *iface.TranscriptionResult)

    go func() {
        defer close(results)
        for chunk := range audio {
            result, err := m.Transcribe(ctx, chunk)
            if err != nil {
                return
            }
            select {
            case results <- result:
            case <-ctx.Done():
                return
            }
        }
    }()

    return results, nil
}

// Close implements iface.Transcriber
func (m *MockTranscriber) Close() error {
    return nil
}
```

### Assertion Helpers

```go
// CallCount returns number of Transcribe calls
func (m *MockTranscriber) CallCount() int {
    m.mu.RLock()
    defer m.mu.RUnlock()
    return m.callCount
}

// LastAudio returns last audio input
func (m *MockTranscriber) LastAudio() []byte {
    m.mu.RLock()
    defer m.mu.RUnlock()
    return m.lastAudio
}

// Reset clears call history
func (m *MockTranscriber) Reset() {
    m.mu.Lock()
    defer m.mu.Unlock()
    m.callCount = 0
    m.lastAudio = nil
    m.currentIndex = 0
}
```

## Wrapper Composite Mocks

```go
// pkg/voice/test_utils.go
package voice

import (
    "github.com/lookatitude/beluga-ai/pkg/voice/stt"
    "github.com/lookatitude/beluga-ai/pkg/voice/tts"
    "github.com/lookatitude/beluga-ai/pkg/voice/vad"
)

// MockVoiceAgent composes sub-package mocks
type MockVoiceAgent struct {
    STT *stt.MockTranscriber
    TTS *tts.MockSpeaker
    VAD *vad.MockDetector
}

type MockVoiceOption func(*MockVoiceAgent)

// NewMockVoiceAgent creates voice agent with mock sub-packages
func NewMockVoiceAgent(opts ...MockVoiceOption) *MockVoiceAgent {
    m := &MockVoiceAgent{
        STT: stt.NewMockTranscriber(),
        TTS: tts.NewMockSpeaker(),
        VAD: vad.NewMockDetector(),
    }
    for _, opt := range opts {
        opt(m)
    }
    return m
}

// WithMockSTT replaces STT mock
func WithMockSTT(mock *stt.MockTranscriber) MockVoiceOption {
    return func(m *MockVoiceAgent) {
        m.STT = mock
    }
}

// WithMockTTS replaces TTS mock
func WithMockTTS(mock *tts.MockSpeaker) MockVoiceOption {
    return func(m *MockVoiceAgent) {
        m.TTS = mock
    }
}

// WithMockVAD replaces VAD mock
func WithMockVAD(mock *vad.MockDetector) MockVoiceOption {
    return func(m *MockVoiceAgent) {
        m.VAD = mock
    }
}
```

## Test Configuration Helpers

```go
// pkg/voice/stt/test_utils.go

// NewTestConfig creates config for testing
func NewTestConfig(opts ...ConfigOption) *Config {
    cfg := &Config{
        Provider:   "mock",
        SampleRate: 16000,
        Language:   "en-US",
        Encoding:   "linear16",
    }
    for _, opt := range opts {
        opt(cfg)
    }
    return cfg
}

type ConfigOption func(*Config)

func WithProvider(provider string) ConfigOption {
    return func(c *Config) {
        c.Provider = provider
    }
}

func WithAPIKey(key string) ConfigOption {
    return func(c *Config) {
        c.APIKey = key
    }
}
```

## Context Helpers

```go
// TestContext creates context with timeout for tests
func TestContext(t *testing.T) context.Context {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    t.Cleanup(cancel)
    return ctx
}

// TestContextWithCancel creates cancellable context
func TestContextWithCancel(t *testing.T) (context.Context, context.CancelFunc) {
    ctx, cancel := context.WithCancel(context.Background())
    t.Cleanup(cancel)
    return ctx, cancel
}
```

## Usage Examples

### Sub-Package Unit Test

```go
// pkg/voice/stt/advanced_test.go
func TestTranscriber(t *testing.T) {
    tests := []struct {
        name           string
        mockOpts       []MockOption
        input          []byte
        wantText       string
        wantConfidence float64
        wantErr        bool
    }{
        {
            name:           "successful transcription",
            mockOpts:       []MockOption{WithMockTranscription("hello world", 0.95)},
            input:          []byte("audio data"),
            wantText:       "hello world",
            wantConfidence: 0.95,
        },
        {
            name:     "transcription error",
            mockOpts: []MockOption{WithMockError(errors.New("api error"))},
            input:    []byte("audio data"),
            wantErr:  true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            ctx := TestContext(t)
            mock := NewMockTranscriber(tt.mockOpts...)

            result, err := mock.Transcribe(ctx, tt.input)

            if tt.wantErr {
                require.Error(t, err)
                return
            }

            require.NoError(t, err)
            assert.Equal(t, tt.wantText, result.Text)
            assert.Equal(t, tt.wantConfidence, result.Confidence)
        })
    }
}
```

### Wrapper Integration Test

```go
// pkg/voice/advanced_test.go
func TestVoiceAgentWithMocks(t *testing.T) {
    ctx := TestContext(t)

    // Configure sub-package mocks
    sttMock := stt.NewMockTranscriber(
        stt.WithMockTranscription("hello", 0.95),
    )
    ttsMock := tts.NewMockSpeaker(
        tts.WithMockAudio([]byte("synthesized audio")),
    )

    // Create agent with mocks
    agent := NewVoiceAgentWithMocks(
        WithMockSTT(sttMock),
        WithMockTTS(ttsMock),
    )

    // Test full pipeline
    result, err := agent.ProcessAudio(ctx, []byte("test audio"))
    require.NoError(t, err)

    assert.Equal(t, "hello", result.Transcription.Text)
    assert.Equal(t, []byte("synthesized audio"), result.Audio)

    // Verify mock calls
    assert.Equal(t, 1, sttMock.CallCount())
    assert.Equal(t, 1, ttsMock.CallCount())
}
```

## Related Standards

- [test-utils.md](./test-utils.md) - General test utilities
- [advanced-test.md](./advanced-test.md) - Advanced test patterns
- [wrapper-integration.md](./wrapper-integration.md) - Wrapper integration tests
