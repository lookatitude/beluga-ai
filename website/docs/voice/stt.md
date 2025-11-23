---
title: Speech-to-Text (STT)
sidebar_position: 2
---

# Speech-to-Text (STT)

The STT package provides interfaces and implementations for converting audio to text using various speech recognition providers.

## Overview

The STT package follows the Beluga AI Framework design patterns, providing:

- **Provider abstraction**: Unified interface for multiple STT providers
- **Streaming support**: Real-time transcription with interim results
- **REST fallback**: Automatic fallback to REST API when streaming unavailable
- **Observability**: OTEL metrics and tracing for all operations
- **Error handling**: Comprehensive error codes and retry logic
- **Configuration**: Flexible configuration with validation

## Supported Providers

### Deepgram

- **Features**: WebSocket streaming, REST fallback, high accuracy
- **Models**: Nova-2, Nova, Enhanced, Base
- **Best for**: Real-time applications, high accuracy requirements
- **Streaming**: ✅ Yes (WebSocket)
- **Languages**: 30+ languages

### Google Cloud Speech-to-Text

- **Features**: REST API, gRPC streaming, multiple languages
- **Models**: Standard, Enhanced, Video, Phone Call
- **Best for**: Multi-language support, enterprise features
- **Streaming**: ✅ Yes (gRPC)
- **Languages**: 100+ languages

### Azure Speech Services

- **Features**: WebSocket streaming, REST fallback, custom models
- **Models**: Standard, Neural, Custom
- **Best for**: Enterprise deployments, custom models
- **Streaming**: ✅ Yes (WebSocket)
- **Languages**: 100+ languages

### OpenAI Whisper

- **Features**: REST API, open-source model, batch processing
- **Models**: Whisper-1 (multiple sizes)
- **Best for**: Cost-effective transcription, good accuracy, batch processing
- **Streaming**: ❌ No - Whisper API is designed for batch processing of complete audio files, not real-time streaming. The API requires the full audio file to be uploaded before transcription begins. For real-time streaming, consider Deepgram, Google Cloud, or Azure providers.
- **Languages**: 50+ languages

## Quick Start

### Basic Usage

```go
import (
    "context"
    "os"
    "log"
    
    "github.com/lookatitude/beluga-ai/pkg/voice/stt"
    "github.com/lookatitude/beluga-ai/pkg/voice/stt/providers/deepgram"
)

func main() {
    ctx := context.Background()
    
    // Create STT provider using factory
    config := stt.DefaultConfig()
    config.Provider = "deepgram"
    config.APIKey = os.Getenv("DEEPGRAM_API_KEY")
    config.Model = "nova-2"
    config.Language = "en"
    
    provider, err := stt.NewProvider(ctx, "deepgram", config)
    if err != nil {
        log.Fatal(err)
    }
    
    // Transcribe audio
    audio := []byte{/* your audio data */}
    transcript, err := provider.Transcribe(ctx, audio)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Println("Transcript:", transcript)
}
```

### Direct Provider Usage

```go
import (
    "github.com/lookatitude/beluga-ai/pkg/voice/stt/providers/deepgram"
)

// Create provider directly
sttProvider, err := deepgram.NewDeepgramSTT(ctx, deepgram.Config{
    APIKey: os.Getenv("DEEPGRAM_API_KEY"),
    Model:  "nova-2",
})
```

### Streaming Usage

**Note**: Not all providers support streaming. OpenAI Whisper does not support streaming transcription.

```go
// Start streaming session (only works with Deepgram, Google, Azure)
session, err := provider.StartStreaming(ctx)
if err != nil {
    // OpenAI Whisper will return an error here
    // Error: "OpenAI Whisper API does not support streaming transcription"
    log.Fatal(err)
}
defer session.Close()

// Send audio chunks in a goroutine
go func() {
    for audioChunk := range audioChunks {
        if err := session.SendAudio(ctx, audioChunk); err != nil {
            log.Printf("Error sending audio: %v", err)
            return
        }
    }
    session.Close()
}()

// Receive transcripts
for result := range session.ReceiveTranscript() {
    if result.Error != nil {
        log.Printf("Error: %v", result.Error)
        continue
    }
    
    if result.IsFinal {
        fmt.Printf("Final transcript: %s\n", result.Text)
    } else {
        fmt.Printf("Interim: %s\n", result.Text)
    }
}
```

