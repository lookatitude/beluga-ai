# S2S Package

The S2S (Speech-to-Speech) package provides interfaces and implementations for end-to-end speech conversations using various S2S model providers. This package enables natural, real-time speech conversations without explicit intermediate text steps.

## Overview

The S2S package follows the Beluga AI Framework design patterns, providing:
- **Provider abstraction**: Unified interface for multiple S2S providers
- **Streaming support**: Real-time bidirectional streaming for natural conversations
- **Built-in reasoning**: Support for provider-built-in reasoning or external Beluga AI agent integration
- **Multi-provider support**: Automatic fallback between providers
- **Observability**: OTEL metrics and tracing for all operations
- **Error handling**: Comprehensive error codes and silent retry logic
- **Configuration**: Flexible configuration with validation

## Supported Providers

- **Amazon Nova 2 Sonic**: AWS Bedrock Nova 2 Sonic with bidirectional streaming
- **Grok Voice Agent**: xAI Grok Voice Agent (coming soon)
- **Gemini 2.5 Flash Native Audio**: Google Gemini 2.5 Flash with native audio support (coming soon)
- **OpenAI Realtime**: GPT Realtime API with streaming support (coming soon)

## Quick Start

### Basic Usage

```go
import (
    "context"
    "os"
    
    "github.com/lookatitude/beluga-ai/pkg/voice/s2s"
    "github.com/lookatitude/beluga-ai/pkg/voice/session"
)

func main() {
    ctx := context.Background()
    
    // Create S2S provider
    config := s2s.DefaultConfig()
    config.Provider = "amazon_nova"
    config.APIKey = os.Getenv("AWS_ACCESS_KEY_ID")
    
    provider, err := s2s.NewProvider(ctx, "amazon_nova", config)
    if err != nil {
        log.Fatal(err)
    }
    
    // Create voice session with S2S provider
    voiceSession, err := session.NewVoiceSession(ctx,
        session.WithS2SProvider(provider),
    )
    if err != nil {
        log.Fatal(err)
    }
    
    // Start session
    err = voiceSession.Start(ctx)
    if err != nil {
        log.Fatal(err)
    }
    
    // Process audio - S2S provider handles speech-to-speech conversion
    audio := []byte{/* your audio data */}
    err = voiceSession.ProcessAudio(ctx, audio)
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

### Using with Session Package

The S2S package integrates seamlessly with the Voice Agents session package. You can use S2S as an alternative to the traditional STT+TTS pipeline:

```go
// Option 1: Use S2S provider (end-to-end speech)
voiceSession, err := session.NewVoiceSession(ctx,
    session.WithS2SProvider(s2sProvider),
)

// Option 2: Use traditional STT+TTS pipeline
voiceSession, err := session.NewVoiceSession(ctx,
    session.WithSTTProvider(sttProvider),
    session.WithTTSProvider(ttsProvider),
)

// Note: Cannot specify both S2S and STT+TTS
```

## Configuration

### Basic Configuration

```go
config := s2s.DefaultConfig()
config.Provider = "amazon_nova"
config.APIKey = "your-api-key"
config.SampleRate = 24000
config.Channels = 1
config.Language = "en-US"
```

### Advanced Configuration

```go
config := s2s.DefaultConfig()
config.Provider = "amazon_nova"
config.APIKey = "your-api-key"

// Latency settings
config.LatencyTarget = "low" // Options: low, medium, high

// Reasoning mode
config.ReasoningMode = "built-in" // Options: built-in, external

// Fallback providers
config.FallbackProviders = []string{"grok", "gemini"}

// Retry settings
config.MaxRetries = 3
config.RetryDelay = 1 * time.Second
config.RetryBackoff = 2.0

// Concurrent session limits
config.MaxConcurrentSessions = 100
```

### Functional Options

```go
provider, err := s2s.NewProvider(ctx, "amazon_nova", config,
    s2s.WithSampleRate(48000),
    s2s.WithChannels(2),
    s2s.WithLanguage("en-US"),
    s2s.WithLatencyTarget("low"),
    s2s.WithReasoningMode("external"),
    s2s.WithFallbackProviders("grok", "gemini"),
)
```

## Provider-Specific Configuration

### Amazon Nova 2 Sonic

```go
import "github.com/lookatitude/beluga-ai/pkg/voice/s2s/providers/amazon_nova"

