# Beluga AI LLM Package Usage Example

This example application demonstrates how to use the refactored Beluga AI LLM package following the framework's design patterns.

## Features Demonstrated

- ✅ **Factory Pattern**: Provider registration and management
- ✅ **Configuration Management**: Functional options and validation
- ✅ **Error Handling**: Custom error types and retry logic
- ✅ **Streaming Responses**: Real-time response handling
- ✅ **Batch Processing**: Concurrent request processing
- ✅ **Tool Calling**: Function calling capabilities
- ✅ **Observability**: Metrics and tracing integration
- ✅ **Utility Functions**: Helper functions for common tasks

## Running the Example

### Prerequisites

1. Go 1.24 or later
2. Access to the Beluga AI Framework

### Installation

```bash
# Navigate to the example directory
cd examples/llm-usage

# Download dependencies
go mod tidy

# Run the example
go run main.go
```

### Expected Output

```
🔄 Beluga AI LLM Package Usage Example
=====================================

📋 Example 1: Basic Factory Usage
Creating factory and registering mock provider...
✅ Factory created successfully
✅ Mock configuration created
📊 Available provider types: [anthropic openai bedrock mock]

⚙️  Example 2: Configuration Management
Demonstrating configuration patterns...
✅ Anthropic configuration validated successfully
✅ OpenAI configuration validated successfully
✅ Mock configuration validated successfully
✅ Configuration merging demonstrated

🚨 Example 3: Error Handling
Demonstrating error handling patterns...
✅ Error configuration created
✅ LLM Error detected: rate_limit
✅ Is retryable: true
   rate_limit: retryable=true
   authentication_error: retryable=false
   invalid_request: retryable=true
   network_error: retryable=true
   internal_error: retryable=true

🌊 Example 4: Streaming Responses
Demonstrating streaming responses...
📝 Created 2 messages for streaming
✅ Streaming example prepared (would work with real providers)
💡 Streaming usage pattern:
   streamChan, err := provider.StreamChat(ctx, messages)
   for chunk := range streamChan {
       if chunk.Err != nil { handle error }
       fmt.Print(chunk.Content) // Print as it arrives
   }

📦 Example 5: Batch Processing
Demonstrating batch processing...
📦 Prepared 4 batch inputs
✅ Batch processing example prepared
💡 Batch processing benefits:
   - Concurrent processing of multiple requests
   - Automatic error handling per request
   - Configurable concurrency limits
   - Efficient resource utilization

🛠️  Example 6: Tool Calling (Mock)
Demonstrating tool calling...
🛠️  Created 2 messages for tool calling
✅ Tool calling example prepared
💡 Tool calling workflow:
   1. Create tools: calculator, webSearch, etc.
   2. Bind to provider: provider.BindTools(tools)
   3. Generate: response, err := provider.Generate(ctx, messages)
   4. Handle tool calls from response.ToolCalls
   5. Execute tools and continue conversation

🔧 Example 7: Utility Functions
Testing EnsureMessages utility...
✅ String converted to 1 messages
✅ Message slice handled correctly
Testing GetSystemAndHumanPrompts utility...
✅ System prompt: You are a helpful assistant.
✅ Human prompts: What is AI?How does it work?
Testing ValidateModelName utility...
✅ Model gpt-4 validated for provider openai
✅ Model claude-3-sonnet validated for provider anthropic

📊 Example 8: Metrics and Observability
Recording sample metrics...
✅ Recorded request metrics
✅ Recorded token usage (100 input, 50 output)
✅ Incremented active request counter
📈 Current metrics:
   Total requests: 1
   Total errors: 0
   Total token usage: 150
   Active requests: 1
💡 In production, these metrics would be exported to:
   - Prometheus for monitoring
   - Jaeger/Grafana for tracing
   - ELK stack for logging

📄 Example 9: Configuration File Pattern
Example configuration file structure (YAML):
[configuration example shown]

✨ All examples completed successfully!
```

## Architecture Overview

The example demonstrates the clean architecture of the refactored LLM package:

```
examples/llm-usage/
├── main.go           # Main application demonstrating all features
├── go.mod            # Go module dependencies
└── README.md         # This documentation
```

## Key Concepts Demonstrated

### 1. Factory Pattern
```go
factory := llms.NewFactory()
// Register providers
// Create providers using factory
```

### 2. Configuration Management
```go
config := llms.NewConfig(
    llms.WithProvider("anthropic"),
    llms.WithModelName("claude-3-sonnet"),
    llms.WithAPIKey("your-key"),
    llms.WithTemperature(0.7),
)
```

### 3. Error Handling
```go
if llms.IsLLMError(err) {
    code := llms.GetLLMErrorCode(err)
    if llms.IsRetryableError(err) {
        // Implement retry logic
    }
}
```

### 4. Streaming
```go
streamChan, err := provider.StreamChat(ctx, messages)
for chunk := range streamChan {
    fmt.Print(chunk.Content)
}
```

### 5. Tool Calling
```go
modelWithTools := provider.BindTools(tools)
response, err := modelWithTools.Generate(ctx, messages)
// Handle tool calls from response.ToolCalls
```

## Production Usage

For production deployments, you would:

1. **Set up proper configuration files** (YAML/JSON)
2. **Initialize OpenTelemetry** for observability
3. **Implement proper error handling** with retries
4. **Set up monitoring** with Prometheus/Grafana
5. **Configure logging** with structured logging
6. **Implement health checks** and graceful shutdown

## Next Steps

After running this example, you can:

1. **Explore the actual provider implementations** in `pkg/llms/providers/`
2. **Check the comprehensive documentation** in `pkg/llms/README.md`
3. **Review the test examples** in `pkg/llms/llms_test.go`
4. **Study the configuration patterns** in `pkg/llms/config.go`
5. **Examine the error handling** in `pkg/llms/errors.go`

## Contributing

This example serves as a reference implementation. To contribute:

1. Add new examples demonstrating advanced features
2. Improve error handling demonstrations
3. Add performance benchmarking examples
4. Create examples for specific provider integrations
5. Add configuration file parsing examples

## License

This example is part of the Beluga AI Framework and follows the same license terms.