**Provider Streaming Support**:
- ✅ **Deepgram**: Full streaming support via WebSocket
- ✅ **Google Cloud**: Streaming support via gRPC
- ✅ **Azure**: Streaming support via WebSocket
- ❌ **OpenAI Whisper**: No streaming - designed for batch processing of complete audio files

## Configuration

### Base Configuration

```go
type Config struct {
    Provider        string        // Provider name: "deepgram", "google", "azure", "openai"
    APIKey          string        // API key for authentication
    BaseURL         string        // Custom API endpoint (optional)
    Model           string        // Model name (provider-specific)
    Language        string        // Language code (ISO 639-1, e.g., "en", "es")
    Timeout         time.Duration // Request timeout (default: 30s)
    SampleRate      int           // Audio sample rate: 8000, 16000, 48000
    Channels        int           // Audio channels: 1 (mono) or 2 (stereo)
    EnableStreaming bool          // Enable streaming support (default: true)
    MaxRetries      int           // Maximum retry attempts (default: 3)
    RetryDelay      time.Duration // Retry delay (default: 1s)
    RetryBackoff    float64       // Retry backoff multiplier (default: 2.0)
}
```

### Configuration Options

```go
config := stt.DefaultConfig()
config = stt.WithProvider("deepgram")(config)
config = stt.WithAPIKey("your-api-key")(config)
config = stt.WithModel("nova-2")(config)
config = stt.WithLanguage("en")(config)
config = stt.WithSampleRate(16000)(config)
config = stt.WithTimeout(30*time.Second)(config)
config = stt.WithEnableStreaming(true)(config)
```

### Provider-Specific Configuration

Each provider has additional configuration options. See the [API Reference](../api/packages/voice/stt) for details.

## Error Handling

The STT package uses structured error handling with error codes:

```go
import (
    "errors"
    "github.com/lookatitude/beluga-ai/pkg/voice/stt"
)

if err != nil {
    var sttErr *stt.STTError
    if errors.As(err, &sttErr) {
        switch sttErr.Code {
        case stt.ErrCodeNetworkError:
            // Retryable network error
            // Implement retry logic
        case stt.ErrCodeAuthentication:
            // Authentication failed - check API key
            log.Error("Invalid API key")
        case stt.ErrCodeRateLimit:
            // Rate limit exceeded - wait and retry
            time.Sleep(1 * time.Minute)
            // Retry request
        case stt.ErrCodeTimeout:
            // Request timeout - increase timeout or retry
        case stt.ErrCodeStreamError:
            // Streaming error - reconnect stream
        }
    }
}
```

### Error Codes

| Code | Description | Retryable |
|------|-------------|-----------|
| `ErrCodeInvalidConfig` | Invalid configuration | ❌ No |
| `ErrCodeNetworkError` | Network connectivity issue | ✅ Yes |
| `ErrCodeTimeout` | Request timeout | ✅ Yes |
| `ErrCodeRateLimit` | Rate limit exceeded | ✅ Yes (with delay) |
| `ErrCodeAuthentication` | Authentication failed | ❌ No |
| `ErrCodeStreamError` | Streaming error | ✅ Yes |
| `ErrCodeStreamClosed` | Stream closed | ❌ No |

## Observability

### Metrics

The STT package emits OTEL metrics:

- `stt.transcriptions.total`: Total transcription requests (counter)
- `stt.transcriptions.successful`: Successful transcriptions (counter)
- `stt.transcriptions.failed`: Failed transcriptions (counter)
- `stt.errors.total`: Total errors by error code (counter)
- `stt.streams.total`: Total streaming sessions (counter)
- `stt.transcription.latency`: Transcription latency (histogram)
- `stt.stream.latency`: Stream latency (histogram)
- `stt.streams.active`: Active streaming sessions (gauge)

### Tracing

All operations create OpenTelemetry spans with attributes:

