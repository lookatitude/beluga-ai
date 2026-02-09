---
title: "Minimizing Glass-to-Glass Latency"
description: "Optimize end-to-end voice latency with streaming STT, LLM, and TTS pipeline parallelism."
---

## Problem

You need to minimize end-to-end latency in speech-to-speech systems (from user speaks to AI responds), requiring optimization at every stage: STT, LLM processing, and TTS.

## Solution

Implement latency optimization using streaming at every pipeline stage, parallel processing between stages, and minimal buffering. Start processing audio chunks as they arrive and begin TTS before the complete LLM response is ready.

## Code Example

```go
package main

import (
	"context"
	"fmt"
	"log"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

var tracer = otel.Tracer("beluga.voice.s2s.latency")

// LowLatencyS2S optimizes glass-to-glass latency.
type LowLatencyS2S struct {
	sttStreaming bool
	llmStreaming bool
	ttsStreaming bool
	bufferSize   int
}

func NewLowLatencyS2S(sttStreaming, llmStreaming, ttsStreaming bool, bufferSize int) *LowLatencyS2S {
	return &LowLatencyS2S{
		sttStreaming: sttStreaming,
		llmStreaming: llmStreaming,
		ttsStreaming: ttsStreaming,
		bufferSize:   bufferSize,
	}
}

// Process runs the S2S pipeline optimized for low latency.
func (ls2s *LowLatencyS2S) Process(ctx context.Context, audioInput <-chan []byte) (<-chan []byte, error) {
	_, span := tracer.Start(ctx, "low_latency_s2s.process")
	defer span.End()

	audioOutput := make(chan []byte, ls2s.bufferSize)

	go func() {
		defer close(audioOutput)

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

func (ls2s *LowLatencyS2S) streamingSTT(ctx context.Context, audioIn <-chan []byte, textOut chan<- string) {
	defer close(textOut)
	for range audioIn {
		text := "partial transcription"
		select {
		case textOut <- text:
		case <-ctx.Done():
			return
		}
	}
}

func (ls2s *LowLatencyS2S) batchSTT(ctx context.Context, audioIn <-chan []byte, textOut chan<- string) {
	defer close(textOut)
	// Collect audio, then transcribe in batch
}

func (ls2s *LowLatencyS2S) streamingLLM(ctx context.Context, textIn <-chan string, responseOut chan<- string) {
	defer close(responseOut)
	fullText := ""
	for text := range textIn {
		fullText += text
		response := "partial response"
		select {
		case responseOut <- response:
		case <-ctx.Done():
			return
		}
	}
}

func (ls2s *LowLatencyS2S) batchLLM(ctx context.Context, textIn <-chan string, responseOut chan<- string) {
	defer close(responseOut)
	// Collect text, then generate in batch
}

func (ls2s *LowLatencyS2S) streamingTTS(ctx context.Context, textIn <-chan string, audioOut chan<- []byte) {
	for text := range textIn {
		_ = text
		audio := []byte("audio")
		select {
		case audioOut <- audio:
		case <-ctx.Done():
			return
		}
	}
}

func (ls2s *LowLatencyS2S) batchTTS(ctx context.Context, textIn <-chan string, audioOut chan<- []byte) {
	// Collect text, then synthesize in batch
}

func main() {
	ctx := context.Background()

	s2s := NewLowLatencyS2S(true, true, true, 10)

	audioIn := make(chan []byte, 10)
	audioOut, err := s2s.Process(ctx, audioIn)
	if err != nil {
		log.Fatalf("Failed: %v", err)
	}

	_ = audioOut
	fmt.Println("Low-latency S2S pipeline created")
}
```

## Explanation

1. **Streaming at every stage** -- STT, LLM, and TTS all use streaming mode. Processing starts as soon as data arrives rather than waiting for complete input.

2. **Pipeline parallelism** -- All stages run concurrently via goroutines, with each stage processing data as it becomes available from the previous stage. This maximizes throughput and minimizes idle time.

3. **Minimal buffering** -- Small channel buffers minimize latency. Larger buffers reduce latency spikes but increase baseline latency.

**Key insight:** Stream everywhere and minimize buffering. Start TTS as soon as you have the first words of the LLM response, not after the complete response is ready.

## Variations

### Predictive Prefetching

Predict likely responses and pre-generate audio before the final transcript arrives to further reduce latency.

### Adaptive Buffering

Adjust buffer size based on network conditions. Increase buffers when latency is high to smooth delivery.

## Related Recipes

- **[Handling Speech Interruption](./speech-interruption)** -- Handle user interruptions during AI speech
- **[Jitter Buffer Management](./jitter-buffer)** -- Smooth network delivery irregularities
- **[Voice Backends Configuration](./voice-backends)** -- Backend provider setup
