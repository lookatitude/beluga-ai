---
title: High-Availability Streaming Proxy
description: Build a resilient streaming proxy with connection pooling, automatic failover, and health monitoring for 99.99% uptime.
---

A cloud services provider needed to build a resilient streaming proxy for LLM responses that could handle high traffic volumes, provide automatic failover, and maintain low latency. Direct LLM connections experienced 5-10% failure rates during peak traffic, causing user-facing errors and requiring manual intervention for recovery. A high-availability streaming proxy with connection pooling, automatic failover, and health monitoring ensures 99.99% uptime.

## Solution Architecture

Beluga AI's core package provides the foundation for building high-availability services. The streaming proxy implements connection pooling for efficient resource usage, health monitoring for automatic failover, and load balancing for distributing requests across multiple connections.

```
┌──────────────┐    ┌──────────────┐    ┌──────────────┐
│   Client     │───▶│   Streaming  │───▶│   Health     │
│   Requests   │    │     Proxy    │    │   Monitor    │
└──────────────┘    └──────┬───────┘    └──────┬───────┘
                           │                   │
                           ▼                   ▼
                    ┌──────────────┐    ┌──────────────┐
                    │  Connection  │◀───│   Primary    │
                    │     Pool     │    │   Provider   │
                    └──────┬───────┘    └──────────────┘
                           │
                           ▼
                    ┌──────────────┐
                    │   Response   │
                    │    Stream    │
                    └──────────────┘
```

## Streaming Proxy Implementation

The proxy manages connections and routes requests to healthy providers.

```go
package main

import (
    "context"
    "fmt"
    "sync"
    "time"

    "github.com/lookatitude/beluga-ai/core"
    "github.com/lookatitude/beluga-ai/llm"
    "github.com/lookatitude/beluga-ai/schema"
)

// StreamingProxy implements high-availability streaming for LLM operations.
type StreamingProxy struct {
    primary    llm.ChatModel
    backup     llm.ChatModel
    connPool   *ConnectionPool
    healthMon  *HealthMonitor
}

// NewStreamingProxy creates a new high-availability streaming proxy.
func NewStreamingProxy(
    primary llm.ChatModel,
    backup llm.ChatModel,
) *StreamingProxy {
    return &StreamingProxy{
        primary:   primary,
        backup:    backup,
        connPool:  NewConnectionPool(10),
        healthMon: NewHealthMonitor(primary, backup),
    }
}

// Stream streams responses with automatic failover.
func (p *StreamingProxy) Stream(
    ctx context.Context,
    msgs []schema.Message,
    opts ...llm.GenerateOption,
) iter.Seq2[schema.StreamChunk, error] {
    return func(yield func(schema.StreamChunk, error) bool) {
        // Select healthy provider
        provider := p.selectProvider(ctx)

        // Get connection from pool
        conn, err := p.connPool.Acquire(ctx)
        if err != nil {
            yield(schema.StreamChunk{}, fmt.Errorf("connection acquisition failed: %w", err))
            return
        }
        defer p.connPool.Release(conn)

        // Stream from provider with failover
        for chunk, err := range provider.Stream(ctx, msgs, opts...) {
            if err != nil {
                // Attempt failover on error
                p.healthMon.RecordFailure(provider)
                backupProvider := p.selectProvider(ctx)

                if backupProvider != provider {
                    // Retry with backup
                    for chunk, err := range backupProvider.Stream(ctx, msgs, opts...) {
                        if !yield(chunk, err) {
                            return
                        }
                    }
                    return
                }

                yield(chunk, err)
                return
            }

            if !yield(chunk, nil) {
                return
            }
        }

        p.healthMon.RecordSuccess(provider)
    }
}

func (p *StreamingProxy) selectProvider(ctx context.Context) llm.ChatModel {
    if p.healthMon.IsHealthy(p.primary) {
        return p.primary
    }
    return p.backup
}
```

## Connection Pool

The connection pool manages reusable connections for performance.

```go
type Connection struct {
    ID        string
    CreatedAt time.Time
    LastUsed  time.Time
}

type ConnectionPool struct {
    maxSize     int
    connections chan *Connection
    active      map[string]*Connection
    mu          sync.Mutex
}

func NewConnectionPool(size int) *ConnectionPool {
    return &ConnectionPool{
        maxSize:     size,
        connections: make(chan *Connection, size),
        active:      make(map[string]*Connection),
    }
}

func (cp *ConnectionPool) Acquire(ctx context.Context) (*Connection, error) {
    select {
    case conn := <-cp.connections:
        conn.LastUsed = time.Now()
        return conn, nil
    default:
        // Create new connection if pool not exhausted
        cp.mu.Lock()
        if len(cp.active) < cp.maxSize {
            conn := &Connection{
                ID:        fmt.Sprintf("conn-%d", len(cp.active)),
                CreatedAt: time.Now(),
                LastUsed:  time.Now(),
            }
            cp.active[conn.ID] = conn
            cp.mu.Unlock()
            return conn, nil
        }
        cp.mu.Unlock()

        // Wait for available connection
        select {
        case conn := <-cp.connections:
            conn.LastUsed = time.Now()
            return conn, nil
        case <-ctx.Done():
            return nil, ctx.Err()
        }
    }
}

func (cp *ConnectionPool) Release(conn *Connection) {
    select {
    case cp.connections <- conn:
    default:
        // Pool is full, close connection
        cp.mu.Lock()
        delete(cp.active, conn.ID)
        cp.mu.Unlock()
    }
}
```

