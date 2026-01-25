# OpenAI Realtime API

Welcome, colleague! In this integration guide, we're going to integrate OpenAI Realtime API for end-to-end speech-to-speech with Beluga AI's S2S package. OpenAI Realtime provides natural, low-latency voice conversations.

## What you will build

You will configure Beluga AI to use OpenAI Realtime API for speech-to-speech conversations, enabling natural voice interactions with built-in reasoning and tool calling.

## Learning Objectives

- ✅ Configure OpenAI Realtime with Beluga AI S2S
- ✅ Use bidirectional audio streaming
- ✅ Handle real-time audio processing
- ✅ Understand Realtime API features

## Prerequisites

- Go 1.24 or later installed
- Beluga AI Framework installed
- OpenAI API key with Realtime API access
- Understanding of WebSocket streaming

## Step 1: Setup and Installation

Get OpenAI API key with Realtime API access from https://platform.openai.com

Set environment variable:
bash
```bash
export OPENAI_API_KEY="sk-..."
```

## Step 2: Basic Realtime Configuration

Create an OpenAI Realtime S2S provider:
```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/lookatitude/beluga-ai/pkg/voice/s2s"
    "github.com/lookatitude/beluga-ai/pkg/voice/s2s/providers/openai_realtime"
)

func main() {
    ctx := context.Background()

    // Create OpenAI Realtime provider
    config := openai_realtime.Config{
        APIKey: os.Getenv("OPENAI_API_KEY"),
        Model:  "gpt-4o-realtime-preview-2024-10-01",
    }
    
    provider, err := openai_realtime.NewOpenAIRealtimeS2S(ctx, config)
    if err != nil {
        log.Fatalf("Failed: %v", err)
    }

    // Start streaming conversation
    stream, err := provider.StartStreaming(ctx, &s2s.ConversationContext{
        SystemPrompt: "You are a helpful assistant.",
    })
    if err != nil {
        log.Fatalf("Failed: %v", err)
    }

    // Handle audio streams
    // Implementation handles bidirectional audio streaming
}
```

## Step 3: Streaming Audio

Handle bidirectional audio streaming:
```go
func handleStream(ctx context.Context, stream s2s.Stream) error {
    // Send audio to provider
    audioIn := make(chan []byte)
    go func() {
        for audio := range audioIn {
            stream.SendAudio(ctx, audio)
        }
    }()
    
    // Receive audio from provider
    for chunk := range stream.ReceiveAudio() {
        if chunk.Error != nil {
            return chunk.Error
        }
        // Process audio chunk
        processAudio(chunk.Audio)
    }

    
    return nil
}
```

## Step 4: Complete Integration

Here's a complete, production-ready example:
```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"
    "time"

    "github.com/lookatitude/beluga-ai/pkg/voice/s2s"
    "github.com/lookatitude/beluga-ai/pkg/voice/s2s/providers/openai_realtime"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"
)

func main() {
    ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
    defer cancel()

    config := openai_realtime.Config{
        APIKey:       os.Getenv("OPENAI_API_KEY"),
        Model:        "gpt-4o-realtime-preview-2024-10-01",
        Voice:        "alloy",
        Temperature:  0.8,
    }
    
    tracer := otel.Tracer("beluga.voice.s2s.openai_realtime")
    ctx, span := tracer.Start(ctx, "openai_realtime.start",
        trace.WithAttributes(
            attribute.String("model", config.Model),
            attribute.String("voice", config.Voice),
        ),
    )
    defer span.End()
    
    provider, err := openai_realtime.NewOpenAIRealtimeS2S(ctx, config)
    if err != nil {
        span.RecordError(err)
        log.Fatalf("Failed: %v", err)
    }

    stream, err := provider.StartStreaming(ctx, &s2s.ConversationContext{
        SystemPrompt: "You are a helpful assistant.",
    })
    if err != nil {
        span.RecordError(err)
        log.Fatalf("Failed: %v", err)
    }
    defer stream.Close()


    // Handle streaming
    // Implementation handles audio I/O
    fmt.Println("Streaming started")
}
```

## Configuration Options

| Option | Description | Default | Required |
|--------|-------------|---------|----------|
| `APIKey` | OpenAI API key | - | Yes |
| `Model` | Realtime model | `gpt-4o-realtime-preview-2024-10-01` | No |
| `Voice` | Voice selection | `alloy` | No |
| `Temperature` | Response temperature | `0.8` | No |

## Common Issues

### "API key invalid"

**Problem**: Wrong or missing API key.

**Solution**: Verify API key has Realtime API access.

### "WebSocket connection failed"

**Problem**: Network or firewall issue.

**Solution**: Check network connectivity.

## Production Considerations

When using OpenAI Realtime in production:

- **Latency**: Optimize for low latency
- **Cost management**: Monitor API usage
- **Error handling**: Handle connection drops
- **Audio quality**: Configure appropriate audio settings
- **Streaming**: Use streaming for real-time conversations

## Next Steps

Congratulations! You've integrated OpenAI Realtime with Beluga AI. Next, learn how to:

- **[Amazon Nova Bedrock Streaming](./amazon-nova-bedrock-streaming.md)** - Amazon Nova integration
- **[S2S Package Documentation](../../../api-docs/packages/voice/session.md)** - Deep dive into S2S package
- **[Voice Providers Guide](../../../guides/voice-providers.md)** - Voice provider patterns

---

**Ready for more?** Check out the Integrations Index for more integration guides!
