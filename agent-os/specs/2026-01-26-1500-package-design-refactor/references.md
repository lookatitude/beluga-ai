# Package Design Patterns Refactor - Reference Implementations

## Registry Pattern Reference: pkg/llms/registry.go

The llms package registry is the gold standard for provider management.

### Key Characteristics

```go
package llms

import (
    "sync"
    "github.com/lookatitude/beluga-ai/pkg/llms/iface"
)

var (
    globalRegistry *Registry
    registryOnce   sync.Once
)

type Registry struct {
    providerFactories map[string]func(*Config) (iface.ChatModel, error)
    llmFactories      map[string]func(*Config) (iface.LLM, error)
    mu                sync.RWMutex
}

func GetRegistry() *Registry {
    registryOnce.Do(func() {
        globalRegistry = &Registry{
            providerFactories: make(map[string]func(*Config) (iface.ChatModel, error)),
            llmFactories:      make(map[string]func(*Config) (iface.LLM, error)),
        }
    })
    return globalRegistry
}

func (r *Registry) Register(name string, factory func(*Config) (iface.ChatModel, error)) {
    r.mu.Lock()
    defer r.mu.Unlock()
    r.providerFactories[name] = factory
}

func (r *Registry) GetProvider(name string, config *Config) (iface.ChatModel, error) {
    r.mu.RLock()
    factory, exists := r.providerFactories[name]
    r.mu.RUnlock()

    if !exists {
        return nil, &Error{Op: "GetProvider", Code: ErrNotFound}
    }

    if config.Provider == "" {
        config.Provider = name
    }

    return factory(config)
}

func (r *Registry) ListProviders() []string {
    r.mu.RLock()
    defer r.mu.RUnlock()

    names := make([]string, 0, len(r.providerFactories))
    for name := range r.providerFactories {
        names = append(names, name)
    }
    return names
}
```

### Pattern Elements
1. **Singleton with sync.Once**: Thread-safe lazy initialization
2. **RWMutex protection**: Read operations use RLock, writes use Lock
3. **Factory functions**: Stored in maps, called with config
4. **Config defaulting**: Sets provider name if empty
5. **Error wrapping**: Uses Op/Err/Code pattern

## Wrapper Package Reference: pkg/voice/

The voice package demonstrates the wrapper/aggregation pattern.

### Structure

```
pkg/voice/
├── iface/                    # Shared interfaces
│   ├── session.go           # Session interface
│   └── config.go            # Config interface
├── backend/                  # Backend orchestration
│   ├── iface/               # Backend interfaces
│   ├── internal/            # Session manager, pipeline, etc.
│   └── providers/           # LiveKit, Vapi, Vocode, etc.
├── stt/                      # Speech-to-Text
│   ├── iface/               # Transcriber interface
│   ├── providers/           # Deepgram, OpenAI, etc.
│   └── registry.go          # STT registry
├── tts/                      # Text-to-Speech
│   ├── iface/               # Speaker interface
│   ├── providers/           # ElevenLabs, Google, etc.
│   └── registry.go          # TTS registry
├── vad/                      # Voice Activity Detection
│   ├── iface/               # VAD interface
│   └── providers/           # Silero, Deepgram, etc.
├── session/                  # Session management
├── transport/                # Audio transport
├── noise/                    # Noise cancellation
├── config.go                 # Root config with sub-package configs
├── metrics.go                # Aggregated metrics
├── errors.go                 # Error definitions
└── voice.go                  # Facade API
```

### Facade Pattern

```go
// voice.go - Facade API
package voice

type VoiceAgent interface {
    StartSession(ctx context.Context, cfg *SessionConfig) (Session, error)
    ProcessAudio(ctx context.Context, audio []byte) (*ProcessResult, error)
    Close() error
}

func NewVoiceAgent(opts ...VoiceOption) (VoiceAgent, error) {
    cfg := defaultConfig()
    for _, opt := range opts {
        opt(cfg)
    }

    // Initialize sub-packages from config
    sttProvider, err := stt.GetRegistry().GetProvider(cfg.STT.Provider, &cfg.STT)
    if err != nil {
        return nil, err
    }

    ttsProvider, err := tts.GetRegistry().GetProvider(cfg.TTS.Provider, &cfg.TTS)
    if err != nil {
        return nil, err
    }

    // ... initialize other sub-packages

    return &voiceAgent{
        stt: sttProvider,
        tts: ttsProvider,
        // ...
    }, nil
}
```

### Config Propagation

```go
// config.go - Hierarchical config
package voice

type Config struct {
    // Sub-package configs embedded
    STT       stt.Config       `yaml:"stt"`
    TTS       tts.Config       `yaml:"tts"`
    VAD       vad.Config       `yaml:"vad"`
    Transport transport.Config `yaml:"transport"`
    Session   session.Config   `yaml:"session"`

    // Root-level options
    Timeout     time.Duration `yaml:"timeout" validate:"required"`
    MaxSessions int           `yaml:"max_sessions" validate:"gte=1"`
}
```

### Span Aggregation

```go
// metrics.go - Aggregated observability
package voice

func (m *Metrics) StartSession(ctx context.Context) (context.Context, trace.Span) {
    ctx, span := m.tracer.Start(ctx, "voice.session")

    // Sub-package spans are children of this span
    // STT, TTS, VAD operations inherit trace context

    return ctx, span
}
```

## Sub-Package Reference: pkg/voice/stt/

### Structure

