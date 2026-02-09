---
title: Event-Driven Agents with Message Bus
description: Build event-driven AI architectures where agents react to system events asynchronously using publish-subscribe patterns.
---

Standard chains are synchronous and tightly coupled. Event-driven architectures decouple producers from consumers, enabling asynchronous processing, parallel reactions to the same event, and clean separation of concerns. The `agent` package's `EventBus` provides publish-subscribe messaging for coordinating agents.

## What You Will Build

An event-driven system where a main agent publishes events and multiple subscriber agents react independently. You will build an audit logger, a notification agent, and a cross-agent communication pipeline.

## Prerequisites

- Familiarity with the `agent` package
- Understanding of Go concurrency patterns

## Core Concepts

### Publish-Subscribe Pattern

The EventBus allows agents to communicate through named topics without direct dependencies. Producers publish events; subscribers receive them asynchronously:

```go
import "github.com/lookatitude/beluga-ai/agent"

bus := agent.NewEventBus()

// Subscribe to a topic.
bus.Subscribe("user.query", func(ctx context.Context, event agent.Event) {
    fmt.Printf("Received: %v\n", event.Payload)
})

// Publish an event.
bus.Publish(ctx, agent.Event{
    Topic:   "user.query",
    Payload: "How do I use the message bus?",
})
```

### Event Types

Events carry a topic name, a payload, and optional metadata:

```go
type Event struct {
    Topic    string
    Payload  any
    Metadata map[string]any
}
```

## Step 1: Initialize the Event Bus

```go
package main

import (
    "context"
    "fmt"
    "sync"
    "time"

    "github.com/lookatitude/beluga-ai/agent"
)

func main() {
    ctx := context.Background()
    bus := agent.NewEventBus()

    var wg sync.WaitGroup

    // Register subscribers before publishing.
    registerAuditLogger(bus, &wg)
    registerNotifier(bus, &wg)

    // Publish events.
    wg.Add(2) // Expect 2 subscribers to handle the event.
    bus.Publish(ctx, agent.Event{
        Topic:   "user.query",
        Payload: "What is the company revenue?",
        Metadata: map[string]any{
            "user_id":    "user-123",
            "session_id": "sess-456",
        },
    })

    // Wait for all subscribers to process.
    wg.Wait()
    fmt.Println("All events processed.")
}
```

## Step 2: Build an Audit Logger Subscriber

Create a subscriber that logs every user query for compliance:

```go
func registerAuditLogger(bus *agent.EventBus, wg *sync.WaitGroup) {
    bus.Subscribe("user.query", func(ctx context.Context, event agent.Event) {
        defer wg.Done()
        userID := event.Metadata["user_id"]
        fmt.Printf("[AUDIT] User %v asked: %v\n", userID, event.Payload)
    })
}
```

## Step 3: Build a Notification Subscriber

Create a subscriber that sends notifications for specific events:

```go
func registerNotifier(bus *agent.EventBus, wg *sync.WaitGroup) {
    bus.Subscribe("user.query", func(ctx context.Context, event agent.Event) {
        defer wg.Done()
        fmt.Printf("[NOTIFY] Processing query from session %v\n",
            event.Metadata["session_id"])
    })
}
```

## Step 4: Cross-Agent Communication

Agents can publish their own completion events, allowing other agents to react:

```go
func registerResearchAgent(bus *agent.EventBus) {
    bus.Subscribe("user.query", func(ctx context.Context, event agent.Event) {
        query := event.Payload.(string)

        // Simulate research work.
        result := fmt.Sprintf("Research result for: %s", query)

        // Publish a completion event for downstream agents.
        bus.Publish(ctx, agent.Event{
            Topic:   "research.complete",
            Payload: result,
            Metadata: map[string]any{
                "original_query": query,
            },
        })
    })
}

func registerEmailAgent(bus *agent.EventBus) {
    bus.Subscribe("research.complete", func(ctx context.Context, event agent.Event) {
        result := event.Payload.(string)
        fmt.Printf("[EMAIL] Sending summary: %s\n", result)
    })
}
```

## Step 5: Error Handling and Resilience

Wrap subscriber logic with error handling to prevent one subscriber from affecting others:

```go
func safeSubscribe(bus *agent.EventBus, topic string, handler func(context.Context, agent.Event) error) {
    bus.Subscribe(topic, func(ctx context.Context, event agent.Event) {
        if err := handler(ctx, event); err != nil {
            fmt.Printf("[ERROR] Subscriber on %q failed: %v\n", topic, err)
        }
    })
}
```

## Architecture Pattern

The event-driven pattern enables a clean separation of concerns:

```
User Input
    |
    v
[Main Agent] --publish("user.query")--> EventBus
                                           |
                           +---------------+---------------+
                           |               |               |
                           v               v               v
                     [Audit Log]    [Research Agent]  [Notifier]
                                         |
                               publish("research.complete")
                                         |
                                         v
                                   [Email Agent]
```

## Verification

1. Start the application with multiple subscribers.
2. Publish test events.
3. Verify each subscriber handles events independently without blocking others.
4. Verify cross-agent communication by checking that downstream agents receive completion events.

## Next Steps

- [DAG Workflows](/tutorials/orchestration/dag-workflows) -- Graph-based flows as an alternative to event-driven patterns
- [Temporal Workflows](/tutorials/orchestration/temporal-workflows) -- Durable event handling for workflows that must survive restarts
