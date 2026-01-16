# Twilio Voice Provider

The Twilio voice provider enables real-time voice interactions with AI agents via Twilio's Programmable Voice API. This provider implements the `VoiceBackend` interface and provides bidirectional audio streaming, webhook handling, and transcription integration.

## Features

- **Real-Time Voice Calls**: Answer and make phone calls with AI agents
- **Bidirectional Audio Streaming**: WebSocket-based audio streaming with mu-law codec
- **Webhook Integration**: Handle Twilio webhook events for call management
- **Transcription & RAG**: Store and search call transcriptions for retrieval-augmented generation
- **Orchestration Support**: Event-driven workflows for complex call flows
- **OTEL Observability**: Comprehensive metrics, tracing, and structured logging
- **Multi-Account Support**: Support multiple Twilio accounts via provider instances

## Quick Start

```go
import (
    "context"
    "github.com/lookatitude/beluga-ai/pkg/voice/backend"
    vbiface "github.com/lookatitude/beluga-ai/pkg/voice/backend/iface"
)

func main() {
    ctx := context.Background()
    
    config := &vbiface.Config{
        Provider:    "twilio",
        PipelineType: vbiface.PipelineTypeSTTTTS,
        STTProvider: "openai",
        TTSProvider: "openai",
        ProviderConfig: map[string]any{
            "account_sid":  "AC...",
            "auth_token":   "your_auth_token",
            "phone_number": "+15551234567",
        },
    }
    
    backend, err := backend.NewBackend(ctx, "twilio", config)
    if err != nil {
        log.Fatal(err)
    }
    
    if err := backend.Start(ctx); err != nil {
        log.Fatal(err)
    }
    defer backend.Stop(ctx)
    
    // Create session
    session, err := backend.CreateSession(ctx, &vbiface.SessionConfig{
        To: "+15559876543",
        AgentCallback: func(ctx context.Context, transcript string) (string, error) {
            return "Hello! How can I help you?", nil
        },
    })
    if err != nil {
        log.Fatal(err)
    }
    
    if err := session.Start(ctx); err != nil {
        log.Fatal(err)
    }
}
```

## Configuration

### TwilioConfig

```go
type TwilioConfig struct {
    AccountSID  string // Twilio Account SID (required)
    AuthToken   string // Twilio Auth Token (required)
    PhoneNumber string // Twilio phone number (required)
    WebhookURL  string // Webhook URL for call events
    AccountName string // Optional identifier for multi-account support
}
```

## Performance Requirements

- **Latency**: <2s from speech completion to agent audio response start (FR-009)
- **Concurrency**: Support 100 concurrent calls (SC-003)
- **Webhooks**: <1s event processing (SC-007)

## Observability

All operations are instrumented with OTEL:
- Metrics: Call quality, latency, success rates
- Tracing: Distributed tracing for all operations
- Logging: Structured logging with trace/span IDs

## Advanced Features

### Session Package Integration

The Twilio provider uses the `pkg/voice/session` package for session management, which provides:
- Automatic error recovery with exponential backoff
- Interruption handling
- Preemptive generation
- Long utterance handling
- State machine with proper transitions
- OTEL metrics and tracing

### S2S (Speech-to-Speech) Support

The provider supports S2S providers for lower latency voice interactions:

```go
config := &vbiface.Config{
    Provider: "twilio",
    PipelineType: vbiface.PipelineTypeS2S,
    S2SProvider: "amazon_nova",
    ProviderConfig: map[string]any{
        "s2s": map[string]any{
            "api_key": "your-api-key",
            "reasoning_mode": "built-in",
        },
    },
}
```

### VAD (Voice Activity Detection)

Optional VAD can be configured to filter non-speech audio:

```go
config := &vbiface.Config{
    Provider: "twilio",
    VADProvider: "silero",
    ProviderConfig: map[string]any{
        "vad": map[string]any{
            "model_path": "/path/to/model",
            "threshold": 0.5,
        },
    },
}
```

### Turn Detection

Turn detection can be configured to identify complete user utterances:

```go
config := &vbiface.Config{
    Provider: "twilio",
    TurnDetectorProvider: "silence",
    ProviderConfig: map[string]any{
        "turn_detection": map[string]any{
            "min_silence_duration": "1s",
            "threshold": 0.3,
        },
    },
}
```

### Memory Integration

Memory configuration is supported for conversation context:

```go
config := &vbiface.Config{
    Provider: "twilio",
    ProviderConfig: map[string]any{
        "memory_config": map[string]any{
            "type": "buffer",
            "window_size": 10,
            "max_token_limit": 2000,
        },
    },
}
```

Note: Memory is managed at the agent level. The session package maintains conversation history internally in AgentContext.

### Noise Cancellation

Optional noise cancellation can be configured:

```go
config := &vbiface.Config{
    Provider: "twilio",
    NoiseCancellationProvider: "rnnoise",
    ProviderConfig: map[string]any{
        "noise_cancellation": map[string]any{
            "model_path": "/path/to/model",
            "noise_reduction_level": 0.7,
        },
    },
}
```

## Transport Evaluation

The Twilio provider uses a custom `TwilioTransportAdapter` for audio transport rather than the generic `pkg/voice/transport` package. This decision was made for the following reasons:

1. **Protocol Compatibility**: Twilio Media Streams use a specific WebSocket protocol with a custom message format (`MediaStreamMessage`) that includes base64-encoded mu-law audio payloads.

2. **Codec Requirements**: Twilio requires mu-law (PCMU) codec conversion, which is handled by the `AudioStream` implementation with dedicated conversion functions.

3. **Reconnection Logic**: Twilio-specific reconnection logic is needed for handling Twilio Media Stream connection failures.

4. **Custom Requirements**: The `AudioStream` implementation provides Twilio-specific features like stream SID management and Twilio-specific error handling.

**Migration Path**: If the generic transport package is extended in the future to support Twilio's protocol, the `TwilioTransportAdapter` could be refactored to use the transport package as a backend. For now, the custom implementation provides the necessary functionality and maintains compatibility with Twilio's Media Streams API.

## Event-Driven Orchestration

The orchestration manager supports event-driven workflows using event handlers:

```go
orchestrator := // ... create orchestrator
manager, err := NewOrchestrationManager(backend, orchestrator)

// Register custom event handler
manager.RegisterEventHandler("call.answered", func(ctx context.Context, event *WebhookEvent) error {
    // Custom handling logic
    return nil
})

// Publish event (automatically triggers registered handlers)
event := &WebhookEvent{
    EventType: "call.answered",
    EventData: map[string]any{"CallSid": "CA..."},
}
err = manager.PublishEvent(ctx, event)
```

## Related Documentation

- [Voice Backend API Contract](../../../../specs/001-twilio-integration/contracts/voice-backend-api.md)
- [Webhook API Contract](../../../../specs/001-twilio-integration/contracts/webhook-api.md)
- [Quickstart Guide](../../../../specs/001-twilio-integration/quickstart.md)
- [Integration Analysis](../../../../docs/twilio-integration-analysis.md)
- [Integration Roadmap](../../../../docs/twilio-integration-roadmap.md)
