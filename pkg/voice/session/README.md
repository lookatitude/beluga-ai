# Session Package

The Session package provides interfaces and implementations for managing voice interaction sessions between users and AI agents.

## Overview

The Session package follows the Beluga AI Framework design patterns, providing:
- **Complete lifecycle management**: Start, stop, and state management for voice sessions
- **Multi-provider integration**: Seamless integration with STT, TTS, VAD, Turn Detection, Transport, and Noise Cancellation
- **Advanced features**: Error recovery, timeouts, interruption handling, preemptive generation, and long utterance handling
- **Streaming support**: Real-time streaming for STT, TTS, and agent responses
- **Observability**: OTEL metrics and tracing for all operations
- **Error handling**: Comprehensive error codes and retry logic
- **Configuration**: Flexible configuration with validation

## Quick Start

### Basic Usage

```go
import (
    "context"
    "github.com/lookatitude/beluga-ai/pkg/voice/session"
    "github.com/lookatitude/beluga-ai/pkg/voice/iface"
)

// Create session with providers
sttProvider := // ... your STT provider
ttsProvider := // ... your TTS provider

agentCallback := func(ctx context.Context, transcript string) (string, error) {
    // Process transcript and generate response
    return "Hello! How can I help you?", nil
}

voiceSession, err := session.NewVoiceSession(ctx,
    session.WithSTTProvider(sttProvider),
    session.WithTTSProvider(ttsProvider),
    session.WithAgentCallback(agentCallback),
    session.WithConfig(session.DefaultConfig()),
)
if err != nil {
    log.Fatal(err)
}

// Start session
err = voiceSession.Start(ctx)
if err != nil {
    log.Fatal(err)
}

// Process audio
audio := []byte{...} // Your audio data
err = voiceSession.ProcessAudio(ctx, audio)
if err != nil {
    log.Fatal(err)
}

// Say something
handle, err := voiceSession.Say(ctx, "Hello, user!")
if err != nil {
    log.Fatal(err)
}

// Wait for playback
err = handle.WaitForPlayout(ctx)
if err != nil {
    log.Fatal(err)
}

// Stop session
err = voiceSession.Stop(ctx)
if err != nil {
    log.Fatal(err)
}
```

## Configuration

### Session Configuration

```go
type Config struct {
    SessionID         string        // Custom session ID (auto-generated if empty)
    Timeout           time.Duration // Session timeout
    AutoStart         bool          // Auto-start session
    EnableKeepAlive   bool          // Enable keep-alive
    KeepAliveInterval time.Duration // Keep-alive interval
    MaxRetries        int           // Maximum retry attempts
    RetryDelay        time.Duration // Retry delay
}
```

### Voice Options

```go
type VoiceOptions struct {
    STTProvider          iface.STTProvider
    TTSProvider          iface.TTSProvider
    VADProvider          iface.VADProvider
    TurnDetector         iface.TurnDetector
    Transport            iface.Transport
    NoiseCancellation    iface.NoiseCancellation
    AgentCallback        func(ctx context.Context, transcript string) (string, error)
    OnStateChanged       func(state SessionState)
    Config               *Config
}
```

## Session States

The session follows a state machine with the following states:

- `initial`: Session created but not started
- `listening`: Session active, listening for user input
- `processing`: Processing user input (STT, agent processing)
- `speaking`: Playing agent response (TTS)
- `away`: User is away (inactivity detected)
- `ended`: Session ended

## Advanced Features

### Error Recovery

Automatic retry with exponential backoff for transient errors:

```go
recovery := internal.NewErrorRecovery(3, 1*time.Second)
err := recovery.RetryWithBackoff(ctx, "operation", func() error {
    return performOperation()
})
```

### Circuit Breaker

Circuit breaker pattern for provider failures:

```go
breaker := internal.NewCircuitBreaker(5, 2, 30*time.Second)
err := breaker.Call(func() error {
    return provider.Operation()
})
```

### Session Timeout

Automatic session timeout on inactivity:

```go
timeout := internal.NewSessionTimeout(30*time.Minute, func() {
    // Handle timeout
})
timeout.Start()
timeout.UpdateActivity() // Call on user activity
```

### Interruption Handling

Configurable interruption detection:

```go
config := internal.DefaultInterruptionConfig()
config.WordCountThreshold = 3
config.DurationThreshold = 500 * time.Millisecond

detector := internal.NewInterruptionDetector(config)
if detector.CheckInterruption(wordCount, duration) {
    // Handle interruption
}
```

### Preemptive Generation

Generate responses based on interim transcripts:

```go
pg := internal.NewPreemptiveGeneration(true, internal.ResponseStrategyUseIfSimilar)
pg.SetInterimHandler(func(transcript string) {
    // Handle interim transcript
})
pg.SetFinalHandler(func(transcript string) {
    // Handle final transcript
})
```

### Long Utterance Handling

Chunking and buffering for long utterances:

```go
config := internal.DefaultChunkingConfig()
config.ChunkSize = 8192
chunking := internal.NewChunking(config)

chunks := chunking.Chunk(largeAudioData)
```

## Error Handling

The Session package uses structured error handling with error codes:

```go
if err != nil {
    var sessionErr *session.SessionError
    if errors.As(err, &sessionErr) {
        switch sessionErr.Code {
        case session.ErrCodeSessionNotActive:
            // Session not active - need to start first
        case session.ErrCodeTimeout:
            // Operation timeout - retryable
        }
    }
}
```

### Error Codes

- `ErrCodeInvalidConfig`: Invalid configuration
- `ErrCodeInternalError`: Internal processing error
- `ErrCodeInvalidState`: Invalid state transition
- `ErrCodeTimeout`: Operation timeout
- `ErrCodeSessionNotFound`: Session not found
- `ErrCodeSessionAlreadyActive`: Session already active
- `ErrCodeSessionNotActive`: Session not active
- `ErrCodeSessionExpired`: Session expired

## Observability

### Metrics

The Session package emits OTEL metrics:

- `session.started.total`: Total sessions started
- `session.stopped.total`: Total sessions stopped
- `session.active`: Active sessions (up/down counter)
- `session.errors.total`: Total Session errors
- `session.duration`: Session duration histogram
- `session.operation.latency`: Operation latency histogram

### Tracing

All operations create OpenTelemetry spans with attributes:
- `session_id`: Session identifier
- `state`: Current session state
- `provider`: Provider names

## Testing

The package includes comprehensive test utilities:

```go
import "github.com/lookatitude/beluga-ai/pkg/voice/session"

// Create mock session
mockSession := session.NewAdvancedMockSession("test-session",
    session.WithActive(true),
)

// Use in tests
err := mockSession.Start(ctx)
```

## Examples

See the [examples directory](../../../examples/voice/session/) for complete usage examples.

## Performance

- **Latency**: Sub-100ms for session operations
- **Throughput**: Supports 100+ concurrent sessions
- **Concurrency**: Thread-safe, supports concurrent operations

## License

Part of the Beluga AI Framework. See main LICENSE file.

