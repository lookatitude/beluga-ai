---
title: Preemptive Generation Strategies
description: Reduce perceived latency by generating agent replies from interim STT transcripts before the user finishes speaking.
---

Preemptive generation uses interim (partial) STT transcripts to begin generating a response before the user finishes speaking. The core idea is that most of a sentence's meaning is established in the first few words -- "What is the weather in" almost certainly ends with a location -- so starting generation early produces a usable response by the time the final transcript arrives. When the final transcript confirms the interim, the preemptive response can be used immediately, reducing perceived latency by hundreds of milliseconds.

## What You Will Build

A voice session that processes interim STT results to speculatively generate responses, with strategies for deciding when to use the preemptive result versus regenerating from the final transcript.

## Prerequisites

- Go 1.23+
- A streaming STT provider (e.g., Deepgram, Google) that supports interim results
- Completion of [Real-time STT Streaming](/tutorials/voice/stt-realtime-streaming)

## The Latency Problem

In a standard pipeline, the agent waits for the final transcript before generating a response. This means the LLM processing and TTS synthesis happen sequentially after the user finishes speaking, creating a noticeable gap:

```
User speaks ──▶ Silence detected ──▶ Final transcript ──▶ LLM ──▶ TTS ──▶ Audio
                                     ▲ latency starts here
```

With preemptive generation, the agent starts the LLM call during the user's speech, overlapping generation with the remaining speech. If the interim and final transcripts are sufficiently similar, the preemptive response is ready before the user finishes:

```
User speaks ──▶ Interim transcript ──▶ LLM (preemptive) ──▶ Buffer
                    ...
               ──▶ Final transcript ──▶ Compare ──▶ Use preemptive or regenerate
                                                         │
                                                         ▼
                                                   TTS ──▶ Audio
```

## Step 1: Session with Streaming STT

Create a voice session with a streaming-capable STT provider. Streaming STT is essential for preemptive generation because batch STT only produces a single final transcript after the entire audio is processed, providing no interim results to act on.

```go
package main

import (
	"context"
	"log"

	"github.com/lookatitude/beluga-ai/voice/session"
	"github.com/lookatitude/beluga-ai/voice/stt"
	"github.com/lookatitude/beluga-ai/voice/tts"
	_ "github.com/lookatitude/beluga-ai/voice/stt/providers/deepgram"
	_ "github.com/lookatitude/beluga-ai/voice/tts/providers/openai"
)

func main() {
	ctx := context.Background()

	sttProvider, err := stt.NewProvider(ctx, "deepgram", stt.DefaultConfig(),
		stt.WithAPIKey("your-key"),
		stt.WithEnableStreaming(true),
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

	_ = sess
}
```

## Step 2: Preemptive Generation Strategies

There are three strategies for handling preemptive responses. The choice depends on your tolerance for incorrect responses versus your latency requirements:

### Use-If-Similar

Generate from the interim transcript. When the final arrives, compare the two. If they are sufficiently similar (high word overlap), use the preemptive response. This is the safest strategy that still provides latency benefits.

```go
// PreemptiveStrategy determines how to handle preemptive responses.
type PreemptiveStrategy int

const (
	// UseIfSimilar uses the preemptive response if the final transcript
	// is similar to the interim that generated it.
	UseIfSimilar PreemptiveStrategy = iota
	// AlwaysUse always uses the preemptive response, regardless of
	// how the final transcript differs.
	AlwaysUse
	// Discard ignores preemptive responses and always generates
	// from the final transcript.
	Discard
)
```

### Strategy Comparison

| Strategy     | Latency Reduction | Risk                                         |
|-------------|-------------------|----------------------------------------------|
| UseIfSimilar | Moderate          | Low -- falls back to final if interim was wrong |
| AlwaysUse    | Maximum           | Medium -- may respond to wrong transcript      |
| Discard      | None              | None -- safest, standard behavior              |

## Step 3: Implement the Preemptive Handler

The handler tracks interim transcripts, preemptive responses, and applies the chosen strategy when the final transcript arrives. The `isSimilar` method uses word overlap as a simple but effective similarity metric -- if 80% or more of the words match, the transcripts are considered equivalent for generation purposes.

