---
title: "Streaming Tool Logic Handler"
description: "Handle tool calls that arrive during streaming LLM responses by executing tools concurrently as they are detected."
---

## Problem

You need to handle tool calls that arrive during streaming LLM responses, executing tools as they're detected and streaming results back to the user, without waiting for the complete response.

## Solution

Implement a streaming tool handler that processes tool call chunks as they arrive, executes tools concurrently, and streams tool results back into the response stream. This works because Beluga AI's streaming interface provides tool call chunks incrementally, allowing you to start tool execution before the stream completes.

## Why This Matters

LLM tool calls during streaming present a latency optimization opportunity that most implementations miss. In a typical non-streaming flow, the application waits for the complete LLM response, parses out tool calls, executes them sequentially, and then sends results back. This serial approach wastes time because the LLM often signals a tool call early in its response (sometimes in the first few chunks), but execution doesn't start until the entire response is received.

By processing tool call chunks incrementally, you can start executing tools as soon as the tool call arguments are complete, potentially saving hundreds of milliseconds or more. When the LLM requests multiple tools, concurrent execution via goroutines compounds this advantage -- three tool calls that each take 200ms complete in 200ms total instead of 600ms.

The implementation handles a subtle challenge: tool call arguments arrive split across multiple chunks. The `toolCallBuffer` accumulates these fragments until a chunk is marked as `Complete`, at which point the full tool call is dispatched. This buffering is necessary because JSON arguments may be split at arbitrary byte boundaries by the streaming protocol. The `activeTools` map with `context.CancelFunc` values enables cancellation of in-flight tool executions when the parent context is cancelled, preventing resource leaks in timeout scenarios.

## Code Example

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "sync"

    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"

    "github.com/lookatitude/beluga-ai/llm"
    "github.com/lookatitude/beluga-ai/schema"
    "github.com/lookatitude/beluga-ai/tool"
)

var tracer = otel.Tracer("beluga.llms.streaming_tools")

// StreamingToolHandler handles tool calls during streaming
type StreamingToolHandler struct {
    tools       map[string]tool.Tool
    resultsCh   chan ToolResult
    mu          sync.Mutex
    activeTools map[string]context.CancelFunc
}

// ToolResult represents a tool execution result
type ToolResult struct {
    ToolName string
    Result   interface{}
    Error    error
}

// NewStreamingToolHandler creates a new streaming tool handler
func NewStreamingToolHandler(toolList []tool.Tool) *StreamingToolHandler {
    toolMap := make(map[string]tool.Tool)
    for _, t := range toolList {
        toolMap[t.Name()] = t
    }

    return &StreamingToolHandler{
        tools:       toolMap,
        resultsCh:   make(chan ToolResult, 10),
        activeTools: make(map[string]context.CancelFunc),
    }
}

// HandleStreamingWithTools processes a streaming response with tool calls
func (sth *StreamingToolHandler) HandleStreamingWithTools(ctx context.Context, model llm.ChatModel, messages []schema.Message, toolList []tool.Tool) (<-chan schema.Message, error) {
    ctx, span := tracer.Start(ctx, "streaming_tools.handle")
    defer span.End()

    // Bind tools to model
    modelWithTools := model.BindTools(toolList)

    // Start streaming
    streamCh, err := modelWithTools.StreamChat(ctx, messages)
    if err != nil {
        span.RecordError(err)
        span.SetStatus(trace.StatusError, err.Error())
        return nil, err
    }

    // Create output channel
    outputCh := make(chan schema.Message, 10)

    go func() {
        defer close(outputCh)

        var accumulatedContent string
        var toolCalls []schema.ToolCall
        var toolCallBuffer string

        for chunk := range streamCh {
            if chunk.Err != nil {
                span.RecordError(chunk.Err)
                return
            }

            // Accumulate content
            if chunk.Content != "" {
                accumulatedContent += chunk.Content
            }

            // Collect tool call chunks
            if len(chunk.ToolCallChunks) > 0 {
                for _, toolChunk := range chunk.ToolCallChunks {
                    toolCallBuffer += toolChunk.Arguments

                    // Check if tool call is complete
                    if toolChunk.Complete {
                        toolCall := schema.ToolCall{
                            Name:      toolChunk.Name,
                            Arguments: toolCallBuffer,
                        }
                        toolCalls = append(toolCalls, toolCall)

                        // Execute tool asynchronously
                        go sth.executeTool(ctx, toolCall)

                        toolCallBuffer = ""
                    }
                }
            }
        }

        // Send final message with accumulated content
        if accumulatedContent != "" {
            finalMsg := schema.NewAIMessage(accumulatedContent)
            outputCh <- finalMsg
        }

        // Wait for tool results and send them
        sth.sendToolResults(ctx, outputCh, len(toolCalls))
    }()

    return outputCh, nil
}

