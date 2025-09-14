# Schema Package - Data Contract Layer

The `schema` package serves as the **data contract layer** for the Beluga AI Framework, providing standardized, well-defined data structures and interfaces that ensure consistent communication and interoperability across all framework components. It establishes clear contracts for messages, configurations, agent interactions, and system events.

## Overview

As the data contract layer, this package:

- **Standardizes Data Formats**: Defines canonical representations for all data exchanged between components
- **Ensures Type Safety**: Provides strongly-typed interfaces and validation
- **Enables Interoperability**: Allows different parts of the framework to communicate reliably
- **Supports Evolution**: Facilitates backward-compatible changes through versioning
- **Provides Observability**: Comprehensive metrics and tracing for data operations

This package implements the Beluga AI Framework design patterns including:

- **Interface Segregation Principle (ISP)**: Small, focused interfaces
- **Dependency Inversion Principle (DIP)**: High-level modules depend on abstractions
- **Factory Pattern**: Creator functions for all major types
- **Functional Options**: Flexible configuration with option functions
- **OpenTelemetry Integration**: Comprehensive observability and metrics
- **Comprehensive Validation**: Schema-level validation with configurable rules

## Package Structure

```
pkg/schema/
├── iface/              # Interface definitions and error handling
│   ├── message.go      # Message and ChatHistory interfaces
│   └── errors.go       # Structured error types and codes
├── internal/           # Private implementations
│   ├── message.go      # Message type implementations
│   ├── document.go     # Document implementation
│   ├── history.go      # Chat history implementation
│   └── agent_io.go     # Agent I/O, A2A communication, and event types
├── config.go           # Configuration structs and validation
├── schema.go           # Main package API and factory functions
├── metrics.go          # OpenTelemetry metrics integration
├── schema_test.go      # Comprehensive package tests
└── README.md           # This documentation
```

## Data Contract Categories

### Core Data Types
- **Messages**: Standardized message formats for human-AI-system communication
- **Documents**: Structured document representations with metadata and embeddings
- **Configurations**: Validated configuration schemas for all framework components

### Agent-to-Agent Communication (A2A)
- **Agent Messages**: Structured inter-agent communication protocols
- **Requests/Responses**: Request-response patterns for agent interactions
- **Agent Errors**: Standardized error handling in A2A scenarios

### Event System
- **System Events**: Framework-wide event definitions and handling
- **Agent Lifecycle Events**: Agent state change notifications
- **Task Events**: Task execution progress and status updates
- **Workflow Events**: Multi-agent workflow coordination events

### Validation and Configuration
- **Schema Validation**: Configurable validation rules for data integrity
- **Error Codes**: Comprehensive error classification system
- **Metrics**: Observability and monitoring for all data operations

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

## Advanced Features

### Agent-to-Agent (A2A) Communication

The package provides comprehensive support for agent-to-agent communication with structured message passing:

```go
// Create agent messages
message := schema.NewAgentMessage(
    "agent-1",
    "msg-123",
    schema.AgentMessageRequest,
    schema.NewAgentRequest("analyze_data", map[string]interface{}{
        "data": "sample data",
    }),
)

// Handle responses
response := schema.NewAgentResponse("msg-123", "success", result)
```

### Event System

A robust event system for system-wide notifications and state changes:

```go
// Agent lifecycle events
event := schema.NewAgentLifecycleEvent("agent-1", schema.AgentStarted)

// Task events
taskEvent := schema.NewTaskEvent("task-123", "agent-1", schema.TaskCompleted)

// Workflow events
workflowEvent := schema.NewWorkflowEvent("workflow-456", schema.WorkflowStepCompleted)
```

### Schema Validation Configuration

Comprehensive validation rules for data integrity:

```go
validationConfig, err := schema.NewSchemaValidationConfig(
    schema.WithStrictValidation(true),
    schema.WithMaxMessageLength(10000),
    schema.WithMaxMetadataSize(100),
    schema.WithAllowedMessageTypes([]string{"human", "ai", "system", "tool"}),
    schema.WithRequiredMetadataFields([]string{"source"}),
)
```

### Error Handling

Structured error handling with comprehensive error codes:

