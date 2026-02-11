---
title: Voice Session Interruptions
description: Implement barge-in detection and playback cancellation so users can interrupt a speaking agent naturally.
---

In natural conversation, speakers interrupt each other. A voice agent that cannot be interrupted feels unresponsive and frustrating -- users expect to be able to cut in with a correction or follow-up without waiting for the agent to finish its entire response. This tutorial demonstrates how to configure a voice session that detects user speech during agent playback and cancels the current response to switch back to listening mode. The approach uses Beluga's session state machine and VAD integration to create a responsive barge-in experience.

## What You Will Build

A voice session with interrupt support that allows users to barge in while the agent is speaking, immediately cancels agent playback, and transitions to listening for the new user input.

## Prerequisites

- Go 1.23+
- Working STT and TTS provider configuration
- Completion of [Sensitivity Tuning](/tutorials/voice/sensitivity-tuning) is recommended

## Step 1: Create a Session with STT and TTS

The voice session orchestrates STT, TTS, and VAD providers into a unified pipeline. By passing providers as functional options, you can swap implementations without changing the session logic. This design follows Beluga's composability principle -- each component is independent and pluggable.

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

Use `SayWithOptions` with `AllowInterruptions: true` to enable barge-in. The returned `SayHandle` provides methods to cancel playback and wait for completion. The handle pattern decouples the initiation of speech from its lifecycle management, allowing you to start playback, register interruption callbacks, and wait for completion in separate parts of your code.

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

Register a state change callback to detect when the session transitions from `speaking` to `listening`. The session state machine manages transitions automatically -- when VAD detects user speech during the `speaking` state, the session transitions to `listening` and fires the callback. This callback is the integration point where your application cancels the current `SayHandle` and prepares to process the new user input.

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

Combine speaking, state monitoring, and handle management into a complete interruption-aware conversation loop. The `agentSpeak` function stores the handle for the state change callback to access, then blocks until either the utterance completes normally or the user interrupts. Treating interruption as a normal event (returning `nil`) rather than an error ensures the conversation loop continues naturally.

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

For more reliable interruption detection, add a VAD provider to the session. VAD detects user speech at the audio level, which is faster and more reliable than waiting for STT to produce a transcript. This matters for interruptions because STT needs several hundred milliseconds of audio to generate even a partial transcript, while VAD can detect speech onset in a single audio frame (typically 20-30ms). The faster detection means the agent stops speaking sooner, creating a more natural interaction.

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

The voice session transitions between these states. The state machine enforces valid transitions and prevents race conditions -- for example, the session cannot transition from `initial` directly to `speaking`, which would bypass the listening phase:

```
 initial ──▶ listening ──▶ processing ──▶ speaking ──▶ listening
                 ▲                                        │
                 │              (interruption)             │
                 └────────────────────────────────────────┘
```

When `AllowInterruptions` is enabled and the user speaks during the `speaking` state, the session transitions directly to `listening`, triggering the `OnStateChanged` callback. This shortcut transition is the mechanism that enables barge-in.

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
