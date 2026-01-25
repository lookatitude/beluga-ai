---
title: "Streaming Chunks with Metadata"
package: "chatmodels"
category: "streaming"
complexity: "intermediate"
---

# Streaming Chunks with Metadata

## Problem

You need to stream LLM responses while preserving and forwarding metadata (token counts, model info, finish reasons) that arrives with each chunk, not just the text content.

## Solution

Implement a streaming wrapper that extracts and forwards metadata from streaming chunks, allowing downstream consumers to track token usage, model information, and completion status in real-time. This works because Beluga AI's streaming interface provides chunks with metadata, and you can extract and forward this information.

## Code Example
```go
package main

import (
    "context"
    "fmt"
    "log"
    
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"
    
    "github.com/lookatitude/beluga-ai/pkg/llms/iface"
    "github.com/lookatitude/beluga-ai/pkg/schema"
)

var tracer = otel.Tracer("beluga.chatmodels.streaming_metadata")

// ChunkWithMetadata represents a chunk with extracted metadata
type ChunkWithMetadata struct {
    Content     string
    Metadata    map[string]interface{}
    TokenCount  int
    FinishReason string
}

// StreamingMetadataExtractor extracts metadata from streaming chunks
type StreamingMetadataExtractor struct {
    model    iface.ChatModel
    metadata map[string]interface{}
    mu       sync.RWMutex
}

// NewStreamingMetadataExtractor creates a new extractor
func NewStreamingMetadataExtractor(model iface.ChatModel) *StreamingMetadataExtractor {
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
        defer span.End()
        
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
        
        // Send final metadata
        sme.mu.RLock()
        finalMeta := make(map[string]interface{})
        for k, v := range sme.metadata {
            finalMeta[k] = v
        }
        sme.mu.RUnlock()
        
        span.SetAttributes(
            attribute.Int("total_tokens", totalTokens),
            attribute.String("model", sme.model.GetModelName()),
        )
        span.SetStatus(trace.StatusOK, "streaming completed")
    }()
    
    return outputCh, nil
}

// extractMetadata extracts metadata from chunk
func (sme *StreamingMetadataExtractor) extractMetadata(chunk iface.AIMessageChunk) map[string]interface{} {
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
    // Simple estimation: ~4 characters per token
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

    // Create extractor
    // model := yourChatModel
    extractor := NewStreamingMetadataExtractor(model)
    
    // Stream with metadata
    messages := []schema.Message{
        schema.NewHumanMessage("Hello"),
    }
    
    chunkCh, err := extractor.StreamWithMetadata(ctx, messages)
    if err != nil {
        log.Fatalf("Failed to stream: %v", err)
    }
    
    // Process chunks
    for chunk := range chunkCh {
        fmt.Printf("Content: %s, Tokens: %d, Metadata: %v\n", 
            chunk.Content, chunk.TokenCount, chunk.Metadata)
    }
    
    // Get final metadata
    finalMeta := extractor.GetAccumulatedMetadata()
    fmt.Printf("Final metadata: %v\n", finalMeta)
}
```

## Explanation

Let's break down what's happening:

1. **Metadata extraction** - Notice how we extract metadata from each chunk (finish reason, tool calls, etc.). This metadata provides context about the streaming response beyond just the text.

2. **Accumulation** - We accumulate metadata across chunks (like total token count). This gives you a complete picture of the streaming operation when it completes.

3. **Real-time forwarding** - Metadata is forwarded with each chunk, allowing downstream consumers to track progress and make decisions in real-time.

```go
**Key insight:** Preserve metadata during streaming. It contains valuable information about token usage, completion status, and model behavior that's useful for monitoring and cost tracking.

## Testing

```
Here's how to test this solution:
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

## Variations

### Metadata Filtering

Filter which metadata to forward:
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

- **[Chatmodels Multi-step History Trimming](./chatmodels-multi-step-history-trimming.md)** - Manage conversation history
- **[LLMs Streaming Tool Logic Handler](./llms-streaming-tool-logic-handler.md)** - Handle streaming with tools
- **[Chatmodels Package Guide](../package_design_patterns.md)** - For a deeper understanding of chatmodels
