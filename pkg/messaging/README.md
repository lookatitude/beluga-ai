# Messaging Package

The Messaging package provides a comprehensive framework for building multi-channel conversational AI agents, supporting SMS, WhatsApp, and other messaging channels with memory persistence and context management.

## Overview

The Messaging package follows the Beluga AI Framework design patterns, providing:
- **Multi-Provider Support**: Unified interface for different messaging providers (Twilio, etc.)
- **Conversation Management**: Full lifecycle management for messaging conversations
- **Memory Integration**: Persistent conversation history with context preservation
- **Webhook Handling**: Event-driven architecture for real-time message processing
- **Observability**: OTEL metrics and tracing throughout
- **Configuration**: Flexible configuration with validation

## Package Structure

```
pkg/messaging/
├── iface/              # Interfaces and types
├── internal/           # Private implementation details
├── providers/          # Provider implementations
│   └── twilio/         # Twilio Conversations API provider
├── config.go           # Configuration structs and validation
├── metrics.go          # OTEL metrics implementation
├── errors.go           # Custom error types
├── messaging.go        # Main interfaces and factory
├── registry.go         # Global registry
├── test_utils.go       # Advanced testing utilities
├── advanced_test.go    # Comprehensive test suites
└── README.md           # Package documentation
```

## Quick Start

### Basic Messaging Backend

```go
import (
    "context"
    "log"
    
    "github.com/lookatitude/beluga-ai/pkg/messaging"
)

func main() {
    ctx := context.Background()
    
    config := &messaging.Config{
        Provider: "twilio",
        // Provider-specific config
    }
    
    backend, err := messaging.NewBackend(ctx, "twilio", config)
    if err != nil {
        log.Fatal(err)
    }
    
    // Start backend
    if err := backend.Start(ctx); err != nil {
        log.Fatal(err)
    }
    defer backend.Stop(ctx)
    
    // Create conversation
    conversation, err := backend.CreateConversation(ctx, &iface.ConversationConfig{
        FriendlyName: "Customer Support",
    })
    if err != nil {
        log.Fatal(err)
    }
    
    // Send message
    err = backend.SendMessage(ctx, conversation.ConversationSID, &iface.Message{
        Body: "Hello! How can I help you?",
    })
    if err != nil {
        log.Fatal(err)
    }
}
```

## Configuration

### Config

```go
type Config struct {
    Provider                string         // Provider name (required)
    Timeout                 time.Duration  // Operation timeout
    MaxRetries              int            // Maximum retry attempts
    RetryDelay              time.Duration  // Retry delay
    RetryBackoff            float64        // Retry backoff multiplier
    EnableTracing           bool           // Enable OTEL tracing
    EnableMetrics           bool           // Enable OTEL metrics
    EnableStructuredLogging bool           // Enable structured logging
    ProviderSpecific        map[string]any // Provider-specific settings
}
```

### Functional Options

```go
config := messaging.NewConfig(
    messaging.WithProvider("twilio"),
    messaging.WithTimeout(30*time.Second),
    messaging.WithRetryConfig(3, time.Second, 2.0),
    messaging.WithObservability(true, true, true),
)
```

## Interface

### ConversationalBackend

The `ConversationalBackend` interface provides methods for:
- **Lifecycle**: Start, Stop
- **Conversation Management**: CreateConversation, GetConversation, ListConversations, CloseConversation
- **Message Operations**: SendMessage, ReceiveMessages
- **Participant Management**: AddParticipant, RemoveParticipant
- **Webhook Handling**: HandleWebhook
- **Health & Status**: HealthCheck, GetConfig

## Error Handling

The package uses custom error types with error codes:

```go
// Error codes
ErrCodeInvalidConfig
ErrCodeNetworkError
ErrCodeTimeout
ErrCodeRateLimit
ErrCodeConversationNotFound
ErrCodeInvalidMessage
// ... and more
```

## Observability

### Metrics

- `messaging_operations_total`: Total operations (counter)
- `messaging_operation_duration_seconds`: Operation duration (histogram)
- `messaging_errors_total`: Total errors (counter)
- `messaging_conversations_total`: Conversations created (counter)
- `messaging_messages_total`: Messages sent (counter)
- `messaging_active_conversations`: Active conversations (gauge)

### Tracing

All public methods create OTEL spans with attributes:
- `operation`: Operation name
- `provider`: Provider name
- `conversation_sid`: Conversation identifier (when applicable)
- `message_sid`: Message identifier (when applicable)

### Logging

Structured logging with OTEL context:
- Trace ID and span ID included
- Log levels: DEBUG (detailed), INFO (operations), WARN (errors), ERROR (failures)

## Integration with Beluga Packages

### Agents Integration
- Use `pkg/agents` for AI agent responses
- Integrate with messaging sessions for text-based conversations

### Memory Integration
- Use `pkg/memory/VectorStoreMemory` for conversation persistence
- Store conversation history across sessions
- Enable multi-channel context preservation

### Orchestration Integration
- Use `pkg/orchestration` for event-driven workflows
- Trigger workflows from webhook events
- Create DAG workflows for complex message flows

## Testing

The package includes comprehensive testing utilities:

```go
// Advanced mock
mock := messaging.NewAdvancedMockMessaging(
    messaging.WithMockError(false, nil),
    messaging.WithHealthState("healthy"),
)

// Concurrent testing
runner := messaging.NewConcurrentTestRunner(10, 5*time.Second, testFunc)
err := runner.Run()
```

## Related Documentation

- [Conversational Backend API Contract](../../specs/001-twilio-integration/contracts/conversational-backend-api.md)
- [Data Model](../../specs/001-twilio-integration/data-model.md)
- [Quickstart Guide](../../specs/001-twilio-integration/quickstart.md)
