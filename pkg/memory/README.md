# Memory Package

The `memory` package provides conversation memory implementations following the Beluga AI Framework design patterns. It manages conversation history and context for AI agents and applications, supporting multiple memory types with configurable behavior.

## Current Capabilities

**✅ Production Ready:**
- Buffer memory for storing all conversation messages
- Window buffer memory with configurable size limits
- Configuration management with validation
- Factory pattern for memory creation
- Comprehensive error handling with custom error types
- Unit tests for core functionality

**🚧 Available as Framework:**
- Summary memory with LLM-based conversation condensation
- Vector store memory for semantic retrieval
- OpenTelemetry metrics/tracing interfaces (placeholders)

## Features

- **Multiple Memory Types**: Support for different memory architectures (Buffer, Window, Summary, Vector Store)
- **Configurable Behavior**: Extensive configuration options via functional options
- **Factory Pattern**: Clean memory instantiation with validation
- **Error Handling**: Comprehensive error types with proper error wrapping
- **Observability**: OpenTelemetry tracing and metrics framework (interfaces ready)
- **Flexible Storage**: Support for different underlying storage backends
- **Type Safety**: Strong typing with Go interfaces and generics

## Architecture

The package follows SOLID principles and the Beluga AI Framework patterns:

```
pkg/memory/
├── iface/                    # Interface definitions (ISP compliant)
│   └── memory.go            # Core Memory and ChatMessageHistory interfaces
├── internal/                 # Private implementations
│   ├── buffer/              # Buffer memory implementation
│   ├── summary/             # Summary-based memory implementations
│   ├── vectorstore/         # Vector store memory implementations
│   └── window/              # Window-based memory implementations
├── providers/               # Provider implementations
│   └── base_history.go      # Base chat message history implementation
├── config.go                # Configuration management and validation
├── memory.go                # Main package API and factory functions
├── errors.go                # Custom error types with proper wrapping
├── metrics.go               # OpenTelemetry observability framework
├── memory_test.go           # Comprehensive unit tests
└── README.md                # This documentation
```

### Key Design Principles

- **Interface Segregation**: Small, focused interfaces serving specific purposes
- **Dependency Inversion**: High-level modules don't depend on low-level modules
- **Single Responsibility**: Each package/component has one clear purpose
- **Composition over Inheritance**: Behaviors composed through embedding
- **Clean Architecture**: Clear separation between business logic and infrastructure

## Core Interfaces

### Memory Interface
```go
type Memory interface {
    // MemoryVariables returns the list of variable names that the memory makes available.
    MemoryVariables() []string

    // LoadMemoryVariables loads memory variables given the context and input values.
    LoadMemoryVariables(ctx context.Context, inputs map[string]any) (map[string]any, error)

    // SaveContext saves the current context and new inputs/outputs to memory.
    SaveContext(ctx context.Context, inputs map[string]any, outputs map[string]any) error

    // Clear clears the memory contents.
    Clear(ctx context.Context) error
}
```

### ChatMessageHistory Interface
```go
type ChatMessageHistory interface {
    // AddMessage adds a message to the history.
    AddMessage(ctx context.Context, message schema.Message) error

    // AddUserMessage adds a human message to the history.
    AddUserMessage(ctx context.Context, content string) error

    // AddAIMessage adds an AI message to the history.
    AddAIMessage(ctx context.Context, content string) error

    // GetMessages returns all messages in the history.
    GetMessages(ctx context.Context) ([]schema.Message, error)

    // Clear removes all messages from the history.
    Clear(ctx context.Context) error
}
```

### Key Interfaces

- **`Memory`**: Core memory interface for storing and retrieving conversation context
- **`ChatMessageHistory`**: Interface for managing chat message sequences
- **`Factory`**: Factory pattern for creating memory instances
- **`MemoryRegistry`**: Global registry for managing memory types
- **`Option`**: Functional options for memory configuration

## Quick Start

### Creating Buffer Memory

```go
package main

import (
    "context"
    "log"

    "github.com/lookatitude/beluga-ai/pkg/memory"
)

func main() {
    // Create buffer memory with default settings
    bufferMem, err := memory.NewMemory(memory.MemoryTypeBuffer)
    if err != nil {
        log.Fatal(err)
    }

    ctx := context.Background()

    // Save conversation context
    inputs := map[string]any{"input": "Hello, how are you?"}
    outputs := map[string]any{"output": "I'm doing well, thank you!"}

    if err := bufferMem.SaveContext(ctx, inputs, outputs); err != nil {
        log.Printf("Failed to save context: %v", err)
        return
    }

    // Load memory variables
    vars, err := bufferMem.LoadMemoryVariables(ctx, map[string]any{})
    if err != nil {
        log.Printf("Failed to load memory: %v", err)
        return
    }

    log.Printf("Memory variables: %v", vars["history"])
}
```