```go
import "strings"

// PreemptiveHandler manages preemptive generation for a voice session.
type PreemptiveHandler struct {
	strategy          PreemptiveStrategy
	similarityThreshold float64
	lastInterim       string
	preemptiveReply   string
	generateFn        func(ctx context.Context, transcript string) (string, error)
}

// NewPreemptiveHandler creates a handler with the given strategy.
func NewPreemptiveHandler(
	strategy PreemptiveStrategy,
	threshold float64,
	generateFn func(ctx context.Context, transcript string) (string, error),
) *PreemptiveHandler {
	return &PreemptiveHandler{
		strategy:          strategy,
		similarityThreshold: threshold,
		generateFn:        generateFn,
	}
}

// OnInterim processes an interim transcript. If the strategy allows,
// it generates a preemptive response in the background.
func (h *PreemptiveHandler) OnInterim(ctx context.Context, interim string) {
	h.lastInterim = interim

	if h.strategy == Discard {
		return
	}

	// Generate preemptive response (in production, run in a goroutine)
	reply, err := h.generateFn(ctx, interim)
	if err != nil {
		return
	}
	h.preemptiveReply = reply
}

// OnFinal processes the final transcript and returns the response to use.
func (h *PreemptiveHandler) OnFinal(ctx context.Context, final string) (string, error) {
	switch h.strategy {
	case AlwaysUse:
		if h.preemptiveReply != "" {
			reply := h.preemptiveReply
			h.reset()
			return reply, nil
		}
	case UseIfSimilar:
		if h.preemptiveReply != "" && h.isSimilar(h.lastInterim, final) {
			reply := h.preemptiveReply
			h.reset()
			return reply, nil
		}
	}

	// Fall back to generating from the final transcript
	h.reset()
	return h.generateFn(ctx, final)
}

func (h *PreemptiveHandler) isSimilar(a, b string) bool {
	wordsA := strings.Fields(strings.ToLower(a))
	wordsB := strings.Fields(strings.ToLower(b))
	if len(wordsA) == 0 || len(wordsB) == 0 {
		return false
	}

	matches := 0
	bSet := make(map[string]bool, len(wordsB))
	for _, w := range wordsB {
		bSet[w] = true
	}
	for _, w := range wordsA {
		if bSet[w] {
			matches++
		}
	}

	overlap := float64(matches) / float64(max(len(wordsA), len(wordsB)))
	return overlap >= h.similarityThreshold
}

func (h *PreemptiveHandler) reset() {
	h.lastInterim = ""
	h.preemptiveReply = ""
}
```

## Step 4: Wire into the Voice Pipeline

Integrate the preemptive handler with your STT streaming session. The handler receives both interim and final transcript results and decides which response to use. The `IsFinal` flag from the STT provider distinguishes between partial results (which trigger preemptive generation) and confirmed results (which trigger the strategy comparison).

```go
	handler := NewPreemptiveHandler(UseIfSimilar, 0.8, func(ctx context.Context, transcript string) (string, error) {
		// Call your LLM or agent here
		return "Response to: " + transcript, nil
	})

	// In your STT transcript processing loop:
	streamSession, err := sttProvider.StartStreaming(ctx)
	if err != nil {
		log.Fatalf("start streaming: %v", err)
	}
	defer streamSession.Close()

	for result := range streamSession.ReceiveTranscript() {
		if result.Error != nil {
			continue
		}

		if result.IsFinal {
			response, err := handler.OnFinal(ctx, result.Text)
			if err != nil {
				log.Printf("generate response: %v", err)
				continue
			}
			// Use response for TTS
			log.Printf("Response: %s", response)
		} else {
			handler.OnInterim(ctx, result.Text)
		}
	}
```

## Verification

1. Run the session with streaming STT and log interim vs. final transcripts.
2. Measure time-to-first-audio-response with preemptive generation enabled vs. disabled.
3. Use the `UseIfSimilar` strategy and speak a sentence that changes significantly at the end. Verify the handler falls back to the final transcript.
4. Use the `AlwaysUse` strategy and verify the fastest response time.

## Next Steps

- [Voice Session Interruptions](/tutorials/voice/session-interruptions) -- Cancel playback when the user interrupts
- [Sentence-Boundary Turn Detection](/tutorials/voice/sentence-boundary) -- Detect when the user finishes speaking
- [Real-time STT Streaming](/tutorials/voice/stt-realtime-streaming) -- Streaming STT fundamentals