```
pkg/voice/stt/
├── iface/
│   └── transcriber.go       # Transcriber interface
├── internal/
│   └── audio_buffer.go      # Internal utilities
├── providers/
│   ├── deepgram/
│   │   ├── deepgram.go      # Provider implementation
│   │   └── init.go          # Auto-registration
│   ├── openai/
│   │   ├── openai.go
│   │   └── init.go
│   └── ...
├── config.go                 # STT-specific config
├── metrics.go                # STT metrics
├── errors.go                 # STT errors
├── registry.go               # STT registry
├── stt.go                    # Main API
├── test_utils.go             # Test utilities
└── advanced_test.go          # Comprehensive tests
```

### Interface Definition

```go
// iface/transcriber.go
package iface

type Transcriber interface {
    Transcribe(ctx context.Context, audio []byte) (*TranscriptionResult, error)
    StreamTranscribe(ctx context.Context, audio <-chan []byte) (<-chan *TranscriptionResult, error)
    Close() error
}

type TranscriptionResult struct {
    Text       string
    Confidence float64
    Language   string
    Segments   []Segment
}
```

### Auto-Registration

```go
// providers/deepgram/init.go
package deepgram

import "github.com/lookatitude/beluga-ai/pkg/voice/stt"

func init() {
    stt.GetRegistry().Register("deepgram", NewDeepgramTranscriber)
}

func NewDeepgramTranscriber(cfg *stt.Config) (stt.iface.Transcriber, error) {
    // Implementation
}
```

## Metrics Reference: pkg/llms/metrics.go

### Standard Structure

```go
package llms

import (
    "context"
    "time"

    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/metric"
    "go.opentelemetry.io/otel/trace"
)

type Metrics struct {
    operationsTotal   metric.Int64Counter
    operationDuration metric.Float64Histogram
    errorsTotal       metric.Int64Counter
    tokensUsed        metric.Int64Counter
    tracer            trace.Tracer
}

func NewMetrics(meterProvider metric.MeterProvider) (*Metrics, error) {
    meter := meterProvider.Meter("beluga.llms")

    operationsTotal, err := meter.Int64Counter(
        "llms.operations.total",
        metric.WithDescription("Total LLM operations"),
    )
    if err != nil {
        return nil, err
    }

    operationDuration, err := meter.Float64Histogram(
        "llms.operation.duration",
        metric.WithDescription("LLM operation duration in seconds"),
        metric.WithUnit("s"),
    )
    if err != nil {
        return nil, err
    }

    errorsTotal, err := meter.Int64Counter(
        "llms.errors.total",
        metric.WithDescription("Total LLM errors"),
    )
    if err != nil {
        return nil, err
    }

    return &Metrics{
        operationsTotal:   operationsTotal,
        operationDuration: operationDuration,
        errorsTotal:       errorsTotal,
        tracer:            otel.Tracer("beluga.llms"),
    }, nil
}

func (m *Metrics) RecordOperation(ctx context.Context, op string, duration time.Duration) {
    m.operationsTotal.Add(ctx, 1, metric.WithAttributes(
        attribute.String("operation", op),
    ))
    m.operationDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(
        attribute.String("operation", op),
    ))
}

func (m *Metrics) RecordError(ctx context.Context, op string, err error) {
    m.errorsTotal.Add(ctx, 1, metric.WithAttributes(
        attribute.String("operation", op),
        attribute.String("error_type", errorType(err)),
    ))
}
```

## Error Pattern Reference: pkg/llms/errors.go

```go
package llms

type ErrorCode string

const (
    ErrNotFound       ErrorCode = "NOT_FOUND"
    ErrInvalidConfig  ErrorCode = "INVALID_CONFIG"
    ErrProviderError  ErrorCode = "PROVIDER_ERROR"
    ErrTimeout        ErrorCode = "TIMEOUT"
    ErrRateLimit      ErrorCode = "RATE_LIMIT"
)

type Error struct {
    Op   string    // Operation that failed
    Err  error     // Underlying error
    Code ErrorCode // Error classification
}

func (e *Error) Error() string {
    if e.Err != nil {
        return fmt.Sprintf("%s: %s: %v", e.Op, e.Code, e.Err)
    }
    return fmt.Sprintf("%s: %s", e.Op, e.Code)
}

func (e *Error) Unwrap() error {
    return e.Err
}
```

## Test Utils Reference: pkg/llms/test_utils.go

```go
package llms

import (
    "context"
    "github.com/lookatitude/beluga-ai/pkg/llms/iface"
)

type MockLLM struct {
    GenerateFunc func(ctx context.Context, prompt string) (string, error)
    response     string
    err          error
}

type MockOption func(*MockLLM)

func WithMockResponse(response string) MockOption {
    return func(m *MockLLM) {
        m.response = response
    }
}

func WithMockError(err error) MockOption {
    return func(m *MockLLM) {
        m.err = err
    }
}

func NewMockLLM(opts ...MockOption) *MockLLM {
    m := &MockLLM{
        response: "mock response",
    }
    for _, opt := range opts {
        opt(m)
    }
    return m
}

func (m *MockLLM) Generate(ctx context.Context, prompt string) (string, error) {
    if m.GenerateFunc != nil {
        return m.GenerateFunc(ctx, prompt)
    }
    if m.err != nil {
        return "", m.err
    }
    return m.response, nil
}

func NewTestConfig(opts ...ConfigOption) *Config {
    cfg := &Config{
        Provider: "mock",
        Timeout:  30 * time.Second,
    }
    for _, opt := range opts {
        opt(cfg)
    }
    return cfg
}

func TestContext(t *testing.T) context.Context {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    t.Cleanup(cancel)
    return ctx
}
```
