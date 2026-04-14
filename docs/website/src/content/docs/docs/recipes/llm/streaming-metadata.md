---
title: "Streaming Chunks with Metadata"
description: "Recipe for streaming LLM responses in Go while preserving token counts, model info, and finish reasons using iter.Seq2 and Beluga AI."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Beluga AI, Go LLM streaming, streaming metadata, token counts, iter.Seq2, finish reasons, streaming recipe"
---

## Problem

You need to stream LLM responses while preserving and forwarding metadata (token counts, model info, finish reasons) that arrives with each chunk, not just the text content.

## Solution

Use `model.Stream()` which returns `iter.Seq2[schema.StreamChunk, error]`. Inspect each `StreamChunk` for metadata fields (`FinishReason`, `Usage`, `ModelID`) and accumulate them across the stream.

## Why This Matters

Streaming responses create a fundamental tension between responsiveness and observability. When you consume a stream chunk by chunk, metadata embedded in each chunk (token counts, finish reasons, tool call indicators) gets discarded unless you actively extract it. In a non-streaming call, this metadata arrives with the complete response and is easy to capture. In a streaming call, it arrives incrementally and must be accumulated.

This matters for three reasons. First, cost tracking: LLM APIs charge per token, and knowing exact token counts per request lets you attribute costs to users, features, or tenants. Second, operational monitoring: finish reasons tell you whether a response completed normally, hit a length limit, or was filtered. Third, downstream decision-making: some consumers need to know whether tool calls are present in the stream before it completes.

## Code Example

```go
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync/atomic"
	"time"

	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/llm"
	"github.com/lookatitude/beluga-ai/schema"
	_ "github.com/lookatitude/beluga-ai/llm/providers/openai"
)

// StreamResult holds the accumulated metadata from a streaming call.
type StreamResult struct {
	Text         string
	FinishReason string
	InputTokens  int
	OutputTokens int
	TotalTokens  int
	ModelID      string
	Chunks       int
	Duration     time.Duration
}

// StreamWithMetadata streams a model response and accumulates metadata.
func StreamWithMetadata(ctx context.Context, model llm.ChatModel, msgs []schema.Message) (*StreamResult, error) {
	start := time.Now()
	result := &StreamResult{}

	for chunk, err := range model.Stream(ctx, msgs) {
		if err != nil {
			return result, err
		}

		result.Chunks++

		// Accumulate text delta.
		if chunk.Delta != "" {
			result.Text += chunk.Delta
			fmt.Print(chunk.Delta) // Print in real-time.
		}

		// Capture finish reason when generation completes.
		if chunk.FinishReason != "" {
			result.FinishReason = chunk.FinishReason
		}

		// Capture model ID.
		if chunk.ModelID != "" {
			result.ModelID = chunk.ModelID
		}

		// Accumulate token usage (usually arrives on the final chunk).
		if chunk.Usage != nil {
			result.InputTokens = chunk.Usage.InputTokens
			result.OutputTokens = chunk.Usage.OutputTokens
			result.TotalTokens = chunk.Usage.TotalTokens
		}
	}

	result.Duration = time.Since(start)
	return result, nil
}

// TokenTracker provides atomic counters for concurrent token tracking.
type TokenTracker struct {
	inputTokens  atomic.Int64
	outputTokens atomic.Int64
}

// Add records token usage from a completed stream result.
func (t *TokenTracker) Add(result *StreamResult) {
	t.inputTokens.Add(int64(result.InputTokens))
	t.outputTokens.Add(int64(result.OutputTokens))
}

// Stats returns cumulative input and output token counts.
func (t *TokenTracker) Stats() (input, output int64) {
	return t.inputTokens.Load(), t.outputTokens.Load()
}

func main() {
	ctx := context.Background()

	model, err := llm.New("openai", config.ProviderConfig{
		APIKey: os.Getenv("OPENAI_API_KEY"),
		Model:  "gpt-4o",
	})
	if err != nil {
		slog.Error("model creation failed", "error", err)
		return
	}

	msgs := []schema.Message{
		schema.NewHumanMessage("Explain Go interfaces in one paragraph."),
	}

	tracker := &TokenTracker{}

	result, err := StreamWithMetadata(ctx, model, msgs)
	if err != nil {
		slog.Error("stream failed", "error", err)
		return
	}
	fmt.Println() // Newline after streamed text.

	tracker.Add(result)

	slog.Info("stream complete",
		"finish_reason", result.FinishReason,
		"model", result.ModelID,
		"input_tokens", result.InputTokens,
		"output_tokens", result.OutputTokens,
		"chunks", result.Chunks,
		"duration", result.Duration,
	)

	input, output := tracker.Stats()
	fmt.Printf("Cumulative tokens -- Input: %d, Output: %d\n", input, output)
}
```

## Explanation

1. **`iter.Seq2[schema.StreamChunk, error]`** -- The streaming API is a pull-based iterator. Errors propagate as the second return value from the `for range` loop. Breaking out of the loop signals the producer to stop.

2. **Metadata accumulation** -- `FinishReason`, `ModelID`, and `Usage` are captured as they arrive. `Usage` typically appears on the final chunk from providers that support usage reporting in streaming mode.

3. **Real-time display** -- `chunk.Delta` is printed as it arrives. Accumulate it into `result.Text` if you need the full response string after streaming completes.

4. **`atomic.Int64` for concurrent tracking** -- When multiple goroutines stream simultaneously and share a single `TokenTracker`, atomic operations ensure correct counts without mutex overhead.

## Variations

### Stream with First-Chunk Timing

Track time-to-first-token (TTFT), an important latency metric for user experience:

```go
func streamWithTTFT(ctx context.Context, model llm.ChatModel, msgs []schema.Message) (string, time.Duration, error) {
	start := time.Now()
	var ttft time.Duration
	var firstChunk bool
	var text string

	for chunk, err := range model.Stream(ctx, msgs) {
		if err != nil {
			return text, ttft, err
		}
		if !firstChunk && chunk.Delta != "" {
			ttft = time.Since(start)
			firstChunk = true
		}
		text += chunk.Delta
	}
	return text, ttft, nil
}
```

### Abort Stream on Token Budget

Stop streaming early when accumulated output exceeds a token budget:

```go
func streamWithBudget(ctx context.Context, model llm.ChatModel, msgs []schema.Message, maxTokens int) (string, error) {
	var text string
	var outputTokens int

	for chunk, err := range model.Stream(ctx, msgs) {
		if err != nil {
			return text, err
		}
		text += chunk.Delta
		outputTokens += len(chunk.Delta) / 4 // Approximate token count.
		if outputTokens >= maxTokens {
			break // Signals producer to stop.
		}
	}
	return text, nil
}
```

## Related Recipes

- **[History Trimming](/docs/recipes/llm/history-trimming)** -- Manage conversation history within context limits
- **[Streaming Tool Calls](/docs/recipes/llm/streaming-tool-calls)** -- Handle tool calls in streaming responses
