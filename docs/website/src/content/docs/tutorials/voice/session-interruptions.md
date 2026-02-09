---
title: Voice Session Interruptions
description: Implement barge-in detection and playback cancellation so users can interrupt a speaking agent naturally.
---

In natural conversation, speakers interrupt each other. A voice agent that cannot be interrupted feels unresponsive and frustrating. This tutorial demonstrates how to configure a voice session that detects user speech during agent playback and cancels the current response to switch back to listening mode.

## What You Will Build

A voice session with interrupt support that allows users to barge in while the agent is speaking, immediately cancels agent playback, and transitions to listening for the new user input.

## Prerequisites

- Go 1.23+
- Working STT and TTS provider configuration
- Completion of [Sensitivity Tuning](/tutorials/voice/sensitivity-tuning) is recommended

## Step 1: Create a Session with STT and TTS

```go
package main

import (
	"context"
	"log"

	"github.com/lookatitude/beluga-ai/voice/session"
	"github.com/lookatitude/beluga-ai/voice/stt"
	"github.com/lookatitude/beluga-ai/voice/tts"
	_ "github.com/lookatitude/beluga-ai/voice/stt/providers/openai"
	_ "github.com/lookatitude/beluga-ai/voice/tts/providers/openai"
)

func main() {
	ctx := context.Background()

	sttProvider, err := stt.NewProvider(ctx, "openai", stt.DefaultConfig(),
		stt.WithAPIKey("your-key"),
	)
	if err != nil {
		log.Fatalf("create STT: %v", err)
	}

	ttsProvider, err := tts.NewProvider(ctx, "openai", tts.DefaultConfig(),
		tts.WithAPIKey("your-key"),
	)
	if err != nil {
		log.Fatalf("create TTS: %v", err)
	}

	sess, err := session.NewVoiceSession(ctx,
		session.WithSTTProvider(sttProvider),
		session.WithTTSProvider(ttsProvider),
		session.WithConfig(session.DefaultConfig()),
	)
	if err != nil {
		log.Fatalf("create session: %v", err)
	}
	defer sess.Stop(ctx)

	if err := sess.Start(ctx); err != nil {
		log.Fatalf("start session: %v", err)
	}
}
```

## Step 2: Speak with Interruption Support

Use `SayWithOptions` with `AllowInterruptions: true` to enable barge-in. The returned `SayHandle` provides methods to cancel playback and wait for completion.

```go
	opts := session.SayOptions{AllowInterruptions: true}
	handle, err := sess.SayWithOptions(ctx, "Hello, how can I help you today?", opts)
	if err != nil {
		log.Fatalf("say: %v", err)
	}

	// Wait for the utterance to complete or be cancelled
	if err := handle.WaitForPlayout(ctx); err != nil {
		log.Printf("playout interrupted: %v", err)
	}
```

The `SayHandle` interface provides:

| Method              | Description                                     |
|--------------------|-------------------------------------------------|
| `WaitForPlayout(ctx)` | Blocks until the audio finishes playing or is cancelled |
| `Cancel()`          | Immediately stops playback and releases resources |

## Step 3: React to State Changes

Register a state change callback to detect when the session transitions from `speaking` to `listening`. When this happens during active playback, cancel the current `SayHandle`.

```go
	var currentHandle session.SayHandle

	sess.OnStateChanged(func(state session.SessionState) {
		switch state {
		case session.SessionStateListening:
			// User started speaking; cancel agent playback if active
			if currentHandle != nil {
				if err := currentHandle.Cancel(); err != nil {
					log.Printf("cancel handle: %v", err)
				}
				currentHandle = nil
			}
		case session.SessionStateSpeaking:
			log.Println("Agent is speaking")
		case session.SessionStateProcessing:
			log.Println("Processing user input")
		}
	})
```

## Step 4: Complete Interruption Flow

Combine speaking, state monitoring, and handle management into a complete interruption-aware conversation loop.

```go
func agentSpeak(ctx context.Context, sess session.VoiceSession, text string) error {
	opts := session.SayOptions{AllowInterruptions: true}
	handle, err := sess.SayWithOptions(ctx, text, opts)
	if err != nil {
		return fmt.Errorf("say: %w", err)
	}

	// Store handle for interruption callback
	currentHandle = handle

	// Wait for completion or interruption
	if err := handle.WaitForPlayout(ctx); err != nil {
		// Playout was interrupted; this is expected behavior
		log.Println("Playback interrupted by user")
		return nil
	}

	currentHandle = nil
	return nil
}
```

## Step 5: Add VAD for Robust Detection

For more reliable interruption detection, add a VAD provider to the session. VAD detects user speech at the audio level, which is faster and more reliable than waiting for STT to produce a transcript.

```go
import (
	"github.com/lookatitude/beluga-ai/voice/vad"
	_ "github.com/lookatitude/beluga-ai/voice/vad/providers/silero"
)

func main() {
	ctx := context.Background()

	vadProvider, err := vad.NewProvider(ctx, "silero", vad.DefaultConfig(),
		vad.WithThreshold(0.5),
	)
	if err != nil {
		log.Fatalf("create VAD: %v", err)
	}

	sess, err := session.NewVoiceSession(ctx,
		session.WithSTTProvider(sttProvider),
		session.WithTTSProvider(ttsProvider),
		session.WithVADProvider(vadProvider),
		session.WithConfig(session.DefaultConfig()),
	)
	if err != nil {
		log.Fatalf("create session: %v", err)
	}

	// VAD-aware sessions automatically detect barge-in
	// when AllowInterruptions is true
}
```

## Session State Machine

The voice session transitions between these states:

```
 initial ──▶ listening ──▶ processing ──▶ speaking ──▶ listening
                 ▲                                        │
                 │              (interruption)             │
                 └────────────────────────────────────────┘
```

When `AllowInterruptions` is enabled and the user speaks during the `speaking` state, the session transitions directly to `listening`, triggering the `OnStateChanged` callback.

## Verification

1. Start the session and trigger `Say` with `AllowInterruptions: true`.
2. While the agent is speaking, begin speaking yourself (or send audio via `ProcessAudio`).
3. Confirm that `OnStateChanged` fires with `SessionStateListening`.
4. Confirm that `SayHandle.Cancel()` stops playback immediately.
5. Verify the agent processes your new input correctly.

## Next Steps

- [Preemptive Generation](/tutorials/voice/preemptive-generation) -- Start generating replies from interim transcripts
- [Sensitivity Tuning](/tutorials/voice/sensitivity-tuning) -- Tune VAD thresholds for interruption detection
- [Custom Silero VAD](/tutorials/voice/custom-silero-vad) -- Use custom models for precise speech detection
