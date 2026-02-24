---
title: Azure Speech TTS Integration
description: "Integrate Azure Cognitive Services Speech for neural TTS with 400+ voices, SSML support, and 140 languages in Beluga AI."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Azure Speech, neural TTS, SSML, Beluga AI, Azure Cognitive Services, text-to-speech Go, enterprise voice"
---

Azure Cognitive Services Speech is the strongest TTS option for enterprises already invested in the Microsoft ecosystem. It offers the widest selection of neural voices (400+ across 140 languages), full SSML support for fine-grained prosody control, and deployment within Azure's compliance boundary. Choose Azure Speech when you need enterprise-grade voice synthesis with SSML customization or when regulatory requirements demand that audio processing stays within Azure. This guide covers integrating Azure Speech as a TTS provider within Beluga AI.

## Overview

The Azure Speech integration uses the `voice/tts` package registry to create a TTS provider backed by Azure Cognitive Services. It supports neural voices, SSML markup for prosody control, and multiple output audio formats.

## Prerequisites

- Go 1.23 or later
- Azure account with Speech Services resource
- Azure subscription key and region

## Installation

```bash
go get github.com/lookatitude/beluga-ai
```

Set environment variables:

```bash
export AZURE_SPEECH_KEY="your-key"
export AZURE_SPEECH_REGION="eastus"
```

## Configuration

Create an Azure TTS provider:

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
        Provider: "azure",
        APIKey:   os.Getenv("AZURE_SPEECH_KEY"),
        Region:   os.Getenv("AZURE_SPEECH_REGION"),
        Voice:    "en-US-JennyNeural",
    }

    provider, err := tts.NewProvider(ctx, "azure", config)
    if err != nil {
        log.Fatalf("Failed to create provider: %v", err)
    }

    text := "Hello from Azure Cognitive Services."
    audio, err := provider.GenerateSpeech(ctx, text)
    if err != nil {
        log.Fatalf("Speech generation failed: %v", err)
    }

    fmt.Printf("Generated %d bytes of audio\n", len(audio))
}
```

## Usage

### SSML Support

Use SSML for advanced voice control including prosody, pitch, and rate:

```go
ssml := `<speak version="1.0" xmlns="http://www.w3.org/2001/10/synthesis" xml:lang="en-US">
    <voice name="en-US-JennyNeural">
        <prosody rate="fast" pitch="high">
            Hello, this is SSML text-to-speech.
        </prosody>
    </voice>
</speak>`

audio, err := provider.GenerateSpeechFromSSML(ctx, ssml)
if err != nil {
    log.Fatalf("SSML generation failed: %v", err)
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

tracer := otel.Tracer("beluga.voice.tts.azure")
ctx, span := tracer.Start(ctx, "azure.generate",
    trace.WithAttributes(
        attribute.String("voice", config.Voice),
        attribute.String("region", config.Region),
    ),
)
defer span.End()
```

## Configuration Reference

| Option     | Description         | Default              | Required |
|------------|---------------------|----------------------|----------|
| `APIKey`   | Azure Speech key    | -                    | Yes      |
| `Region`   | Azure region        | `eastus`             | No       |
| `Voice`    | Voice name          | `en-US-JennyNeural`  | No       |
| `Language` | Language code       | `en-US`              | No       |

## Troubleshooting

### Invalid subscription key

Verify the key in the Azure portal under your Speech Services resource. Ensure the key matches the configured region.

### Region not found

Use a valid Azure region such as `eastus`, `westus2`, or `westeurope`. The region must match the region where your Speech Services resource was created.

## Advanced Topics

### Production Deployment

- Use neural voices for the best audio quality
- Leverage SSML for fine-grained prosody and style control
- Monitor usage through the Azure portal for cost management
- Deploy the Speech Services resource in a region close to your users
- Implement retry logic for transient Azure API failures

## Related Resources

- [ElevenLabs Streaming API](/integrations/elevenlabs-streaming)
- [Voice Services Overview](/integrations/voice-services)
