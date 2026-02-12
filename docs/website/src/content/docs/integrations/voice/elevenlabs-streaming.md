---
title: ElevenLabs Streaming TTS
description: "Integrate ElevenLabs for natural text-to-speech with streaming, voice cloning, and 29-language support in Beluga AI pipelines."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "ElevenLabs, text-to-speech, TTS streaming, voice cloning, Beluga AI, multilingual TTS, Go TTS provider"
---

ElevenLabs produces some of the most natural-sounding synthetic voices available, making it the go-to choice when voice quality directly affects user experience -- customer-facing phone agents, audiobook narration, or branded voice assistants. Its voice cloning capability lets you create a custom voice from a small audio sample, and multilingual support covers 29 languages with a single model. This guide covers integrating ElevenLabs as a TTS provider within Beluga AI.

## Overview

The ElevenLabs integration uses the `voice/tts` package registry to create a streaming TTS provider. It supports voice selection, stability and similarity tuning, and real-time audio generation with the multilingual v2 model.

## Prerequisites

- Go 1.23 or later
- ElevenLabs API key (from [elevenlabs.io](https://elevenlabs.io))

## Installation

```bash
go get github.com/lookatitude/beluga-ai
```

Set your API key:

```bash
export ELEVENLABS_API_KEY="your-api-key"
```

## Configuration

Create an ElevenLabs TTS provider:

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/lookatitude/beluga-ai/voice/tts"
)

func main() {
    ctx := context.Background()

    config := tts.Config{
        Provider: "elevenlabs",
        APIKey:   os.Getenv("ELEVENLABS_API_KEY"),
        VoiceID:  "21m00Tcm4TlvDq8ikWAM",
    }

    provider, err := tts.NewProvider(ctx, "elevenlabs", config)
    if err != nil {
        log.Fatalf("Failed to create provider: %v", err)
    }

    text := "Hello, this is a test of ElevenLabs text-to-speech."
    audio, err := provider.GenerateSpeech(ctx, text)
    if err != nil {
        log.Fatalf("Generation failed: %v", err)
    }

    fmt.Printf("Generated %d bytes of audio\n", len(audio))
}
```

## Usage

### Voice Selection and Tuning

Customize voice parameters for quality and consistency:

```go
config := tts.Config{
    Provider:        "elevenlabs",
    APIKey:          os.Getenv("ELEVENLABS_API_KEY"),
    VoiceID:         "your-voice-id",
    ModelID:         "eleven_multilingual_v2",
    Stability:       0.5,
    SimilarityBoost: 0.75,
}
```

### Streaming Audio Generation

Stream audio chunks as they are generated for low-latency playback:

```go
func streamAudio(ctx context.Context, provider tts.Provider, text string) (<-chan []byte, error) {
    audioChan := make(chan []byte, 10)

    go func() {
        defer close(audioChan)
        // ElevenLabs supports streaming audio generation.
        // Implementation handles chunked API responses.
    }()

    return audioChan, nil
}
```

### Observability

Add OpenTelemetry tracing:

```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"
)

tracer := otel.Tracer("beluga.voice.tts.elevenlabs")
ctx, span := tracer.Start(ctx, "elevenlabs.generate",
    trace.WithAttributes(
        attribute.String("voice_id", config.VoiceID),
    ),
)
defer span.End()
```

## Configuration Reference

| Option            | Description           | Default                    | Required |
|-------------------|-----------------------|----------------------------|----------|
| `APIKey`          | ElevenLabs API key    | -                          | Yes      |
| `VoiceID`         | Voice identifier      | -                          | Yes      |
| `ModelID`         | Model identifier      | `eleven_multilingual_v2`   | No       |
| `Stability`       | Voice stability (0-1) | `0.5`                      | No       |
| `SimilarityBoost` | Similarity boost (0-1)| `0.75`                     | No       |

## Troubleshooting

### API key invalid

Verify your API key is correct and active in the ElevenLabs dashboard.

### Voice not found

Check that the voice ID exists. List available voices through the ElevenLabs API or dashboard to find valid IDs.

## Advanced Topics

### Production Deployment

- Use voice cloning for custom brand voices
- Enable streaming for low-latency audio generation
- Monitor API usage for cost management
- Balance stability and similarity settings for your use case
- Implement retry logic for transient API failures

## Related Resources

- [Azure Cognitive Services Speech](/integrations/azure-speech)
- [Voice Services Overview](/integrations/voice-services)
