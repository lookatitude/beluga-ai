---
title: Session Management
sidebar_position: 8
---

# Session Management

The Session package provides complete lifecycle management for voice interactions between users and AI agents.

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
    "log"
    
    "github.com/lookatitude/beluga-ai/pkg/voice/session"
    "github.com/lookatitude/beluga-ai/pkg/voice/stt/providers/deepgram"
    "github.com/lookatitude/beluga-ai/pkg/voice/tts/providers/openai"
)

func main() {
    ctx := context.Background()
    
    // Create providers
    sttProvider, _ := deepgram.NewDeepgramSTT(ctx, deepgram.Config{
        APIKey: os.Getenv("DEEPGRAM_API_KEY"),
    })
    
    ttsProvider, _ := openai.NewOpenAITTS(ctx, openai.Config{
        APIKey: os.Getenv("OPENAI_API_KEY"),
    })
    
    // Create agent callback
    agentCallback := func(ctx context.Context, transcript string) (string, error) {
        // Process transcript and generate response
        return "Hello! How can I help you?", nil
    }
    
    // Create and start session
    voiceSession, err := session.NewVoiceSession(ctx,
        session.WithSTTProvider(sttProvider),
        session.WithTTSProvider(ttsProvider),
        session.WithAgentCallback(agentCallback),
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
    audio := []byte{/* your audio data */}
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
}
```

## Session States

The session manages the following states:

- **Initial**: Session created but not started
- **Starting**: Session is starting
- **Listening**: Waiting for user input
- **Processing**: Processing user audio
- **Speaking**: Playing agent response
- **Paused**: Session is paused
- **Ended**: Session has ended

## Configuration

### Session Configuration

```go
config := session.DefaultConfig()
config.SessionID = "custom-session-id"  // Auto-generated if empty
config.Timeout = 30 * time.Minute        // Session timeout
config.AutoStart = false                  // Auto-start on creation
config.EnableKeepAlive = true            // Enable keep-alive
config.KeepAliveInterval = 30 * time.Second
config.MaxRetries = 3                    // Maximum retry attempts
config.RetryDelay = 1 * time.Second
```

### Voice Options

```go
voiceSession, err := session.NewVoiceSession(ctx,
    session.WithSTTProvider(sttProvider),
    session.WithTTSProvider(ttsProvider),
    session.WithVADProvider(vadProvider),
    session.WithTurnDetector(turnDetector),
    session.WithTransport(transport),
    session.WithNoiseCancellation(noiseCancellation),
    session.WithAgentCallback(agentCallback),
    session.WithOnStateChanged(func(state sessioniface.SessionState) {
        log.Printf("State changed: %s", state)
    }),
    session.WithConfig(config),
)
```

## Advanced Features

### Error Recovery

Automatic error recovery with configurable retry logic:

```go
config := session.DefaultConfig()
config.MaxRetries = 3
config.RetryDelay = 1 * time.Second
```

### Timeout Handling

Automatic session timeout on inactivity:

```go
config := session.DefaultConfig()
config.Timeout = 5 * time.Minute
```

### Interruption Detection

Detect and handle user interruptions:

```go
// Configure interruption detection
// Interruptions are automatically detected based on:
// - Word count threshold
// - Duration threshold
// - Voice activity
```

### Preemptive Generation

Generate responses before user finishes speaking:

```go
// Enable preemptive generation
// Responses are generated based on interim transcripts
// and used if similar to final transcript
```

### Long Utterance Handling

Handle long user utterances with chunking:

```go
// Long utterances are automatically chunked
// and processed incrementally
```

## Observability

### Metrics

- `session.sessions.total`: Total sessions (counter)
- `session.sessions.active`: Active sessions (gauge)
- `session.audio.processed`: Audio chunks processed (counter)
- `session.responses.generated`: Responses generated (counter)
- `session.errors.total`: Total errors (counter)
- `session.latency`: Session operation latency (histogram)

### Tracing

All operations create OpenTelemetry spans with session context.

## Error Handling

The Session package uses structured error handling:

```go
import (
    "errors"
    "github.com/lookatitude/beluga-ai/pkg/voice/session"
)

if err != nil {
    var sessErr *session.SessionError
    if errors.As(err, &sessErr) {
        switch sessErr.Code {
        case session.ErrCodeSessionNotActive:
            // Session is not active
        case session.ErrCodeProviderError:
            // Provider error
        }
    }
}
```

## Performance

- **Latency**: Sub-200ms for most operations
- **Throughput**: 1000+ audio chunks per second
- **Concurrency**: 100+ concurrent sessions
- **Reliability**: Automatic error recovery and fallback

## API Reference

For complete API documentation, see the [Session API Reference](../api/packages/voice/session).

## Next Steps

- [Voice Agents Overview](./index) - Complete voice agent guide
- [Speech-to-Text (STT)](./stt) - Convert speech to text
- [Text-to-Speech (TTS)](./tts) - Convert text to speech

