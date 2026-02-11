---
title: "Dynamic Message Chain Templates"
description: "Build message chains dynamically based on runtime conditions, conversation history, and user context."
---

# Dynamic Message Chain Templates

## Problem

You need to build message chains (system, user, assistant messages) dynamically based on runtime conditions, conversation history, or user context, rather than using static templates.

## Solution

Implement a message chain builder that constructs message sequences programmatically, supports conditional message inclusion, and integrates with conversation history. This works because Beluga AI's prompt system supports message arrays, and you can build these arrays dynamically based on your application logic.

## Why This Matters

LLM APIs consume message arrays where the order, type, and content of each message shape the model's behavior. Static templates work for fixed interactions, but most production applications need dynamic message composition: including different system prompts based on user roles, injecting conversation history with length limits, and conditionally adding context based on feature flags or runtime state.

The `MessageChainBuilder` pattern solves this through a fluent builder API that constructs `[]schema.Message` arrays programmatically. The builder pattern is particularly well-suited here because message chains have ordering constraints (system messages should come first, history before the current query) and conditional inclusion rules (admin instructions only for admin users). Without a builder, this logic ends up as scattered `if` statements and slice appends, which is error-prone and hard to test.

The conditional message methods accept `func(map[string]interface{}) bool` predicates rather than simple boolean flags. This allows conditions to depend on multiple context values (e.g., "include this message if the user is admin AND the feature flag is enabled") without requiring the caller to evaluate the condition before calling the builder. The context map acts as a shared state for all conditions, keeping the builder's API clean while supporting arbitrary condition logic.

## Code Example

```go
package main

import (
    "context"
    "fmt"

    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"

    "github.com/lookatitude/beluga-ai/schema"
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

// BuildRoleBasedChain builds a chain based on user role
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

1. **Fluent builder pattern** -- Method chaining builds message chains incrementally with a readable API. Each method returns `*MessageChainBuilder`, enabling chains like `.WithContext(...).AddSystemMessage(...).AddHistory(...).AddUserMessage(...)`. This makes the message construction order explicit in the code, which is important because LLMs are sensitive to message ordering.

2. **Conditional messages with predicate functions** -- The `AddConditionalMessage` and `AddSystemMessage` methods accept `func(map[string]interface{}) bool` predicates that are evaluated against the builder's context map. This allows complex conditions (e.g., "include if admin AND feature flag enabled") without requiring the caller to pre-evaluate conditions. When the condition is `nil`, the message is always included, providing a clean API for unconditional messages.

3. **History integration with limits** -- The `AddHistory` method accepts a `maxMessages` parameter that truncates history from the beginning, keeping the most recent exchanges. This prevents context windows from growing unbounded while maintaining the most relevant conversation context. The truncation is applied before adding to the chain, so the builder never allocates memory for messages that will be discarded.

4. **OTel instrumentation** -- Each builder method creates a span, providing a trace of the message chain construction process. The `Build` method records the final message count, making it easy to correlate prompt size with LLM response quality or latency in your observability dashboard.

## Testing

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
```

## Variations

### Template-based Messages

Use templates within messages:

```go
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

- **[Partial Variable Substitution](/cookbook/prompts/partial-substitution)** -- Incremental variable substitution
- **[Multi-step History Trimming](/cookbook/llm/history-trimming)** -- Manage conversation history
