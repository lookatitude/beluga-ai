---
title: "Streaming Chunks with Metadata"
description: "Stream LLM responses while preserving and forwarding metadata such as token counts, model info, and finish reasons."
---

## Problem

You need to stream LLM responses while preserving and forwarding metadata (token counts, model info, finish reasons) that arrives with each chunk, not just the text content.

## Solution

Implement a streaming wrapper that extracts and forwards metadata from streaming chunks, allowing downstream consumers to track token usage, model information, and completion status in real-time. This works because Beluga AI's streaming interface provides chunks with metadata, and you can extract and forward this information.

## Why This Matters

Streaming responses create a fundamental tension between responsiveness and observability. When you consume a stream chunk by chunk, the metadata embedded in each chunk (token counts, finish reasons, tool call indicators) gets discarded unless you actively extract it. In a non-streaming call, this metadata arrives with the complete response and is easy to capture. In a streaming call, it arrives incrementally and must be accumulated.

This matters for three reasons. First, cost tracking: LLM APIs charge per token, and knowing exact token counts per request lets you attribute costs to users, features, or tenants. Without extraction, you lose this data. Second, operational monitoring: finish reasons tell you whether a response completed normally, hit a length limit, or was filtered by safety checks. Missing this signal means you cannot distinguish between a successful response and a truncated one. Third, downstream decision-making: some consumers need to know whether tool calls are present in the stream before the stream completes, so they can start preparing tool execution in parallel.

The `sync.RWMutex` in the extractor is not incidental. Because metadata is accumulated in a goroutine (the stream consumer) and read from the main goroutine (via `GetAccumulatedMetadata()`), the mutex prevents data races. The buffered output channel (`make(chan ChunkWithMetadata, 10)`) absorbs bursts from fast producers without blocking the stream consumer goroutine, which would otherwise stall the entire stream. OpenTelemetry spans record the final accumulated metadata, making it available in your tracing backend for cost analysis and debugging.

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

    "github.com/lookatitude/beluga-ai/llm"
    "github.com/lookatitude/beluga-ai/schema"
)

var tracer = otel.Tracer("beluga.chatmodels.streaming_metadata")

// ChunkWithMetadata represents a chunk with extracted metadata
type ChunkWithMetadata struct {
    Content      string
    Metadata     map[string]interface{}
    TokenCount   int
    FinishReason string
}

// StreamingMetadataExtractor extracts metadata from streaming chunks
type StreamingMetadataExtractor struct {
    model    llm.ChatModel
    metadata map[string]interface{}
    mu       sync.RWMutex
}

// NewStreamingMetadataExtractor creates a new extractor
func NewStreamingMetadataExtractor(model llm.ChatModel) *StreamingMetadataExtractor {
    return &StreamingMetadataExtractor{
        model:    model,
        metadata: make(map[string]interface{}),
    }
}

// StreamWithMetadata streams responses with metadata
func (sme *StreamingMetadataExtractor) StreamWithMetadata(ctx context.Context, messages []schema.Message) (<-chan ChunkWithMetadata, error) {
    ctx, span := tracer.Start(ctx, "metadata_extractor.stream")
    defer span.End()

    // Start streaming
    streamCh, err := sme.model.StreamChat(ctx, messages)
    if err != nil {
        span.RecordError(err)
        span.SetStatus(trace.StatusError, err.Error())
        return nil, err
    }

    // Create output channel
    outputCh := make(chan ChunkWithMetadata, 10)

    go func() {
        defer close(outputCh)

        totalTokens := 0

        for chunk := range streamCh {
            if chunk.Err != nil {
                span.RecordError(chunk.Err)
                continue
            }

            // Extract metadata
            metadata := sme.extractMetadata(chunk)

            // Count tokens (approximate)
            tokenCount := sme.estimateTokens(chunk.Content)
            totalTokens += tokenCount

            // Update accumulated metadata
            sme.mu.Lock()
            sme.metadata["total_tokens"] = totalTokens
            sme.metadata["model"] = sme.model.GetModelName()
            if chunk.FinishReason != "" {
                sme.metadata["finish_reason"] = chunk.FinishReason
            }
            sme.mu.Unlock()

            // Send chunk with metadata
            chunkWithMeta := ChunkWithMetadata{
                Content:      chunk.Content,
                Metadata:     metadata,
                TokenCount:   tokenCount,
                FinishReason: chunk.FinishReason,
            }

            select {
            case outputCh <- chunkWithMeta:
            case <-ctx.Done():
                return
            }
        }

        // Record final metadata on span
        sme.mu.RLock()
        span.SetAttributes(
            attribute.Int("total_tokens", totalTokens),
            attribute.String("model", sme.model.GetModelName()),
        )
        sme.mu.RUnlock()
        span.SetStatus(trace.StatusOK, "streaming completed")
    }()

    return outputCh, nil
}

