# Streaming LLM Calls with Tool Calling

## Introduction

Welcome to this guide on combining streaming LLM responses with tool calling in Beluga AI. By the end, you'll understand how to build responsive AI applications that can both stream responses to users in real-time *and* execute tools mid-conversation.

**What you'll learn:**
- How streaming works in Beluga AI's LLM package
- How to bind tools to an LLM and receive tool call chunks during streaming
- How to process tool calls and continue the conversation
- Best practices for error handling and OTEL instrumentation

**Why this matters:**
Imagine you're building a chatbot that needs to look up weather data while responding. Streaming lets you show the response as it's generated, while tool calls let you fetch data mid-conversation. Together, they create a responsive, interactive experience that feels natural to users.

## Prerequisites

Before we begin, make sure you have:

- **Go 1.24+** installed ([installation guide](https://go.dev/doc/install))
- **Beluga AI Framework** installed (`go get github.com/lookatitude/beluga-ai`)
- **OpenAI API key** (or another provider) - you'll need this for LLM access. Get one at [OpenAI](https://platform.openai.com/api-keys)
- **Understanding of basic LLM concepts** - if you're new to this, check out our [LLM concepts guide](../concepts/llms.md)

## Concepts

Before we start coding, let's understand the key concepts:

### Streaming in Beluga AI

Streaming allows LLM responses to be delivered as they're generated, rather than waiting for the complete response. In Beluga AI, streaming works through Go channels:

```
┌─────────────────┐       ┌─────────────────┐       ┌─────────────────┐
│   Your Code     │◀──────│   LLM Provider  │◀──────│   LLM API       │
│  (consumes ch)  │ chan  │  (StreamChat)   │ HTTP  │  (OpenAI, etc)  │
└─────────────────┘       └─────────────────┘       └─────────────────┘
```

Each message chunk (`iface.AIMessageChunk`) may contain:
- `Content`: Text content of the chunk
- `ToolCallChunks`: Any tool calls the LLM wants to make
- `Err`: Any error that occurred
- `AdditionalArgs`: Metadata like finish reason

### Tool Calling

Tool calling (also known as function calling) allows the LLM to request that your code execute specific functions. The flow looks like this:

```
┌──────────┐    ┌─────────┐    ┌──────────┐    ┌─────────┐    ┌──────────┐
│  Prompt  │───▶│   LLM   │───▶│ Tool Call│───▶│  Your   │───▶│ Tool     │
│          │    │         │    │  Chunk   │    │  Code   │    │  Result  │
└──────────┘    └─────────┘    └──────────┘    └─────────┘    └──────────┘
                                                                    │
                                                                    ▼
                               ┌──────────┐    ┌─────────┐    ┌──────────┐
                               │ Continue │◀───│   LLM   │◀───│ Result   │
                               │ Response │    │         │    │ Message  │
                               └──────────┘    └─────────┘    └──────────┘
```

### Combining Streaming with Tool Calls

When streaming with tools, you'll receive chunks that may contain text, tool calls, or both. Your code needs to:

1. Stream text chunks to the user immediately
2. Detect when tool call chunks arrive
3. Execute the requested tools
4. Send tool results back to the LLM
5. Continue streaming the final response

## Step-by-Step Tutorial

Now let's build this step by step.

### Step 1: Set up the LLM client with tool binding

First, we'll create an LLM client and define the tools we want to make available.

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "os"

    "github.com/lookatitude/beluga-ai/pkg/llms"
    "github.com/lookatitude/beluga-ai/pkg/llms/iface"
    "github.com/lookatitude/beluga-ai/pkg/schema"
    "github.com/lookatitude/beluga-ai/pkg/tools"
)

func main() {
    ctx := context.Background()
    
    // Create the LLM client - we use OpenAI here, but the pattern
    // works for any provider that supports streaming and tools
    client, err := llms.NewOpenAIChat(
        llms.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
        llms.WithModel("gpt-4"),
    )
    if err != nil {
        log.Fatalf("Failed to create LLM client: %v", err)
    }
    
    // Define our tools - the LLM will see these and can request to call them
    weatherTool := tools.NewSimpleTool(
        "get_weather",
        "Get the current weather for a location",
        func(ctx context.Context, args map[string]any) (string, error) {
            location, _ := args["location"].(string)
            // In production, you'd call a real weather API here
            return fmt.Sprintf(`{"location": "%s", "temp": 72, "conditions": "sunny"}`, location), nil
        },
        tools.WithParameter("location", "string", "The city and state, e.g. San Francisco, CA", true),
    )
    
    // Bind tools to the model - this returns a new model instance with tools attached
    modelWithTools := client.BindTools([]tools.Tool{weatherTool})
    
    // ... continue to streaming
}
```

**What you'll see:**
No output yet - we're just setting up. If there's an error, you'll see a log message about failing to create the client.

**Why this works:** `BindTools` creates a new model instance that includes tool definitions in every request. The LLM provider (OpenAI, Anthropic, etc.) receives these tool definitions and can decide when to call them based on the user's prompt.

### Step 2: Start streaming with the prompt

Now let's send a message that might trigger a tool call:

```go
    // Build our message - something that will likely trigger a tool call
    messages := []schema.Message{
        schema.NewHumanMessage("What's the weather like in San Francisco right now?"),
    }
    
    // Start streaming - this returns a channel, not a single response
    streamChan, err := modelWithTools.StreamChat(ctx, messages)
    if err != nil {
        log.Fatalf("Failed to start streaming: %v", err)
    }
    
    fmt.Println("Streaming response:")
```

**What you'll see:**
```
Streaming response:
```

**Why this works:** `StreamChat` initiates a streaming request to the LLM. It returns a channel (`<-chan iface.AIMessageChunk`) that will receive chunks as they arrive. This is non-blocking - chunks arrive asynchronously.

### Step 3: Process streaming chunks

Here's where the magic happens. We'll process each chunk, handle text content, and detect tool calls:

```go
    var pendingToolCalls []schema.ToolCallChunk
    var responseText strings.Builder
    
    for chunk := range streamChan {
        // Check for errors first
        if chunk.Err != nil {
            log.Fatalf("Stream error: %v", chunk.Err)
        }
        
        // Print text content as it arrives - this is what creates the "typing" effect
        if chunk.Content != "" {
            fmt.Print(chunk.Content)
            responseText.WriteString(chunk.Content)
        }
        
        // Collect tool call chunks - these may arrive across multiple chunks
        if len(chunk.ToolCallChunks) > 0 {
            pendingToolCalls = append(pendingToolCalls, chunk.ToolCallChunks...)
        }
    }
    fmt.Println() // Newline after streaming completes
```

**What you'll see:**
If the LLM decides to call the weather tool, you might see partial text or no text before the tool call. If it doesn't need tools, you'll see the full response streamed.

**Why this works:** The channel receives chunks in order as they're generated. Text content is printed immediately for real-time feedback. Tool call chunks are collected for processing after streaming completes.

### Step 4: Execute tool calls and continue the conversation

When tool calls are detected, we need to execute them and send results back to the LLM:

```go
    // If we have tool calls, execute them and continue the conversation
    if len(pendingToolCalls) > 0 {
        fmt.Println("\n--- Executing tool calls ---")
        
        // Build tool results
        var toolResults []schema.Message
        for _, tc := range pendingToolCalls {
            fmt.Printf("Calling tool: %s with args: %s\n", tc.Name, tc.Arguments)
            
            // Parse arguments and execute the tool
            var args map[string]any
            if err := json.Unmarshal([]byte(tc.Arguments), &args); err != nil {
                log.Printf("Failed to parse tool arguments: %v", err)
                continue
            }
            
            // Find and execute the matching tool
            result, err := weatherTool.Execute(ctx, args)
            if err != nil {
                log.Printf("Tool execution failed: %v", err)
                // Send error result back to LLM
                toolResults = append(toolResults, schema.NewToolMessage(tc.ID, tc.Name, fmt.Sprintf("Error: %v", err)))
                continue
            }
            
            fmt.Printf("Tool result: %s\n", result)
            toolResults = append(toolResults, schema.NewToolMessage(tc.ID, tc.Name, result))
        }
        
        // Continue the conversation with tool results
        messages = append(messages, schema.NewAIMessage(responseText.String()))
        messages = append(messages, toolResults...)
        
        fmt.Println("\n--- Continuing response ---")
        
        // Stream the final response
        finalChan, err := modelWithTools.StreamChat(ctx, messages)
        if err != nil {
            log.Fatalf("Failed to continue streaming: %v", err)
        }
        
        for chunk := range finalChan {
            if chunk.Err != nil {
                log.Fatalf("Stream error: %v", chunk.Err)
            }
            fmt.Print(chunk.Content)
        }
        fmt.Println()
    }
```

**What you'll see:**
```
Streaming response:
--- Executing tool calls ---
Calling tool: get_weather with args: {"location": "San Francisco, CA"}
Tool result: {"location": "San Francisco, CA", "temp": 72, "conditions": "sunny"}

--- Continuing response ---
The current weather in San Francisco is sunny with a temperature of 72°F. Perfect weather for being outdoors!
```

**Why this works:** After collecting tool calls, we execute each one and create `ToolMessage` responses. These are added to the conversation history, and we make another streaming request. The LLM now has the tool results and can generate a final response.

## Code Examples

Here's a complete, production-ready example combining everything we've learned with proper OTEL instrumentation:

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "os"
    "strings"
    "time"

    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/codes"
    "go.opentelemetry.io/otel/trace"

    "github.com/lookatitude/beluga-ai/pkg/llms"
    "github.com/lookatitude/beluga-ai/pkg/llms/iface"
    "github.com/lookatitude/beluga-ai/pkg/schema"
    "github.com/lookatitude/beluga-ai/pkg/tools"
)

// We define a tracer for observability - you'll see this in your tracing dashboard
var tracer = otel.Tracer("beluga.streaming.example")

// StreamingToolCallResult holds the result of a streaming + tool call session
type StreamingToolCallResult struct {
    ResponseText   string
    ToolsCalled    []string
    TotalDuration  time.Duration
    ChunksReceived int
}

// StreamWithTools demonstrates streaming LLM calls with tool calling
func StreamWithTools(ctx context.Context, client iface.ChatModel, prompt string, availableTools []tools.Tool) (*StreamingToolCallResult, error) {
    ctx, span := tracer.Start(ctx, "streaming.with_tools",
        trace.WithAttributes(
            attribute.String("prompt", prompt),
            attribute.Int("tools_count", len(availableTools)),
        ))
    defer span.End()

    start := time.Now()
    result := &StreamingToolCallResult{}

    // Bind tools to model
    modelWithTools := client.BindTools(availableTools)

    // Create initial messages
    messages := []schema.Message{
        schema.NewHumanMessage(prompt),
    }

    // Process streaming response (may include tool calls)
    responseText, toolCalls, chunks, err := processStream(ctx, modelWithTools, messages)
    if err != nil {
        span.RecordError(err)
        span.SetStatus(codes.Error, err.Error())
        return nil, fmt.Errorf("initial stream failed: %w", err)
    }

    result.ResponseText = responseText
    result.ChunksReceived = chunks

    // Handle tool calls if any
    if len(toolCalls) > 0 {
        span.AddEvent("tool_calls_detected", trace.WithAttributes(
            attribute.Int("tool_call_count", len(toolCalls)),
        ))

        toolResults, toolNames, err := executeToolCalls(ctx, toolCalls, availableTools)
        if err != nil {
            span.RecordError(err)
            // Continue with partial results - don't fail completely
            log.Printf("Some tool calls failed: %v", err)
        }

        result.ToolsCalled = toolNames

        // Continue conversation with tool results
        messages = append(messages, schema.NewAIMessage(responseText))
        messages = append(messages, toolResults...)

        finalText, _, finalChunks, err := processStream(ctx, modelWithTools, messages)
        if err != nil {
            span.RecordError(err)
            span.SetStatus(codes.Error, err.Error())
            return nil, fmt.Errorf("continuation stream failed: %w", err)
        }

        result.ResponseText = finalText
        result.ChunksReceived += finalChunks
    }

    result.TotalDuration = time.Since(start)
    span.SetAttributes(
        attribute.Int("total_chunks", result.ChunksReceived),
        attribute.Int("tools_called", len(result.ToolsCalled)),
        attribute.Float64("duration_ms", float64(result.TotalDuration.Milliseconds())),
    )
    span.SetStatus(codes.Ok, "")

    return result, nil
}

// processStream handles the streaming loop and returns text + tool calls
func processStream(ctx context.Context, model iface.ChatModel, messages []schema.Message) (string, []schema.ToolCallChunk, int, error) {
    ctx, span := tracer.Start(ctx, "streaming.process_stream")
    defer span.End()

    streamChan, err := model.StreamChat(ctx, messages)
    if err != nil {
        span.RecordError(err)
        return "", nil, 0, fmt.Errorf("failed to start stream: %w", err)
    }

    var responseText strings.Builder
    var toolCalls []schema.ToolCallChunk
    chunks := 0

    for chunk := range streamChan {
        chunks++

        // Check for stream errors
        if chunk.Err != nil {
            span.RecordError(chunk.Err)
            return responseText.String(), toolCalls, chunks, fmt.Errorf("stream error: %w", chunk.Err)
        }

        // Accumulate text content
        if chunk.Content != "" {
            responseText.WriteString(chunk.Content)
        }

        // Collect tool calls
        if len(chunk.ToolCallChunks) > 0 {
            toolCalls = append(toolCalls, chunk.ToolCallChunks...)
        }
    }

    span.SetAttributes(
        attribute.Int("chunks_received", chunks),
        attribute.Int("tool_calls_count", len(toolCalls)),
        attribute.Int("response_length", responseText.Len()),
    )

    return responseText.String(), toolCalls, chunks, nil
}

// executeToolCalls executes all pending tool calls and returns results
func executeToolCalls(ctx context.Context, toolCalls []schema.ToolCallChunk, availableTools []tools.Tool) ([]schema.Message, []string, error) {
    ctx, span := tracer.Start(ctx, "streaming.execute_tools")
    defer span.End()

    // Build a tool lookup map for efficient access
    toolMap := make(map[string]tools.Tool)
    for _, t := range availableTools {
        toolMap[t.Name()] = t
    }

    var results []schema.Message
    var toolNames []string
    var lastErr error

    for _, tc := range toolCalls {
        toolNames = append(toolNames, tc.Name)

        tool, ok := toolMap[tc.Name]
        if !ok {
            lastErr = fmt.Errorf("unknown tool: %s", tc.Name)
            results = append(results, schema.NewToolMessage(tc.ID, tc.Name, fmt.Sprintf("Error: unknown tool %s", tc.Name)))
            continue
        }

        // Parse arguments
        var args map[string]any
        if err := json.Unmarshal([]byte(tc.Arguments), &args); err != nil {
            lastErr = fmt.Errorf("failed to parse args for %s: %w", tc.Name, err)
            results = append(results, schema.NewToolMessage(tc.ID, tc.Name, fmt.Sprintf("Error: %v", err)))
            continue
        }

        // Execute the tool
        toolResult, err := tool.Execute(ctx, args)
        if err != nil {
            lastErr = fmt.Errorf("tool %s failed: %w", tc.Name, err)
            results = append(results, schema.NewToolMessage(tc.ID, tc.Name, fmt.Sprintf("Error: %v", err)))
            continue
        }

        results = append(results, schema.NewToolMessage(tc.ID, tc.Name, toolResult))
    }

    span.SetAttributes(
        attribute.Int("tools_executed", len(toolCalls)),
        attribute.StringSlice("tool_names", toolNames),
    )

    return results, toolNames, lastErr
}

func main() {
    ctx := context.Background()

    // Create LLM client
    client, err := llms.NewOpenAIChat(
        llms.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
        llms.WithModel("gpt-4"),
    )
    if err != nil {
        log.Fatalf("Failed to create client: %v", err)
    }

    // Define available tools
    weatherTool := tools.NewSimpleTool(
        "get_weather",
        "Get the current weather for a location",
        func(ctx context.Context, args map[string]any) (string, error) {
            location, _ := args["location"].(string)
            return fmt.Sprintf(`{"location": "%s", "temp": 72, "conditions": "sunny"}`, location), nil
        },
        tools.WithParameter("location", "string", "The city and state", true),
    )

    // Run streaming with tools
    result, err := StreamWithTools(ctx, client, "What's the weather in San Francisco?", []tools.Tool{weatherTool})
    if err != nil {
        log.Fatalf("Streaming failed: %v", err)
    }

    fmt.Printf("\n--- Results ---\n")
    fmt.Printf("Response: %s\n", result.ResponseText)
    fmt.Printf("Tools called: %v\n", result.ToolsCalled)
    fmt.Printf("Chunks received: %d\n", result.ChunksReceived)
    fmt.Printf("Duration: %v\n", result.TotalDuration)
}
```

## Testing

Testing streaming with tool calls requires careful handling of channels and asynchronous behavior. Here's how to test what we've built:

### Unit Tests

```go
func TestStreamWithTools(t *testing.T) {
    tests := []struct {
        name           string
        prompt         string
        mockResponse   []string
        mockToolCalls  []schema.ToolCallChunk
        wantToolsCalled int
        wantErr        bool
    }{
        {
            name:          "simple response without tools",
            prompt:        "Hello",
            mockResponse:  []string{"Hello", " there", "!"},
            wantToolsCalled: 0,
        },
        {
            name:   "response with tool call",
            prompt: "What's the weather?",
            mockResponse: []string{"Let me check"},
            mockToolCalls: []schema.ToolCallChunk{
                {Name: "get_weather", Arguments: `{"location": "NYC"}`},
            },
            wantToolsCalled: 1,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Create mock LLM that returns our test data
            mockLLM := llms.NewAdvancedMockChatModel(
                llms.WithResponses(tt.mockResponse),
            )

            // Execute
            result, err := StreamWithTools(
                context.Background(),
                mockLLM,
                tt.prompt,
                []tools.Tool{testWeatherTool},
            )

            // Assert
            if (err != nil) != tt.wantErr {
                t.Errorf("StreamWithTools() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if len(result.ToolsCalled) != tt.wantToolsCalled {
                t.Errorf("ToolsCalled = %d, want %d", len(result.ToolsCalled), tt.wantToolsCalled)
            }
        })
    }
}
```

### Running Tests

```bash
# Run all tests
go test -v ./...

# Run with coverage
go test -cover ./...

# Run specific test
go test -v -run TestStreamWithTools ./...
```

**What to look for:**
- All tests should pass
- Coverage should be above 80%
- No race conditions (run with `-race` flag)

## Best Practices

After using this feature in production, here are the patterns that work best:

### Do

- **Always check `chunk.Err`** - Errors can arrive at any point in the stream. Check every chunk.
- **Use context for cancellation** - Users may navigate away. Respect `ctx.Done()` to clean up properly.
- **Add OTEL instrumentation** - Track chunks received, tool calls made, and duration. You'll need this for debugging.
- **Handle partial tool calls** - Some providers send tool calls across multiple chunks. Accumulate them properly.
- **Buffer appropriately** - For high-frequency chunks, consider buffering before UI updates.

### Don't

- **Don't assume order** - While chunks usually arrive in order, build defensively.
- **Don't block the channel** - Process chunks quickly or use a buffered approach.
- **Don't ignore tool call failures** - Send error results back to the LLM so it can respond appropriately.

### Performance Tips

- **Chunk batching**: For very fast streams, batch multiple chunks before updating UI
- **Goroutine per stream**: Process streams in separate goroutines if handling multiple users
- **Channel buffer size**: The default unbuffered channels work well; buffering can hide backpressure issues

## Troubleshooting

### Q: I see error "stream closed" unexpectedly

**A:** This can happen when:
1. The API rate limit is hit - implement exponential backoff
2. Network timeout - increase context timeout or check connectivity
3. API key issues - verify your key is valid and has proper permissions

Try wrapping the stream in a retry loop for transient failures:

```go
// Retry with exponential backoff
for attempt := 0; attempt < 3; attempt++ {
    streamChan, err := model.StreamChat(ctx, messages)
    if err == nil {
        // process stream...
        break
    }
    time.Sleep(time.Duration(attempt+1) * time.Second)
}
```

### Q: Tool calls aren't being triggered

**A:** Check that:
1. Your tool definitions include clear descriptions
2. The prompt naturally leads to using the tool
3. The model you're using supports tool calling (not all models do)
4. Tools are properly bound with `BindTools()` before calling `StreamChat()`

### Q: Chunks are arriving but no text content

**A:** When the LLM decides to call a tool, it may not emit text content before the tool call. Check `chunk.ToolCallChunks` - the LLM might be making tool calls silently.

### Q: How do I handle multiple concurrent tool calls?

**A:** Execute them in parallel using goroutines:

```go
var wg sync.WaitGroup
results := make(chan schema.Message, len(toolCalls))

for _, tc := range toolCalls {
    wg.Add(1)
    go func(tc schema.ToolCallChunk) {
        defer wg.Done()
        result := executeToolCall(ctx, tc, tools)
        results <- result
    }(tc)
}

wg.Wait()
close(results)
```

## Related Resources

Now that you understand streaming LLM calls with tool calling, explore:

- **[Agent Types Guide](./agent-types.md)** - Learn how agents orchestrate multiple tool calls in a reasoning loop
- **[LLM Providers Guide](./llm-providers.md)** - Deep dive into configuring different LLM providers
- **[Streaming Example](/examples/llms/streaming/README.md)** - Complete implementation with tests and OTEL instrumentation
- **[LLM Error Handling](../cookbook/llm-error-handling.md)** - Handle rate limits and API errors gracefully
- **[Batch Processing Use Case](../use-cases/batch-processing.md)** - See streaming in action for high-volume processing
