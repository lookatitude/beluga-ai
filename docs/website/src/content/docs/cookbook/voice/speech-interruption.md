---
title: "Handling Speech Interruption"
description: "Detect and handle user interruptions during AI speech with immediate cancellation and clean state reset."
---

## Problem

Natural conversations involve frequent interruptions where listeners interject, ask clarifying questions, or redirect the conversation mid-sentence. Voice AI systems that cannot handle interruptions feel unresponsive and frustrating. When a user interrupts while the AI is speaking, three problems arise simultaneously. First, the text-to-speech system continues playing audio that the user no longer wants to hear, wasting time and bandwidth. Second, in-flight LLM requests continue generating tokens for responses the user has rejected, consuming API quota and delaying processing of the new input. Third, the system must distinguish intentional interruptions from background noise or cross-talk, avoiding false triggers that disrupt legitimate AI responses.

The challenge is detecting interruptions with minimal latency (sub-200ms), stopping all ongoing operations without leaving orphaned goroutines or partial state, and processing the new input immediately while maintaining conversation context.

## Solution

Implement interruption detection that continuously monitors for new user speech while TTS is active, immediately stops TTS playback when detected, cancels in-flight operations via context cancellation, and processes the new input with a fresh context. Voice activity detection provides low-latency signals indicating when users start speaking. Context cancellation propagates through the entire pipeline, terminating STT streaming, LLM generation, and TTS synthesis cleanly without requiring explicit coordination between components.

The architecture uses Go's context cancellation as the primary coordination mechanism. Each processing pipeline (STT -> LLM -> TTS) runs with a cancellable context. When interruption is detected, calling the cancel function immediately terminates all downstream operations. Components respect context cancellation by checking `ctx.Done()` in their processing loops and returning promptly when cancelled. This pattern avoids explicit state synchronization and ensures clean shutdown even if components are distributed across goroutines or services.

State tracking (ttsPlaying, llmProcessing) enables quick decisions about what needs cancellation. A mutex guards this state because interruption detection and processing logic run in separate goroutines. After cancellation, creating a new context for the interrupted input ensures the new processing pipeline operates independently from the cancelled one.

## Code Example

```go
package main

import (
	"context"
	"fmt"
	"log"
	"sync"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

var tracer = otel.Tracer("beluga.voice.s2s.interruption")

// InterruptionHandler handles speech interruptions.
type InterruptionHandler struct {
	ttsPlaying    bool
	llmProcessing bool
	cancelFunc    context.CancelFunc
	mu            sync.RWMutex
}

func NewInterruptionHandler() *InterruptionHandler {
	return &InterruptionHandler{}
}

// HandleInterruption cancels ongoing operations and processes new input.
func (ih *InterruptionHandler) HandleInterruption(ctx context.Context, newAudio <-chan []byte) error {
	ctx, span := tracer.Start(ctx, "interruption_handler.handle")
	defer span.End()

	span.SetAttributes(attribute.Bool("interruption_detected", true))

	ih.mu.Lock()

	// Cancel ongoing operations
	if ih.cancelFunc != nil {
		ih.cancelFunc()
		span.SetAttributes(attribute.Bool("operations_cancelled", true))
	}

	// Stop TTS if playing
	if ih.ttsPlaying {
		ih.stopTTS(ctx)
		span.SetAttributes(attribute.Bool("tts_stopped", true))
	}

	// Stop LLM if processing
	if ih.llmProcessing {
		ih.stopLLM(ctx)
		span.SetAttributes(attribute.Bool("llm_stopped", true))
	}

	// Create new context for new input
	newCtx, cancel := context.WithCancel(ctx)
	ih.cancelFunc = cancel
	ih.mu.Unlock()

	// Process new input immediately
	go ih.processNewInput(newCtx, newAudio)

	span.SetStatus(trace.StatusOK, "interruption handled")
	return nil
}

func (ih *InterruptionHandler) stopTTS(ctx context.Context) {
	_, span := tracer.Start(ctx, "interruption_handler.stop_tts")
	defer span.End()

	// Stop TTS output immediately
	// ttsProvider.Stop()

	ih.ttsPlaying = false
	span.SetStatus(trace.StatusOK, "TTS stopped")
}

func (ih *InterruptionHandler) stopLLM(ctx context.Context) {
	_, span := tracer.Start(ctx, "interruption_handler.stop_llm")
	defer span.End()

	// Cancel LLM request via context cancellation
	ih.llmProcessing = false
	span.SetStatus(trace.StatusOK, "LLM stopped")
}

func (ih *InterruptionHandler) processNewInput(ctx context.Context, audio <-chan []byte) {
	_, span := tracer.Start(ctx, "interruption_handler.process_new_input")
	defer span.End()

	ih.mu.Lock()
	ih.llmProcessing = true
	ih.mu.Unlock()

	// Trigger new STT -> LLM -> TTS pipeline
	span.SetStatus(trace.StatusOK, "new input processed")
}

// InterruptibleS2S wraps S2S with interruption handling.
type InterruptibleS2S struct {
	handler *InterruptionHandler
}

func NewInterruptibleS2S() *InterruptibleS2S {
	return &InterruptibleS2S{
		handler: NewInterruptionHandler(),
	}
}

// Process processes audio with interruption detection.
func (is2s *InterruptibleS2S) Process(ctx context.Context, audioIn <-chan []byte, ttsOut chan<- []byte) error {
	_, span := tracer.Start(ctx, "interruptible_s2s.process")
	defer span.End()

	interruptionCh := make(chan []byte, 10)

	go func() {
		for audio := range audioIn {
			// Detect voice activity
			interruptionCh <- audio
		}
	}()

	select {
	case <-interruptionCh:
		return is2s.handler.HandleInterruption(ctx, interruptionCh)
	case <-ctx.Done():
		return ctx.Err()
	}
}

func main() {
	ctx := context.Background()

	is2s := NewInterruptibleS2S()

	audioIn := make(chan []byte, 10)
	ttsOut := make(chan []byte, 10)

	err := is2s.Process(ctx, audioIn, ttsOut)
	if err != nil {
		log.Fatalf("Failed: %v", err)
	}
	fmt.Println("Interruptible S2S created")
}
```

