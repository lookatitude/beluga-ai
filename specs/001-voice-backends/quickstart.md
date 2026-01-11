# Quickstart: Voice Backends

**Date**: 2026-01-11  
**Feature**: Voice Backends (001-voice-backends)

## Overview

This quickstart guide demonstrates how to use the Voice Backends package to build real-time voice-enabled AI agents. The package provides a swappable backend infrastructure that orchestrates voice pipelines (STT/TTS or S2S) with sub-500ms latency and multi-user scalability.

## Prerequisites

- Go 1.24.1+
- Beluga AI Framework installed
- Backend provider credentials (LiveKit API keys, Daily.co tokens, etc.)
- Access to voice package components (STT, TTS, VAD, turn detection providers)

## Installation

```bash
go get github.com/lookatitude/beluga-ai/pkg/voice/backend
```

## Basic Usage

### 1. Create a Voice Backend

```go
package main

import (
    "context"
    "log"
    "time"
    
    "github.com/lookatitude/beluga-ai/pkg/voice/backend"
)

func main() {
    ctx := context.Background()
    
    // Get the global registry
    registry := backend.GetRegistry()
    
    // Create a LiveKit backend
    backend, err := registry.Create(ctx, "livekit", &backend.Config{
        Provider: "livekit",
        ProviderConfig: map[string]any{
            "api_key":    "your-livekit-api-key",
            "api_secret": "your-livekit-api-secret",
            "url":        "wss://your-livekit-server.com",
        },
        PipelineType: backend.PipelineTypeSTTTTS,
        STTProvider:  "deepgram",
        TTSProvider:  "elevenlabs",
        LatencyTarget: 500 * time.Millisecond,
        MaxConcurrentSessions: 100,
    })
    if err != nil {
        log.Fatal(err)
    }
    
    // Start the backend
    err = backend.Start(ctx)
    if err != nil {
        log.Fatal(err)
    }
    defer backend.Stop(ctx)
    
    log.Println("Backend started successfully")
}
```

### 2. Create a Voice Session

```go
// Create a session with agent callback
session, err := backend.CreateSession(ctx, &backend.SessionConfig{
    UserID: "user-123",
    Transport: "webrtc",
    ConnectionURL: "wss://your-signaling-server.com",
    AgentCallback: func(ctx context.Context, transcript string) (string, error) {
        // Process transcript and generate response
        // This is where you'd integrate with your LLM/agent
        return "Hello! I heard you say: " + transcript, nil
    },
})
if err != nil {
    log.Fatal(err)
}

// Start the session
err = session.Start(ctx)
if err != nil {
    log.Fatal(err)
}
defer session.Stop(ctx)
```

### 3. Process Audio

```go
// Process incoming audio
audio := []byte{...} // Your audio data (PCM, Opus, etc.)
err = session.ProcessAudio(ctx, audio)
if err != nil {
    log.Fatal(err)
}

// The session will:
// 1. Process audio through STT (if STT/TTS pipeline)
// 2. Route transcript to agent callback
// 3. Generate response via TTS
// 4. Send audio back to user
// All within 500ms target latency
```

## Advanced Usage

### Using S2S Pipeline

```go
// Create backend with S2S pipeline for ultra-low latency
backend, err := registry.Create(ctx, "livekit", &backend.Config{
    Provider: "livekit",
    PipelineType: backend.PipelineTypeS2S,
    S2SProvider: "your-s2s-provider",
    LatencyTarget: 300 * time.Millisecond, // Lower target for S2S
})

// S2S pipeline bypasses text transcription
// Audio flows: User → S2S Provider → Agent (optional) → User
```

### Using Agent Instance Instead of Callback

```go
import (
    "github.com/lookatitude/beluga-ai/pkg/agents"
)

// Create agent instance
agent, err := agents.NewAgent(ctx, &agents.Config{
    Name: "voice-agent",
    LLM: yourLLM,
    Tools: yourTools,
})

// Create session with agent instance
session, err := backend.CreateSession(ctx, &backend.SessionConfig{
    UserID: "user-123",
    Transport: "webrtc",
    ConnectionURL: "wss://your-signaling-server.com",
    AgentInstance: agent, // Use agent instance instead of callback
})
```

### Provider Swapping (Zero Code Changes)

```go
// Application code is provider-agnostic
func createBackend(providerName string) (backend.VoiceBackend, error) {
    registry := backend.GetRegistry()
    return registry.Create(ctx, providerName, getConfigForProvider(providerName))
}

// Switch from LiveKit to Pipecat - only config change
livekitBackend, _ := createBackend("livekit")
pipecatBackend, _ := createBackend("pipecat") // Same code, different provider
```

### Custom Audio Processors

```go
// Implement custom audio processor
type MyAudioProcessor struct{}

func (p *MyAudioProcessor) Process(ctx context.Context, audio []byte, metadata map[string]any) ([]byte, error) {
    // Custom audio processing (e.g., noise cancellation, effects)
    return processedAudio, nil
}

func (p *MyAudioProcessor) GetName() string { return "my-processor" }
func (p *MyAudioProcessor) GetOrder() int   { return 1 }

// Register custom processor in pipeline config
backend, err := registry.Create(ctx, "livekit", &backend.Config{
    Provider: "livekit",
    PipelineType: backend.PipelineTypeSTTTTS,
    CustomProcessors: []backend.CustomProcessor{
        &MyAudioProcessor{},
    },
})
```

