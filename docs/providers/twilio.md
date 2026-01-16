# Twilio Provider Documentation

**Status**: Production Ready  
**Version**: 1.0.0  
**Date**: 2025-01-07

## Overview

The Twilio provider integration enables voice-enabled AI agents and multi-channel conversational agents using Twilio's Programmable Voice API and Conversations API. This integration provides:

- **Voice-Enabled Agents**: Real-time voice interactions via phone calls with AI agents
- **Multi-Channel Messaging**: SMS and WhatsApp messaging agents with memory persistence
- **Event-Driven Workflows**: Webhook-based orchestration for complex call and message flows
- **Transcription & RAG**: Store and search call transcriptions for retrieval-augmented generation

## Features

### Voice API Integration
- Real-time bidirectional audio streaming via WebSocket (wss://)
- Mu-law codec support for telephony audio
- Call management (create, list, update, delete)
- Webhook handling for call events
- Transcription integration for RAG

### Conversations API Integration
- Multi-channel messaging (SMS, WhatsApp)
- Conversation management with participants
- Message sending and receiving
- Webhook handling for message events
- Memory persistence across sessions

### Observability
- OTEL metrics for call quality, latency, success rates
- OTEL tracing for telephony operations
- Structured logging with trace/span IDs

## Quick Start

### Voice Agent Example

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
    
    config := &vbiface.Config{
        Provider: "twilio",
        // Twilio-specific config
    }
    
    backend, err := backend.NewBackend(ctx, "twilio", config)
    if err != nil {
        log.Fatal(err)
    }
    
    // Start backend
    if err := backend.Start(ctx); err != nil {
        log.Fatal(err)
    }
    defer backend.Stop(ctx)
    
    // Create session
    sessionConfig := &vbiface.SessionConfig{
        UserID:        "user-123",
        Transport:     "websocket",
        ConnectionURL: "wss://example.com/voice",
        PipelineType:  vbiface.PipelineTypeSTTTTS,
        Metadata: map[string]any{
            "to":   "+15559876543",
            "from": "+15551234567",
        },
    }
    
    session, err := backend.CreateSession(ctx, sessionConfig)
    if err != nil {
        log.Fatal(err)
    }
    
    // Use session for voice interactions
    // ...
}
```

### Messaging Agent Example

```go
package main

import (
    "context"
    "log"
    
    "github.com/lookatitude/beluga-ai/pkg/messaging"
    "github.com/lookatitude/beluga-ai/pkg/messaging/iface"
)