config := s2s.DefaultConfig()
config.Provider = "amazon_nova"
config.APIKey = os.Getenv("AWS_ACCESS_KEY_ID")

// Provider-specific settings
config.ProviderSpecific = map[string]any{
    "region": "us-east-1",
    "model": "nova-2-sonic",
    "voice_id": "Ruth",
    "language_code": "en-US",
}

provider, err := s2s.NewProvider(ctx, "amazon_nova", config)
```

## Streaming Support

S2S providers support bidirectional streaming for real-time conversations:

```go
streamingProvider, ok := provider.(s2siface.StreamingS2SProvider)
if !ok {
    log.Fatal("Provider does not support streaming")
}

// Start streaming session
convCtx := s2s.NewConversationContext("session-123")
session, err := streamingProvider.StartStreaming(ctx, convCtx)
if err != nil {
    log.Fatal(err)
}

// Send audio
audio := []byte{/* audio chunk */}
err = session.SendAudio(ctx, audio)
if err != nil {
    log.Fatal(err)
}

// Receive audio
audioCh := session.ReceiveAudio()
for chunk := range audioCh {
    if chunk.Error != nil {
        log.Printf("Error: %v", chunk.Error)
        break
    }
    // Process audio chunk
    processAudio(chunk.Audio)
}

// Close session
err = session.Close()
if err != nil {
    log.Fatal(err)
}
```

## Reasoning Modes

S2S providers support two reasoning modes:

### Built-in Reasoning

Uses the provider's built-in reasoning capabilities (default):

```go
config.ReasoningMode = "built-in"
```

### External Agent Integration

Routes audio through Beluga AI agents for custom reasoning:

```go
config.ReasoningMode = "external"

// Create session with agent integration
voiceSession, err := session.NewVoiceSession(ctx,
    session.WithS2SProvider(provider),
    session.WithAgentInstance(agentInstance, agentConfig),
)
```

## Error Handling

The S2S package implements silent retry logic with automatic recovery:

```go
// Errors are automatically retried with exponential backoff
// User may notice brief pause, but no explicit error unless recovery fails

config.MaxRetries = 3
config.RetryDelay = 1 * time.Second
config.RetryBackoff = 2.0
```

## Observability

### Metrics

S2S operations are automatically instrumented with OTEL metrics:

- `s2s.requests.total`: Total number of S2S requests
- `s2s.requests.latency`: Latency histogram for S2S operations
- `s2s.errors.total`: Total number of errors
- `s2s.provider.usage`: Provider usage counters

### Tracing

Distributed tracing is enabled by default:

```go
config.EnableTracing = true
```

### Logging

Structured logging with context:

```go
config.EnableStructuredLogging = true
```

## Performance

### Latency Targets

- **Low**: Aim for 200ms (60% of interactions)
- **Medium**: Up to 1 second (default)
- **High**: Up to 2 seconds (95% of interactions)

```go
config.LatencyTarget = "low"
```

### Concurrent Sessions

Configure maximum concurrent sessions per provider:

```go
config.MaxConcurrentSessions = 100
```

## Testing

### Mock Provider

Use the advanced mock provider for testing:

```go
mockProvider := s2s.NewAdvancedMockS2SProvider("test",
    s2s.WithAudioOutputs(&s2s.AudioOutput{
        Data: []byte{1, 2, 3, 4, 5},
        Format: s2s.AudioFormat{
            SampleRate: 24000,
            Channels:   1,
            BitDepth:   16,
            Encoding:   "PCM",
        },
        Timestamp: time.Now(),
        Provider:  "test",
        Latency:   100 * time.Millisecond,
    }),
    s2s.WithMockDelay(10 * time.Millisecond),
)
```

## Integration with Voice Agents

The S2S package integrates with the Voice Agents session package:

```go
import (
    "github.com/lookatitude/beluga-ai/pkg/voice/session"
    "github.com/lookatitude/beluga-ai/pkg/voice/s2s"
)

