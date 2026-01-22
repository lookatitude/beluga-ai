# NATS Message Bus

Welcome, colleague! In this integration guide, we're going to integrate NATS message bus with Beluga AI's orchestration package. NATS provides high-performance, distributed messaging for multi-agent coordination and workflow orchestration.

## What you will build

You will configure Beluga AI to use NATS for distributed messaging between agents and orchestration components, enabling scalable, event-driven multi-agent systems with pub/sub and request/reply patterns.

## Learning Objectives

- ✅ Configure NATS with Beluga AI orchestration
- ✅ Implement pub/sub messaging
- ✅ Use request/reply patterns
- ✅ Coordinate multi-agent systems

## Prerequisites

- Go 1.24 or later installed
- Beluga AI Framework installed
- NATS server
- NATS Go client

## Step 1: Setup and Installation

Install NATS Go client:
bash
```bash
go get github.com/nats-io/nats.go
```

Start NATS server:
nats-server
```

Or use Docker:
docker run -p 4222:4222 nats:latest
```

## Step 2: Create NATS Message Bus

Create a NATS-based message bus:
```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "time"

    "github.com/lookatitude/beluga-ai/pkg/orchestration/iface"
    "github.com/nats-io/nats.go"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"
)

type NATSMessageBus struct {
    conn   *nats.Conn
    tracer trace.Tracer
}

type Message struct {
    Type      string
    Payload   map[string]interface{}
    Timestamp time.Time
    MessageID string
}

func NewNATSMessageBus(natsURL string) (*NATSMessageBus, error) {
    conn, err := nats.Connect(natsURL)
    if err != nil {
        return nil, fmt.Errorf("failed to connect: %w", err)
    }
    
    return &NATSMessageBus{
        conn:   conn,
        tracer: otel.Tracer("beluga.orchestration.nats"),
    }, nil
}

func (b *NATSMessageBus) Publish(ctx context.Context, subject string, msg Message) error {
    ctx, span := b.tracer.Start(ctx, "nats.publish",
        trace.WithAttributes(
            attribute.String("subject", subject),
            attribute.String("message_type", msg.Type),
        ),
    )
    defer span.End()
    
    data, err := json.Marshal(msg)
    if err != nil {
        span.RecordError(err)
        return fmt.Errorf("failed to marshal: %w", err)
    }
    
    err = b.conn.Publish(subject, data)
    if err != nil {
        span.RecordError(err)
        return fmt.Errorf("publish failed: %w", err)
    }
    
    return nil
}

func (b *NATSMessageBus) Subscribe(ctx context.Context, subject string, handler func(Message) error) error {
    _, err := b.conn.Subscribe(subject, func(natsMsg *nats.Msg) {
        var msg Message
        if err := json.Unmarshal(natsMsg.Data, &msg); err != nil {
            log.Printf("Failed to unmarshal message: %v", err)
            return
        }
        
        if err := handler(msg); err != nil {
            log.Printf("Handler error: %v", err)
        }
    })

    
    return err
}
```

## Step 3: Use with Multi-Agent System

Use NATS for agent coordination:
```go
func main() {
    ctx := context.Background()
    
    // Create message bus
    bus, err := NewNATSMessageBus("nats://localhost:4222")
    if err != nil {
        log.Fatalf("Failed to create bus: %v", err)
    }
    defer bus.conn.Close()
    
    // Agent 1: Subscribe to requests
    bus.Subscribe(ctx, "agent.requests", func(msg Message) error {
        fmt.Printf("Agent 1 received: %v\n", msg)
        // Process request
        response := Message{
            Type:      "response",
            Payload:   map[string]interface{}{"result": "processed"},
            Timestamp: time.Now(),
        }
        return bus.Publish(ctx, "agent.responses", response)
    })
    
    // Agent 2: Publish request
    request := Message{
        Type:      "request",
        Payload:   map[string]interface{}{"task": "process"},
        Timestamp: time.Now(),
    }
    bus.Publish(ctx, "agent.requests", request)

    
    time.Sleep(1 * time.Second) // Wait for processing
}
```

## Step 4: Request/Reply Pattern

Implement request/reply:
```go
func (b *NATSMessageBus) Request(ctx context.Context, subject string, msg Message, timeout time.Duration) (Message, error) {
    data, _ := json.Marshal(msg)
    
    natsMsg, err := b.conn.RequestWithContext(ctx, subject, data, timeout)
    if err != nil {
        return Message{}, fmt.Errorf("request failed: %w", err)
    }
    
    var response Message
    if err := json.Unmarshal(natsMsg.Data, &response); err != nil {
        return Message{}, fmt.Errorf("failed to unmarshal: %w", err)
    }

    
    return response, nil
}
```

## Step 5: Complete Integration

