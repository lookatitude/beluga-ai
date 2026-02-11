---
title: Deepgram Live Streams
description: Integrate Deepgram for real-time speech-to-text with WebSocket streaming in Beluga AI.
---

Deepgram is the default STT choice for real-time voice applications that prioritize low latency. Its Nova-2 model consistently delivers sub-300ms transcription with high accuracy, and interim results let your agent begin processing before the speaker finishes. Deepgram also supports punctuation, speaker diarization, and 36+ languages out of the box. This guide covers integrating the Deepgram STT provider with Beluga AI for live transcription.

## Overview

The Deepgram integration uses the `voice/stt` package to connect to Deepgram's streaming API. It supports multiple models (Nova-2 for accuracy, Base for speed), punctuation, speaker diarization, and interim results.

## Prerequisites

- Go 1.23 or later
- Deepgram API key (from [deepgram.com](https://deepgram.com))

## Installation

```bash
go get github.com/lookatitude/beluga-ai
```

Set your API key:

```bash
export DEEPGRAM_API_KEY="your-api-key"
```

## Configuration

Create a Deepgram STT provider:

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/lookatitude/beluga-ai/voice/stt/providers/deepgram"
)

func main() {
    ctx := context.Background()

    config := deepgram.Config{
        APIKey: os.Getenv("DEEPGRAM_API_KEY"),
        Model:  "nova-2",
    }

    provider, err := deepgram.NewDeepgramSTT(ctx, config)
    if err != nil {
        log.Fatalf("Failed to create provider: %v", err)
    }

    audio := loadAudioData()
    transcript, err := provider.Transcribe(ctx, audio)
    if err != nil {
        log.Fatalf("Transcription failed: %v", err)
    }

    fmt.Printf("Transcript: %s\n", transcript)
}
```

## Usage

### Streaming Transcription

Stream audio chunks for real-time transcription:

```go
func streamTranscription(ctx context.Context, provider *deepgram.DeepgramSTT, audioStream <-chan []byte) (<-chan string, error) {
    transcriptChan := make(chan string, 10)

    go func() {
        defer close(transcriptChan)
        for audioChunk := range audioStream {
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

### Voice Session Integration

Use the Deepgram provider in a voice session:

```go
import "github.com/lookatitude/beluga-ai/voice/session"

sttProvider, err := deepgram.NewDeepgramSTT(ctx, deepgram.Config{
    APIKey: os.Getenv("DEEPGRAM_API_KEY"),
    Model:  "nova-2",
})
if err != nil {
    log.Fatalf("Failed to create STT provider: %v", err)
}

sess, err := session.NewVoiceSession(ctx,
    session.WithSTTProvider(sttProvider),
    session.WithConfig(session.DefaultConfig()),
)
```

### Observability

Add OpenTelemetry tracing:

```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"
)

tracer := otel.Tracer("beluga.voice.stt.deepgram")
ctx, span := tracer.Start(ctx, "deepgram.transcribe",
    trace.WithAttributes(
        attribute.String("model", config.Model),
    ),
)
defer span.End()
```

## Configuration Reference

| Option      | Description           | Default  | Required |
|-------------|-----------------------|----------|----------|
| `APIKey`    | Deepgram API key      | -        | Yes      |
| `Model`     | Deepgram model        | `nova-2` | No       |
| `Language`  | Language code         | `en`     | No       |
| `Punctuate` | Add punctuation       | `true`   | No       |
| `Diarize`   | Speaker diarization   | `false`  | No       |

## Troubleshooting

### API key invalid

Verify your API key is correct and active in the Deepgram dashboard.

### WebSocket connection failed

Check network connectivity and firewall rules. Deepgram uses WebSocket connections that may be blocked by corporate firewalls.

## Advanced Topics

### Production Deployment

- Use WebSocket streaming for all real-time use cases
- Select the appropriate model: `nova-2` for accuracy, `base` for speed
- Monitor API usage through the Deepgram dashboard
- Implement reconnection logic for connection drops
- Optimize chunk size and frequency for your latency requirements

## Related Resources

- [Amazon Transcribe WebSockets](/integrations/transcribe-websockets)
- [Voice Services Overview](/integrations/voice-services)