### Creating Window Buffer Memory

```go
// Create window buffer memory that keeps only the last 3 interactions
windowMem, err := memory.NewMemory(memory.MemoryTypeBufferWindow,
    memory.WithWindowSize(3),
    memory.WithMemoryKey("recent_history"),
)
if err != nil {
    log.Fatal(err)
}

// Use the same SaveContext and LoadMemoryVariables methods
```

### Using Factory Pattern

```go
// Create factory
factory := memory.NewFactory()

// Configure memory
config := memory.Config{
    Type:          memory.MemoryTypeBuffer,
    MemoryKey:     "chat_history",
    ReturnMessages: true,
    Enabled:       true,
}

// Create memory instance
mem, err := factory.CreateMemory(ctx, config)
if err != nil {
    log.Fatal(err)
}
```

### Using Global Registry

```go
// Use the global registry to create memory
mem, err := memory.CreateMemory(
    ctx,
    string(memory.MemoryTypeBuffer),
    config,
)

// List available memory types
types := memory.ListAvailableMemoryTypes()
// Returns: ["buffer", "buffer_window", "summary", "summary_buffer", "vector_store", "vector_store_retriever"]

// Register a custom memory type
memory.RegisterMemoryType("custom", func(ctx context.Context, config memory.Config) (memory.Memory, error) {
    // Custom memory creation logic
    return customMemory, nil
})

// Get the global registry for advanced usage
registry := memory.GetGlobalMemoryRegistry()
```

### Creating Custom Memory with Functional Options

```go
customMem, err := memory.NewMemory(memory.MemoryTypeBuffer,
    memory.WithMemoryKey("conversation"),
    memory.WithInputKey("user_input"),
    memory.WithOutputKey("assistant_output"),
    memory.WithReturnMessages(true),
    memory.WithHumanPrefix("User"),
    memory.WithAIPrefix("Assistant"),
)
if err != nil {
    log.Fatal(err)
}
```

## Memory Types

### Buffer Memory
**Status**: ✅ Production Ready

Stores all conversation messages without any size limits:
```go
bufferMem := memory.NewChatMessageBufferMemory(
    memory.NewBaseChatMessageHistory(),
)
```

### Window Buffer Memory
**Status**: ✅ Production Ready

Keeps only the most recent K interactions (2K messages):
```go
windowMem := memory.NewConversationBufferWindowMemory(
    memory.NewBaseChatMessageHistory(),
    5, // Keep last 5 interactions
    "history",
    false, // Return formatted strings
)
```

### Summary Memory
**Status**: 🚧 Framework (Placeholder)

Uses LLMs to condense conversation history:
```go
// Note: Requires LLM dependency injection
summaryMem := memory.NewConversationSummaryMemory(
    memory.NewBaseChatMessageHistory(),
    llm, // core.Runnable implementation
    "summary",
)
```

### Summary Buffer Memory
**Status**: 🚧 Framework (Placeholder)

Combines buffer and summarization approaches.

### Vector Store Memory
**Status**: 🚧 Framework (Placeholder)

Uses vector stores for semantic retrieval of relevant context.

## Configuration

The package supports extensive configuration through functional options:

### Basic Configuration
```go
config := memory.Config{
    Type:           memory.MemoryTypeBuffer,
    MemoryKey:      "chat_history",
    InputKey:       "user_input",
    OutputKey:      "assistant_response",
    ReturnMessages: false,
    WindowSize:     10,
    MaxTokenLimit:  2000,
    TopK:           5,
    HumanPrefix:    "Human",
    AIPrefix:       "Assistant",
    Enabled:        true,
    Timeout:        30 * time.Second,
}
```

### Functional Options
```go
mem, err := memory.NewMemory(memory.MemoryTypeBufferWindow,
    // Memory configuration
    memory.WithMemoryKey("conversation_history"),
    memory.WithInputKey("query"),
    memory.WithOutputKey("response"),

    // Behavior configuration
    memory.WithReturnMessages(true),
    memory.WithWindowSize(8),

    // Formatting configuration
    memory.WithHumanPrefix("User"),
    memory.WithAIPrefix("AI"),

    // Performance configuration
    memory.WithTimeout(60*time.Second),
)
```

