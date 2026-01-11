# Voice Backend Package

The `voice/backend` package provides a comprehensive framework for managing real-time voice pipelines for AI agents. It supports multiple voice backend providers (LiveKit, Pipecat, Vocode, Vapi, Cartesia) with a unified interface, enabling seamless integration of voice capabilities into Beluga AI applications.

## Features

- **Multi-Provider Support**: Unified interface for LiveKit, Pipecat, Vocode, Vapi, and Cartesia
- **Pipeline Types**: Support for both STT/TTS (Speech-to-Text/Text-to-Speech) and S2S (Speech-to-Speech) pipelines
- **Real-Time Processing**: Ultra-low latency voice interactions (<300ms for S2S, <500ms for STT/TTS)
- **Multi-User Scalability**: Support for 100+ concurrent voice conversations per backend instance
- **Extensibility Hooks**: Custom authentication, rate limiting, data retention, and telephony integration
- **Custom Processors**: Pluggable audio processors for pre/post-processing
- **OTEL Observability**: Comprehensive metrics, tracing, and structured logging
- **Session Management**: Thread-safe session lifecycle management with isolation
- **Error Recovery**: Automatic retry logic and connection recovery
- **Provider Swapping**: Dynamic switching between providers with minimal code changes

## Quick Start

### Basic Usage

```go
package main

import (
    "context"
    "log"

    "github.com/lookatitude/beluga-ai/pkg/voice/backend"
    vbiface "github.com/lookatitude/beluga-ai/pkg/voice/backend/iface"
)

func main() {
    ctx := context.Background()

    // Create backend configuration
    config := backend.DefaultConfig()
    config.Provider = "mock" // or "livekit", "pipecat", etc.
    config.PipelineType = vbiface.PipelineTypeSTTTTS
    config.STTProvider = "openai"
    config.TTSProvider = "openai"

    // Create backend instance
    voiceBackend, err := backend.NewBackend(ctx, "mock", config)
    if err != nil {
        log.Fatal(err)
    }

    // Start backend
    if err := voiceBackend.Start(ctx); err != nil {
        log.Fatal(err)
    }
    defer voiceBackend.Stop(ctx)

    // Create a voice session
    sessionConfig := &vbiface.SessionConfig{
        UserID:       "user-123",
        Transport:    "websocket",
        PipelineType: vbiface.PipelineTypeSTTTTS,
        AgentCallback: func(ctx context.Context, transcript string) (string, error) {
            // Process transcript and return response
            return "Hello! I received: " + transcript, nil
        },
    }

    session, err := voiceBackend.CreateSession(ctx, sessionConfig)
    if err != nil {
        log.Fatal(err)
    }

    // Start session
    if err := session.Start(ctx); err != nil {
        log.Fatal(err)
    }
    defer session.Stop(ctx)

    // Process audio
    audio := []byte{1, 2, 3, 4, 5}
    if err := session.ProcessAudio(ctx, audio); err != nil {
        log.Fatal(err)
    }
}
```

### Provider-Specific Examples

#### LiveKit Provider

```go
config := backend.DefaultConfig()
config.Provider = "livekit"
config.PipelineType = vbiface.PipelineTypeSTTTTS
config.STTProvider = "openai"
config.TTSProvider = "openai"
config.ProviderConfig = map[string]any{
    "api_key":    "your-livekit-api-key",
    "api_secret": "your-livekit-api-secret",
    "url":        "wss://your-livekit-server.com",
}

voiceBackend, err := backend.NewBackend(ctx, "livekit", config)
```

#### Pipecat Provider

```go
config := backend.DefaultConfig()
config.Provider = "pipecat"
config.PipelineType = vbiface.PipelineTypeSTTTTS
config.STTProvider = "openai"
config.TTSProvider = "openai"
config.ProviderConfig = map[string]any{
    "daily_api_key":      "your-daily-api-key",
    "pipecat_server_url": "ws://localhost:8080/ws",
}

voiceBackend, err := backend.NewBackend(ctx, "pipecat", config)
```

#### S2S Pipeline (Speech-to-Speech)

```go
config := backend.DefaultConfig()
config.Provider = "livekit"
config.PipelineType = vbiface.PipelineTypeS2S
config.S2SProvider = "elevenlabs" // or other S2S provider
config.ProviderConfig = map[string]any{
    "api_key": "your-api-key",
}

voiceBackend, err := backend.NewBackend(ctx, "livekit", config)
```

### Extensibility Examples

#### Custom Authentication Hook

```go
type MyAuthHook struct{}

func (h *MyAuthHook) Authenticate(ctx context.Context, token string, metadata map[string]any) (*vbiface.AuthResult, error) {
    // Validate token and return user info
    return &vbiface.AuthResult{
        UserID:     "user-123",
        Authorized: true,
        Metadata:   map[string]any{"role": "admin"},
    }, nil
}

func (h *MyAuthHook) Authorize(ctx context.Context, userID string, operation string) (bool, error) {
    // Check if user is authorized for operation
    return true, nil
}

config.AuthHook = &MyAuthHook{}
```

