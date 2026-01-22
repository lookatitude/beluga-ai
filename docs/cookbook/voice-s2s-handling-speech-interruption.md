---
title: "Handling Speech Interruption"
package: "voice/s2s"
category: "interaction"
complexity: "advanced"
---

# Handling Speech Interruption

## Problem

You need to handle cases where users interrupt the AI's speech mid-sentence, requiring immediate stopping of TTS output, processing the new input, and resuming appropriately.

## Solution

Implement interruption detection that monitors for new user speech while TTS is playing, immediately stops TTS output, cancels in-flight LLM requests, and processes the new input with minimal delay. This works because you can detect voice activity, cancel ongoing operations, and prioritize new input.

## Code Example
```go
package main

import (
    "context"
    "fmt"
    "log"
    "sync"
    "time"
    
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"
)

var tracer = otel.Tracer("beluga.voice.s2s.interruption")

// InterruptionHandler handles speech interruptions
type InterruptionHandler struct {
    ttsPlaying    bool
    llmProcessing bool
    cancelFunc    context.CancelFunc
    mu            sync.RWMutex
}

// NewInterruptionHandler creates a new interruption handler
func NewInterruptionHandler() *InterruptionHandler {
    return &InterruptionHandler{}
}

// HandleInterruption handles user interruption
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

// stopTTS stops TTS playback
func (ih *InterruptionHandler) stopTTS(ctx context.Context) {
    ctx, span := tracer.Start(ctx, "interruption_handler.stop_tts")
    defer span.End()
    
    // Stop TTS output immediately
    // ttsProvider.Stop()
    
    ih.mu.Lock()
    ih.ttsPlaying = false
    ih.mu.Unlock()
    
    span.SetStatus(trace.StatusOK, "TTS stopped")
}

// stopLLM cancels LLM processing
func (ih *InterruptionHandler) stopLLM(ctx context.Context) {
    ctx, span := tracer.Start(ctx, "interruption_handler.stop_llm")
    defer span.End()
    
    // Cancel LLM request
    // This is handled by context cancellation
    
    ih.mu.Lock()
    ih.llmProcessing = false
    ih.mu.Unlock()
    
    span.SetStatus(trace.StatusOK, "LLM stopped")
}

// processNewInput processes new user input after interruption
func (ih *InterruptionHandler) processNewInput(ctx context.Context, audio <-chan []byte) {
    ctx, span := tracer.Start(ctx, "interruption_handler.process_new_input")
    defer span.End()
    
    // Mark LLM as processing
    ih.mu.Lock()
    ih.llmProcessing = true
    ih.mu.Unlock()
    
    // Process new input
    // This would typically trigger new STT -> LLM -> TTS pipeline
    
    span.SetStatus(trace.StatusOK, "new input processed")
}

// InterruptibleS2S wraps S2S with interruption handling
type InterruptibleS2S struct {
    handler     *InterruptionHandler
    vadDetector interface{} // Voice Activity Detector
}

// NewInterruptibleS2S creates interruptible S2S
func NewInterruptibleS2S() *InterruptibleS2S {
    return &InterruptibleS2S{
        handler: NewInterruptionHandler(),
    }
}

// Process processes audio with interruption detection
func (is2s *InterruptibleS2S) Process(ctx context.Context, audioIn <-chan []byte, ttsOut chan<- []byte) error {
    ctx, span := tracer.Start(ctx, "interruptible_s2s.process")
    defer span.End()
    
    // Monitor for interruptions while processing
    interruptionCh := make(chan []byte, 10)
    
    go func() {
        for audio := range audioIn {
            // Detect voice activity
            // if vadDetector.Detect(audio) {
            interruptionCh <- audio
            // }
        }
    }()
    
    // Process with interruption handling
    select {
    case newAudio := <-interruptionCh:
        return is2s.handler.HandleInterruption(ctx, interruptionCh)
    case <-ctx.Done():
        return ctx.Err()
    }
    
    return nil
}

func main() {
    ctx := context.Background()

    // Create interruptible S2S
    is2s := NewInterruptibleS2S()
    
    // Process with interruption support
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

Let's break down what's happening:

1. **Immediate cancellation** - Notice how we cancel all ongoing operations (TTS, LLM) immediately when interruption is detected. This minimizes delay in processing new input.

2. **Voice activity detection** - We monitor for voice activity while processing, allowing early detection of interruptions.

3. **Clean state reset** - We reset processing state (ttsPlaying, llmProcessing) to allow new processing to start cleanly.

```go
**Key insight:** Detect interruptions early and cancel operations immediately. The faster you detect and respond, the more natural the conversation feels.

## Testing

```
Here's how to test this solution:
```go
func TestInterruptionHandler_HandlesInterruption(t *testing.T) {
    handler := NewInterruptionHandler()
    
    handler.mu.Lock()
    handler.ttsPlaying = true
    handler.mu.Unlock()
    
    audioCh := make(chan []byte, 1)
    err := handler.HandleInterruption(context.Background(), audioCh)
    
    require.NoError(t, err)
    require.False(t, handler.ttsPlaying)
}

## Variations

### Partial Response Handling

Handle partial LLM responses during interruption:
func (ih *InterruptionHandler) HandlePartialResponse(partialResponse string) {
    // Save or discard partial response
}
```

### Graceful Interruption

Fade out TTS instead of stopping abruptly:
```go
func (ih *InterruptionHandler) GracefulStopTTS(ctx context.Context) {
    // Fade out over brief period
}
```

## Related Recipes

- **[Voice S2S Minimizing Glass-to-Glass Latency](./voice-s2s-minimizing-glass-to-glass-latency.md)** - Optimize latency
- **[Voice STT Jitter Buffer Management](./voice-stt-jitter-buffer-management.md)** - Handle audio issues
- **[Voice Providers Guide](../guides/voice-providers.md)** - For a deeper understanding of S2S
