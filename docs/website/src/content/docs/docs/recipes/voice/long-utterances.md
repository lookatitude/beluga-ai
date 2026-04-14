---
title: "Handling Long Utterances"
description: "Recipe for chunking and buffering long user speech in Go with turn-aware flushing to keep voice sessions responsive and avoid timeouts."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Beluga AI, long utterances, Go speech chunking, audio buffering, turn-aware flushing, voice session, timeout prevention recipe"
---

## Problem

Users sometimes deliver extended monologues without clear turn boundaries: detailed problem descriptions, lengthy instructions, storytelling, or open-ended brainstorming. Long utterances create multiple challenges for voice systems. First, accumulating audio in memory until the user stops speaking risks exhausting buffer capacity, causing dropped audio or system instability. Second, many STT providers impose duration limits (typically 30-60 seconds per request) or perform poorly on very long continuous audio due to model context windows. Third, waiting for complete utterance transcription before beginning LLM processing increases latency, making the system feel unresponsive during long input. Fourth, timeouts configured for normal turn lengths may expire during long monologues, terminating processing prematurely and frustrating users.

The challenge is processing long audio incrementally without fragmenting coherent thoughts, maintaining responsive feedback so users know the system is listening, and avoiding resource exhaustion or timeout failures.

## Solution

Chunk audio into fixed-duration segments that STT providers handle efficiently (typically 2-5 seconds), use turn detection to identify natural boundaries within long speech for flushing accumulated transcripts, and optionally provide incremental feedback to users indicating continued processing. Chunking keeps buffer sizes bounded and ensures each STT request stays within provider limits. Turn detection identifies sentence boundaries or natural pauses within long monologues where it is safe to flush accumulated transcripts and begin agent processing without fragmenting the user's thought.

The architecture buffers incoming audio until reaching either a chunk duration limit or a turn boundary detected by turn detection. At chunk boundaries, the system sends audio to STT and accumulates the resulting transcript. At turn boundaries (detected via sentence markers and silence), the system flushes accumulated transcripts to the agent for processing and begins a new accumulation cycle. This two-level buffering (chunk-level for STT, turn-level for agent processing) balances STT efficiency with natural conversation flow.

Optional incremental feedback provides users with acknowledgment during long input. Simple strategies include playing a brief tone or saying "I'm listening" after each processed chunk. More sophisticated approaches generate partial agent responses based on interim transcripts, then refine them as more context arrives. This feedback reassures users that the system has not crashed or stopped listening, particularly important in voice-only interfaces where visual feedback is unavailable.

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

The code calculates chunk size based on duration and sample rate (3 seconds x 16kHz x 2 bytes per sample = 96,000 bytes per chunk). This ensures consistent chunk sizes regardless of sample rate changes. Turn detection configuration uses high `MaxTurnLength` (10,000 characters) and long silence duration (600ms) appropriate for extended monologues.

## Explanation

1. **Chunk size** -- Deriving chunk size from duration and sample rate ensures audio chunks contain the right amount of data for efficient STT processing. Three-second chunks provide enough context for accurate transcription without overwhelming provider APIs or exceeding request size limits. The calculation (`chunkDuration * sampleRate * bytesPerSample`) produces byte counts that work consistently across different audio formats. For 16-bit mono audio at 16kHz (common for speech), this yields 96,000 bytes per chunk. For different formats (8kHz narrow-band, 48kHz wideband), adjust the sample rate and bytes per sample accordingly. Keep chunks between 2-5 seconds: shorter chunks increase API call overhead and may lack sufficient context for accurate transcription, longer chunks approach provider limits and increase memory usage.

2. **Turn detection** -- High `MaxTurnLength` (10,000 characters) and extended `MinSilenceDuration` (600ms) accommodate long-form speech patterns. Users delivering detailed explanations or narratives naturally pause for breath or thought, creating silence periods that should not trigger premature turn boundaries. The 600ms threshold is long enough to distinguish true turn boundaries from mid-thought pauses but short enough to detect natural segment endings. `MaxTurnLength` ensures that even rambling monologues eventually flush accumulated transcripts and trigger agent processing, preventing unbounded memory growth and providing periodic feedback to users. When turn detection signals turn completion, the session flushes all accumulated transcript chunks to the agent, processes them as a single coherent input, and begins accumulating a new turn.

3. **Buffering** -- Accumulating audio until reaching chunk size or turn boundary keeps memory usage bounded while maintaining conversation coherence. The buffer holds unprocessed audio (not yet sent to STT) and accumulated transcripts (STT results not yet sent to agent). Double-buffering prevents loss during processing: while one buffer is being transcribed, the other accumulates new audio. Flush buffers at turn boundaries detected by turn detection rather than arbitrary time limits to avoid fragmenting user thoughts. For very long monologues where users never pause, the `MaxTurnLength` acts as a safety valve, forcing a flush after a configured character count to provide periodic feedback and prevent resource exhaustion.

**Key insight:** Long utterances are manageable with chunking for STT efficiency, turn-aware flushing for conversation coherence, and optional incremental processing for user feedback. The combination keeps resource usage bounded, maintains natural conversation flow by respecting turn boundaries, and prevents the system from appearing unresponsive during extended input. Users should receive acknowledgment at least every 10-15 seconds during long monologues, either through partial agent responses ("I understand so far...") or simple auditory cues, confirming the system has not stopped listening.

## Variations

### Variable Chunk Size

Adjust chunk size based on environment or provider constraints. Mobile connections with limited bandwidth benefit from smaller chunks (1-2 seconds) to reduce upload latency. High-bandwidth connections can use larger chunks (4-5 seconds) to reduce API call overhead. Check provider documentation for optimal chunk sizes: some STT services perform better with specific durations.

### Incremental Feedback

Use the session's `Say` method to play short acknowledgments during long turns, reassuring users the system is processing their input.

```go
// After each chunk or after N seconds of continuous input
sess.Say(ctx, "Got it, go on")
```

### Max Utterance Duration

Enforce a hard time-based cap (e.g., 30 seconds) in addition to character-based `MaxTurnLength`. Flush and process when reached even without turn end signals, ensuring timely feedback and preventing abuse or accidental microphone activation from consuming unlimited resources.

## Related Recipes

- **[Preemptive Generation](./voice-preemptive-gen)** -- Interim results and latency reduction
- **[Sentence-Boundary Turns](./sentence-boundary-turns)** -- Turn boundaries with sentence detection
- **[Voice Backends Configuration](./voice-backends)** -- Backend provider setup