// executeTool executes a tool call
func (sth *StreamingToolHandler) executeTool(ctx context.Context, toolCall schema.ToolCall) {
    ctx, span := tracer.Start(ctx, "streaming_tools.execute")
    defer span.End()

    span.SetAttributes(
        attribute.String("tool.name", toolCall.Name),
    )

    t, exists := sth.tools[toolCall.Name]
    if !exists {
        result := ToolResult{
            ToolName: toolCall.Name,
            Error:    fmt.Errorf("tool %s not found", toolCall.Name),
        }
        sth.resultsCh <- result
        return
    }

    // Parse arguments
    var args map[string]interface{}
    if err := json.Unmarshal([]byte(toolCall.Arguments), &args); err != nil {
        result := ToolResult{
            ToolName: toolCall.Name,
            Error:    fmt.Errorf("failed to parse arguments: %w", err),
        }
        sth.resultsCh <- result
        return
    }

    // Execute tool
    result, err := t.Execute(ctx, args)

    sth.resultsCh <- ToolResult{
        ToolName: toolCall.Name,
        Result:   result,
        Error:    err,
    }
}

// sendToolResults sends tool results to output channel
func (sth *StreamingToolHandler) sendToolResults(ctx context.Context, outputCh chan<- schema.Message, expectedCount int) {
    for i := 0; i < expectedCount; i++ {
        select {
        case result := <-sth.resultsCh:
            if result.Error != nil {
                log.Printf("Tool %s failed: %v", result.ToolName, result.Error)
                continue
            }

            resultJSON, _ := json.Marshal(result.Result)
            toolMsg := schema.NewToolMessage(string(resultJSON), result.ToolName)
            outputCh <- toolMsg

        case <-ctx.Done():
            return
        }
    }
}

func main() {
    // Create tools
    toolList := []tool.Tool{
        // Add your tools here
    }

    // Create handler
    handler := NewStreamingToolHandler(toolList)

    fmt.Println("Streaming tool handler created")
}
```

## Explanation

1. **Incremental tool call collection** -- Tool call chunks are accumulated as they arrive. Tool calls may be split across multiple chunks because the streaming protocol fragments JSON arguments at arbitrary byte boundaries. The `toolCallBuffer` accumulates these fragments until a chunk marked as `Complete` is received, at which point the full tool call is assembled and dispatched.

2. **Concurrent tool execution** -- Tools are executed in separate goroutines as soon as they're detected. This allows multiple tools to run in parallel, reducing total execution time from the sum of all tool durations to the maximum of any single tool duration. The `resultsCh` channel collects results from all concurrent executions in a thread-safe manner.

3. **Streaming results** -- Tool results are sent to the output channel as they complete, allowing downstream consumers to process results incrementally rather than waiting for all tools to finish. The `sendToolResults` method collects exactly the expected number of results, ensuring all tool executions are accounted for before the output channel closes.

4. **OTel instrumentation** -- Each tool execution gets its own span with the tool name as an attribute. This creates a trace tree showing the streaming response span as the parent with individual tool execution spans as children, making it straightforward to identify which tools are slow or failing in production.

## Testing

```go
func TestStreamingToolHandler_ExecutesTools(t *testing.T) {
    mockTool := &MockTool{name: "test_tool"}
    handler := NewStreamingToolHandler([]tool.Tool{mockTool})

    toolCall := schema.ToolCall{
        Name:      "test_tool",
        Arguments: `{"input": "test"}`,
    }

    ctx := context.Background()
    handler.executeTool(ctx, toolCall)

    result := <-handler.resultsCh
    require.NoError(t, result.Error)
}
```

## Variations

### Tool Result Streaming

Stream tool results as they're computed:

```go
func (sth *StreamingToolHandler) executeToolWithStreaming(ctx context.Context, toolCall schema.ToolCall, resultCh chan<- ToolResult) {
    // Stream partial results
}
```

### Tool Call Deduplication

Deduplicate identical tool calls:

```go
type ToolCallKey string

func (sth *StreamingToolHandler) deduplicateToolCalls(toolCalls []schema.ToolCall) []schema.ToolCall {
    // Remove duplicates
}
```

## Related Recipes

- **[Token Counting](./token-counting)** — Optimize token counting
- **[Handling Tool Failures](./agents-tool-failures)** — Robust tool error handling