- `provider`: Provider name (e.g., "deepgram")
- `model`: Model name (e.g., "nova-2")
- `language`: Language code (e.g., "en")
- `audio_size`: Audio data size in bytes
- `sample_rate`: Audio sample rate
- `channels`: Number of audio channels

### Logging

Structured logging with context:

```go
// Logs include:
// - provider name
// - model name
// - language
// - operation type
// - duration
// - error (if any)
```

## Provider Registry

Providers are automatically registered via `init()` functions. You can also manually register custom providers:

```go
import (
    "github.com/lookatitude/beluga-ai/pkg/voice/stt"
    sttiface "github.com/lookatitude/beluga-ai/pkg/voice/stt/iface"
)

registry := stt.GetRegistry()
registry.Register("custom-provider", func(config *stt.Config) (sttiface.STTProvider, error) {
    return NewCustomProvider(config)
})

// Use custom provider
provider, err := stt.NewProvider(ctx, "custom-provider", config)
```

## Testing

The package includes comprehensive test utilities:

```go
import (
    "github.com/lookatitude/beluga-ai/pkg/voice/stt"
)

// Create mock provider
mockProvider := stt.NewAdvancedMockSTTProvider("test",
    stt.WithTranscriptions("Hello world", "How are you?"),
    stt.WithStreamingDelay(10*time.Millisecond),
    stt.WithError(nil),
)

// Use in tests
text, err := mockProvider.Transcribe(ctx, audio)
assert.NoError(t, err)
assert.Equal(t, "Hello world", text)
```

## Performance

### Latency

- **Streaming** (Deepgram, Google, Azure): Sub-200ms for interim results
- **REST/Batch** (All providers): 500ms - 2s depending on audio length
- **Long audio files**: 1-5s for complete transcription

### Throughput

- **Streaming**: 1000+ audio chunks per second
- **REST**: Limited by API rate limits
- **Concurrent**: Thread-safe, supports concurrent requests

### Why OpenAI Whisper Doesn't Support Streaming

OpenAI Whisper is **architecturally designed for batch processing** of complete audio files, not real-time streaming:

1. **Full-file processing**: The Whisper model processes the entire audio file at once to provide context-aware transcription with better accuracy
2. **API design**: The OpenAI Whisper API requires uploading the complete audio file before transcription begins - there's no streaming endpoint
3. **Model architecture**: Whisper's transformer-based architecture is optimized for batch inference on complete audio segments, not incremental processing
4. **No streaming endpoint**: OpenAI doesn't provide a streaming API endpoint for Whisper - it's designed as a batch transcription service

**For real-time streaming applications**, use providers that support streaming:
- **Deepgram**: WebSocket-based streaming with low latency (<200ms)
- **Google Cloud Speech-to-Text**: gRPC streaming support
- **Azure Speech Services**: WebSocket streaming support

**Use Whisper when**:
- You have complete audio files to transcribe
- Cost-effectiveness is important (Whisper is typically cheaper)
- You don't need real-time results
- Batch processing fits your workflow
- You want high accuracy with context from the full audio

### Best Practices

1. **Choose the right provider for your use case**:
   - **Real-time streaming**: Use Deepgram, Google Cloud, or Azure
   - **Batch processing**: OpenAI Whisper is cost-effective for complete audio files
   - **Low latency requirements**: Deepgram typically offers the lowest latency
2. **Use streaming for real-time applications**: Lower latency, better UX (not available with Whisper)
3. **Batch small audio files**: More efficient for short clips (works well with Whisper)
4. **Configure appropriate timeouts**: Balance between reliability and responsiveness
5. **Monitor metrics**: Track latency and error rates
6. **Implement retry logic**: Handle transient errors gracefully

## Examples

See the [examples directory](https://github.com/lookatitude/beluga-ai/tree/main/examples/voice) for complete usage examples.

## API Reference

For complete API documentation, see the [STT API Reference](../api/packages/voice/stt).

## Next Steps

- [Text-to-Speech (TTS)](./tts) - Convert text to speech
- [Voice Activity Detection (VAD)](./vad) - Detect voice in audio
- [Session Management](./session) - Manage voice interactions

