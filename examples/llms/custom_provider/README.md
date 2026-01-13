# Custom LLM Provider Example

> **Learn how to create and register your own LLM provider with the Beluga AI framework.**

## Description

This example demonstrates how to build a custom LLM provider that integrates with Beluga AI's provider registry system. You'll see how to:

- Implement the `ChatModel` interface completely
- Add OTEL instrumentation for observability (metrics, tracing)
- Use functional options for flexible configuration
- Register your provider with the global registry
- Handle errors properly with provider-specific error codes
- Support streaming responses
- Bind tools to your provider

The example creates a simulated provider for demonstration purposes, but the patterns shown apply directly to integrating real LLM APIs.

## Prerequisites

| Requirement | Version | Why |
|-------------|---------|-----|
| Go | 1.24+ | Required for Beluga AI framework |
| Beluga AI | latest | The framework we're extending |

### Installation

```bash
# Clone the repository if you haven't
git clone https://github.com/lookatitude/beluga-ai.git
cd beluga-ai

# Navigate to this example
cd examples/llms/custom_provider
```

## Usage

### Running the Example

```bash
go run custom_llm_provider.go
```

### Expected Output

```
Registered providers: [example-custom openai anthropic ...]

--- Generate Example ---
Response: This is a response from the custom LLM provider.

--- Streaming Example ---
Streaming: I'm a simulated response for demonstration purposes.

--- Tool Binding Example ---
Tool calls: [{ID:call_example_123 Name:calculator Arguments:{"example": "value"}}]

--- Health Check ---
Health: map[api_key_set:true model:example-model-v1 provider:example-custom state:healthy timestamp:1706541234 tools_count:0]

Custom provider example completed successfully!
```

### Using in Your Own Code

1. **Copy the provider structure** from `custom_llm_provider.go`
2. **Replace the simulated API calls** with your actual LLM API client
3. **Register your provider**:

```go
import (
    _ "your/package/path/custom"  // Auto-registers via init()
)
```

Or register manually:

```go
llms.GetRegistry().Register("my-provider", myProviderFactory)
```

4. **Use your provider**:

```go
config := llms.NewConfig(
    llms.WithProvider("my-provider"),
    llms.WithModelName("my-model"),
    llms.WithAPIKey("your-api-key"),
)

provider, err := llms.GetRegistry().GetProvider("my-provider", config)
// Use provider.Generate(), StreamChat(), etc.
```

## Code Structure

```
custom_provider/
├── custom_llm_provider.go       # Main provider implementation
├── custom_llm_provider_test.go  # Comprehensive test suite
└── README.md                    # This file
```

### Key Components in `custom_llm_provider.go`

| Component | Lines | Description |
|-----------|-------|-------------|
| `CustomProvider` struct | 55-80 | Main provider struct with all dependencies |
| `CustomMetrics` | 85-120 | OTEL metrics wrapper |
| `NewCustomProvider` | 150-190 | Constructor with validation and setup |
| `Generate` | 200-260 | Core generation with tracing and metrics |
| `StreamChat` | 270-320 | Streaming implementation |
| `BindTools` | 330-345 | Immutable tool binding |
| `Batch` | 380-420 | Concurrent batch processing |
| `RegisterCustomProvider` | 450-455 | Registry registration |

## Testing

### Run All Tests

```bash
go test -v ./...
```

### Run Specific Test Categories

```bash
# Interface compliance tests
go test -v -run TestCustomProviderImplements

# Registration tests
go test -v -run TestProviderRegistration

# Generate tests
go test -v -run TestCustomProviderGenerate

# Streaming tests
go test -v -run TestCustomProviderStreamChat

# Concurrency tests
go test -v -run TestCustomProviderConcurrent

# Error handling tests
go test -v -run TestCustomProviderErrorHandling
```

### Run Benchmarks

```bash
go test -bench=. -benchmem
```

Example benchmark output:

```
BenchmarkGenerate-8              50000       25000 ns/op     1200 B/op      20 allocs/op
BenchmarkStreamChat-8            20000       75000 ns/op     2400 B/op      35 allocs/op
BenchmarkBindTools-8            500000        3000 ns/op      400 B/op       8 allocs/op
BenchmarkConcurrentGenerate-8    30000       40000 ns/op     1500 B/op      25 allocs/op
```

### Test Coverage

```bash
go test -cover ./...
```

Target coverage: 80%+

## What the Tests Verify

| Test Category | What It Tests |
|--------------|---------------|
| Interface Compliance | Provider implements `ChatModel` and `LLM` interfaces |
| Registration | Provider registers and retrieves from global registry |
| Generate | Multiple message types, conversations, error cases |
| Streaming | Chunk delivery, cancellation handling |
| Tool Binding | Immutability, concurrent binding |
| Batch Processing | Parallel execution, concurrency limits |
| Health Check | Status reporting |
| Concurrency | Thread safety of all operations |
| Error Handling | Nil config, context cancellation, timeouts |
| Benchmarks | Performance baselines |

## Customization Points

When adapting this example for your LLM API:

### 1. Replace the API Client

```go
type CustomProvider struct {
    // Add your real API client
    client *yourapi.Client
    // ...
}
```

### 2. Implement Message Conversion

```go
func (c *CustomProvider) convertMessages(messages []schema.Message) []yourapi.Message {
    // Convert schema.Message to your API's format
}
```

### 3. Handle API-Specific Errors

```go
func (c *CustomProvider) handleAPIError(err error) error {
    switch {
    case errors.Is(err, yourapi.ErrRateLimit):
        return llms.NewLLMError("generate", ErrCodeRateLimit, err)
    // Handle other error types
    }
}
```

### 4. Add Provider-Specific Options

```go
func WithYourAPIOption(value string) llms.ConfigOption {
    return llms.WithProviderSpecific("your_option", value)
}
```

## Related Examples

- [Streaming with Tool Calls](../streaming/) - Advanced streaming patterns
- [LLM Usage Basics](../llm-usage/) - Basic LLM operations
- [Error Handling](../../llm-usage/) - Error handling patterns

## Related Documentation

- [LLM Provider Integration Guide](/docs/guides/llm-providers.md) - Complete guide to provider integration
- [Extensibility Guide](/docs/guides/extensibility.md) - Framework extension patterns
- [LLM Error Handling Cookbook](/docs/cookbook/llm-error-handling.md) - Error handling recipes
- [Observability Guide](/docs/guides/observability-tracing.md) - OTEL integration details

## Troubleshooting

### Provider not found in registry

**Symptom**: `error: provider 'my-provider' not registered`

**Solution**: Ensure you either:
- Import the package with `_` to trigger `init()`: `import _ "your/package"`
- Or call `RegisterCustomProvider()` explicitly before using

### Metrics not appearing

**Symptom**: No metrics in your observability backend

**Solution**: 
1. Verify OTEL is initialized: `otel.SetMeterProvider(yourProvider)`
2. Check that `llms.InitMetrics(meter, tracer)` is called at startup
3. Confirm your exporter is configured correctly

### Streaming stops unexpectedly

**Symptom**: Stream ends without all chunks

**Solution**:
1. Check for context cancellation in your streaming loop
2. Ensure errors are sent through the chunk channel
3. Verify the `defer close(outputChan)` is in place

### Tool calls not working

**Symptom**: `ToolCalls()` returns empty even when expected

**Solution**:
1. Verify `BindTools()` returns a new instance (check with `NotSame`)
2. Check your API request includes tools in correct format
3. Parse tool calls from your API's response format correctly