#### Custom Rate Limiter

```go
type MyRateLimiter struct{}

func (rl *MyRateLimiter) Allow(ctx context.Context, key string) (bool, error) {
    // Check if request is allowed
    return true, nil
}

func (rl *MyRateLimiter) Wait(ctx context.Context, key string) error {
    // Wait until request is allowed
    return nil
}

config.RateLimiter = &MyRateLimiter{}
```

#### Custom Audio Processor

```go
type NoiseReductionProcessor struct{}

func (p *NoiseReductionProcessor) Process(ctx context.Context, audio []byte, metadata map[string]any) ([]byte, error) {
    // Apply noise reduction algorithm
    return processedAudio, nil
}

func (p *NoiseReductionProcessor) GetName() string {
    return "noise_reduction"
}

func (p *NoiseReductionProcessor) GetOrder() int {
    return 1 // Lower order = earlier processing
}

config.CustomProcessors = []vbiface.CustomProcessor{
    &NoiseReductionProcessor{},
}
```

#### Telephony Hook

```go
type MyTelephonyHook struct{}

func (h *MyTelephonyHook) RouteCall(ctx context.Context, phoneNumber string, metadata map[string]any) (string, error) {
    // Route call to appropriate backend provider
    if strings.HasPrefix(phoneNumber, "+1") {
        return "livekit", nil
    }
    return "pipecat", nil
}

func (h *MyTelephonyHook) HandleSIP(ctx context.Context, message []byte, metadata map[string]any) ([]byte, error) {
    // Handle SIP protocol messages
    return response, nil
}

func (h *MyTelephonyHook) GetCallMetadata(ctx context.Context, callID string) (map[string]any, error) {
    // Retrieve call metadata
    return map[string]any{"caller_id": "1234567890"}, nil
}

config.TelephonyHook = &MyTelephonyHook{}
```

## Architecture

The voice backend package follows a provider-based architecture:

```
┌─────────────────────────────────────────────────────────┐
│                    Voice Backend                        │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐ │
│  │   Registry   │  │   Config     │  │   Metrics    │ │
│  └──────────────┘  └──────────────┘  └──────────────┘ │
└─────────────────────────────────────────────────────────┘
                        │
        ┌───────────────┼───────────────┐
        │               │               │
┌───────▼──────┐ ┌───────▼──────┐ ┌───────▼──────┐
│   LiveKit    │ │   Pipecat     │ │   Vocode     │
│   Provider   │ │   Provider    │ │   Provider   │
└───────┬──────┘ └───────┬──────┘ └───────┬──────┘
        │                 │                 │
        └───────────────┬───────────────────┘
                        │
            ┌───────────▼───────────┐
            │  Pipeline Orchestrator│
            │  (STT/TTS or S2S)    │
            └───────────┬───────────┘
                        │
        ┌───────────────┼───────────────┐
        │               │               │
┌───────▼──────┐ ┌───────▼──────┐ ┌───────▼──────┐
│   Session    │ │   Agent      │ │   Memory     │
│   Manager    │ │   Integration│ │   Integration│
└──────────────┘ └──────────────┘ └──────────────┘
```

### Key Components

- **Registry**: Global provider registry for dynamic provider registration
- **Config**: Unified configuration with provider-specific options
- **Session Manager**: Thread-safe session lifecycle management
- **Pipeline Orchestrator**: Handles STT/TTS and S2S pipeline processing
- **Providers**: Provider-specific implementations (LiveKit, Pipecat, etc.)

## Observability

### OTEL Metrics

The package exposes comprehensive OpenTelemetry metrics:

- `backend.requests.total`: Total voice backend requests
- `backend.errors.total`: Total errors by error code
- `backend.latency.seconds`: Request latency histogram
- `backend.sessions.active`: Active session count (gauge)
- `backend.sessions.total`: Total sessions created (counter)
- `backend.throughput.bytes`: Audio throughput (counter)
- `backend.concurrent.operations`: Concurrent operations (gauge)
- `backend.session.creation.time.seconds`: Session creation time (histogram)
- `backend.throughput.per.session.bytes`: Per-session throughput (counter)

### OTEL Tracing

All public methods include distributed tracing with:
- Span creation for each operation
- Span attributes (provider, session ID, user ID, etc.)
- Error recording with error codes
- Context propagation across operations

### Structured Logging

Structured logging with OTEL context:
- Trace IDs and span IDs in log messages
- Context-aware logging with slog
- Log levels: Debug, Info, Warn, Error

Example:
```go
backend.LogWithOTELContext(ctx, slog.LevelInfo, "Session created",
    "session_id", sessionID,
    "user_id", userID,
    "provider", "livekit")
```

## Error Handling

### Error Codes

The package defines standard error codes for programmatic handling:

