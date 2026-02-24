---
title: Real-Time STT Streaming Tutorial
description: "Implement real-time speech-to-text streaming in Go using Deepgram with WebSocket sessions — process audio incrementally with interim and final transcripts."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Beluga AI, Go, tutorial, STT streaming, Deepgram, WebSocket, real-time, transcription"
---

Real-time STT streaming enables voice applications to process audio incrementally, delivering interim and final transcripts with minimal delay. Unlike batch transcription, which waits for the complete audio before producing output, streaming STT processes audio as it arrives and returns partial results in real time. This is essential for voice agents because the agent can begin understanding user intent before the user finishes speaking, enabling features like preemptive generation and turn-taking. This tutorial demonstrates how to configure a streaming STT provider, open a session, and handle transcripts as they arrive.

## What You Will Build

A streaming speech-to-text pipeline using the Deepgram provider that processes audio chunks in real time and delivers both interim (partial) and final transcripts over a WebSocket connection.

## Prerequisites

- Deepgram API key
- Go 1.23+
- Basic understanding of audio buffers and PCM encoding

## Step 1: Initialize the STT Provider

Use the registry pattern to create a Deepgram STT provider with streaming enabled. The blank import `_ "github.com/lookatitude/beluga-ai/voice/stt/providers/deepgram"` triggers the provider's `init()` function, which registers the `"deepgram"` factory with the STT registry. This is the same registration pattern used across all Beluga extensible packages.

The `WithEnableStreaming(true)` option tells the provider to establish a persistent WebSocket connection for continuous audio processing, rather than using the batch HTTP API.

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/lookatitude/beluga-ai/voice/stt"
	_ "github.com/lookatitude/beluga-ai/voice/stt/providers/deepgram"
)

func main() {
	ctx := context.Background()

	// Create a Deepgram STT provider with streaming and interim results
	provider, err := stt.NewProvider(ctx, "deepgram", stt.DefaultConfig(),
		stt.WithAPIKey(os.Getenv("DEEPGRAM_API_KEY")),
		stt.WithModel("nova-2"),
		stt.WithLanguage("en-US"),
		stt.WithSampleRate(16000),
		stt.WithEnableStreaming(true),
	)
	if err != nil {
		log.Fatalf("create STT provider: %v", err)
	}

	_ = provider
}
```

## Step 2: Start a Streaming Session

A streaming session establishes a persistent WebSocket connection for continuous audio input and transcript output. The session remains open for the duration of the conversation, avoiding the overhead of establishing a new connection for each utterance.

```go
	// Open a streaming session
	session, err := provider.StartStreaming(ctx)
	if err != nil {
		log.Fatalf("start streaming: %v", err)
	}
	defer session.Close()
```

The `StartStreaming` method returns a `StreamingSession` that supports bidirectional communication: you send audio chunks in, and receive `TranscriptResult` values out. This bidirectional pattern is why Beluga uses Go channels for STT streaming rather than `iter.Seq2` -- both the send and receive directions operate concurrently.

## Step 3: Handle Transcripts

Transcripts arrive on a Go channel. Each result indicates whether it is an interim (partial) or final transcript. Interim transcripts update continuously as the STT model refines its prediction; final transcripts represent the model's confirmed output for a segment of speech. Processing these in a goroutine ensures the receive loop does not block the audio send path.

```go
	// Process transcripts in a separate goroutine
	go func() {
		for result := range session.ReceiveTranscript() {
			if result.Error != nil {
				fmt.Printf("transcript error: %v\n", result.Error)
				continue
			}

			if result.IsFinal {
				fmt.Printf("\n[Final] %s (confidence: %.2f)\n", result.Text, result.Confidence)
			} else {
				fmt.Printf("\r[Interim] %s", result.Text)
			}
		}
	}()
```

The `TranscriptResult` struct contains both the transcript text and metadata about the recognition:

| Field        | Type      | Description                              |
|-------------|-----------|------------------------------------------|
| `Text`      | `string`  | Transcribed text                         |
| `IsFinal`   | `bool`    | Whether this is a final or interim result |
| `Confidence`| `float64` | Model confidence score (0.0 to 1.0)      |
| `Language`  | `string`  | Detected language code                   |
| `Error`     | `error`   | Non-nil if an error occurred             |

## Step 4: Send Audio Data

Stream audio chunks to the session. In production, these come from a microphone, WebRTC track, or transport layer. The frame size determines the tradeoff between latency and efficiency: smaller frames reduce latency (the model receives data sooner) but increase overhead (more WebSocket messages).

```go
	// Send audio chunks (e.g., 20ms frames of 16kHz mono PCM)
	// Each frame: 16000 samples/sec * 0.020 sec * 2 bytes/sample = 640 bytes
	audioFrame := make([]byte, 640)

	err = session.SendAudio(ctx, audioFrame)
	if err != nil {
		log.Fatalf("send audio: %v", err)
	}
```

For continuous streaming from a microphone, wrap the send loop:

```go
	// Continuous streaming from an audio source
	for frame := range audioSource {
		if err := session.SendAudio(ctx, frame); err != nil {
			log.Printf("send error: %v", err)
			break
		}
	}
```

## Step 5: Close the Session

Always close the session to flush pending transcripts and release the WebSocket connection. The `Close` method signals the server that no more audio will be sent, which triggers the server to finalize any pending transcriptions and return remaining results before terminating the connection.

```go
	// Close flushes any remaining audio and shuts down the connection
	if err := session.Close(); err != nil {
		log.Printf("close session: %v", err)
	}
```

## Architecture

The streaming STT pipeline follows this flow:

```
Audio Source ──▶ SendAudio() ──▶ [WebSocket] ──▶ Deepgram API
                                                      │
                                                      ▼
Application ◀── ReceiveTranscript() ◀── [WebSocket] ◀─┘
```

Beluga's STT interface uses Go channels for streaming rather than `iter.Seq2`, because audio processing requires true bidirectional communication where the sender and receiver operate concurrently. The `iter.Seq2` pattern is used for unidirectional streaming (like LLM token output) where the consumer pulls values from a producer.

## Verification

1. Set the `DEEPGRAM_API_KEY` environment variable.
2. Run the application and pipe audio from a microphone or WAV file.
3. Confirm that interim transcripts update in place and final transcripts appear on new lines.
4. Verify that closing the session produces any remaining final transcripts.

## Next Steps

- [Fine-tuning Whisper for Industry Terms](/docs/tutorials/voice/whisper-finetuning) -- Improve accuracy for specialized vocabulary
- [Voice Session Interruptions](/docs/tutorials/voice/session-interruptions) -- Combine STT with full session management
- [Custom Silero VAD](/docs/tutorials/voice/custom-silero-vad) -- Add voice activity detection to filter silence