// extractMetadata extracts metadata from chunk
func (sme *StreamingMetadataExtractor) extractMetadata(chunk llm.AIMessageChunk) map[string]interface{} {
    metadata := make(map[string]interface{})

    if chunk.FinishReason != "" {
        metadata["finish_reason"] = chunk.FinishReason
    }

    if len(chunk.ToolCallChunks) > 0 {
        metadata["tool_calls"] = len(chunk.ToolCallChunks)
    }

    return metadata
}

// estimateTokens estimates token count (simplified)
func (sme *StreamingMetadataExtractor) estimateTokens(text string) int {
    return len(text) / 4
}

// GetAccumulatedMetadata returns accumulated metadata
func (sme *StreamingMetadataExtractor) GetAccumulatedMetadata() map[string]interface{} {
    sme.mu.RLock()
    defer sme.mu.RUnlock()

    result := make(map[string]interface{})
    for k, v := range sme.metadata {
        result[k] = v
    }
    return result
}

func main() {
    ctx := context.Background()

    // Create extractor with your ChatModel
    // model, _ := llm.New("openai", llm.ProviderConfig{...})
    // extractor := NewStreamingMetadataExtractor(model)

    // Stream with metadata
    // messages := []schema.Message{
    //     schema.NewHumanMessage("Hello"),
    // }
    //
    // chunkCh, err := extractor.StreamWithMetadata(ctx, messages)
    // if err != nil {
    //     log.Fatalf("Failed to stream: %v", err)
    // }
    //
    // for chunk := range chunkCh {
    //     fmt.Printf("Content: %s, Tokens: %d, Metadata: %v\n",
    //         chunk.Content, chunk.TokenCount, chunk.Metadata)
    // }
    //
    // finalMeta := extractor.GetAccumulatedMetadata()
    // fmt.Printf("Final metadata: %v\n", finalMeta)

    fmt.Println("Streaming metadata extractor ready")
}
```

## Explanation

1. **Metadata extraction** -- Each chunk is inspected for metadata (finish reason, tool calls, etc.). This metadata provides context about the streaming response beyond just the text content. The `extractMetadata` method creates a per-chunk metadata map that captures transient signals like tool call counts and finish reasons that would otherwise be lost as the stream progresses.

2. **Accumulation with thread safety** -- Metadata is accumulated across chunks using a `sync.RWMutex`-protected map. Total token count grows incrementally, and the model name and finish reason are captured as they appear. The `RWMutex` allows concurrent reads from `GetAccumulatedMetadata()` without blocking the stream consumer goroutine, which holds a write lock only briefly when updating the map.

3. **Real-time forwarding** -- Metadata is forwarded with each chunk via the `ChunkWithMetadata` struct, allowing downstream consumers to track progress and make decisions in real-time. For example, a UI component could display a running token count, or a rate limiter could throttle based on accumulated usage before the stream completes.

4. **OTel span enrichment** -- Final accumulated metadata is recorded on the OTel span after the stream completes. This makes token usage and model information queryable in your tracing backend, enabling dashboards that correlate cost with latency, user, or feature.

## Testing

```go
func TestStreamingMetadataExtractor_ExtractsMetadata(t *testing.T) {
    mockModel := &MockChatModel{}
    extractor := NewStreamingMetadataExtractor(mockModel)

    messages := []schema.Message{schema.NewHumanMessage("test")}
    chunkCh, err := extractor.StreamWithMetadata(context.Background(), messages)
    require.NoError(t, err)

    chunk := <-chunkCh
    require.NotNil(t, chunk.Metadata)
}
```

## Variations

### Metadata Filtering

Filter which metadata to forward:

```go
func (sme *StreamingMetadataExtractor) StreamWithFilteredMetadata(ctx context.Context, messages []schema.Message, filter func(string) bool) (<-chan ChunkWithMetadata, error) {
    // Filter metadata
}
```

### Metadata Aggregation

Aggregate metadata across multiple streams:

```go
type MetadataAggregator struct {
    streams []*StreamingMetadataExtractor
}
```

## Related Recipes

- **[Multi-step History Trimming](./history-trimming)** — Manage conversation history
- **[Streaming Tool Calls](./streaming-tool-calls)** — Handle streaming with tools