- `invalid_config`: Configuration validation failed
- `provider_not_found`: Provider not registered
- `connection_failed`: Backend connection failed
- `connection_timeout`: Connection timeout
- `session_not_found`: Session not found
- `session_limit_exceeded`: Maximum concurrent sessions exceeded
- `rate_limit_exceeded`: Rate limit exceeded
- `authentication_failed`: Authentication failed
- `authorization_failed`: Authorization failed
- `pipeline_error`: Pipeline processing error
- `agent_error`: Agent processing error
- `timeout`: Operation timeout
- `context_canceled`: Context canceled

### Error Handling Patterns

```go
session, err := voiceBackend.CreateSession(ctx, sessionConfig)
if err != nil {
    backendErr := backend.AsError(err)
    if backendErr != nil {
        switch backendErr.Code {
        case backend.ErrCodeSessionLimitExceeded:
            // Handle session limit
        case backend.ErrCodeRateLimitExceeded:
            // Handle rate limit
        default:
            // Handle other errors
        }
    }
    return err
}
```

### Retry Logic

The package includes automatic retry logic for transient errors:

```go
// Retry configuration
config.MaxRetries = 3
config.RetryDelay = 1 * time.Second

// Retryable errors are automatically retried with exponential backoff
```

## Performance

### Latency Targets

- **STT/TTS Pipeline**: <500ms end-to-end latency (SC-001)
- **S2S Pipeline**: <300ms end-to-end latency (SC-005)
- **Session Creation**: <2 seconds (SC-007)

### Scalability

- **Concurrent Sessions**: 100+ concurrent sessions per backend instance (SC-002)
- **Session Isolation**: Each session has independent state and resources
- **Thread Safety**: All operations are thread-safe with proper locking

### Optimization Tips

1. **Connection Pooling**: Backend instances reuse connections across sessions
2. **Audio Buffering**: Buffered channels prevent audio overflow
3. **Concurrent Processing**: Pipeline orchestrator supports concurrent audio processing
4. **Session Capacity**: Configure `MaxConcurrentSessions` to match your infrastructure

## Configuration

### Configuration Options

```go
config := &vbiface.Config{
    // Provider selection
    Provider:     "livekit",
    ProviderConfig: map[string]any{
        "api_key": "your-key",
    },

    // Pipeline configuration
    PipelineType:  vbiface.PipelineTypeSTTTTS,
    STTProvider:   "openai",
    TTSProvider:    "openai",
    S2SProvider:    "elevenlabs", // For S2S pipeline

    // Optional providers
    VADProvider:              "silero",
    TurnDetectionProvider:    "webrtc",
    NoiseCancellationProvider: "rnnoise",

    // Performance
    LatencyTarget:           500 * time.Millisecond,
    Timeout:                 30 * time.Second,
    MaxRetries:              3,
    RetryDelay:              1 * time.Second,
    MaxConcurrentSessions:   100,

    // Observability
    EnableTracing:           true,
    EnableMetrics:           true,
    EnableStructuredLogging: true,

    // Extensibility hooks
    AuthHook:          &MyAuthHook{},
    RateLimiter:       &MyRateLimiter{},
    DataRetentionHook: &MyDataRetentionHook{},
    TelephonyHook:     &MyTelephonyHook{},
    CustomProcessors:  []vbiface.CustomProcessor{...},
}
```

### Functional Options

```go
config := backend.DefaultConfig()
config = backend.WithProvider("livekit")(config)
config = backend.WithPipelineType(vbiface.PipelineTypeSTTTTS)(config)
config = backend.WithSTTProvider("openai")(config)
config = backend.WithTTSProvider("openai")(config)
config = backend.WithMaxConcurrentSessions(100)(config)
```

## Integration with Beluga AI Framework

The voice backend package integrates deeply with other Beluga AI packages:

- **`pkg/agents`**: Agent integration for voice-driven conversations
- **`pkg/memory`**: Conversation context and history management
- **`pkg/orchestration`**: Workflow triggers based on voice events
- **`pkg/retrievers`**: RAG-enabled voice agents
- **`pkg/vectorstores`**: Knowledge base integration
- **`pkg/embeddings`**: Semantic search for voice queries
- **`pkg/multimodal`**: Audio content handling
- **`pkg/prompts`**: Agent prompt management
- **`pkg/chatmodels`**: LLM integration for voice agents

## Testing

The package includes comprehensive test utilities:

```go
// Advanced mock backend
mockBackend := backend.NewAdvancedMockVoiceBackend(
    backend.WithMockError("connection_failed"),
    backend.WithMockDelay(100 * time.Millisecond),
    backend.WithAudioData([]byte{1, 2, 3}),
)

// Concurrent testing
runner := backend.NewConcurrentTestRunner(10, 5, 30*time.Second)
err := runner.Run(ctx, func(ctx context.Context, workerID, iteration int) error {
    // Test operation
    return nil
})

// Load testing
err := backend.RunLoadTest(ctx, voiceBackend, 100, 10, 60*time.Second)
```

## Examples

See the `examples/` directory for complete examples:
- Basic voice agent conversation
- S2S pipeline usage
- Multi-user scenarios
- Custom processor implementation
- Provider swapping

## License

This package is part of the Beluga AI framework and follows the same license.
