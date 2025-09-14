# Schema Package

The `schema` package provides core data structures and interfaces for the Beluga AI Framework. It defines types for messages, documents, configurations, and agent interactions, following the framework's design patterns.

## Overview

This package implements the Beluga AI Framework design patterns including:

- **Interface Segregation Principle (ISP)**: Small, focused interfaces
- **Dependency Inversion Principle (DIP)**: High-level modules depend on abstractions
- **Factory Pattern**: Creator functions for all major types
- **Functional Options**: Flexible configuration with option functions
- **OpenTelemetry Integration**: Comprehensive observability and metrics

## Package Structure

```
pkg/schema/
├── iface/              # Interface definitions
│   └── message.go      # Message and ChatHistory interfaces
├── internal/           # Private implementations
│   ├── message.go      # Message type implementations
│   ├── document.go     # Document implementation
│   ├── history.go      # Chat history implementation
│   └── agent_io.go     # Agent I/O types
├── config.go           # Configuration structs and validation
├── schema.go           # Main package API and factory functions
├── metrics.go          # OpenTelemetry metrics integration
├── schema_test.go      # Package tests
└── README.md           # This file
```

## Core Interfaces

### Message Interface

```go
type Message interface {
    GetType() MessageType
    GetContent() string
    ToolCalls() []ToolCall
    AdditionalArgs() map[string]interface{}
}
```

The `Message` interface is implemented by various message types:
- `ChatMessage` - Generic chat messages with roles
- `AIMessage` - AI responses with optional tool calls
- `ToolMessage` - Tool execution results
- `FunctionMessage` - Function call results
- `Document` - Document content (treated as system messages)

### ChatHistory Interface

```go
type ChatHistory interface {
    AddMessage(message Message) error
    AddUserMessage(message string) error
    AddAIMessage(message string) error
    Messages() ([]Message, error)
    Clear() error
}
```

## Factory Functions

The package provides factory functions for all major types:

### Messages

```go
// Create different types of messages
humanMsg := schema.NewHumanMessage("Hello, world!")
aiMsg := schema.NewAIMessage("Hello from AI!")
systemMsg := schema.NewSystemMessage("You are a helpful assistant.")
toolMsg := schema.NewToolMessage("Tool result", "call_123")
```

### Documents

```go
// Create documents
doc := schema.NewDocument("Document content", map[string]string{"author": "AI"})
docWithID := schema.NewDocumentWithID("doc-123", "Content", metadata)
docWithEmbedding := schema.NewDocumentWithEmbedding("Content", metadata, embedding)
```

### Agent I/O

```go
// Create agent actions and observations
action := schema.NewAgentAction("tool_name", input, "Action log")
observation := schema.NewAgentObservation("action log", "output", parsedOutput)
step := schema.NewStep(action, observation)
```

### Chat History

```go
// Create chat history with configuration
history, err := schema.NewBaseChatHistory(
    schema.WithMaxMessages(100),
    schema.WithPersistence(true),
)
```

## Configuration

The package uses functional options for configuration:

### Agent Configuration

```go
agentConfig, err := schema.NewAgentConfig("my-agent", "openai-provider",
    schema.WithToolNames([]string{"calculator", "search"}),
    schema.WithMemoryProvider("vector-store", "buffer"),
    schema.WithMaxIterations(10),
    schema.WithPromptTemplate("You are a helpful assistant..."),
)
```

### LLM Provider Configuration

```go
llmConfig, err := schema.NewLLMProviderConfig("openai-gpt4", "openai", "gpt-4",
    schema.WithAPIKey("your-api-key"),
    schema.WithBaseURL("https://api.openai.com/v1"),
    schema.WithDefaultCallOptions(map[string]interface{}{
        "temperature": 0.7,
        "max_tokens": 1024,
    }),
)
```

### Embedding Provider Configuration

```go
embedConfig, err := schema.NewEmbeddingProviderConfig("openai-embed", "openai", "text-embedding-ada-002", "api-key",
    schema.WithEmbeddingBaseURL("https://api.openai.com/v1"),
)
```

## Observability

The package integrates with OpenTelemetry for comprehensive observability:

### Metrics

```go
// Global metrics instance
metrics := schema.NewMetrics(meter)
schema.SetGlobalMetrics(metrics)

// Metrics are automatically recorded for:
// - Message creation (by type)
// - Document creation
// - Chat history operations
// - Agent actions and observations
// - Configuration validations
```

### Tracing

Context-aware factory functions support distributed tracing:

```go
// Context-aware message creation
ctx, span := tracer.Start(ctx, "create_message")
defer span.End()

msg := schema.NewHumanMessageWithContext(ctx, "Hello!")
```

## Usage Examples

### Basic Message Handling

```go
package main

import (
    "fmt"
    "github.com/lookatitude/beluga-ai/pkg/schema"
)

func main() {
    // Create messages
    humanMsg := schema.NewHumanMessage("What's the weather?")
    aiMsg := schema.NewAIMessage("I'll check the weather for you.")

    // Access message properties
    fmt.Printf("Human: %s\n", humanMsg.GetContent())
    fmt.Printf("AI: %s\n", aiMsg.GetContent())
    fmt.Printf("Message types: %s, %s\n", humanMsg.GetType(), aiMsg.GetType())
}
```

### Chat History Management

```go
// Create chat history
history, err := schema.NewBaseChatHistory(
    schema.WithMaxMessages(50),
)
if err != nil {
    log.Fatal(err)
}

// Add messages
err = history.AddUserMessage("Hello!")
err = history.AddAIMessage("Hi there!")

// Retrieve conversation
messages, err := history.Messages()
for _, msg := range messages {
    fmt.Printf("%s: %s\n", msg.GetType(), msg.GetContent())
}
```

### Agent Interaction

```go
// Create agent action
action := schema.NewAgentAction(
    "search",
    map[string]interface{}{"query": "AI frameworks"},
    "Searching for AI frameworks",
)

// Create observation
observation := schema.NewAgentObservation(
    "Searching for AI frameworks",
    "Found several AI frameworks including Beluga AI",
    nil, // parsed output
)

// Combine into step
step := schema.NewStep(action, observation)

// Create final answer
finalAnswer := schema.NewFinalAnswer(
    "Beluga AI is a great framework for building AI applications",
    []interface{}{doc1, doc2}, // source documents
    []schema.Step{step},
)
```

## Extendability

### Adding Custom Message Types

```go
// Define custom message type
type CustomMessage struct {
    schema.BaseMessage
    CustomField string
}

func (m *CustomMessage) GetType() schema.MessageType {
    return "custom"
}

// Implement the Message interface
func NewCustomMessage(content, customField string) schema.Message {
    return &CustomMessage{
        BaseMessage: schema.BaseMessage{Content: content},
        CustomField: customField,
    }
}
```

### Custom Chat History Implementation

```go
type CustomHistory struct {
    // implementation
}

func (h *CustomHistory) AddMessage(msg schema.Message) error {
    // custom implementation
    return nil
}

func (h *CustomHistory) Messages() ([]schema.Message, error) {
    // custom implementation
    return messages, nil
}

// Use as schema.ChatHistory
var history schema.ChatHistory = &CustomHistory{}
```

### Provider-Specific Extensions

```go
// Extend configuration for custom provider
type CustomProviderConfig struct {
    schema.LLMProviderConfig
    CustomSetting string `yaml:"custom_setting"`
}

// Factory function with custom options
func NewCustomProvider(name, model string, opts ...CustomOption) (*CustomProviderConfig, error) {
    config := &CustomProviderConfig{
        LLMProviderConfig: schema.LLMProviderConfig{
            Name:   name,
            ModelName: model,
        },
    }

    for _, opt := range opts {
        opt(config)
    }

    return config, nil
}
```

## Validation

All configuration structs include validation:

```go
config, err := schema.NewAgentConfig("agent", "llm-provider")
if err != nil {
    // Handle validation error
    log.Printf("Invalid config: %v", err)
}
```

## Error Handling

The package provides structured error handling:

```go
// Message validation
if err := schema.ValidateMessage(msg); err != nil {
    return fmt.Errorf("invalid message: %w", err)
}

// Document validation
if err := schema.ValidateDocument(doc); err != nil {
    return fmt.Errorf("invalid document: %w", err)
}
```

## Testing

The package includes comprehensive tests:

```bash
go test ./pkg/schema/
```

Test coverage includes:
- Message creation and validation
- Document operations
- Chat history functionality
- Configuration validation
- Factory function behavior

## Dependencies

- `github.com/go-playground/validator/v10` - Configuration validation
- `go.opentelemetry.io/otel/metric` - Metrics collection
- `go.opentelemetry.io/otel/trace` - Distributed tracing

## Contributing

When contributing to this package:

1. Follow the Beluga AI Framework design patterns
2. Add tests for new functionality
3. Update documentation
4. Ensure backward compatibility
5. Use functional options for configuration
6. Add appropriate metrics and tracing

## License

This package is part of the Beluga AI Framework and follows the same license terms.
