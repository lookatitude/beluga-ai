---
title: "Handling Speech Interruption"
description: "Detect and handle user interruptions during AI speech with immediate cancellation and clean state reset."
---

## Problem

You need to handle cases where users interrupt the AI's speech mid-sentence, requiring immediate stopping of TTS output, cancellation of in-flight LLM requests, and processing the new input with minimal delay.

## Solution

Implement interruption detection that monitors for new user speech while TTS is playing, immediately stops TTS output, cancels ongoing operations via context cancellation, and processes the new input. Voice activity detection provides low-latency interruption signals while context cancellation propagates through the pipeline.

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

## Explanation

1. **Immediate cancellation** -- All ongoing operations (TTS, LLM) are cancelled immediately when an interruption is detected. This minimizes delay in processing new input.

2. **Voice activity detection** -- Audio input is monitored for voice activity while processing, allowing early detection of interruptions.

3. **Clean state reset** -- Processing state (`ttsPlaying`, `llmProcessing`) is reset to allow new processing to start cleanly with a fresh context.

**Key insight:** Detect interruptions early and cancel operations immediately. The faster you detect and respond, the more natural the conversation feels.

## Variations

### Partial Response Handling

Save or discard partial LLM responses during interruption for context in subsequent turns.

### Graceful Interruption

Fade out TTS over a brief period instead of stopping abruptly for a smoother user experience.

## Related Recipes

- **[Glass-to-Glass Latency](./glass-to-glass-latency)** -- Optimize end-to-end latency
- **[Jitter Buffer Management](./jitter-buffer)** -- Handle network audio issues
- **[ML-Based Barge-In](./ml-barge-in)** -- ML-based interruption detection
