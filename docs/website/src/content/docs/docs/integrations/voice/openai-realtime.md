---
title: OpenAI Realtime Voice API
description: "Integrate OpenAI Realtime API for sub-second speech-to-speech conversations with tool calling in Beluga AI voice pipelines."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "OpenAI Realtime, speech-to-speech, low-latency voice, Beluga AI, GPT-4o voice, WebSocket audio, Go S2S provider"
---

OpenAI Realtime is the lowest-latency option for speech-to-speech conversations that also need tool calling and reasoning capabilities. Because the model processes audio natively rather than transcribing first, it preserves tone and nuance while delivering sub-second response times. Choose this provider when conversational quality and built-in function calling matter more than infrastructure lock-in. This guide covers integrating the OpenAI Realtime S2S provider with Beluga AI for end-to-end voice interactions.

## Overview

The OpenAI Realtime integration uses the `voice/s2s` package to enable bidirectional audio streaming over WebSocket. It supports configurable voices, temperature control, and real-time audio processing.

## Prerequisites

- Go 1.23 or later
- OpenAI API key with Realtime API access
- Understanding of WebSocket streaming concepts

## Installation

```bash
go get github.com/lookatitude/beluga-ai
```

Set your API key:

```bash
export OPENAI_API_KEY="sk-..."
```

## Configuration

Create an OpenAI Realtime S2S provider:

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/lookatitude/beluga-ai/voice/s2s"
    "github.com/lookatitude/beluga-ai/voice/s2s/providers/openai_realtime"
)

func main() {
    ctx := context.Background()

    config := openai_realtime.Config{
        APIKey: os.Getenv("OPENAI_API_KEY"),
        Model:  "gpt-4o-realtime-preview-2024-10-01",
        Voice:  "alloy",
    }

    provider, err := openai_realtime.NewOpenAIRealtimeS2S(ctx, config)
    if err != nil {
        log.Fatalf("Failed to create provider: %v", err)
    }

    stream, err := provider.StartStreaming(ctx, &s2s.ConversationContext{
        SystemPrompt: "You are a helpful assistant.",
    })
    if err != nil {
        log.Fatalf("Failed to start streaming: %v", err)
    }
    defer stream.Close()

    fmt.Println("Realtime streaming started")
}
```

## Usage

### Bidirectional Audio Streaming

Handle concurrent audio input and output:

```go
func handleStream(ctx context.Context, stream s2s.Stream) error {
    audioIn := make(chan []byte)
    go func() {
        for audio := range audioIn {
            if err := stream.SendAudio(ctx, audio); err != nil {
                log.Printf("Send error: %v", err)
                return
            }
        }
    }()

    for chunk := range stream.ReceiveAudio() {
        if chunk.Error != nil {
            return chunk.Error
        }
        processAudio(chunk.Audio)
    }

    return nil
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

tracer := otel.Tracer("beluga.voice.s2s.openai_realtime")
ctx, span := tracer.Start(ctx, "openai_realtime.start",
    trace.WithAttributes(
        attribute.String("model", config.Model),
        attribute.String("voice", config.Voice),
    ),
)
defer span.End()
```

## Configuration Reference

| Option        | Description            | Default                                  | Required |
|---------------|------------------------|------------------------------------------|----------|
| `APIKey`      | OpenAI API key         | -                                        | Yes      |
| `Model`       | Realtime model         | `gpt-4o-realtime-preview-2024-10-01`     | No       |
| `Voice`       | Voice selection        | `alloy`                                  | No       |
| `Temperature` | Response temperature   | `0.8`                                    | No       |

## Troubleshooting

### API key invalid

Verify your API key has Realtime API access enabled. Standard API keys may not include Realtime capabilities.

### WebSocket connection failed

Check network connectivity and firewall rules. The Realtime API uses WebSocket connections that some corporate firewalls may block.

## Advanced Topics

### Production Deployment

- Optimize audio settings for your latency requirements
- Monitor API usage through the OpenAI dashboard for cost management
- Implement reconnection logic for WebSocket drops
- Configure appropriate context timeouts for streaming sessions
- Use streaming mode for all real-time conversation scenarios

## Related Resources

- [Amazon Nova Bedrock Streaming](/docs/integrations/nova-bedrock-streaming)
- [Voice Services Overview](/docs/integrations/voice-services)