### Authentication Hooks

```go
// Implement custom authentication hook
type MyAuthHook struct{}

func (h *MyAuthHook) Authenticate(ctx context.Context, token string, metadata map[string]any) (*backend.AuthResult, error) {
    // Custom authentication logic
    return &backend.AuthResult{UserID: "user-123", Authorized: true}, nil
}

func (h *MyAuthHook) Authorize(ctx context.Context, userID string, operation string) (bool, error) {
    // Custom authorization logic
    return true, nil
}

// Register auth hook in config
backend, err := registry.Create(ctx, "livekit", &backend.Config{
    Provider: "livekit",
    AuthHook: &MyAuthHook{},
})
```

### Rate Limiting

```go
// Implement custom rate limiter
type MyRateLimiter struct{}

func (r *MyRateLimiter) Allow(ctx context.Context, key string) (bool, error) {
    // Custom rate limiting logic
    return true, nil
}

func (r *MyRateLimiter) Wait(ctx context.Context, key string) error {
    // Wait until rate limit allows
    return nil
}

// Register rate limiter in config
backend, err := registry.Create(ctx, "livekit", &backend.Config{
    Provider: "livekit",
    RateLimiter: &MyRateLimiter{},
})
```

## Provider-Specific Examples

### LiveKit Backend

```go
backend, err := registry.Create(ctx, "livekit", &backend.Config{
    Provider: "livekit",
    ProviderConfig: map[string]any{
        "api_key":    os.Getenv("LIVEKIT_API_KEY"),
        "api_secret": os.Getenv("LIVEKIT_API_SECRET"),
        "url":        os.Getenv("LIVEKIT_URL"),
    },
    PipelineType: backend.PipelineTypeSTTTTS,
    STTProvider:  "deepgram",
    TTSProvider:  "elevenlabs",
})
```

### Pipecat Backend

```go
backend, err := registry.Create(ctx, "pipecat", &backend.Config{
    Provider: "pipecat",
    ProviderConfig: map[string]any{
        "daily_api_key": os.Getenv("DAILY_API_KEY"),
        "pipecat_url":   os.Getenv("PIPECAT_SERVER_URL"),
    },
    PipelineType: backend.PipelineTypeSTTTTS,
    STTProvider:  "deepgram",
    TTSProvider:  "elevenlabs",
})
```

## Error Handling

```go
backend, err := registry.Create(ctx, "livekit", config)
if err != nil {
    var backendErr *backend.BackendError
    if errors.As(err, &backendErr) {
        switch backendErr.Code {
        case backend.ErrCodeProviderNotFound:
            log.Fatal("Provider not registered")
        case backend.ErrCodeInvalidConfig:
            log.Fatal("Invalid configuration:", backendErr.Message)
        case backend.ErrCodeConnectionFailed:
            log.Fatal("Connection failed, retrying...")
        default:
            log.Fatal("Unknown error:", backendErr)
        }
    }
    log.Fatal(err)
}
```

## Observability

The package automatically emits OTEL metrics and traces:

```go
// Metrics are automatically recorded:
// - backend.latency: End-to-end latency histogram
// - backend.sessions.active: Active session count
// - backend.errors.total: Error counter
// - backend.throughput: Throughput gauge

// Traces are automatically created for:
// - Backend creation and lifecycle
// - Session creation and management
// - Audio processing pipeline
// - Agent integration calls
```

## Health Checks

```go
// Check backend health
health, err := backend.HealthCheck(ctx)
if err != nil {
    log.Fatal(err)
}

if health.Status == backend.HealthStatusUnhealthy {
    log.Fatal("Backend is unhealthy")
}

fmt.Printf("Backend health: %s, active sessions: %d\n", 
    health.Status, backend.GetActiveSessionCount())
```

## Multi-User Scalability

```go
// Backend automatically handles concurrent sessions
// Per SC-002: Supports 100+ concurrent conversations

for i := 0; i < 100; i++ {
    go func(userID int) {
        session, err := backend.CreateSession(ctx, &backend.SessionConfig{
            UserID: fmt.Sprintf("user-%d", userID),
            Transport: "webrtc",
            ConnectionURL: "wss://your-signaling-server.com",
            AgentCallback: agentCallback,
        })
        if err != nil {
            log.Printf("Failed to create session for user %d: %v", userID, err)
            return
        }
        
        session.Start(ctx)
        defer session.Stop(ctx)
        
        // Process audio for this user
        // Each session is isolated and independent
    }(i)
}
```

## Session Persistence

```go
// Active sessions are automatically persisted (per FR-025)
// Completed sessions are ephemeral (per FR-026)

session, err := backend.CreateSession(ctx, config)
// Session state is persisted while active

session.Stop(ctx)
// After stop, session is ephemeral (no persistence)
```

## Next Steps

1. **Choose a Provider**: Start with LiveKit for full-featured backend, or Pipecat for cost-effective solution
2. **Configure Pipeline**: Choose STT/TTS for traditional pipeline or S2S for ultra-low latency
3. **Integrate Agent**: Use agent callback or agent instance for transcript processing
4. **Add Observability**: Metrics and traces are automatic, but you can add custom instrumentation
5. **Scale**: Backend automatically handles multi-user concurrency up to configured limits

## See Also

- [API Contracts](./contracts/voice-backend-api.md) - Complete API reference
- [Data Model](./data-model.md) - Entity relationships and validation
- [Research](./research.md) - Backend provider research and integration patterns
