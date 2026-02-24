---
title: Native S2S with Amazon Nova
description: "Build ultra-low-latency voice conversations in Go using Amazon Nova's Speech-to-Speech capabilities — end-to-end audio processing with sub-second response times."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Beluga AI, Go, tutorial, Amazon Nova, S2S, Speech-to-Speech, voice conversation, low-latency"
---

Speech-to-Speech (S2S) models process audio input and produce audio output directly, bypassing the traditional STT-LLM-TTS pipeline. This architectural shift eliminates the latency overhead of intermediate text conversion and preserves tonal and emotional cues that are lost when speech is transcribed to text and back. This tutorial demonstrates how to configure Amazon Nova's S2S provider, manage bidirectional audio streams, and handle the session lifecycle for real-time voice conversations.

## What You Will Build

A real-time voice conversation system using Amazon Nova's S2S model that processes audio-to-audio with sub-second latency, preserving emotional nuance that is lost in traditional text-based pipelines.

## Prerequisites

- AWS credentials with Bedrock/Nova access
- Go 1.23+
- Basic understanding of bidirectional audio streams

## S2S vs. Traditional Pipeline

Traditional voice agents use a three-stage pipeline where each stage adds latency and loses information:

```
Audio ──▶ STT ──▶ Text ──▶ LLM ──▶ Text ──▶ TTS ──▶ Audio
```

Each transition introduces delay: STT must wait for enough audio to produce a transcript, the LLM processes text, and TTS synthesizes audio from scratch. Emotional cues like hesitation, excitement, or sarcasm are lost during the STT step and cannot be recovered by TTS.

Native S2S processes audio end-to-end in a single step:

```
Audio ──▶ S2S Model ──▶ Audio
```

The S2S approach eliminates latency from intermediate text conversion and preserves tonal and emotional cues from the speaker's voice. The model can also respond with appropriate intonation because it processes the audio signal directly rather than a text approximation.

## Step 1: Initialize the S2S Provider

Beluga uses the registry pattern for S2S providers, the same pattern used across all extensible packages. The blank import triggers the Amazon Nova provider's `init()` function, which registers its factory with the S2S registry. The `s2s.NewProvider` call then looks up the `"amazon_nova"` factory and creates a configured provider instance.

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/lookatitude/beluga-ai/voice/s2s"
	_ "github.com/lookatitude/beluga-ai/voice/s2s/providers/amazon_nova"
)

func main() {
	ctx := context.Background()

	provider, err := s2s.NewProvider(ctx, "amazon_nova", s2s.DefaultConfig(),
		s2s.WithAPIKey(os.Getenv("AWS_ACCESS_KEY_ID")),
		s2s.WithLanguage("en-US"),
		s2s.WithLatencyTarget("low"),
	)
	if err != nil {
		log.Fatalf("create S2S provider: %v", err)
	}

	_ = provider
}
```

The `LatencyTarget` option controls the tradeoff between response speed and output quality. S2S models use internal reasoning steps that affect both the depth of the response and the time before audio output begins. Choosing the right target depends on your application's interaction style:

| Target   | Behavior                                              |
|----------|-------------------------------------------------------|
| `"low"`  | Fastest response, suitable for concierge-style agents |
| `"medium"` | Balanced speed and reasoning depth                  |
| `"high"` | Deepest reasoning, suitable for tutoring or complex tasks |

## Step 2: Start a Streaming Session

S2S requires a bidirectional streaming session because the model processes input and generates output concurrently. Unlike request-response APIs, the session remains open for the duration of the conversation, allowing audio to flow in both directions simultaneously.

The `StreamingS2SProvider` interface extends the base provider with session-based interaction. This type assertion pattern is used because not all S2S providers support streaming -- some may only offer batch processing for specific use cases.

```go
	// Cast to streaming provider for session-based interaction
	streamingProvider, ok := provider.(s2s.StreamingS2SProvider)
	if !ok {
		log.Fatal("provider does not support streaming")
	}

	conversationCtx := &s2s.ConversationContext{
		SessionID:      "session-001",
		ConversationID: "conv-001",
		UserID:         "user-001",
	}

	session, err := streamingProvider.StartStreaming(ctx, conversationCtx,
		s2s.WithVoiceID("Brian"),
		s2s.WithEnableStreaming(true),
	)
	if err != nil {
		log.Fatalf("start streaming: %v", err)
	}
	defer session.Close()
```

## Step 3: Handle Incoming Audio

The session produces audio output chunks on a Go channel. Processing these concurrently in a goroutine ensures that the receive loop does not block the send path, which is essential for maintaining bidirectional flow. Blocking either direction would create backpressure that increases latency or causes dropped frames.

```go
	// Handle incoming model audio in a separate goroutine
	go func() {
		for chunk := range session.ReceiveAudio() {
			if chunk.Error != nil {
				fmt.Printf("receive error: %v\n", chunk.Error)
				continue
			}

			if chunk.IsFinal {
				fmt.Println("[S2S] Response complete")
			}

			// Forward audio to the user's speaker or transport layer
			playAudio(chunk.Audio)
		}
	}()
```

The `AudioOutputChunk` contains both the audio data and metadata needed for playback synchronization:

| Field       | Type    | Description                             |
|------------|---------|------------------------------------------|
| `Audio`    | `[]byte`| Raw audio data in the configured format  |
| `Timestamp`| `int64` | Server-side timestamp                    |
| `IsFinal`  | `bool`  | Whether this is the last chunk in a turn |
| `Error`    | `error` | Non-nil if an error occurred             |

## Step 4: Send User Audio

Pipe audio from the user's microphone (or any audio source) into the session. The send loop runs concurrently with the receive goroutine, creating the full-duplex audio channel that S2S models require.

```go
	// Stream user audio to the model
	for audioChunk := range microphoneStream(ctx) {
		if err := session.SendAudio(ctx, audioChunk); err != nil {
			log.Printf("send audio error: %v", err)
			break
		}
	}
```

For optimal performance, send audio in small frames (20ms at 16kHz = 640 bytes per frame). Smaller frames reduce latency because the model begins processing sooner, but increase packet overhead. The 20ms frame size is the standard tradeoff used by most real-time audio systems, including WebRTC.

## Step 5: Interruption Handling

S2S models handle interruptions naturally because they continuously monitor the input audio stream. When the user begins speaking while the model is generating a response, the model detects the interruption and adjusts its output. This is a fundamental advantage of S2S over traditional pipelines, where the application must explicitly coordinate STT silence detection with TTS playback cancellation -- a complex orchestration problem that often results in clipped responses or delayed interruption recognition.

## Architecture

```
User Microphone                              User Speaker
      │                                           ▲
      ▼                                           │
 SendAudio(ctx, audio)                    ReceiveAudio()
      │                                           │
      ▼                                           │
┌─────────────────── S2S Session ──────────────────┐
│                                                   │
│  Audio In ──▶ Amazon Nova Model ──▶ Audio Out     │
│                                                   │
└───────────────────────────────────────────────────┘
```

## Verification

1. Set AWS credentials in the environment.
2. Run the application and say "Hello, who are you?"
3. Verify the model responds with voice within 500ms.
4. Try interrupting the model while it is speaking and confirm it adapts.

## Next Steps

- [Configuring S2S Reasoning Modes](/docs/tutorials/voice/s2s-reasoning-modes) -- Optimize for speed or quality
- [Voice Session Interruptions](/docs/tutorials/voice/session-interruptions) -- Session-level interruption management
- [Scalable Voice Backend](/docs/tutorials/voice/scalable-backend) -- Production-grade backend with LiveKit