// Create S2S provider
provider, err := s2s.NewProvider(ctx, "amazon_nova", config)

// Create voice session with S2S
voiceSession, err := session.NewVoiceSession(ctx,
    session.WithS2SProvider(provider),
    session.WithAgentInstance(agent, agentConfig), // Optional: for external reasoning
)
```

## Security

### Authentication and Authorization

All S2S providers handle authentication and authorization through their respective SDKs:

- **Amazon Nova**: Uses AWS IAM credentials (access keys, IAM roles, or temporary credentials via STS)
- **Grok Voice Agent**: Uses xAI API keys via `XAI_API_KEY` environment variable
- **Gemini**: Uses Google Cloud API keys or service account credentials
- **OpenAI Realtime**: Uses OpenAI API keys via `OPENAI_API_KEY` environment variable

**Best Practices:**
- Store API keys securely using environment variables or secret management systems
- Use IAM roles for AWS deployments (preferred over access keys)
- Rotate API keys regularly
- Never commit API keys to version control

### Encryption in Transit

All communication with S2S providers uses TLS (Transport Layer Security) encryption:

- **TLS 1.2+**: All provider SDKs enforce TLS encryption for API calls
- **Certificate Validation**: Provider SDKs validate SSL certificates automatically
- **No Configuration Required**: Encryption is handled transparently by provider SDKs

**Note**: The S2S package does not implement custom encryption. All encryption is handled by the underlying provider SDKs, which use industry-standard TLS protocols.

## Examples

See `examples/voice/s2s/` for complete examples:
- `basic_conversation.go`: Basic S2S usage
- `multi_provider.go`: Multi-provider configuration
- `agent_integration.go`: Agent integration examples

### Streaming Examples

#### Basic Streaming Session

```go
import (
    "context"
    "time"
    
    "github.com/lookatitude/beluga-ai/pkg/voice/s2s"
    "github.com/lookatitude/beluga-ai/pkg/voice/s2s/iface"
    "github.com/lookatitude/beluga-ai/pkg/voice/s2s/internal"
)

