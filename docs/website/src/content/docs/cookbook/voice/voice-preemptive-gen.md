---
title: "Preemptive Generation for Voice"
description: "Reduce time-to-first-reply by generating responses from interim STT results before the user finishes speaking."
---

## Problem

You need to reduce time-to-first-reply in voice sessions by generating a response from interim STT results before the user has finished speaking, then either use that reply (when the final transcript is similar) or fall back to a new reply from the final transcript.

## Solution

Use streaming STT with interim and final results. On each interim transcript, call your agent/LLM and store the preemptive reply. When the final transcript arrives, apply a strategy: **use-if-similar** (compare interim vs final, use preemptive if similar), **always-use** (use preemptive when available), or **discard** (always generate from final).

## Code Example

```go
package main

import (
	"context"
	"log"
	"strings"
	"sync"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/lookatitude/beluga-ai/voice/session"
	"github.com/lookatitude/beluga-ai/voice/stt"
	"github.com/lookatitude/beluga-ai/voice/tts"
)

var tracer = otel.Tracer("beluga.voice.session.preemptive")

func main() {
	ctx := context.Background()
	ctx, span := tracer.Start(ctx, "preemptive_generation")
	defer span.End()

	sttProv, err := stt.NewProvider(ctx, "deepgram", stt.DefaultConfig(), stt.WithAPIKey("your-key"))
	if err != nil {
		span.RecordError(err)
		log.Fatalf("stt: %v", err)
	}
	ttsProv, err := tts.NewProvider(ctx, "openai", tts.DefaultConfig(), tts.WithAPIKey("your-key"))
	if err != nil {
		span.RecordError(err)
		log.Fatalf("tts: %v", err)
	}

	sess, err := session.NewVoiceSession(ctx,
		session.WithSTTProvider(sttProv),
		session.WithTTSProvider(ttsProv),
		session.WithConfig(session.DefaultConfig()),
	)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(trace.StatusError, err.Error())
		log.Fatalf("session: %v", err)
	}

	var (
		mu           sync.Mutex
		lastInterim  string
		lastPreempt  string
		useIfSimilar = true
	)

	// In your STT streaming loop:
	// - On interim: call agent, store in lastPreempt; set lastInterim.
	// - On final: if useIfSimilar && similar(lastInterim, final) && lastPreempt != "" -> use lastPreempt;
	//   else generate from final. Then sess.Say(ctx, reply).
	_, _ = lastInterim, lastPreempt
	span.SetAttributes(attribute.Bool("preemptive.use_if_similar", useIfSimilar))
	_ = sess
	_ = mu
}

func similar(a, b string) bool {
	a = strings.TrimSpace(strings.ToLower(a))
	b = strings.TrimSpace(strings.ToLower(b))
	if len(a) == 0 || len(b) == 0 {
		return false
	}
	// Simplified: use proper similarity (e.g. Levenshtein) in production
	return strings.Contains(b, a) || strings.Contains(a, b)
}
```

## Explanation

1. **Interim vs final** -- Streaming STT emits partial (interim) and final transcripts. Use interims to start agent/LLM work early, overlapping "user still speaking" with "agent thinking."

2. **Preemptive buffer** -- Store the last preemptive reply and last interim. When final arrives, decide whether to use the preemptive reply or generate anew.

3. **Use-if-similar** -- Compare final to last interim (e.g. overlap, edit distance). If similar and preemptive exists, use it to avoid an extra round-trip.

**Key insight:** Preemptive generation trades some complexity (similarity check, buffering) for lower perceived latency. Use discard or use-if-similar when correctness matters.

## Variations

### Always-Use

Skip similarity; use preemptive whenever available. Lowest latency, higher risk of mismatched answers.

### Discard

Never use preemptive; always generate from final. Safest, no similarity logic needed.

### Similarity Threshold

Tune the similarity threshold (e.g. 0.8) per domain or A/B test for best tradeoff.

## Related Recipes

- **[Handling Long Utterances](./long-utterances)** -- Chunking and buffering for long user input
- **[Glass-to-Glass Latency](./glass-to-glass-latency)** -- End-to-end latency optimization
- **[Voice Backends Configuration](./voice-backends)** -- Backend provider setup
