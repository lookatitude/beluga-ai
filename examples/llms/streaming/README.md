# Streaming LLM with Tool Calls

## Description

This example shows you how to combine streaming LLM responses with tool calling in Beluga AI. You'll learn:

- How to stream LLM responses in real-time via Go channels
- How to bind tools to an LLM client
- How to detect and execute tool calls during streaming
- How to continue conversations after tool execution
- Best practices for OTEL instrumentation and error handling

## Prerequisites

Before running this example, you need:

| Requirement | Version | Notes |
|-------------|---------|-------|
| Go | 1.24+ | [Install Go](https://go.dev/doc/install) |
| Beluga AI | latest | `go get github.com/lookatitude/beluga-ai` |
| OpenAI API Key | - | Get one at [OpenAI](https://platform.openai.com/api-keys) |

### Environment Setup

Set these environment variables:

```bash
export OPENAI_API_KEY="your-api-key-here"

# Optional: Enable debug logging
export BELUGA_LOG_LEVEL=debug
```

## Usage

### Running the Example

1. **Navigate to the example directory**:

```bash
cd examples/llms/streaming
```

2. **Run the example**:

```bash
go run streaming_tool_call.go
```

### Expected Output

When successful, you'll see:

```
=== Streaming LLM with Tool Calls Example ===

Prompt: What's the weather like in San Francisco right now?

Response: The current weather in San Francisco is sunny with a temperature of 72°F (22°C) and humidity at 45%. It's a beautiful day!
Tools called: [get_weather]
Chunks received: 15
Duration: 2.345s

Prompt: What's 2 + 2?

Response: 2 + 2 equals 4.
Tools called: []
Chunks received: 5
Duration: 0.567s
```

### Using Different LLM Providers

The streaming pattern works with any provider that supports streaming:

```go
// OpenAI
client, _ := llms.NewOpenAIChat(llms.WithAPIKey(os.Getenv("OPENAI_API_KEY")))

// Anthropic
client, _ := llms.NewAnthropicChat(llms.WithAPIKey(os.Getenv("ANTHROPIC_API_KEY")))

// Ollama (local)
client, _ := llms.NewOllamaChat(llms.WithModel("llama2"))
```

## Code Structure

```
streaming/
├── README.md                   # This file
├── streaming_tool_call.go      # Main example implementation
└── streaming_tool_call_test.go # Comprehensive test suite
```

### Key Components

| File | Purpose |
|------|---------|
| `streaming_tool_call.go` | Main implementation showing streaming with tools |
| `streaming_tool_call_test.go` | Tests covering success, error, and edge cases |

### Design Decisions

This example demonstrates these Beluga AI patterns:

- **Dependency Injection**: The `StreamingToolCallExample` accepts an `iface.ChatModel` interface, making it testable with mock clients
- **OTEL Instrumentation**: All operations are traced with `go.opentelemetry.io/otel`. Metrics follow the `beluga.{package}.{metric}` naming convention
- **Error Handling**: Errors are wrapped with context using `fmt.Errorf("context: %w", err)`
- **Context Propagation**: All functions accept `context.Context` for cancellation support

## Testing

### Running Tests

```bash
# Run all tests
go test -v ./...

# Run with race detection
go test -race ./...

# Run with coverage
go test -cover ./...

# Run benchmarks
go test -bench=. ./...
```

### Test Structure

| Test | What it verifies |
|------|-----------------|
| `TestStreamingToolCallExample_Run` | Main run method with various scenarios |
| `TestStreamingToolCallExample_ContextCancellation` | Proper context cancellation handling |
| `TestStreamingToolCallExample_ErrorHandling` | Error scenarios are handled correctly |
| `TestExecuteToolCalls` | Tool execution with success, failure, and edge cases |
| `TestProcessStream` | Stream processing accumulates text correctly |
| `TestCreateWeatherTool` | Weather tool works as expected |
| `TestCreateCalculatorTool` | Calculator tool works as expected |
| `BenchmarkStreamProcessing` | Performance of stream processing |
| `BenchmarkToolExecution` | Performance of tool execution |

### Expected Test Output

```
=== RUN   TestStreamingToolCallExample_Run
=== RUN   TestStreamingToolCallExample_Run/simple_response_without_tool_calls
=== RUN   TestStreamingToolCallExample_Run/handles_empty_response_gracefully
--- PASS: TestStreamingToolCallExample_Run (0.01s)
    --- PASS: TestStreamingToolCallExample_Run/simple_response_without_tool_calls (0.00s)
    --- PASS: TestStreamingToolCallExample_Run/handles_empty_response_gracefully (0.00s)
PASS
coverage: 85.2% of statements
```

## Troubleshooting

### Common Issues

<details>
<summary>❌ Error: "OPENAI_API_KEY environment variable is required"</summary>

**Cause:** The `OPENAI_API_KEY` environment variable is not set.

**Solution:**
```bash
export OPENAI_API_KEY="sk-..."
# Then run the example again
```
</details>

<details>
<summary>❌ Error: "connection refused" or timeout</summary>

**Cause:** Network issues or API rate limiting.

**Solution:**
1. Check your internet connection
2. Verify your API key is valid and has credits
3. Try again after a few seconds (rate limiting)
</details>

<details>
<summary>❌ Tool calls not triggering</summary>

**Cause:** The prompt may not naturally lead to tool usage.

**Solution:**
1. Make your prompt more specific: "What's the weather in NYC?" instead of "Tell me about weather"
2. Verify tools are properly bound with `client.BindTools(tools)`
3. Check that the model supports tool calling (e.g., gpt-4, claude-3)
</details>

<details>
<summary>❌ Empty response from streaming</summary>

**Cause:** The LLM may have decided to make tool calls without text content.

**Solution:**
Check for tool call chunks in the response - when the LLM decides to call a tool, it may not emit text content first.
</details>

## Related Examples

After completing this example, you might want to explore:

- **[Agent with Tools](../../agents/with_tools/README.md)** - See how agents manage multi-turn tool calling automatically
- **[React Agent](../../agents/react/README.md)** - Agents that reason about when to use tools

## Learn More

- **[Streaming LLM Guide](/docs/guides/llm-streaming-tool-calls.md)** - In-depth guide on this topic
- **[LLM Error Handling](/docs/cookbook/llm-error-handling.md)** - Handling rate limits and API errors
- **[Observability Guide](/docs/guides/observability-tracing.md)** - Setting up OTEL for production