Here's a complete, production-ready example:
```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "time"

    "github.com/lookatitude/beluga-ai/pkg/orchestration/iface"
    "github.com/nats-io/nats.go"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"
)

type ProductionNATSMessageBus struct {
    conn   *nats.Conn
    tracer trace.Tracer
}

func NewProductionNATSMessageBus(natsURL string) (*ProductionNATSMessageBus, error) {
    opts := []nats.Option{
        nats.Name("Beluga AI Message Bus"),
        nats.ReconnectWait(2 * time.Second),
        nats.MaxReconnects(10),
        nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
            log.Printf("Disconnected: %v", err)
        }),
        nats.ReconnectHandler(func(nc *nats.Conn) {
            log.Printf("Reconnected to %v", nc.ConnectedUrl())
        }),
    }
    
    conn, err := nats.Connect(natsURL, opts...)
    if err != nil {
        return nil, fmt.Errorf("failed to connect: %w", err)
    }
    
    return &ProductionNATSMessageBus{
        conn:   conn,
        tracer: otel.Tracer("beluga.orchestration.nats"),
    }, nil
}

func (b *ProductionNATSMessageBus) Publish(ctx context.Context, subject string, msg Message) error {
    ctx, span := b.tracer.Start(ctx, "nats.publish",
        trace.WithAttributes(
            attribute.String("subject", subject),
            attribute.String("message_type", msg.Type),
        ),
    )
    defer span.End()
    
    data, err := json.Marshal(msg)
    if err != nil {
        span.RecordError(err)
        return fmt.Errorf("marshal failed: %w", err)
    }
    
    err = b.conn.Publish(subject, data)
    if err != nil {
        span.RecordError(err)
        return fmt.Errorf("publish failed: %w", err)
    }
    
    span.SetAttributes(attribute.Int("message_size", len(data)))
    return nil
}

func (b *ProductionNATSMessageBus) Request(ctx context.Context, subject string, msg Message, timeout time.Duration) (Message, error) {
    ctx, span := b.tracer.Start(ctx, "nats.request",
        trace.WithAttributes(
            attribute.String("subject", subject),
            attribute.Float64("timeout_s", timeout.Seconds()),
        ),
    )
    defer span.End()
    
    data, _ := json.Marshal(msg)
    
    natsMsg, err := b.conn.RequestWithContext(ctx, subject, data, timeout)
    if err != nil {
        span.RecordError(err)
        return Message{}, fmt.Errorf("request failed: %w", err)
    }
    
    var response Message
    if err := json.Unmarshal(natsMsg.Data, &response); err != nil {
        span.RecordError(err)
        return Message{}, fmt.Errorf("unmarshal failed: %w", err)
    }
    
    span.SetAttributes(attribute.Bool("success", true))
    return response, nil
}

func main() {
    ctx := context.Background()
    
    bus, err := NewProductionNATSMessageBus("nats://localhost:4222")
    if err != nil {
        log.Fatalf("Failed: %v", err)
    }
    defer bus.conn.Close()
    
    // Use message bus
    msg := Message{
        Type:    "task",
        Payload: map[string]interface{}{"action": "process"},
    }

    

    err = bus.Publish(ctx, "tasks", msg)
    if err != nil {
        log.Fatalf("Failed: %v", err)
    }
    
    fmt.Println("Message published successfully")
}
```

## Configuration Options

| Option | Description | Default | Required |
|--------|-------------|---------|----------|
| `NATSURL` | NATS server URL | `nats://localhost:4222` | No |
| `ReconnectWait` | Reconnect delay | `2s` | No |
| `MaxReconnects` | Maximum reconnects | `10` | No |
| `Timeout` | Request timeout | `30s` | No |

## Common Issues

### "Connection refused"

**Problem**: NATS server not running.

**Solution**: Start NATS:nats-server
```

### "Subject not found"

**Problem**: No subscribers for subject.

**Solution**: Ensure subscribers are registered before publishing.

## Production Considerations

When using NATS in production:

- **Clustering**: Use NATS cluster for high availability
- **JetStream**: Use JetStream for persistence
- **Security**: Enable TLS and authentication
- **Monitoring**: Monitor message rates and latency
- **Error handling**: Implement retry and dead letter queues

## Next Steps

Congratulations! You've integrated NATS with Beluga AI. Next, learn how to:

- **[Kubernetes Job Scheduler](./kubernetes-job-scheduler.md)** - Kubernetes integration
- **[Orchestration Package Documentation](../../api/packages/orchestration.md)** - Deep dive into orchestration
- **[Orchestration Tutorial](../../getting-started/06-orchestration-basics.md)** - Orchestration patterns

---

**Ready for more?** Check out the [Integrations Index](./README.md) for more integration guides!
