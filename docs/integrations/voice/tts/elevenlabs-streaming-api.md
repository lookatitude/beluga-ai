# ElevenLabs Streaming API

Welcome, colleague! In this integration guide, we're going to integrate ElevenLabs for high-quality text-to-speech with Beluga AI's TTS package. ElevenLabs provides natural-sounding voices with streaming support and voice cloning.

## What you will build

You will configure Beluga AI to use ElevenLabs for text-to-speech generation with streaming support, enabling real-time audio generation with high-quality, natural-sounding voices.

## Learning Objectives

- ✅ Configure ElevenLabs with Beluga AI TTS
- ✅ Use streaming for real-time audio generation
- ✅ Select and customize voices
- ✅ Understand ElevenLabs-specific features

## Prerequisites

- Go 1.24 or later installed
- Beluga AI Framework installed
- ElevenLabs API key
- Understanding of audio streaming

## Step 1: Setup and Installation

Get your ElevenLabs API key from https://elevenlabs.io

Set environment variable:
bash
```bash
export ELEVENLABS_API_KEY="your-api-key"
```

## Step 2: Basic ElevenLabs Configuration

Create an ElevenLabs TTS provider:
```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/lookatitude/beluga-ai/pkg/voice/tts"
)

func main() {
    ctx := context.Background()

    // Create ElevenLabs provider
    // Note: This is a conceptual example
    // Actual implementation would use ElevenLabs SDK
    config := tts.Config{
        Provider: "elevenlabs",
        APIKey:   os.Getenv("ELEVENLABS_API_KEY"),
        VoiceID:  "21m00Tcm4TlvDq8ikWAM", // Default voice
    }
    
    provider, err := tts.NewProvider(ctx, "elevenlabs", config)
    if err != nil {
        log.Fatalf("Failed to create provider: %v", err)
    }

    // Generate speech
    text := "Hello, this is a test of ElevenLabs text-to-speech."
    audio, err := provider.GenerateSpeech(ctx, text)
    if err != nil {
        log.Fatalf("Generation failed: %v", err)
    }


    fmt.Printf("Generated %d bytes of audio\n", len(audio))
}
```

## Step 3: Streaming Audio Generation

Use streaming for real-time audio:
```go
func streamAudio(ctx context.Context, provider tts.Provider, text string) (<-chan []byte, error) {
    audioChan := make(chan []byte, 10)

    

    go func() {
        defer close(audioChan)
        
        // Stream audio chunks as they're generated
        // ElevenLabs supports streaming audio generation
        // Implementation would handle streaming API calls
    }()
    
    return audioChan, nil
}
```

## Step 4: Voice Selection

Select and customize voices:
```text
go
go
config := tts.Config{
    Provider: "elevenlabs",
    APIKey:   os.Getenv("ELEVENLABS_API_KEY"),
    VoiceID:  "your-voice-id",
    ModelID:  "eleven_multilingual_v2",
    Stability: 0.5,
    SimilarityBoost: 0.75,
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

    "github.com/lookatitude/beluga-ai/pkg/voice/tts"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"
)

func main() {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    // Create ElevenLabs provider
    config := tts.Config{
        Provider: "elevenlabs",
        APIKey:   os.Getenv("ELEVENLABS_API_KEY"),
        VoiceID:  "21m00Tcm4TlvDq8ikWAM",
        ModelID:  "eleven_multilingual_v2",
    }
    
    tracer := otel.Tracer("beluga.voice.tts.elevenlabs")
    ctx, span := tracer.Start(ctx, "elevenlabs.generate",
        trace.WithAttributes(
            attribute.String("voice_id", config.VoiceID),
        ),
    )
    defer span.End()
    
    provider, err := tts.NewProvider(ctx, "elevenlabs", config)
    if err != nil {
        span.RecordError(err)
        log.Fatalf("Failed: %v", err)
    }

    text := "Hello, this is ElevenLabs text-to-speech."
    audio, err := provider.GenerateSpeech(ctx, text)
    if err != nil {
        span.RecordError(err)
        log.Fatalf("Failed: %v", err)
    }



    span.SetAttributes(
        attribute.Int("audio_length", len(audio)),
        attribute.Int("text_length", len(text)),
    )

    fmt.Printf("Generated %d bytes of audio\n", len(audio))
}
```

## Configuration Options

| Option | Description | Default | Required |
|--------|-------------|---------|----------|
| `APIKey` | ElevenLabs API key | - | Yes |
| `VoiceID` | Voice identifier | - | Yes |
| `ModelID` | Model identifier | `eleven_multilingual_v2` | No |
| `Stability` | Voice stability | `0.5` | No |
| `SimilarityBoost` | Similarity boost | `0.75` | No |

## Common Issues

### "API key invalid"

**Problem**: Wrong or missing API key.

**Solution**: Verify API key:export ELEVENLABS_API_KEY="your-api-key"
```

### "Voice not found"

**Problem**: Invalid voice ID.

**Solution**: List available voices via API or dashboard.

## Production Considerations

When using ElevenLabs in production:

- **Voice cloning**: Use voice cloning for custom voices
- **Streaming**: Use streaming for low latency
- **Cost management**: Monitor API usage
- **Quality settings**: Balance quality and latency
- **Error handling**: Handle API failures gracefully

## Next Steps

Congratulations! You've integrated ElevenLabs with Beluga AI. Next, learn how to:

- **[Azure Cognitive Services Speech](./azure-cognitive-services-speech.md)** - Azure TTS integration
- **[TTS Package Documentation](../../../api/packages/voice/tts.md)** - Deep dive into TTS package
- **[Voice Providers Guide](../../../guides/voice-providers.md)** - Voice provider patterns

---

**Ready for more?** Check out the [Integrations Index](./README.md) for more integration guides!
