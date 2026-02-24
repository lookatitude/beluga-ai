---
title: NATS Message Bus Integration
description: "Integrate NATS messaging with Beluga AI for sub-millisecond distributed agent coordination using pub/sub and JetStream in Go."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "NATS messaging, distributed agents, Beluga AI, pub/sub, JetStream, agent coordination Go, event-driven AI"
---

When your system grows beyond a single agent, you need a way for agents to communicate without tight coupling. NATS provides sub-millisecond messaging with minimal operational overhead -- no broker clusters to manage for core pub/sub, and JetStream adds persistence when you need guaranteed delivery. This makes NATS the natural choice for distributed agent coordination, event-driven orchestration, and scalable multi-agent architectures. This guide covers using NATS with Beluga AI for inter-agent communication and scalable multi-agent coordination.

## Overview

NATS provides lightweight, cloud-native messaging that pairs well with Beluga AI's agent and orchestration systems:

- **Pub/Sub messaging** -- broadcast events to multiple agent subscribers
- **Request/Reply** -- synchronous communication between agents with timeouts
- **Queue groups** -- load-balanced message delivery across agent replicas
- **JetStream** -- durable message persistence for reliable delivery

This integration uses the official NATS Go client to build a message bus that agents use for coordination.

## Prerequisites

- Go 1.23 or later
- A running NATS server (local or remote)
- Beluga AI framework installed

## Installation

Install the NATS Go client:

```bash
go get github.com/nats-io/nats.go
```

Start a NATS server using one of these methods:

```bash
# Option 1: Direct binary
nats-server

# Option 2: Docker
docker run -p 4222:4222 nats:latest
```

## Configuration

### Message Bus Implementation

Create a NATS-based message bus for agent communication:

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "time"

    "github.com/nats-io/nats.go"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"
)

// Message represents a structured message exchanged between agents.
type Message struct {
    Type      string                 `json:"type"`
    Payload   map[string]interface{} `json:"payload"`
    Timestamp time.Time              `json:"timestamp"`
    MessageID string                 `json:"message_id"`
}

// NATSMessageBus provides pub/sub and request/reply messaging over NATS.
type NATSMessageBus struct {
    conn   *nats.Conn
    tracer trace.Tracer
}

// NewNATSMessageBus connects to a NATS server and returns a message bus.
func NewNATSMessageBus(natsURL string) (*NATSMessageBus, error) {
    conn, err := nats.Connect(natsURL)
    if err != nil {
        return nil, fmt.Errorf("failed to connect to NATS: %w", err)
    }

    return &NATSMessageBus{
        conn:   conn,
        tracer: otel.Tracer("beluga.orchestration.nats"),
    }, nil
}

// Publish sends a message to the specified subject.
func (b *NATSMessageBus) Publish(ctx context.Context, subject string, msg Message) error {
    ctx, span := b.tracer.Start(ctx, "nats.publish",
        trace.WithAttributes(
            attribute.String("messaging.system", "nats"),
            attribute.String("messaging.destination.name", subject),
            attribute.String("messaging.message.type", msg.Type),
        ),
    )
    defer span.End()

    data, err := json.Marshal(msg)
    if err != nil {
        span.RecordError(err)
        return fmt.Errorf("failed to marshal message: %w", err)
    }

    if err := b.conn.Publish(subject, data); err != nil {
        span.RecordError(err)
        return fmt.Errorf("failed to publish to %s: %w", subject, err)
    }

    return nil
}

// Subscribe registers a handler for messages on the specified subject.
func (b *NATSMessageBus) Subscribe(ctx context.Context, subject string, handler func(Message) error) error {
    _, err := b.conn.Subscribe(subject, func(natsMsg *nats.Msg) {
        var msg Message
        if err := json.Unmarshal(natsMsg.Data, &msg); err != nil {
            log.Printf("failed to unmarshal message on %s: %v", subject, err)
            return
        }

        if err := handler(msg); err != nil {
            log.Printf("handler error on %s: %v", subject, err)
        }
    })

    return err
}

// Close closes the NATS connection.
func (b *NATSMessageBus) Close() {
    b.conn.Close()
}
```

### Configuration Reference

| Option | Description | Default | Required |
|--------|-------------|---------|----------|
| `NATS URL` | NATS server URL | `nats://localhost:4222` | No |
| `ReconnectWait` | Delay between reconnection attempts | `2s` | No |
| `MaxReconnects` | Maximum reconnection attempts | `10` | No |
| `Name` | Client connection name | - | No |

## Usage

### Pub/Sub Agent Coordination

Use pub/sub messaging to coordinate multiple agents:

```go
func main() {
    ctx := context.Background()

    bus, err := NewNATSMessageBus("nats://localhost:4222")
    if err != nil {
        log.Fatalf("failed to create message bus: %v", err)
    }
    defer bus.Close()

    // Agent 1: Subscribe to task requests
    err = bus.Subscribe(ctx, "agent.requests", func(msg Message) error {
        fmt.Printf("Agent 1 received request: %v\n", msg.Type)

        response := Message{
            Type:      "response",
            Payload:   map[string]interface{}{"result": "processed"},
            Timestamp: time.Now(),
        }
        return bus.Publish(ctx, "agent.responses", response)
    })
    if err != nil {
        log.Fatalf("failed to subscribe: %v", err)
    }

    // Agent 2: Send a task request
    request := Message{
        Type:      "request",
        Payload:   map[string]interface{}{"task": "analyze-document"},
        Timestamp: time.Now(),
    }

    if err := bus.Publish(ctx, "agent.requests", request); err != nil {
        log.Fatalf("failed to publish: %v", err)
    }

    // Allow time for message processing
    time.Sleep(1 * time.Second)
}
```

### Request/Reply Pattern

Implement synchronous request/reply for agent-to-agent communication:

```go
// Request sends a message and waits for a reply within the given timeout.
func (b *NATSMessageBus) Request(ctx context.Context, subject string, msg Message, timeout time.Duration) (Message, error) {
    ctx, span := b.tracer.Start(ctx, "nats.request",
        trace.WithAttributes(
            attribute.String("messaging.system", "nats"),
            attribute.String("messaging.destination.name", subject),
            attribute.Float64("messaging.timeout_s", timeout.Seconds()),
        ),
    )
    defer span.End()

    data, err := json.Marshal(msg)
    if err != nil {
        span.RecordError(err)
        return Message{}, fmt.Errorf("failed to marshal request: %w", err)
    }

    natsMsg, err := b.conn.RequestWithContext(ctx, subject, data)
    if err != nil {
        span.RecordError(err)
        return Message{}, fmt.Errorf("request to %s failed: %w", subject, err)
    }

    var response Message
    if err := json.Unmarshal(natsMsg.Data, &response); err != nil {
        span.RecordError(err)
        return Message{}, fmt.Errorf("failed to unmarshal response: %w", err)
    }

    return response, nil
}
```

## Advanced Topics

### Production-Ready Connection

Configure reconnection handling, connection monitoring, and named connections for production:

```go
func NewProductionNATSMessageBus(natsURL string) (*NATSMessageBus, error) {
    opts := []nats.Option{
        nats.Name("beluga-ai-message-bus"),
        nats.ReconnectWait(2 * time.Second),
        nats.MaxReconnects(10),
        nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
            log.Printf("NATS disconnected: %v", err)
        }),
        nats.ReconnectHandler(func(nc *nats.Conn) {
            log.Printf("NATS reconnected to %s", nc.ConnectedUrl())
        }),
        nats.ClosedHandler(func(nc *nats.Conn) {
            log.Printf("NATS connection closed: %v", nc.LastError())
        }),
    }

    conn, err := nats.Connect(natsURL, opts...)
    if err != nil {
        return nil, fmt.Errorf("failed to connect to NATS: %w", err)
    }

    return &NATSMessageBus{
        conn:   conn,
        tracer: otel.Tracer("beluga.orchestration.nats"),
    }, nil
}
```

### JetStream for Durable Messaging

Use NATS JetStream for message persistence when agents must not miss events:

```go
func (b *NATSMessageBus) PublishDurable(ctx context.Context, stream string, subject string, msg Message) error {
    js, err := b.conn.JetStream()
    if err != nil {
        return fmt.Errorf("failed to get JetStream context: %w", err)
    }

    data, err := json.Marshal(msg)
    if err != nil {
        return fmt.Errorf("failed to marshal message: %w", err)
    }

    _, err = js.Publish(subject, data)
    if err != nil {
        return fmt.Errorf("failed to publish to JetStream: %w", err)
    }

    return nil
}
```

### Production Considerations

When deploying NATS messaging in production:

- **Clustering**: Deploy a 3-node NATS cluster for high availability. NATS uses Raft consensus for leader election.
- **JetStream**: Enable JetStream for message persistence. Without it, messages sent while a subscriber is offline are lost.
- **TLS**: Enable TLS for all NATS connections. Use mutual TLS (mTLS) in zero-trust environments.
- **Authentication**: Configure NATS accounts and users with appropriate permissions per subject.
- **Monitoring**: Use the NATS monitoring endpoint (`http://localhost:8222/varz`) or Prometheus exporter for metrics on message rates, connection counts, and latency.
- **Dead letter queues**: Implement retry logic with a dead letter subject for messages that fail processing after multiple attempts.

## Troubleshooting

### "Connection refused"

The NATS server is not running or not reachable at the specified URL.

```bash
# Verify the server is running
nats-server --addr 0.0.0.0 --port 4222

# Or start with Docker
docker run -p 4222:4222 -p 8222:8222 nats:latest

# Test connectivity
nats sub test.subject
```

### "No responders"

No subscribers are registered on the subject used with `Request`. Ensure the subscribing agent is running before the requesting agent sends a message. For durable patterns, use JetStream instead of core NATS.

## Related Resources

- [Kubernetes Job Scheduler](/docs/integrations/kubernetes-scheduler) -- Kubernetes workflow scheduling
- [Infrastructure](/docs/integrations/infrastructure) -- Infrastructure integration patterns
- [Messaging Platforms](/docs/integrations/messaging) -- Twilio, Slack, and custom messaging channels