```go
// Check for specific error types
if schema.IsSchemaError(err, schema.ErrCodeInvalidMessage) {
    // Handle invalid message error
}

// Create new errors with context
err := schema.WrapError(cause, schema.ErrCodeValidationFailed, "validation failed for message: %s", msgID)
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

### A2A Communication

```go
// Create agent messages
message := schema.NewAgentMessage("agent-1", "msg-123", schema.AgentMessageRequest, payload)
request := schema.NewAgentRequest("action", parameters)
response := schema.NewAgentResponse("msg-123", "success", result)
agentError := schema.NewAgentError("code", "message", details)
```

### Events

```go
// Create various event types
event := schema.NewEvent("event-123", "user_action", "web_app", payload)
lifecycleEvent := schema.NewAgentLifecycleEvent("agent-1", schema.AgentStarted)
taskEvent := schema.NewTaskEvent("task-123", "agent-1", schema.TaskCompleted)
workflowEvent := schema.NewWorkflowEvent("workflow-456", schema.WorkflowStarted)
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

The package provides comprehensive metrics for all data operations:

```go
// Global metrics instance
metrics := schema.NewMetrics(meter)
schema.SetGlobalMetrics(metrics)

// Metrics are automatically recorded for:
// - Message creation and validation (by type)
// - Document creation and validation
// - Chat history operations
// - Agent actions and observations
// - Configuration validations
// - A2A communication (messages, requests, responses)
// - Event publishing and consumption
// - Agent lifecycle, task, and workflow events
// - Factory creation operations
// - Schema validation operations
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

The package provides structured error handling with comprehensive error codes:

```go
// Message validation
if err := schema.ValidateMessage(msg); err != nil {
    return fmt.Errorf("invalid message: %w", err)
}

// Document validation
if err := schema.ValidateDocument(doc); err != nil {
    return fmt.Errorf("invalid document: %w", err)
}

// Check for specific error types
if schema.IsSchemaError(err, schema.ErrCodeInvalidMessage) {
    // Handle invalid message error
}

if schema.IsSchemaError(err, schema.ErrCodeAgentMessageInvalid) {
    // Handle invalid agent message error
}

// Create structured errors
err := schema.WrapError(cause, schema.ErrCodeValidationFailed, "validation failed for message: %s", msgID)
```

### Error Code Categories

- **Validation Errors**: `ErrCodeInvalidMessage`, `ErrCodeMessageTooLong`, `ErrCodeInvalidFieldValue`
- **A2A Communication Errors**: `ErrCodeAgentMessageInvalid`, `ErrCodeCommunicationFailed`, `ErrCodeAgentNotFound`
- **Event Errors**: `ErrCodeEventInvalid`, `ErrCodeEventPublishFailed`, `ErrCodeEventHandlerNotFound`
- **Configuration Errors**: `ErrCodeConfigValidationFailed`, `ErrCodeInvalidConfigFormat`
- **Factory Errors**: `ErrCodeFactoryCreationFailed`, `ErrCodeFactoryNotFound`
- **Storage Errors**: `ErrCodeStorageOperationFailed`, `ErrCodePersistenceFailed`

## Testing

The package includes comprehensive tests covering all functionality:

```bash
go test ./pkg/schema/
```

Test coverage includes:
- Message creation and validation
- Document operations
- Chat history functionality
- Configuration validation
- Factory function behavior
- A2A communication types and factories
- Event system types and factories
- Schema validation configuration
- Error code handling
- Functional options for all types
- Edge cases and error conditions

## Dependencies

- `github.com/go-playground/validator/v10` - Configuration validation
- `go.opentelemetry.io/otel/metric` - Metrics collection
- `go.opentelemetry.io/otel/trace` - Distributed tracing

## Contributing

When contributing to this package as the data contract layer:

1. **Maintain Data Contract Stability**: Changes to data structures must be backward compatible
2. **Follow Framework Patterns**: Adhere to ISP, DIP, SRP, and composition over inheritance
3. **Add Comprehensive Tests**: Include tests for new types, validation, and error conditions
4. **Update Documentation**: Keep README and code comments current with new features
5. **Ensure Type Safety**: Use strong typing and validation for all data structures
6. **Add Observability**: Include metrics, tracing, and structured logging for new operations
7. **Use Factory Pattern**: Provide factory functions for all new data types
8. **Handle Errors Properly**: Use appropriate error codes and structured error handling
9. **Consider Interoperability**: Ensure new types work well with existing framework components
10. **Document Data Contracts**: Clearly specify the contract (fields, validation rules, etc.) for new types

### Adding New Data Types

When adding new data structures to the schema:

1. Define the type in `internal/` with proper JSON/YAML tags
2. Add factory functions in `schema.go`
3. Add validation methods if needed
4. Include metrics recording
5. Add comprehensive tests
6. Update documentation with usage examples
7. Consider adding to configuration if applicable

## License

This package is part of the Beluga AI Framework and follows the same license terms.
