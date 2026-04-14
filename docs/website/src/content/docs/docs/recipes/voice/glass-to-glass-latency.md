---
title: "Glass-to-Glass Latency Optimization"
description: "Recipe for minimizing end-to-end voice latency in Go with streaming STT, parallel LLM processing, and TTS pipeline overlap for sub-second responses."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Beluga AI, glass-to-glass latency, Go voice optimization, streaming STT, pipeline parallelism, low latency voice, performance recipe"
---

## Problem

End-to-end latency in speech-to-speech systems determines whether conversations feel natural or frustratingly slow. Glass-to-glass latency spans from the moment a user stops speaking until they hear the AI's response, encompassing STT transcription, LLM processing, and TTS synthesis. Each stage introduces delays: STT waits for complete utterances, LLMs process entire inputs before generating responses, and TTS synthesizes full texts before playback begins. Users expect sub-second response times similar to human conversations, where response latencies beyond 300-500ms feel sluggish. Reducing this latency requires optimizing every pipeline stage and eliminating sequential processing wherever possible.

## Solution

Minimize latency by streaming at every pipeline stage and processing stages in parallel rather than sequentially. Streaming STT produces partial transcripts as audio arrives rather than waiting for silence. Streaming LLMs generate token-by-token responses as soon as they have sufficient context from partial transcripts. Streaming TTS begins synthesizing audio from the first few LLM tokens rather than waiting for complete sentences. Pipeline parallelism means all stages run concurrently via goroutines, with each stage consuming data as it becomes available from the previous stage.

This architecture eliminates wait times between stages. The LLM starts processing while STT is still receiving audio. TTS begins synthesis while the LLM is still generating. Users hear responses before the full interaction completes, dramatically reducing perceived latency. Minimal buffering further reduces delays by transmitting data as soon as it is ready rather than batching into larger chunks.

The key architectural decision is choosing stream-everywhere over batch-and-wait patterns. Every interface uses streaming (Beluga's `iter.Seq2[T, error]`), every goroutine processes data immediately, and buffer sizes remain small to minimize queueing delays.

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

The code uses channels to connect pipeline stages, with each stage processing data as it arrives. Streaming flags control whether each component uses streaming or batch mode, allowing you to measure latency impact per stage. Buffer sizes control how much data queues between stages: small buffers minimize latency but may cause backpressure under load.

## Explanation

1. **Streaming at every stage** -- Each component (STT, LLM, TTS) uses streaming mode, producing partial results as soon as data arrives. This eliminates the wait-for-completion pattern where each stage must finish before the next begins. Streaming STT emits partial transcripts after each audio chunk. Streaming LLMs generate tokens incrementally. Streaming TTS synthesizes audio sentence-by-sentence or even word-by-word. The cumulative latency reduction is substantial: instead of waiting seconds for complete STT + LLM + TTS, you wait milliseconds for the first partial results to flow through all stages.

2. **Pipeline parallelism** -- All stages run concurrently in separate goroutines, connected by channels. This parallelism means the LLM processes the first few words of a transcript while STT is still transcribing later words. TTS synthesizes audio for the beginning of a response while the LLM is still generating the end. Each stage operates at maximum throughput because it never waits idle for upstream stages to complete. This pattern is fundamental to Beluga's frame-based voice architecture, where FrameProcessor components compose into concurrent pipelines.

3. **Minimal buffering** -- Small channel buffers (10 items) ensure data flows immediately rather than accumulating. Larger buffers smooth over latency spikes and network jitter but increase baseline latency because data queues rather than processing immediately. The optimal buffer size balances latency and robustness: too small and you get backpressure stalls, too large and you add unnecessary delay. Start with 10-item buffers and adjust based on your throughput requirements and tolerance for latency variance.

**Key insight:** Stream everywhere and minimize buffering. The fastest speech-to-speech systems start TTS synthesis as soon as they have the first few tokens of an LLM response, not after the complete response is ready. This incremental processing pattern reduces perceived latency by 50-80% compared to batch-and-wait architectures. Measure glass-to-glass latency using OpenTelemetry traces to identify bottlenecks in your specific configuration.

## Variations

### Predictive Prefetching

Predict likely responses and pre-generate audio before the final transcript arrives to further reduce latency. Use n-gram models or lightweight LLMs to predict common continuations and speculatively synthesize audio that may be needed.

### Adaptive Buffering

Adjust buffer size based on network conditions. Increase buffers when latency is high to smooth delivery, decrease when latency is low to minimize queueing delay. Monitor channel length metrics to detect when buffers fill.

## Related Recipes

- **[Handling Speech Interruption](./speech-interruption)** -- Handle user interruptions during AI speech
- **[Jitter Buffer Management](./jitter-buffer)** -- Smooth network delivery irregularities
- **[Voice Backends Configuration](./voice-backends)** -- Backend provider setup
