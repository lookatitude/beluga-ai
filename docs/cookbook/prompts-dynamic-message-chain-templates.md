---
title: "Dynamic Message Chain Templates"
package: "prompts"
category: "templating"
complexity: "advanced"
---

# Dynamic Message Chain Templates

## Problem

You need to build message chains (system, user, assistant messages) dynamically based on runtime conditions, conversation history, or user context, rather than using static templates.

## Solution

Implement a message chain builder that constructs message sequences programmatically, supports conditional message inclusion, and integrates with conversation history. This works because Beluga AI's prompt system supports message arrays, and you can build these arrays dynamically based on your application logic.

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
    
    "github.com/lookatitude/beluga-ai/pkg/prompts"
    "github.com/lookatitude/beluga-ai/pkg/schema"
)

var tracer = otel.Tracer("beluga.prompts.message_chain")

// MessageChainBuilder builds message chains dynamically
type MessageChainBuilder struct {
    messages []schema.Message
    context  map[string]interface{}
}

// NewMessageChainBuilder creates a new builder
func NewMessageChainBuilder() *MessageChainBuilder {
    return &MessageChainBuilder{
        messages: []schema.Message{},
        context:  make(map[string]interface{}),
    }
}

// WithContext sets builder context
func (mcb *MessageChainBuilder) WithContext(key string, value interface{}) *MessageChainBuilder {
    mcb.context[key] = value
    return mcb
}

// AddSystemMessage adds a system message conditionally
func (mcb *MessageChainBuilder) AddSystemMessage(ctx context.Context, content string, condition func(map[string]interface{}) bool) *MessageChainBuilder {
    ctx, span := tracer.Start(ctx, "message_chain.add_system")
    defer span.End()
    
    if condition == nil || condition(mcb.context) {
        mcb.messages = append(mcb.messages, schema.NewSystemMessage(content))
        span.SetAttributes(attribute.Bool("message.added", true))
    } else {
        span.SetAttributes(attribute.Bool("message.added", false))
    }
    
    return mcb
}

// AddUserMessage adds a user message
func (mcb *MessageChainBuilder) AddUserMessage(ctx context.Context, content string) *MessageChainBuilder {
    ctx, span := tracer.Start(ctx, "message_chain.add_user")
    defer span.End()
    
    mcb.messages = append(mcb.messages, schema.NewHumanMessage(content))
    span.SetAttributes(attribute.Int("message_count", len(mcb.messages)))
    
    return mcb
}

// AddHistory adds conversation history
func (mcb *MessageChainBuilder) AddHistory(ctx context.Context, history []schema.Message, maxMessages int) *MessageChainBuilder {
    ctx, span := tracer.Start(ctx, "message_chain.add_history")
    defer span.End()
    
    // Trim history if needed
    if maxMessages > 0 && len(history) > maxMessages {
        history = history[len(history)-maxMessages:]
    }
    
    mcb.messages = append(mcb.messages, history...)
    span.SetAttributes(
        attribute.Int("history_count", len(history)),
        attribute.Int("total_messages", len(mcb.messages)),
    )
    
    return mcb
}

// AddConditionalMessage adds a message based on condition
func (mcb *MessageChainBuilder) AddConditionalMessage(ctx context.Context, msg schema.Message, condition func(map[string]interface{}) bool) *MessageChainBuilder {
    if condition == nil || condition(mcb.context) {
        mcb.messages = append(mcb.messages, msg)
    }
    return mcb
}

// Build returns the message chain
func (mcb *MessageChainBuilder) Build(ctx context.Context) []schema.Message {
    ctx, span := tracer.Start(ctx, "message_chain.build")
    defer span.End()
    
    span.SetAttributes(attribute.Int("message_count", len(mcb.messages)))
    span.SetStatus(trace.StatusOK, "message chain built")
    
    return mcb.messages
}

// Example: Build chain based on user role
func BuildRoleBasedChain(ctx context.Context, userRole string, query string, history []schema.Message) []schema.Message {
    builder := NewMessageChainBuilder().
        WithContext("user_role", userRole)
    
    // Add role-specific system message
    builder.AddSystemMessage(ctx, "You are a helpful assistant.", nil)
    
    if userRole == "admin" {
        builder.AddSystemMessage(ctx, "You have admin privileges and can access all features.", 
            func(ctx map[string]interface{}) bool {
                return ctx["user_role"] == "admin"
            })
    }
    
    // Add history
    builder.AddHistory(ctx, history, 10)
    
    // Add current query
    builder.AddUserMessage(ctx, query)
    
    return builder.Build(ctx)
}

func main() {
    ctx := context.Background()

    // Build dynamic chain
    history := []schema.Message{
        schema.NewHumanMessage("Previous question"),
        schema.NewAIMessage("Previous answer"),
    }
    
    messages := BuildRoleBasedChain(ctx, "admin", "What can I do?", history)
    fmt.Printf("Built chain with %d messages\n", len(messages))
}
```

## Explanation

Let's break down what's happening:

1. **Fluent builder pattern** - Notice how we use method chaining to build message chains incrementally. This makes the code readable and allows conditional inclusion of messages.

2. **Conditional messages** - We support conditional message inclusion based on context. This allows you to customize message chains based on user roles, features, or other runtime conditions.

3. **History integration** - We can add conversation history with limits. This prevents context windows from growing too large while maintaining conversation continuity.

```go
**Key insight:** Build message chains programmatically rather than using static templates. This gives you flexibility to adapt to different scenarios and user contexts.

## Testing

```
Here's how to test this solution:
```go
func TestMessageChainBuilder_BuildsConditionally(t *testing.T) {
    builder := NewMessageChainBuilder().
        WithContext("role", "admin").
        AddSystemMessage(context.Background(), "Base message", nil).
        AddSystemMessage(context.Background(), "Admin message", 
            func(ctx map[string]interface{}) bool {
                return ctx["role"] == "admin"
            })
    
    messages := builder.Build(context.Background())
    require.Len(t, messages, 2)
}

## Variations

### Template-based Messages

Use templates within messages:
func (mcb *MessageChainBuilder) AddTemplatedMessage(ctx context.Context, template string, vars map[string]string) *MessageChainBuilder {
    // Format template and add message
}
```

### Message Validation

Validate message chains before building:
```go
func (mcb *MessageChainBuilder) Validate() error {
    // Check message order, types, etc.
}
```

## Related Recipes

- **[Prompts Partial Variable Substitution](./prompts-partial-variable-substitution.md)** - Incremental variable substitution
- **[Chatmodels Multi-step History Trimming](./chatmodels-multi-step-history-trimming.md)** - Manage conversation history
- **[Prompts Package Guide](../package_design_patterns.md)** - For a deeper understanding of prompts
