---
title: "Handling Long Utterances"
description: "Chunk and buffer long user speech with turn-aware flushing to keep voice sessions responsive."
---

## Problem

Users sometimes speak for a long time without clear turn boundaries. You need to chunk or buffer audio, handle multi-sentence input, and avoid truncation or timeouts while keeping the session responsive.

## Solution

Use configurable chunking and buffering before passing audio to the session's `ProcessAudio`. Send STT-friendly chunks (2-5 seconds) and use turn detection to decide when to flush. For long monologues, aggregate interim transcripts and trigger agent processing at sensible boundaries (sentence end, max duration) so the session can respond incrementally.

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

	"github.com/lookatitude/beluga-ai/voice/session"
	"github.com/lookatitude/beluga-ai/voice/stt"
	"github.com/lookatitude/beluga-ai/voice/tts"
	"github.com/lookatitude/beluga-ai/voice/turndetection"
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

1. **Chunk size** -- Derived from duration and sample rate (e.g. 3s x 16kHz x 2 bytes). Keeps `ProcessAudio` inputs bounded and STT-friendly.

2. **Turn detection** -- `MaxTurnLength` and `MinSilenceDuration` split long speech at turn boundaries when possible. Flush buffers on turn end.

3. **Buffering** -- Accumulate audio until you have a chunk or turn end, then pass to `ProcessAudio`. Avoid sending single giant buffers.

**Key insight:** Long utterances are manageable with chunking, turn-aware flushing, and optional incremental processing (e.g. "I'm listening" or partial replies) so users know the system is handling their input.

## Variations

### Variable Chunk Size

Adjust by environment (mobile vs broadband) or STT provider limits.

### Incremental Feedback

Use `Say` to play short acknowledgments ("Got it, go on") during long turns so users know the system has not dropped.

### Max Utterance Duration

Enforce a hard cap (e.g. 30s); flush and process when reached even without turn end.

## Related Recipes

- **[Preemptive Generation](./voice-preemptive-gen)** -- Interim results and latency reduction
- **[Sentence-Boundary Turns](./sentence-boundary-turns)** -- Turn boundaries with sentence detection
- **[Voice Backends Configuration](./voice-backends)** -- Backend provider setup