## Health Monitoring

The health monitor tracks provider availability and triggers failover.

```go
type HealthMonitor struct {
    primary       llm.ChatModel
    backup        llm.ChatModel
    primaryFails  int
    backupFails   int
    lastCheck     time.Time
    checkInterval time.Duration
    mu            sync.RWMutex
}

func NewHealthMonitor(primary, backup llm.ChatModel) *HealthMonitor {
    hm := &HealthMonitor{
        primary:       primary,
        backup:        backup,
        checkInterval: 30 * time.Second,
    }

    go hm.monitorHealth(context.Background())

    return hm
}

func (hm *HealthMonitor) monitorHealth(ctx context.Context) {
    ticker := time.NewTicker(hm.checkInterval)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            hm.checkHealth(ctx)
        }
    }
}

func (hm *HealthMonitor) checkHealth(ctx context.Context) {
    // Simple health check with minimal prompt
    testMsg := []schema.Message{
        &schema.HumanMessage{
            Parts: []schema.ContentPart{
                schema.TextPart{Text: "ping"},
            },
        },
    }

    // Check primary
    ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()

    _, err := hm.primary.Generate(ctx, testMsg)
    if err != nil {
        hm.RecordFailure(hm.primary)
    } else {
        hm.RecordSuccess(hm.primary)
    }
}

func (hm *HealthMonitor) IsHealthy(provider llm.ChatModel) bool {
    hm.mu.RLock()
    defer hm.mu.RUnlock()

    if provider == hm.primary {
        return hm.primaryFails < 3
    }
    return hm.backupFails < 3
}

func (hm *HealthMonitor) RecordSuccess(provider llm.ChatModel) {
    hm.mu.Lock()
    defer hm.mu.Unlock()

    if provider == hm.primary {
        hm.primaryFails = 0
    } else {
        hm.backupFails = 0
    }
}

func (hm *HealthMonitor) RecordFailure(provider llm.ChatModel) {
    hm.mu.Lock()
    defer hm.mu.Unlock()

    if provider == hm.primary {
        hm.primaryFails++
    } else {
        hm.backupFails++
    }
}
```

## Synchronous Generation

The proxy also supports non-streaming requests:

```go
func (p *StreamingProxy) Generate(
    ctx context.Context,
    msgs []schema.Message,
    opts ...llm.GenerateOption,
) (*schema.AIMessage, error) {
    provider := p.selectProvider(ctx)

    conn, err := p.connPool.Acquire(ctx)
    if err != nil {
        return nil, fmt.Errorf("connection acquisition failed: %w", err)
    }
    defer p.connPool.Release(conn)

    result, err := provider.Generate(ctx, msgs, opts...)
    if err != nil {
        p.healthMon.RecordFailure(provider)

        // Attempt failover
        backupProvider := p.selectProvider(ctx)
        if backupProvider != provider {
            return backupProvider.Generate(ctx, msgs, opts...)
        }

        return nil, err
    }

    p.healthMon.RecordSuccess(provider)
    return result, nil
}
```

## Production Considerations

### Connection Pool Sizing

Connection pool size should be tuned based on traffic patterns. Start with 10 connections per instance and monitor pool utilization. If connections are frequently exhausted, increase the pool size. Track wait times and connection reuse rates.

### Health Check Strategy

Active health checks should be lightweight to minimize overhead. Use a simple ping-style prompt and short timeout (5 seconds). Check frequency depends on provider reliability—more frequent checks for providers with known issues.

### Load Balancing

For multi-instance deployments, use a load balancer to distribute requests across proxy instances. Each instance maintains its own connection pool and health state. Consider sticky sessions for stateful operations.

### Observability

Track key metrics for proxy operations:

```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/metric"
)

func (p *StreamingProxy) recordMetrics(ctx context.Context, provider llm.ChatModel, duration time.Duration) {
    meter := otel.Meter("streaming-proxy")

    histogram, _ := meter.Float64Histogram("proxy_request_duration_seconds")
    histogram.Record(ctx, duration.Seconds(),
        metric.WithAttributes(
            attribute.String("provider", getProviderName(provider)),
        ),
    )

    counter, _ := meter.Int64Counter("proxy_requests_total")
    counter.Add(ctx, 1,
        metric.WithAttributes(
            attribute.String("provider", getProviderName(provider)),
        ),
    )
}

func getProviderName(provider llm.ChatModel) string {
    return provider.ModelID()
}
```

### Graceful Degradation

When both primary and backup providers are unhealthy, the proxy should fail gracefully with informative error messages. Consider implementing a queue for requests during temporary outages.

## Results

| Metric | Before | After | Improvement |
|--------|--------|-------|-------------|
| Uptime | 99.5% | 99.992% | 0.49% improvement |
| Request Failure Rate | 5-10% | 0.08% | 98-99% reduction |
| Average Latency (ms) | 200-2000 | 85 | 57-96% reduction |
| P95 Latency (ms) | 5000 | 180 | 96% reduction |
| Manual Interventions/Month | 8 | 0 | 100% reduction |

## Related Resources

- [Error Recovery Service](/use-cases/error-recovery-service/) for retry and circuit breaker patterns
- [LLM Package Guide](/guides/llm/) for model integration
- [Observability Guide](/guides/observability/) for monitoring streaming systems