### Configuration Validation
```go
factory := memory.NewFactory()
mem, err := factory.CreateMemory(ctx, config)
if err != nil {
    // Configuration validation error
    var memErr *memory.MemoryError
    if errors.As(err, &memErr) {
        switch memErr.Code {
        case memory.ErrCodeInvalidConfig:
            log.Printf("Invalid configuration: %v", memErr.Err)
        }
    }
}
```

## Error Handling

The package provides comprehensive error handling with custom error types:

```go
mem, err := memory.NewMemory(memory.MemoryTypeBuffer)
if err != nil {
    var memErr *memory.MemoryError
    if errors.As(err, &memErr) {
        switch memErr.Code {
        case memory.ErrCodeInvalidConfig:
            log.Printf("Configuration error: %v", memErr.Err)
        case memory.ErrCodeStorageError:
            log.Printf("Storage error: %v", memErr.Err)
        case memory.ErrCodeRetrievalError:
            log.Printf("Retrieval error: %v", memErr.Err)
        }
    }
}
```

### Error Types
- `MemoryError`: General memory operation errors with context
- `ErrCodeInvalidConfig`: Configuration validation errors
- `ErrCodeInvalidInput`: Invalid input parameter errors
- `ErrCodeStorageError`: Data storage operation errors
- `ErrCodeRetrievalError`: Data retrieval operation errors
- `ErrCodeTimeout`: Operation timeout errors

## Observability

### Metrics
**Status**: Framework with placeholders

The package includes a metrics framework using OpenTelemetry interfaces:

```go
// Initialize metrics
memory.SetGlobalMetrics(meter)

// Metrics are automatically recorded for:
// - Memory operations (save, load, clear)
// - Operation duration
// - Error rates
// - Memory size/length
// - Active memory instances
```

### Tracing
**Status**: Framework with placeholders

Distributed tracing support for end-to-end observability:

```go
// Tracing is automatically integrated
// Spans are created for major memory operations
tracer := memory.GetGlobalTracer()
ctx, span := tracer.StartSpan(ctx, "memory_operation", memory.MemoryTypeBuffer, "history_key")
defer span.End()
```

### Structured Logging
**Status**: Framework with placeholders

Structured logging support is planned for comprehensive observability.

## Storage Backends

### Base Chat Message History
**Status**: ✅ Production Ready

Simple in-memory storage for chat messages:

```go
history := memory.NewBaseChatMessageHistory()

// Add messages
history.AddUserMessage(ctx, "Hello")
history.AddAIMessage(ctx, "Hi there!")

// Retrieve messages
messages, err := history.GetMessages(ctx)
```

### Custom Storage Implementations
The package supports custom storage backends by implementing the `ChatMessageHistory` interface:

```go
type CustomHistory struct {
    // Custom storage implementation
}

func (h *CustomHistory) AddMessage(ctx context.Context, message schema.Message) error {
    // Custom implementation
    return nil
}

// ... implement other interface methods
```

## Utility Functions

### Input/Output Key Detection
```go
// Automatically detect input/output keys from context
inputKey, outputKey, err := memory.GetInputOutputKeys(inputs, outputs)
if err != nil {
    log.Printf("Could not determine keys: %v", err)
}
```

### Message Formatting
```go
// Format messages as strings with custom prefixes
formatted := memory.GetBufferString(messages, "Human", "AI")
```

## Testing

The package includes comprehensive unit tests:

```go
func TestBufferMemory(t *testing.T) {
    ctx := context.Background()
    history := memory.NewBaseChatMessageHistory()
    bufferMem := memory.NewChatMessageBufferMemory(history)

    // Test saving context
    inputs := map[string]any{"input": "test"}
    outputs := map[string]any{"output": "response"}

    err := bufferMem.SaveContext(ctx, inputs, outputs)
    require.NoError(t, err)

    // Test loading memory
    vars, err := bufferMem.LoadMemoryVariables(ctx, map[string]any{})
    require.NoError(t, err)
    assert.Contains(t, vars, "history")
}

func TestFactory(t *testing.T) {
    ctx := context.Background()
    factory := memory.NewFactory()

    config := memory.Config{
        Type:          memory.MemoryTypeBuffer,
        MemoryKey:     "test_history",
        ReturnMessages: false,
        Enabled:       true,
    }

    mem, err := factory.CreateMemory(ctx, config)
    require.NoError(t, err)
    assert.Equal(t, []string{"test_history"}, mem.MemoryVariables())
}
```

