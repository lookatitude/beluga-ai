# Deepgram Live Streams

Welcome, colleague! In this integration guide, we're going to integrate Deepgram for real-time speech transcription with Beluga AI's STT package. Deepgram provides high-accuracy, low-latency speech-to-text with WebSocket streaming.

## What you will build

You will configure Beluga AI to use Deepgram for real-time speech transcription with WebSocket streaming, enabling live transcription with interim results and low latency.

## Learning Objectives

- ✅ Configure Deepgram with Beluga AI STT
- ✅ Use WebSocket streaming for real-time transcription
- ✅ Handle interim and final results
- ✅ Understand Deepgram-specific features

## Prerequisites

- Go 1.24 or later installed
- Beluga AI Framework installed
- Deepgram API key
- Understanding of WebSocket streaming

## Step 1: Setup and Installation

Get your Deepgram API key from https://deepgram.com

Set environment variable:
bash
```bash
export DEEPGRAM_API_KEY="your-api-key"
```

## Step 2: Basic Deepgram Configuration

Create a Deepgram STT provider:
```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/lookatitude/beluga-ai/pkg/voice/stt"
    "github.com/lookatitude/beluga-ai/pkg/voice/stt/providers/deepgram"
)

func main() {
    ctx := context.Background()

    // Create Deepgram provider
    config := deepgram.Config{
        APIKey: os.Getenv("DEEPGRAM_API_KEY"),
        Model:  "nova-2",
    }
    
    provider, err := deepgram.NewDeepgramSTT(ctx, config)
    if err != nil {
        log.Fatalf("Failed to create provider: %v", err)
    }

    // Transcribe audio
    audio := []byte{/* your audio data */}
    transcript, err := provider.Transcribe(ctx, audio)
    if err != nil {
        log.Fatalf("Transcription failed: %v", err)
    }


    fmt.Printf("Transcript: %s\n", transcript)
}
```

### Verification

Run the example:
bash
```bash
export DEEPGRAM_API_KEY="your-api-key"
go run main.go
```

You should see the transcription result.

## Step 3: Streaming Transcription

Use WebSocket streaming for real-time transcription:
```go
func streamTranscription(ctx context.Context, provider stt.Provider, audioStream <-chan []byte) (<-chan string, error) {
    transcriptChan := make(chan string, 10)
    
    go func() {
        defer close(transcriptChan)
        
        for audioChunk := range audioStream {
            // Stream transcription
            // Deepgram provider handles streaming internally
            transcript, err := provider.Transcribe(ctx, audioChunk)
            if err != nil {
                log.Printf("Stream error: %v", err)
                continue
            }

            

            transcriptChan <- transcript
        }
    }()
    
    return transcriptChan, nil
}
```

## Step 4: Use with Beluga AI Voice Session

Integrate with voice sessions:
```go
func main() {
    ctx := context.Background()
    
    // Create Deepgram STT
    sttConfig := deepgram.Config{
        APIKey: os.Getenv("DEEPGRAM_API_KEY"),
        Model:  "nova-2",
    }
    
    sttProvider, err := deepgram.NewDeepgramSTT(ctx, sttConfig)
    if err != nil {
        log.Fatalf("Failed: %v", err)
    }

    
    // Use in voice session
    // The voice session will use this provider for transcription
    // See voice session integration guide
}
```

## Step 5: Complete Integration

Here's a complete, production-ready example:
```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"
    "time"

    "github.com/lookatitude/beluga-ai/pkg/voice/stt"
    "github.com/lookatitude/beluga-ai/pkg/voice/stt/providers/deepgram"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"
)

func main() {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    // Create Deepgram provider
    config := deepgram.Config{
        APIKey:     os.Getenv("DEEPGRAM_API_KEY"),
        Model:      "nova-2",
        Language:   "en",
        Punctuate:  true,
        Diarize:    false,
    }
    
    tracer := otel.Tracer("beluga.voice.stt.deepgram")
    ctx, span := tracer.Start(ctx, "deepgram.setup",
        trace.WithAttributes(
            attribute.String("model", config.Model),
        ),
    )
    defer span.End()
    
    provider, err := deepgram.NewDeepgramSTT(ctx, config)
    if err != nil {
        span.RecordError(err)
        log.Fatalf("Failed to create provider: %v", err)
    }

    // Transcribe audio
    audio := []byte{/* audio data */}
    transcript, err := provider.Transcribe(ctx, audio)
    if err != nil {
        span.RecordError(err)
        log.Fatalf("Transcription failed: %v", err)
    }



    span.SetAttributes(
        attribute.String("transcript", transcript),
        attribute.Int("audio_length", len(audio)),
    )

    fmt.Printf("Transcript: %s\n", transcript)
}
```

## Configuration Options

| Option | Description | Default | Required |
|--------|-------------|---------|----------|
| `APIKey` | Deepgram API key | - | Yes |
| `Model` | Deepgram model | `nova-2` | No |
| `Language` | Language code | `en` | No |
| `Punctuate` | Add punctuation | `true` | No |
| `Diarize` | Speaker diarization | `false` | No |

## Common Issues

### "API key invalid"

**Problem**: Wrong or missing API key.

**Solution**: Verify API key:export DEEPGRAM_API_KEY="your-api-key"
```

### "WebSocket connection failed"

**Problem**: Network or firewall issue.

**Solution**: Check network connectivity and firewall rules.

## Production Considerations

When using Deepgram in production:

- **Streaming**: Use WebSocket streaming for real-time
- **Model selection**: Choose appropriate model (nova-2 for accuracy, base for speed)
- **Cost management**: Monitor API usage
- **Error handling**: Handle connection drops gracefully
- **Latency**: Optimize for low-latency use cases

## Next Steps

Congratulations! You've integrated Deepgram with Beluga AI. Next, learn how to:

- **[Amazon Transcribe Audio Websockets](./amazon-transcribe-websockets.md)** - AWS Transcribe integration
- **[STT Package Documentation](../../../api/packages/voice/stt.md)** - Deep dive into STT package
- **[Voice Providers Guide](../../../guides/voice-providers.md)** - Voice provider patterns

---

**Ready for more?** Check out the [Integrations Index](./README.md) for more integration guides!
