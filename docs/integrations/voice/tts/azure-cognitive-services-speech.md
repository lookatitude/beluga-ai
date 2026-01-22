# Azure Cognitive Services Speech

Welcome, colleague! In this integration guide, we're going to integrate Azure Cognitive Services Speech for text-to-speech with Beluga AI's TTS package. Azure provides neural voices with SSML support and multiple languages.

## What you will build

You will configure Beluga AI to use Azure Speech Services for text-to-speech generation, enabling high-quality neural voices with SSML support and style customization.

## Learning Objectives

- ✅ Configure Azure Speech Services with Beluga AI TTS
- ✅ Use SSML for voice customization
- ✅ Select neural voices
- ✅ Understand Azure-specific features

## Prerequisites

- Go 1.24 or later installed
- Beluga AI Framework installed
- Azure account with Speech Services
- Azure subscription key and region

## Step 1: Setup and Installation

Install Azure Speech SDK:
bash
```bash
go get github.com/Microsoft/cognitive-services-speech-sdk-go
```

Get Azure credentials:
- Subscription key
- Region (e.g., `eastus`)

Set environment variables:
bash
```bash
export AZURE_SPEECH_KEY="your-key"
export AZURE_SPEECH_REGION="eastus"
```

## Step 2: Basic Azure Configuration

Create an Azure TTS provider:
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

    // Create Azure TTS provider
    config := tts.Config{
        Provider: "azure",
        APIKey:   os.Getenv("AZURE_SPEECH_KEY"),
        Region:   os.Getenv("AZURE_SPEECH_REGION"),
        Voice:    "en-US-JennyNeural",
    }
    
    provider, err := tts.NewProvider(ctx, "azure", config)
    if err != nil {
        log.Fatalf("Failed: %v", err)
    }

    // Generate speech
    text := "Hello from Azure Cognitive Services."
    audio, err := provider.GenerateSpeech(ctx, text)
    if err != nil {
        log.Fatalf("Failed: %v", err)
    }

    fmt.Printf("Generated %d bytes\n", len(audio))
}
```

## Step 3: SSML Support

Use SSML for advanced voice control:
```go
ssml := `<speak version="1.0" xmlns="http://www.w3.org/2001/10/synthesis" xml:lang="en-US">
    <voice name="en-US-JennyNeural">
        <prosody rate="fast" pitch="high">
            Hello, this is SSML text-to-speech.
        </prosody>
    </voice>
</speak>`

audio, err := provider.GenerateSpeechFromSSML(ctx, ssml)
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

    "github.com/lookatitude/beluga-ai/pkg/voice/tts"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"
)

func main() {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    config := tts.Config{
        Provider: "azure",
        APIKey:   os.Getenv("AZURE_SPEECH_KEY"),
        Region:   os.Getenv("AZURE_SPEECH_REGION"),
        Voice:    "en-US-JennyNeural",
    }
    
    tracer := otel.Tracer("beluga.voice.tts.azure")
    ctx, span := tracer.Start(ctx, "azure.generate",
        trace.WithAttributes(
            attribute.String("voice", config.Voice),
            attribute.String("region", config.Region),
        ),
    )
    defer span.End()
    
    provider, err := tts.NewProvider(ctx, "azure", config)
    if err != nil {
        span.RecordError(err)
        log.Fatalf("Failed: %v", err)
    }

    text := "Hello from Azure Speech Services."
    audio, err := provider.GenerateSpeech(ctx, text)
    if err != nil {
        span.RecordError(err)
        log.Fatalf("Failed: %v", err)
    }



    span.SetAttributes(
        attribute.Int("audio_length", len(audio)),
    )

    fmt.Printf("Generated %d bytes\n", len(audio))
}
```

## Configuration Options

| Option | Description | Default | Required |
|--------|-------------|---------|----------|
| `APIKey` | Azure Speech key | - | Yes |
| `Region` | Azure region | `eastus` | No |
| `Voice` | Voice name | `en-US-JennyNeural` | No |
| `Language` | Language code | `en-US` | No |

## Common Issues

### "Invalid subscription key"

**Problem**: Wrong API key.

**Solution**: Verify key:export AZURE_SPEECH_KEY="your-key"
```

### "Region not found"

**Problem**: Invalid region.

**Solution**: Use valid Azure region (e.g., `eastus`, `westus2`).

## Production Considerations

When using Azure Speech in production:

- **Neural voices**: Use neural voices for best quality
- **SSML**: Leverage SSML for advanced control
- **Cost optimization**: Monitor usage and optimize
- **Regional deployment**: Deploy close to users
- **Error handling**: Handle API failures gracefully

## Next Steps

Congratulations! You've integrated Azure Speech with Beluga AI. Next, learn how to:

- **[ElevenLabs Streaming API](./elevenlabs-streaming-api.md)** - ElevenLabs integration
- **[TTS Package Documentation](../../../api/packages/voice/tts.md)** - Deep dive into TTS package
- **[Voice Providers Guide](../../../guides/voice-providers.md)** - Voice provider patterns

---

**Ready for more?** Check out the [Integrations Index](./README.md) for more integration guides!