## Best Practices

### 1. Error Handling
```go
// Always check for specific error types
if err := mem.SaveContext(ctx, inputs, outputs); err != nil {
    if memory.IsMemoryError(err, memory.ErrCodeStorageError) {
        // Handle storage-specific errors
        log.Printf("Storage failed, retrying...")
    } else {
        // Handle other errors
        return err
    }
}
```

### 2. Configuration Validation
```go
// Validate configuration before creating memory
factory := memory.NewFactory()
config := memory.Config{ /* ... */ }

mem, err := factory.CreateMemory(ctx, config)
if err != nil {
    // Handle configuration errors
    return err
}
```

### 3. Resource Management
```go
// Always clear memory when done
defer func() {
    if err := mem.Clear(ctx); err != nil {
        log.Printf("Failed to clear memory: %v", err)
    }
}()
```

### 4. Memory Size Management
```go
// Use window memory for large conversations
windowMem, err := memory.NewMemory(memory.MemoryTypeBufferWindow,
    memory.WithWindowSize(10), // Keep only last 10 interactions
)
```

### 5. Custom Storage
```go
// Implement custom storage for persistence
type PersistentHistory struct {
    // Database or file storage implementation
}

// Implement ChatMessageHistory interface
func (h *PersistentHistory) AddMessage(ctx context.Context, message schema.Message) error {
    // Persist to database/file
    return h.saveToStorage(message)
}
```

## Implementation Status

### ✅ Completed Features
- **Architecture**: SOLID principles with clean separation of concerns
- **Buffer Memory**: Complete implementation with all features
- **Window Memory**: Complete implementation with configurable window size
- **Interfaces**: Comprehensive interface definitions (ISP compliant)
- **Configuration**: Full configuration management and validation
- **Factory Pattern**: Complete factory implementation with error handling
- **Error Handling**: Custom error types with proper error wrapping
- **Testing**: Comprehensive unit tests for all core functionality

### 🚧 In Development / Placeholder
- **Summary Memory**: LLM-based conversation summarization (framework ready)
- **Vector Store Memory**: Semantic retrieval from vector stores (framework ready)
- **Metrics & Tracing**: OpenTelemetry integration (interfaces ready)
- **Advanced Storage**: Database/file system backends (framework ready)

### 📋 Roadmap
1. **Complete Summary Memory**: Full LLM-based summarization implementation
2. **Vector Store Integration**: Complete semantic retrieval support
3. **OpenTelemetry Integration**: Full metrics and tracing implementation
4. **Persistent Storage**: Database and file system backends
5. **Streaming Support**: Real-time memory updates
6. **Memory Compression**: Advanced compression algorithms
7. **Multi-tenant Support**: Isolated memory spaces

## Contributing

When adding new memory types:

1. **Create implementation** in `internal/` directory following existing patterns
2. **Follow SOLID principles** and interface segregation
3. **Add comprehensive tests** with mocks and edge cases
4. **Update configuration** with appropriate options
5. **Update documentation** with examples and usage patterns
6. **Maintain backward compatibility** with existing interfaces

### Development Guidelines
- Use functional options for configuration
- Implement proper error handling with custom error types
- Add comprehensive unit tests
- Follow the existing package structure
- Update this README when adding new features

### Adding a New Memory Type

1. **Define the memory type constant** in `config.go`:
```go
const MemoryTypeCustom MemoryType = "custom"
```

2. **Create implementation** in `internal/custom/`:
```go
type CustomMemory struct {
    // Implementation
}

func (m *CustomMemory) MemoryVariables() []string { /* ... */ }
func (m *CustomMemory) LoadMemoryVariables(ctx context.Context, inputs map[string]any) (map[string]any, error) { /* ... */ }
func (m *CustomMemory) SaveContext(ctx context.Context, inputs map[string]any, outputs map[string]any) error { /* ... */ }
func (m *CustomMemory) Clear(ctx context.Context) error { /* ... */ }
```

3. **Add factory support** in `memory.go`:
```go
func (f *DefaultFactory) createCustomMemory(ctx context.Context, config Config) (Memory, error) {
    // Implementation
}
```

4. **Add convenience function**:
```go
func NewCustomMemory(/* params */) Memory {
    // Implementation
}
```

## License

This package is part of the Beluga AI Framework and follows the same license terms.
