# Preemptive Generation Strategies

**What you will build:** A voice session that uses **interim** STT results to optionally start generating a reply before the user has finished speaking (preemptive generation). You'll wire streaming STT, an agent or callback, and a strategy for when to use preemptive vs final transcripts.

## Learning Objectives

- Use streaming STT with interim and final results
- Trigger agent or LLM calls on interim transcripts (optional)
- Decide when to use preemptive vs final transcript (e.g. similarity, always, or discard)
- Integrate with `VoiceSession` and `ProcessAudio`

## Prerequisites

- Go 1.24+
- Beluga AI (`go get github.com/lookatitude/beluga-ai`)
- [Voice Sessions](../../../use-cases/voice-sessions.md) and [Real-time STT Streaming](./voice-stt-realtime-streaming.md)

## Step 1: Session with Streaming STT and Agent

Create a voice session with an STT provider that supports streaming (e.g. Deepgram, Google) and an agent or callback. The session's `ProcessAudio` feeds STT; you handle interim vs final in your agent integration.
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

	sttProv, _ := stt.NewProvider(ctx, "deepgram", stt.DefaultConfig(), stt.WithAPIKey("your-key"))
	ttsProv, _ := tts.NewProvider(ctx, "openai", tts.DefaultConfig(), tts.WithAPIKey("your-key"))

	sess, err := session.NewVoiceSession(ctx,
		session.WithSTTProvider(sttProv),
		session.WithTTSProvider(ttsProv),
		session.WithConfig(session.DefaultConfig()),
	)
	if err != nil {
		log.Fatalf("session: %v", err)
	}
	_ = sess
}
```

## Step 2: Preemptive Strategy (Concept)

**Use-if-similar:** Generate a reply from the **interim** transcript. When the **final** transcript arrives, compare it to the interim. If similar (e.g. high overlap), use the preemptive reply to reduce latency; otherwise, generate from the final.

**Always-use:** Use the preemptive reply whenever it exists and the final transcript is available (aggressive latency optimization).

**Discard:** Ignore preemptive replies; always generate from the final transcript (safest, no risk of mismatched answers).

## Step 3: Wire Interim and Final Handlers

If you use an agent callback or custom pipeline:

- **Interim handler:** On each interim transcript, optionally call your agent/LLM and store the result (e.g. in a buffer).
- **Final handler:** On final transcript, either use the preemptive result (if similar or always-use) or generate a new reply from the final.

The session package uses internal `PreemptiveGeneration` and `FinalHandler` for this; see [session README](https://pkg.go.dev/github.com/lookatitude/beluga-ai/pkg/voice/session) and `internal/preemptive.go`. Your integration can follow the same pattern: track last interim, last preemptive reply, and apply your strategy when final arrives.

## Step 4: Example Flow

1. User speaks → `ProcessAudio` → streaming STT.
2. Interim results → optional preemptive agent call → buffer reply.
3. Final result → compare with interim (if use-if-similar); use preemptive or generate from final.
4. `Say` or `SayWithOptions` with the chosen reply.

## Verification

- Run with streaming STT and log interim vs final.
- Measure time-to-first-reply with preemptive on vs off.
- Confirm correct answers when final differs significantly from interim (use-if-similar should fall back to final).

## Next Steps

- **[Voice Session Interruptions](./voice-session-interruptions.md)** — Cancel playback when the user interrupts.
- **[Preemptive Generation Cookbook](../../../cookbook/voice-session-preemptive-generation.md)** — Recipe and code.
- **[Long Utterances Cookbook](../../../cookbook/voice-session-long-utterances.md)** — Chunking and buffering.
