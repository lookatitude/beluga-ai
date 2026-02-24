---
title: Event-Driven Agents with Message Bus
description: "Build event-driven AI architectures in Go where agents react asynchronously to system events using publish-subscribe patterns with Beluga AI's EventBus."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Beluga AI, Go, tutorial, event-driven, message bus, pub-sub, async agents, EventBus"
---

Standard chains are synchronous and tightly coupled -- the caller waits for each step to complete before proceeding. Event-driven architectures decouple producers from consumers, enabling asynchronous processing, parallel reactions to the same event, and clean separation of concerns. Instead of wiring agents together with direct function calls, agents subscribe to named topics and react independently when events are published. This pattern is particularly valuable when multiple systems need to respond to the same trigger (audit logging, notifications, downstream processing) without creating a dependency web.

## What You Will Build

An event-driven system where a main agent publishes events and multiple subscriber agents react independently. You will build an audit logger, a notification agent, and a cross-agent communication pipeline.

## Prerequisites

- Familiarity with the `agent` package
- Understanding of Go concurrency patterns

## Core Concepts

### Publish-Subscribe Pattern

The EventBus allows agents to communicate through named topics without direct dependencies. Producers publish events to topics; subscribers registered on those topics receive them asynchronously. This is the same decoupling principle behind message queues like NATS and Kafka, but implemented in-process for agent coordination. The key benefit is that publishers do not need to know who is listening -- you can add or remove subscribers without modifying the publisher code.

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

Events carry a topic name (used for routing to subscribers), a payload (the actual data), and optional metadata (for cross-cutting concerns like user IDs and session tracking):

```go
type Event struct {
    Topic    string
    Payload  any
    Metadata map[string]any
}
```

## Step 1: Initialize the Event Bus

The event bus is created once and shared across all agents. Subscribers must be registered before events are published, since the bus does not buffer events for late subscribers. The `sync.WaitGroup` ensures the main goroutine waits for all subscribers to finish processing before exiting -- in a long-running server, you would typically not need this since the process stays alive.

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

Audit logging is a classic use case for the pub-sub pattern. Every user query needs to be logged for compliance, but the logging logic should not be mixed into the main agent's processing path. By subscribing to the `user.query` topic, the audit logger receives every query without the main agent needing to know about it.

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

Multiple subscribers can listen to the same topic. Each subscriber runs independently -- if the audit logger is slow, it does not block the notification subscriber. This isolation is a key advantage over synchronous middleware chains where a slow handler delays the entire pipeline.

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

Agents can publish their own events after completing work, creating event chains where one agent's output triggers another agent's input. This enables reactive pipelines: the research agent completes its work and publishes a `research.complete` event, which the email agent picks up to send a summary. Neither agent needs to know about the other -- they are connected only through the topic namespace.

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

In a multi-subscriber system, one failing subscriber should not bring down the others. Wrapping subscriber logic with error handling ensures that failures are logged and contained rather than propagated. In production, you would also add metrics (count of failed handlers per topic) and potentially dead-letter queues for events that consistently fail processing.

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

- [DAG Workflows](/docs/tutorials/orchestration/dag-workflows) -- Graph-based flows as an alternative to event-driven patterns
- [Temporal Workflows](/docs/tutorials/orchestration/temporal-workflows) -- Durable event handling for workflows that must survive restarts
