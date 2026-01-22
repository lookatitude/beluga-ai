# Event-Driven Agents with Message Bus

In this tutorial, you'll learn how to use Beluga AI's internal Message Bus to build event-driven AI architectures, where agents react to system events asynchronously.

## Learning Objectives

- ✅ Understand the Pub/Sub pattern in Orchestration
- ✅ Create and publish events
- ✅ Implement agent subscribers
- ✅ Build a decoupled "Audit Log" agent

## Prerequisites

- [Orchestration Basics](../../getting-started/06-orchestration-basics.md)

## Why a Message Bus?

Standard chains are synchronous and tightly coupled. A Message Bus allows:
- **Asynchronous tasks**: "Send an email after the research agent is done".
- **Monitoring**: "Log every tool call to a security audit service".
- **Parallel reactions**: Multiple agents reacting to the same user input.

## Step 1: Initialize the Message Bus
```go
package main

import (
    "github.com/lookatitude/beluga-ai/pkg/orchestration"
    "github.com/lookatitude/beluga-ai/pkg/orchestration/messagebus"
)

func main() {
    bus := messagebus.NewLocalBus()
    // LocalBus is in-memory. For distributed, use NATS or Redis providers.
}

## Step 2: Defining Events
type UserQueryEvent struct {
    UserID string
    Query  string
}

func (e UserQueryEvent) Topic() string {
    return "user.query"
}
```

## Step 3: Publishing Events

Trigger events from your main application loop or within agents.

```go
bus.Publish(ctx, UserQueryEvent{
    UserID: "123",
    Query:  "How do I use message bus?",
})
```

## Step 4: Subscribing Agents

Create a "Logger Agent" that listens for all queries.
```go
bus.Subscribe("user.query", func(ctx context.Context, event any) {
    e := event.(UserQueryEvent)
    fmt.Printf("[AUDIT] User %s asked: %s\n", e.UserID, e.Query)
    // You could also call another agent here!
    // auditAgent.Invoke(ctx, e.Query)
})


## Step 5: Advanced - Cross-Agent Communication

Agents can publish their own completion events.
// Inside Research Agent
res, _ := agent.Invoke(ctx, input)
bus.Publish(ctx, ResearchCompleteEvent{Result: res})

// Subscribed Email Agent
bus.Subscribe("research.complete", func(...) {
    emailAgent.Invoke(ctx, "Send summary...")
})
```

## Verification

1. Start the application with a subscriber.
2. Publish a few test events.
3. Verify the subscriber handles them without blocking the main thread.

## Next Steps

- **[Building DAG-based Agents](../orchestration-dag-agents.md)** - Compare with graph-based flows.
- **[Long-running Workflows with Temporal](./orchestration-temporal-workflows.md)** - For durable events.
