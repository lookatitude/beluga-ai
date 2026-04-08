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

The `EventBus` interface allows agents to communicate through named topics without direct dependencies. Producers publish events to topics; subscribers registered on those topics receive them synchronously in the order they subscribed. This is the same decoupling principle behind message queues like NATS and Kafka, but implemented in-process for agent coordination. The key benefit is that publishers do not need to know who is listening -- you can add or remove subscribers without modifying the publisher code.

```go
import (
    "context"
    "fmt"

    "github.com/lookatitude/beluga-ai/agent"
)

bus := agent.NewInMemoryBus()
ctx := context.Background()

// Subscribe to a topic.
sub, err := bus.Subscribe(ctx, "user.query", func(event agent.AgentEvent) {
    fmt.Printf("Received type=%s payload=%v\n", event.Type, event.Payload)
})
if err != nil {
    // handle error
}
defer sub.Unsubscribe()

// Publish an event.
err = bus.Publish(ctx, "user.query", agent.AgentEvent{
    Type:     "user.query",
    SourceID: "main-agent",
    Payload:  "How do I use the message bus?",
})
if err != nil {
    // handle error
}
```

### AgentEvent

Events carry a type (used for identification), a source ID (which agent published it), and a payload (the actual data):

```go
type AgentEvent struct {
    Type      string
    SourceID  string
    Payload   any
    Timestamp time.Time
}
```

The `Timestamp` is set automatically by `InMemoryBus.Publish` when zero.

## Step 1: Initialize the Event Bus

The event bus is created once and shared across all agents. Subscribers must be registered before events are published, since the built-in bus does not buffer events for late subscribers.

```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/lookatitude/beluga-ai/agent"
)

func main() {
    ctx := context.Background()
    bus := agent.NewInMemoryBus()

    // Register subscribers before publishing.
    auditSub, err := registerAuditLogger(ctx, bus)
    if err != nil {
        fmt.Printf("subscribe error: %v\n", err)
        return
    }
    defer auditSub.Unsubscribe()

    notifySub, err := registerNotifier(ctx, bus)
    if err != nil {
        fmt.Printf("subscribe error: %v\n", err)
        return
    }
    defer notifySub.Unsubscribe()

    // Publish an event.
    if err := bus.Publish(ctx, "user.query", agent.AgentEvent{
        Type:     "user.query",
        SourceID: "gateway",
        Payload:  "What is the company revenue?",
    }); err != nil {
        fmt.Printf("publish error: %v\n", err)
        return
    }

    // Give async handlers time to complete (in production use sync patterns or wait groups).
    time.Sleep(50 * time.Millisecond)
    fmt.Println("All events processed.")
}
```

## Step 2: Build an Audit Logger Subscriber

Audit logging is a classic use case for the pub-sub pattern. Every user query needs to be logged for compliance, but the logging logic should not be mixed into the main agent's processing path. By subscribing to the `user.query` topic, the audit logger receives every query without the main agent needing to know about it.

```go
func registerAuditLogger(ctx context.Context, bus agent.EventBus) (agent.Subscription, error) {
    return bus.Subscribe(ctx, "user.query", func(event agent.AgentEvent) {
        fmt.Printf("[AUDIT] source=%s type=%s payload=%v\n",
            event.SourceID, event.Type, event.Payload)
    })
}
```

## Step 3: Build a Notification Subscriber

Multiple subscribers can listen to the same topic. Each subscriber runs in the order it was registered. Handlers are called synchronously, so slow handlers do delay subsequent handlers -- wrap expensive work in a goroutine if isolation is required.

```go
func registerNotifier(ctx context.Context, bus agent.EventBus) (agent.Subscription, error) {
    return bus.Subscribe(ctx, "user.query", func(event agent.AgentEvent) {
        fmt.Printf("[NOTIFY] Processing query from source %s\n", event.SourceID)
    })
}
```

## Step 4: Cross-Agent Communication

Agents can publish their own events after completing work, creating event chains where one agent's output triggers another agent's input. This enables reactive pipelines: the research agent completes its work and publishes a `research.complete` event, which the email agent picks up to send a summary. Neither agent needs to know about the other -- they are connected only through the topic namespace.

```go
func registerResearchAgent(ctx context.Context, bus agent.EventBus) (agent.Subscription, error) {
    return bus.Subscribe(ctx, "user.query", func(event agent.AgentEvent) {
        query, _ := event.Payload.(string)

        // Simulate research work.
        result := fmt.Sprintf("Research result for: %s", query)

        // Publish a completion event for downstream agents.
        _ = bus.Publish(ctx, "research.complete", agent.AgentEvent{
            Type:     "research.complete",
            SourceID: "research-agent",
            Payload:  result,
        })
    })
}

func registerEmailAgent(ctx context.Context, bus agent.EventBus) (agent.Subscription, error) {
    return bus.Subscribe(ctx, "research.complete", func(event agent.AgentEvent) {
        result, _ := event.Payload.(string)
        fmt.Printf("[EMAIL] Sending summary: %s\n", result)
    })
}
```

## Step 5: Error Handling and Resilience

In a multi-subscriber system, one failing subscriber should not prevent others from being called. The `InMemoryBus` calls handlers synchronously in registration order; wrap subscriber logic with error handling to prevent panics from escaping the handler.

```go
func safeSubscribe(ctx context.Context, bus agent.EventBus, topic string, handler func(agent.AgentEvent) error) (agent.Subscription, error) {
    return bus.Subscribe(ctx, topic, func(event agent.AgentEvent) {
        if err := handler(event); err != nil {
            fmt.Printf("[ERROR] Subscriber on %q failed: %v\n", topic, err)
        }
    })
}
```

To unsubscribe when a subscriber is no longer needed:

```go
sub, err := safeSubscribe(ctx, bus, "user.query", myHandler)
if err != nil {
    return err
}
// Later:
if err := sub.Unsubscribe(); err != nil {
    // handle error
}
```

## Architecture Pattern

The event-driven pattern enables a clean separation of concerns:

```
User Input
    |
    v
[Main Agent] --publish("user.query")--> InMemoryBus
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

1. Start the application with multiple subscribers registered before publishing.
2. Publish a test event.
3. Verify each subscriber handler is called in registration order.
4. Register a downstream subscriber on `research.complete` and verify it receives the chained event.
5. Call `Unsubscribe()` on one subscription and verify it no longer receives events.

## Next Steps

- [DAG Workflows](/docs/tutorials/orchestration/dag-workflows) -- Graph-based flows as an alternative to event-driven patterns
- [Temporal Workflows](/docs/tutorials/orchestration/temporal-workflows) -- Durable event handling for workflows that must survive restarts
