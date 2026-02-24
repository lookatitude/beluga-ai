---
title: "Preemptive Voice Generation"
description: "Recipe for reducing voice response latency in Go by generating LLM responses from interim STT transcripts before the user finishes speaking."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Beluga AI, preemptive generation, Go voice latency, interim STT, speculative response, time-to-first-reply, voice optimization"
---

## Problem

You need to reduce time-to-first-reply in voice sessions by generating a response from interim STT results before the user has finished speaking, then either use that reply (when the final transcript is similar) or fall back to a new reply from the final transcript. Voice applications suffer from serial latency accumulation: STT waits for user silence, then LLM generates a response, then TTS synthesizes audio. This pipeline introduces hundreds of milliseconds to multiple seconds of latency, making conversations feel sluggish compared to human-to-human interaction where overlapping speech and rapid turn-taking are natural.

The challenge is balancing latency reduction with accuracy. Generating responses from interim transcripts (which are partial and may change as more audio arrives) risks producing irrelevant answers if the final transcript differs significantly. However, in many conversations, interim transcripts stabilize well before the user finishes speaking, especially for common phrases or questions. By speculatively generating a response from interim transcripts and validating it against the final transcript, you can hide much of the LLM latency within the user's speaking time.

## Solution

Use streaming STT with interim and final results. On each interim transcript, call your agent/LLM and store the preemptive reply. When the final transcript arrives, apply a strategy: **use-if-similar** (compare interim vs final, use preemptive if similar), **always-use** (use preemptive when available), or **discard** (always generate from final).

The reasoning behind this pattern is opportunistic parallelism. Streaming STT emits interim transcripts as the user speaks, often updating every 100-200ms. These interims converge toward the final transcript as more audio context arrives. By calling the LLM on each interim (or the last stable interim), you overlap "user speaking" time with "LLM thinking" time. When the user finishes speaking and the final transcript arrives, you may already have a generated response ready. The similarity check guards against drift: if the final transcript differs significantly from the interim used to generate the preemptive response, discard the preemptive response and generate anew.

This pattern trades computational cost (multiple LLM calls per user turn) for latency reduction. In practice, most conversations see 2-3 interim transcripts before final, and you can optimize by only generating from interims that differ by more than a threshold from the previous interim. The functional option pattern (`WithSimilarityThreshold`, `WithPreemptiveStrategy`) allows tuning this tradeoff per deployment.

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

1. **Interim vs final** -- Streaming STT emits partial (interim) and final transcripts. Use interims to start agent/LLM work early, overlapping "user still speaking" with "agent thinking." This parallelism is the core benefit: instead of waiting for the user to finish, then starting generation, you start generation while the user is still speaking. In typical conversations, users speak for 2-5 seconds, and LLM generation takes 500ms-2s. By starting generation 1-2 seconds before the user finishes, you can hide most or all of the LLM latency.

2. **Preemptive buffer** -- Store the last preemptive reply and last interim. When final arrives, decide whether to use the preemptive reply or generate anew. The mutex protects shared state because interim results arrive asynchronously on a WebSocket stream while the final transcript arrives on a separate event. Without synchronization, you risk data races where the preemptive reply is read while being written. The buffer is simple (single string), but production systems might maintain a queue of (interim, preemptive_reply) pairs to handle rapid interim updates.

3. **Use-if-similar** -- Compare final to last interim (e.g. overlap, edit distance). If similar and preemptive exists, use it to avoid an extra round-trip. Similarity checking guards against wasted work: if the final transcript is "What is the weather in Paris" but the last interim was "What is the weather in London", the preemptive reply (about London) is wrong for the final transcript. The `similar` function here uses substring containment, which is fast but crude. Production systems should use Levenshtein distance, Jaro-Winkler, or semantic embedding similarity with a tunable threshold (e.g., 0.8 similarity score).

**Key insight:** Preemptive generation trades some complexity (similarity check, buffering) for lower perceived latency. Use discard or use-if-similar when correctness matters. The pattern here prioritizes correctness: when in doubt, regenerate from the final transcript. This conservative approach avoids incorrect responses at the cost of occasional wasted preemptive generation. For applications where speed is paramount (e.g., rapid-fire Q&A bots), you might use `always-use` and accept occasional mismatches, relying on users to rephrase if the response seems off.

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
