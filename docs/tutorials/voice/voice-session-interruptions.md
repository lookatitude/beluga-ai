# Implementing Voice Interruptions

**What you will build:** A voice session that allows users to interrupt the agent while it is speaking. You'll use `SayWithOptions` with `AllowInterruptions: true`, `SayHandle.Cancel()`, and `OnStateChanged` to react when the user starts speaking during playback.

## Learning Objectives

- Create a voice session with STT, TTS, and optional VAD/turn detection
- Use `SayWithOptions` and `AllowInterruptions`
- Cancel playback via `SayHandle.Cancel()` when the user interrupts
- React to state changes with `OnStateChanged`

## Prerequisites

- Go 1.24+
- Beluga AI (`go get github.com/lookatitude/beluga-ai`)
- [Voice Sessions](../../use-cases/voice-sessions.md) and [Barge-In Detection](../../use-cases/voice-turn-barge-in-detection.md) (recommended)

## Step 1: Create a Session with STT and TTS
```go
package main

import (
	"context"
	"log"

	"github.com/lookatitude/beluga-ai/pkg/voice/session"
	"github.com/lookatitude/beluga-ai/pkg/voice/stt"
	"github.com/lookatitude/beluga-ai/pkg/voice/tts"
)

func main() {
	ctx := context.Background()

	sttProv, _ := stt.NewProvider(ctx, "openai", stt.DefaultConfig(), stt.WithAPIKey("your-key"))
	ttsProv, _ := tts.NewProvider(ctx, "openai", tts.DefaultConfig(), tts.WithAPIKey("your-key"))

	sess, err := session.NewVoiceSession(ctx,
		session.WithSTTProvider(sttProv),
		session.WithTTSProvider(ttsProv),
		session.WithConfig(session.DefaultConfig()),
	)
	if err != nil {
		log.Fatalf("session: %v", err)
	}
	defer sess.Stop(ctx)
}
```

## Step 2: Use SayWithOptions and AllowInterruptions

When the agent speaks, set `AllowInterruptions: true` so you can cancel playback if the user interrupts:
```text
go
go
	opts := session.SayOptions{AllowInterruptions: true}
	handle, err := sess.SayWithOptions(ctx, "Hello, how can I help you today?", opts)
	if err != nil {
		log.Fatalf("say: %v", err)
	}
	// Later: handle.Cancel() when user interrupts
	_ = handle
```

## Step 3: React to State and Cancel on Interrupt

Use `OnStateChanged` to detect when the session moves to `listening` (e.g. user started speaking). Cancel the current `SayHandle` if you're in `speaking`:

```
	var currentSayHandle session.SayHandle
	sess.OnStateChanged(func(state session.SessionState) {
		switch state {
		case session.SessionStateListening:
			if currentSayHandle != nil {
				_ = currentSayHandle.Cancel()
				currentSayHandle = nil
			}
		case session.SessionStateSpeaking:
			// Playing back; handle set by SayWithOptions
		}
	})

go
```go
	handle, _ := sess.SayWithOptions(ctx, "Here are your options...", session.SayOptions{AllowInterruptions: true})
	currentSayHandle = handle
	defer func() {
		if currentSayHandle != nil {
			_ = currentSayHandle.Cancel()
		}
	}()
	_ = handle.WaitForPlayout(ctx)
	currentSayHandle = nil
```

## Step 4: Optional VAD + Turn Detection

For more robust interruption detection, add VAD and turn detection. When VAD detects speech during playback, call `SayHandle.Cancel()` and ensure the session switches to listening (e.g. via `ProcessAudio` and your pipeline). See [Barge-In Detection](../../use-cases/voice-turn-barge-in-detection.md).

## Verification

1. Start the session and trigger `Say` with `AllowInterruptions: true`.
2. Simulate user speech (e.g. send audio via `ProcessAudio`) during playback.
3. Confirm `OnStateChanged` fires `listening` and that `SayHandle.Cancel()` stops playback.

## Next Steps

- **[Preemptive Generation](./voice-session-preemptive-generation.md)** — Start generating replies from interim transcripts.
- **[Voice S2S Handling Interruption](../../cookbook/voice-s2s-handling-speech-interruption.md)** — Interruption in S2S pipelines.
- **[Barge-In Detection](../../use-cases/voice-turn-barge-in-detection.md)** — Full use case.
