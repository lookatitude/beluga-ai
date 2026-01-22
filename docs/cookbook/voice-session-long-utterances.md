---
title: "Handling Long Utterances in Voice Sessions"
package: "voice/session"
category: "voice"
complexity: "intermediate"
---

# Handling Long Utterances in Voice Sessions

## Problem

Users sometimes speak for a long time without clear turn boundaries. You need to chunk or buffer audio, handle multi-sentence input, and avoid truncation or timeouts while keeping the session responsive.

## Solution

Use configurable chunking and buffering before passing audio to the session's `ProcessAudio`. Send STT-friendly chunks (e.g. 2–5 s) and optionally use turn detection to decide when to flush. For very long monologues, aggregate interim transcripts and trigger agent processing at sensible boundaries (e.g. sentence end, max duration) so the session can respond incrementally or confirm receipt. This works because smaller, bounded chunks align with STT limits and reduce memory use while preserving context.

## Code Example
```go
package main

import (
	"context"
	"log"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/lookatitude/beluga-ai/pkg/voice/session"
	"github.com/lookatitude/beluga-ai/pkg/voice/stt"
	"github.com/lookatitude/beluga-ai/pkg/voice/tts"
	"github.com/lookatitude/beluga-ai/pkg/voice/turndetection"
)

var tracer = otel.Tracer("beluga.voice.session.long_utterances")

const (
	chunkDuration = 3 * time.Second
	sampleRate    = 16000
	bytesPerSec   = sampleRate * 2 // 16-bit mono
)

func main() {
	ctx := context.Background()
	ctx, span := tracer.Start(ctx, "long_utterance_handler")
	defer span.End()

	sttProv, _ := stt.NewProvider(ctx, "openai", stt.DefaultConfig(), stt.WithAPIKey("your-key"))
	ttsProv, _ := tts.NewProvider(ctx, "openai", tts.DefaultConfig(), tts.WithAPIKey("your-key"))
	td, _ := turndetection.NewProvider(ctx, "heuristic", turndetection.DefaultConfig(),
		turndetection.WithMaxTurnLength(10000),
		turndetection.WithMinSilenceDuration(600*time.Millisecond),
	)

	sess, err := session.NewVoiceSession(ctx,
		session.WithSTTProvider(sttProv),
		session.WithTTSProvider(ttsProv),
		session.WithTurnDetector(td),
		session.WithConfig(session.DefaultConfig()),
	)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(trace.StatusError, err.Error())
		log.Fatalf("session: %v", err)
	}

	chunkSize := int(chunkDuration.Seconds() * float64(bytesPerSec))
	span.SetAttributes(attribute.Int("chunk_size_bytes", chunkSize))
	_ = sess
}

func chunkAudio(audio []byte, size int) [][]byte {
	var chunks [][]byte
	for len(audio) > 0 {
		n := size
		if n > len(audio) {
			n = len(audio)
		}
		chunks = append(chunks, audio[:n])
		audio = audio[n:]
	}
	return chunks
}
```

## Explanation

1. **Chunk size** — Derive from duration and sample rate (e.g. 3 s × 16 kHz × 2 bytes). Keeps `ProcessAudio` inputs bounded and STT-friendly.

2. **Turn detection** — Use `MaxTurnLength` and `MinSilenceDuration` so long runs of speech are split at turn boundaries when possible. Flush buffers on turn end.

3. **Buffering** — Accumulate audio until you have a chunk or turn end, then pass to `ProcessAudio`. Avoid sending single giant buffers.

4. **OTEL** — Record chunk size and counts. Use session metrics for latency and throughput.

**Key insight:** Long utterances are manageable with chunking, turn-aware flushing, and optional incremental processing (e.g. "I'm listening" or partial replies) so users know the system is handling their input.

## Testing

- Unit-test `chunkAudio` for various lengths and chunk sizes.
- Integration-test with mock audio; verify `ProcessAudio` receives expected chunks and session state stays consistent.

## Variations

### Variable chunk size

Adjust by environment (e.g. mobile vs broadband) or STT provider limits.

### Incremental feedback

Use `Say` to play short acknowledgments ("Got it, go on") during long turns so users don’t think the system dropped.

### Max utterance duration

Enforce a hard cap (e.g. 30 s); flush and process when reached even without turn end.

## Related Recipes

- **[Preemptive Generation](./voice-session-preemptive-generation.md)** — Interim results and latency.
- **[Voice Turn Sentence-Boundary-Aware](./voice-turn-sentence-boundary-aware.md)** — Turn boundaries.
- **[Heuristic Tuning](../../integrations/voice/turn/heuristic-tuning.md)** — MinSilenceDuration and turn length.