The code demonstrates the state tracking and context management pattern. `InterruptionHandler` maintains boolean flags guarded by a mutex, ensuring safe concurrent access. The `HandleInterruption` method cancels the previous context, stops active components, creates a new context, and starts new processing in a single atomic operation.

## Explanation

1. **Immediate cancellation** -- Calling `cancelFunc()` propagates cancellation through all goroutines and remote calls using the cancelled context. STT providers stop reading audio streams. LLM providers abort generation requests. TTS providers stop synthesizing. This happens automatically because Beluga components respect context cancellation in their processing loops (they check `ctx.Done()` and return promptly). Immediate cancellation is critical for responsive interruption handling: every 100ms of delay in stopping the AI's speech makes the interaction feel sluggish. Well-behaved components must check context cancellation frequently (at least once per processing iteration) to enable fast shutdown.

2. **Voice activity detection** -- Continuous monitoring of audio input for voice activity enables early interruption detection. VAD runs in parallel with TTS playback, emitting events as soon as user speech is detected (typically within 50-100ms of speech onset). This low-latency signal triggers cancellation before users finish their first syllable, creating the perception of immediate responsiveness. The VAD threshold must balance sensitivity: too sensitive and background noise or the AI's own audio (in full-duplex scenarios) triggers false interruptions; too insensitive and users must speak loudly or repeat themselves.

3. **Clean state reset** -- Resetting processing state (`ttsPlaying = false`, `llmProcessing = false`) prevents stale flags from affecting subsequent interactions. Creating a new cancellable context for the new input ensures the new processing pipeline is independent: cancelling it later (due to another interruption) will not affect unrelated operations. This pattern avoids the subtle bugs that arise when contexts or state leak across conversation turns. The mutex-guarded state update ensures the cancellation and reset happen atomically, preventing race conditions where interruption detection and processing logic read inconsistent state.

**Key insight:** Detect interruptions early and cancel operations immediately. The faster you detect and respond to interruptions, the more natural the conversation feels. Users should hear the AI stop speaking within 100-200ms of their first word, matching the responsiveness of human conversations. Implement this by running VAD in parallel with all other processing and ensuring every component respects context cancellation with low latency (sub-10ms from cancellation to actual stop).

## Variations

### Partial Response Handling

Save partial LLM responses during interruption, storing them in conversation context for subsequent turns. This allows the AI to reference what it was saying before being interrupted, creating more coherent multi-turn conversations.

### Graceful Interruption

Fade out TTS over 50-100ms instead of stopping abruptly for a smoother auditory experience. Apply a volume ramp to the remaining audio samples before stopping playback, reducing the jarring effect of sudden silence.

## Related Recipes

- **[Glass-to-Glass Latency](./glass-to-glass-latency)** -- Optimize end-to-end latency
- **[Jitter Buffer Management](./jitter-buffer)** -- Handle network audio issues
- **[ML-Based Barge-In](./ml-barge-in)** -- ML-based interruption detection