func streamingExample(ctx context.Context, provider iface.S2SProvider) error {
    // Check if provider supports streaming
    streamingProvider, ok := provider.(iface.StreamingS2SProvider)
    if !ok {
        return fmt.Errorf("provider does not support streaming")
    }
    
    // Create conversation context
    convCtx := &internal.ConversationContext{
        ConversationID: "conv-123",
        SessionID:      "session-456",
    }
    
    // Start streaming session
    session, err := streamingProvider.StartStreaming(ctx, convCtx)
    if err != nil {
        return fmt.Errorf("failed to start streaming: %w", err)
    }
    defer session.Close()
    
    // Send audio chunks
    audioChunks := [][]byte{
        []byte{1, 2, 3, 4},
        []byte{5, 6, 7, 8},
    }
    
    for _, chunk := range audioChunks {
        if err := session.SendAudio(ctx, chunk); err != nil {
            return fmt.Errorf("failed to send audio: %w", err)
        }
        time.Sleep(100 * time.Millisecond) // Small delay between chunks
    }
    
    // Receive audio output
    for {
        select {
        case outputChunk := <-session.ReceiveAudio():
            if outputChunk.Error != nil {
                return fmt.Errorf("streaming error: %w", outputChunk.Error)
            }
            // Process audio output
            processAudioOutput(outputChunk.Audio)
            
            if outputChunk.IsFinal {
                return nil // Stream complete
            }
        case <-ctx.Done():
            return ctx.Err()
        case <-time.After(5 * time.Second):
            return fmt.Errorf("timeout waiting for audio")
        }
    }
}
```

#### Streaming with Error Handling and Retry

```go
func streamingWithRetry(ctx context.Context, provider iface.StreamingS2SProvider) error {
    convCtx := &internal.ConversationContext{
        ConversationID: "conv-123",
        SessionID:      "session-456",
    }
    
    maxRetries := 3
    for attempt := 0; attempt < maxRetries; attempt++ {
        session, err := streamingProvider.StartStreaming(ctx, convCtx)
        if err != nil {
            if attempt < maxRetries-1 {
                time.Sleep(time.Duration(attempt+1) * 100 * time.Millisecond)
                continue
            }
            return fmt.Errorf("failed to start streaming after %d attempts: %w", maxRetries, err)
        }
        
        // Use session...
        defer session.Close()
        
        // Send audio with retry logic
        audio := []byte{1, 2, 3, 4}
        for i := 0; i < 3; i++ {
            err = session.SendAudio(ctx, audio)
            if err == nil {
                break
            }
            if i < 2 {
                time.Sleep(time.Duration(i+1) * 50 * time.Millisecond)
            }
        }
        
        if err != nil {
            return fmt.Errorf("failed to send audio: %w", err)
        }
        
        return nil
    }
    
    return fmt.Errorf("max retries exceeded")
}
```

#### Bidirectional Streaming (OpenAI Realtime)

```go
func bidirectionalStreaming(ctx context.Context, provider iface.StreamingS2SProvider) error {
    convCtx := &internal.ConversationContext{
        ConversationID: "conv-123",
        SessionID:      "session-456",
    }
    
    session, err := provider.StartStreaming(ctx, convCtx)
    if err != nil {
        return err
    }
    defer session.Close()
    
    // Start goroutine to receive audio
    audioOutput := make(chan []byte, 10)
    go func() {
        for chunk := range session.ReceiveAudio() {
            if chunk.Error != nil {
                log.Printf("Streaming error: %v", chunk.Error)
                continue
            }
            audioOutput <- chunk.Audio
            if chunk.IsFinal {
                close(audioOutput)
                return
            }
        }
    }()
    
    // Send audio in a loop
    for {
        select {
        case audio := <-getAudioInput():
            if err := session.SendAudio(ctx, audio); err != nil {
                return fmt.Errorf("failed to send audio: %w", err)
            }
        case output := <-audioOutput:
            processAudioOutput(output)
        case <-ctx.Done():
            return ctx.Err()
        }
    }
}
```

#### Streaming with Context Cancellation

```go
func streamingWithCancellation(ctx context.Context, provider iface.StreamingS2SProvider) error {
    // Create context with timeout
    streamCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
    defer cancel()
    
    convCtx := &internal.ConversationContext{
        ConversationID: "conv-123",
        SessionID:      "session-456",
    }
    
    session, err := provider.StartStreaming(streamCtx, convCtx)
    if err != nil {
        return err
    }
    defer session.Close()
    
    // Send audio with context checking
    audio := []byte{1, 2, 3, 4}
    select {
    case <-streamCtx.Done():
        return streamCtx.Err()
    default:
        if err := session.SendAudio(streamCtx, audio); err != nil {
            if errors.Is(err, context.Canceled) {
                return fmt.Errorf("stream cancelled: %w", err)
            }
            return err
        }
    }
    
    return nil
}
```

### Notes on Streaming Behavior

1. **One-Way Streaming (Nova, Gemini, Grok)**: These providers use one-way streaming APIs. `SendAudio()` creates new streaming requests with accumulated audio, providing functional (though not truly bidirectional) streaming.

2. **Bidirectional Streaming (OpenAI Realtime)**: Supports true bidirectional streaming where audio can be sent and received simultaneously.

3. **Stream Restart Optimization**: The implementation includes debouncing and retry logic to optimize stream restarts when using one-way streaming providers.

4. **Error Recovery**: Automatic retry with exponential backoff is built into the streaming implementation.

## Package Design

The S2S package follows Beluga AI Framework design patterns:
- **Interfaces**: `pkg/voice/s2s/iface/` - S2SProvider, StreamingS2SProvider
- **Providers**: `pkg/voice/s2s/providers/` - Provider implementations
- **Configuration**: `config.go` - Config struct with validation
- **Errors**: `errors.go` - S2SError with Op/Err/Code pattern
- **Metrics**: `metrics.go` - OTEL metrics implementation
- **Registry**: `registry.go` - Global provider registry

## See Also

- [Voice Agents Session Package](../session/README.md)
- [Voice Agents Overview](../../README.md)
- [Beluga AI Framework Documentation](../../../../docs/README.md)