func main() {
    ctx := context.Background()
    
    config := &messaging.Config{
        Provider: "twilio",
        // Twilio-specific config
    }
    
    backend, err := messaging.NewBackend(ctx, "twilio", config)
    if err != nil {
        log.Fatal(err)
    }
    
    // Start backend
    if err := backend.Start(ctx); err != nil {
        log.Fatal(err)
    }
    defer backend.Stop(ctx)
    
    // Create conversation
    conversation, err := backend.CreateConversation(ctx, &iface.ConversationConfig{
        FriendlyName: "Customer Support",
    })
    if err != nil {
        log.Fatal(err)
    }
    
    // Send message
    err = backend.SendMessage(ctx, conversation.ConversationSID, &iface.Message{
        Body:    "Hello, how can I help you?",
        Channel: iface.ChannelSMS,
    })
    if err != nil {
        log.Fatal(err)
    }
}
```

## Configuration

### Voice Backend Configuration

```go
type TwilioVoiceConfig struct {
    AccountSID  string `mapstructure:"account_sid" yaml:"account_sid" env:"TWILIO_ACCOUNT_SID" validate:"required"`
    AuthToken   string `mapstructure:"auth_token" yaml:"auth_token" env:"TWILIO_AUTH_TOKEN" validate:"required"`
    PhoneNumber string `mapstructure:"phone_number" yaml:"phone_number" env:"TWILIO_PHONE_NUMBER" validate:"required"`
    WebhookURL  string `mapstructure:"webhook_url" yaml:"webhook_url" env:"TWILIO_WEBHOOK_URL"`
}
```

### Messaging Backend Configuration

```go
type TwilioMessagingConfig struct {
    AccountSID  string `mapstructure:"account_sid" yaml:"account_sid" env:"TWILIO_ACCOUNT_SID" validate:"required"`
    AuthToken   string `mapstructure:"auth_token" yaml:"auth_token" env:"TWILIO_AUTH_TOKEN" validate:"required"`
    WebhookURL  string `mapstructure:"webhook_url" yaml:"webhook_url" env:"TWILIO_WEBHOOK_URL"`
}
```

## Webhook Setup

Twilio requires publicly accessible webhook endpoints. Configure webhooks in your Twilio console:

1. **Voice API Webhooks**: Set status callback URL for call events
2. **Conversations API Webhooks**: Configure webhook URL for message and conversation events

Example webhook handler:

```go
func handleWebhook(w http.ResponseWriter, r *http.Request) {
    // Validate signature
    // Parse event
    // Trigger orchestration workflow
}
```

## Integration with Beluga Packages

### Session Package Integration ✅
The Twilio provider uses `pkg/voice/session` for session management, providing:
- Automatic error recovery with exponential backoff
- Interruption handling
- Preemptive generation
- Long utterance handling
- State machine with proper transitions
- OTEL metrics and tracing

### S2S (Speech-to-Speech) Integration ✅
Support for S2S providers enables lower latency voice interactions:
- Amazon Nova 2 Sonic
- Grok Voice Agent
- Gemini Native Audio
- OpenAI Realtime

### VAD (Voice Activity Detection) Integration ✅
Optional VAD filtering reduces unnecessary processing:
- Silero VAD
- Energy-based VAD
- WebRTC VAD
- ONNX VAD

### Turn Detection Integration ✅
Optional turn detection identifies complete user utterances:
- Silence-based detection
- ONNX-based detection

### Memory Integration ✅
Memory configuration support for conversation context:
- Buffer memory
- Window buffer memory
- Summary memory
- Vector store memory

Note: Memory is managed at the agent level. The session package maintains conversation history internally in AgentContext.

### Noise Cancellation Integration ✅
Optional noise cancellation improves audio quality:
- RNNoise
- Spectral subtraction
- WebRTC noise suppression

### Agents Integration
- Use `pkg/agents` for AI agent responses
- Integrate with voice sessions for real-time conversations
- Integrate with messaging sessions for text-based conversations

### Orchestration Integration ✅
Enhanced orchestration with event-driven workflows:
- Event handlers for call events (answered, ended, failed)
- DAG workflows for complex call flows
- Event publishing and subscription

### RAG Integration
- Store transcriptions in `pkg/vectorstores`
- Use `pkg/embeddings` for embedding generation
- Use `pkg/retrievers` for semantic search

## Performance Requirements

- **Voice Latency**: <2s from speech completion to agent audio response start
- **Messaging**: <5s message processing for 95% of messages
- **Webhooks**: <1s event processing for 99% of events
- **Concurrency**: Support 100 concurrent voice calls

## Error Handling

The provider implements comprehensive error handling:
- Rate limit handling with backoff
- Network error retry logic
- Graceful degradation for failures
- Context cancellation support

## Observability

All operations are instrumented with OTEL:
- Metrics: Call quality, latency, success rates
- Tracing: Distributed tracing for all operations
- Logging: Structured logging with trace/span IDs

## Advanced Configuration

### S2S Provider Configuration

```go
config := &vbiface.Config{
    Provider: "twilio",
    PipelineType: vbiface.PipelineTypeS2S,
    S2SProvider: "amazon_nova",
    ProviderConfig: map[string]any{
        "s2s": map[string]any{
            "api_key": "your-api-key",
            "reasoning_mode": "built-in", // or "external"
            "latency_target": "low",
        },
    },
}
```

### VAD Configuration

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

### Turn Detection Configuration

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

### Noise Cancellation Configuration

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

## Implementation Notes

### Session Adapter Architecture

The Twilio provider uses `TwilioSessionAdapter` which wraps `pkg/voice/session.NewVoiceSession()`:
- Handles mu-law ↔ PCM codec conversion
- Bridges Twilio `AudioStream` to session's `ProcessAudio()`
- Provides `TwilioTransportAdapter` for audio output
- Maintains backward compatibility with existing API

### Transport Implementation

The provider uses a custom `TwilioTransportAdapter` rather than the generic transport package due to:
- Twilio-specific WebSocket protocol requirements
- Mu-law codec conversion needs
- Twilio-specific reconnection logic
- Custom message format (`MediaStreamMessage`)

See [README](../../pkg/voice/providers/twilio/README.md) for detailed transport evaluation documentation.

## Related Documentation

- [Voice Backend API Contract](../../specs/001-twilio-integration/contracts/voice-backend-api.md)
- [Conversational Backend API Contract](../../specs/001-twilio-integration/contracts/conversational-backend-api.md)
- [Webhook API Contract](../../specs/001-twilio-integration/contracts/webhook-api.md)
- [Quickstart Guide](../../specs/001-twilio-integration/quickstart.md)
- [Integration Analysis](../twilio-integration-analysis.md) - ✅ IMPLEMENTED
- [Integration Roadmap](../twilio-integration-roadmap.md) - ✅ COMPLETED
