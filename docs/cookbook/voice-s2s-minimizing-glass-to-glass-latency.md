---
title: "Minimizing Glass-to-Glass Latency"
package: "voice/s2s"
category: "performance"
complexity: "advanced"
---

# Minimizing Glass-to-Glass Latency

## Problem

You need to minimize end-to-end latency in speech-to-speech systems (from user speaks to AI responds), requiring optimization at every stage: STT, LLM processing, and TTS.

## Solution

Implement latency optimization techniques including streaming STT/TTS, parallel processing where possible, prediction of likely responses, and minimizing buffering. This works because you can start processing audio chunks as they arrive and begin TTS before the complete response is ready.

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

var tracer = otel.Tracer("beluga.voice.s2s.latency")

// LowLatencyS2S optimizes glass-to-glass latency
type LowLatencyS2S struct {
    sttStreaming  bool
    llmStreaming  bool
    ttsStreaming  bool
    parallel      bool
    bufferSize    int
}

// NewLowLatencyS2S creates a low-latency S2S handler
func NewLowLatencyS2S(sttStreaming, llmStreaming, ttsStreaming, parallel bool, bufferSize int) *LowLatencyS2S {
    return &LowLatencyS2S{
        sttStreaming: sttStreaming,
        llmStreaming: llmStreaming,
        ttsStreaming: ttsStreaming,
        parallel:     parallel,
        bufferSize:   bufferSize,
    }
}

// Process optimizes S2S pipeline for low latency
func (ls2s *LowLatencyS2S) Process(ctx context.Context, audioInput <-chan []byte) (<-chan []byte, error) {
    ctx, span := tracer.Start(ctx, "low_latency_s2s.process")
    defer span.End()
    
    audioOutput := make(chan []byte, ls2s.bufferSize)
    
    go func() {
        defer close(audioOutput)
        defer span.End()
        
        // Stage 1: Streaming STT
        textCh := make(chan string, ls2s.bufferSize)
        if ls2s.sttStreaming {
            go ls2s.streamingSTT(ctx, audioInput, textCh)
        } else {
            go ls2s.batchSTT(ctx, audioInput, textCh)
        }
        
        // Stage 2: Streaming LLM
        responseCh := make(chan string, ls2s.bufferSize)
        if ls2s.llmStreaming {
            go ls2s.streamingLLM(ctx, textCh, responseCh)
        } else {
            go ls2s.batchLLM(ctx, textCh, responseCh)
        }
        
        // Stage 3: Streaming TTS
        if ls2s.ttsStreaming {
            ls2s.streamingTTS(ctx, responseCh, audioOutput)
        } else {
            ls2s.batchTTS(ctx, responseCh, audioOutput)
        }
    }()
    
    return audioOutput, nil
}

// streamingSTT processes audio with streaming STT
func (ls2s *LowLatencyS2S) streamingSTT(ctx context.Context, audioIn <-chan []byte, textOut chan<- string) {
    defer close(textOut)
    
    for audio := range audioIn {
        // Process with streaming STT (partial results)
        // text := sttProvider.StreamTranscribe(ctx, audio)
        text := "partial transcription" // Placeholder
        select {
        case textOut <- text:
        case <-ctx.Done():
            return
        }
    }
}

// streamingLLM processes text with streaming LLM
func (ls2s *LowLatencyS2S) streamingLLM(ctx context.Context, textIn <-chan string, responseOut chan<- string) {
    defer close(responseOut)
    
    // Accumulate text
    fullText := ""
    for text := range textIn {
        fullText += text
        
        // Send partial response immediately for low latency
        // response := llmProvider.StreamGenerate(ctx, fullText)
        response := "partial response" // Placeholder
        select {
        case responseOut <- response:
        case <-ctx.Done():
            return
        }
    }
}

// streamingTTS synthesizes with streaming TTS
func (ls2s *LowLatencyS2S) streamingTTS(ctx context.Context, textIn <-chan string, audioOut chan<- []byte) {
    for text := range textIn {
        // Synthesize immediately (streaming TTS)
        // audio := ttsProvider.StreamSynthesize(ctx, text)
        audio := []byte("audio") // Placeholder
        select {
        case audioOut <- audio:
        case <-ctx.Done():
            return
        }
    }
}

func main() {
    ctx := context.Background()
    
    // Create low-latency S2S
    s2s := NewLowLatencyS2S(true, true, true, false, 10)
    
    // Process audio
    audioIn := make(chan []byte, 10)
    audioOut, err := s2s.Process(ctx, audioIn)
    if err != nil {
        log.Fatalf("Failed: %v", err)
    }
    
    fmt.Println("Low-latency S2S created")
}
```

## Explanation

Let's break down what's happening:

1. **Streaming at every stage** - Notice how we use streaming for STT, LLM, and TTS. This allows processing to start as soon as data arrives, rather than waiting for complete input.

2. **Pipeline parallelism** - All stages run concurrently, with each stage processing data as it becomes available. This maximizes throughput.

3. **Minimal buffering** - We use small buffers to minimize latency. Larger buffers reduce latency spikes but increase baseline latency.

**Key insight:** Stream everywhere and minimize buffering. Start TTS as soon as you have the first words of the LLM response, not after the complete response.

## Testing

Here's how to test this solution:

```go
func TestLowLatencyS2S_ProcessesStreaming(t *testing.T) {
    s2s := NewLowLatencyS2S(true, true, true, false, 5)

    audioIn := make(chan []byte, 5)
    audioOut, err := s2s.Process(context.Background(), audioIn)
    require.NoError(t, err)
    
    // Test streaming
    audioIn <- []byte("audio1")
    <-audioOut // Should receive quickly
}
```

## Variations

### Predictive Prefetching

Prefetch likely responses:
```go
func (ls2s *LowLatencyS2S) PredictivePrefetch(ctx context.Context, input string) {
    // Predict likely response and pre-generate
}
```

### Adaptive Buffering

Adjust buffer size based on network conditions:
```go
func (ls2s *LowLatencyS2S) AdaptBufferSize(latency time.Duration) {
    // Increase buffer if latency is high
}
```

## Related Recipes

- **[Voice S2S Handling Speech Interruption](./voice-s2s-handling-speech-interruption.md)** - Handle interruptions
- **[Voice STT Jitter Buffer Management](./voice-stt-jitter-buffer-management.md)** - Handle jitter
- **[Voice Providers Guide](../guides/voice-providers.md)** - For a deeper understanding of S2S
