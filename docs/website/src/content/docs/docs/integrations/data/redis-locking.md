---
title: Redis Distributed Locking
description: "Implement Redis distributed locking for Beluga AI memory operations to ensure thread-safe coordination across multiple agent instances."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Redis locking, distributed lock, Beluga AI, concurrency control, Redis Go, memory coordination, multi-instance AI"
---

Scaling Beluga AI horizontally -- running multiple agent instances behind a load balancer -- introduces a concurrency problem: two instances processing messages for the same conversation can write to the shared memory backend simultaneously, corrupting sequence ordering or duplicating entries. Distributed locking coordinates these writes so only one instance modifies a given conversation at a time.

This guide demonstrates Redis-based distributed locking, which is the right choice when you already have Redis in your infrastructure and need a lightweight coordination mechanism without introducing a separate consensus system like etcd or ZooKeeper.

## Overview

Redis distributed locks use the `SET NX` (set-if-not-exists) command to provide mutual exclusion. Combined with atomic Lua scripts for safe release, this pattern prevents concurrent memory mutations from corrupting shared state.

Key properties:
- Mutual exclusion across processes and machines
- Automatic lock expiration via TTL to prevent deadlocks
- Atomic release using Lua scripts to avoid accidental unlocks
- Retry with backoff for contention handling

## Prerequisites

- Go 1.23 or later
- A Beluga AI project initialized with `go mod init`
- A Redis server (v5.0 or later)
- The `go-redis/v9` client library

## Installation

Install the Redis client:

```bash
go get github.com/redis/go-redis/v9
```

Start a Redis server if you do not already have one:

```bash
# Using Docker
docker run -d --name redis -p 6379:6379 redis:7

# Or directly
redis-server
```

## Configuration

### Distributed Lock Implementation

Create a Redis-based distributed lock:

```go
package distlock

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Lock represents a Redis-based distributed lock.
type Lock struct {
	client *redis.Client
	key    string
	value  string
	ttl    time.Duration
}

// New creates a new distributed lock with the given key and TTL.
func New(client *redis.Client, key string, ttl time.Duration) *Lock {
	return &Lock{
		client: client,
		key:    key,
		value:  generateValue(),
		ttl:    ttl,
	}
}

// generateValue returns a random token to identify the lock holder.
func generateValue() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// Fallback to timestamp if crypto/rand fails.
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

// Acquire attempts to acquire the lock. Returns true if the lock was
// acquired, false if it is held by another instance.
func (l *Lock) Acquire(ctx context.Context) (bool, error) {
	ok, err := l.client.SetNX(ctx, l.key, l.value, l.ttl).Result()
	if err != nil {
		return false, fmt.Errorf("distlock: acquire failed: %w", err)
	}
	return ok, nil
}

// Release releases the lock using an atomic Lua script to ensure
// only the holder can release it.
func (l *Lock) Release(ctx context.Context) error {
	const script = `
		if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("del", KEYS[1])
		else
			return 0
		end
	`

	result, err := l.client.Eval(ctx, script, []string{l.key}, l.value).Result()
	if err != nil {
		return fmt.Errorf("distlock: release failed: %w", err)
	}

	if result.(int64) == 0 {
		return fmt.Errorf("distlock: lock not held by this instance")
	}

	return nil
}
```

### Retry with Backoff

Add retry logic for acquiring locks under contention:

```go
// AcquireWithRetry attempts to acquire the lock with retries and backoff.
func (l *Lock) AcquireWithRetry(ctx context.Context, maxRetries int, retryDelay time.Duration) error {
	for i := 0; i < maxRetries; i++ {
		acquired, err := l.Acquire(ctx)
		if err != nil {
			return err
		}
		if acquired {
			return nil
		}

		if i < maxRetries-1 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(retryDelay):
			}
		}
	}

	return fmt.Errorf("distlock: failed to acquire lock after %d attempts", maxRetries)
}
```

## Usage

### Wrapping Memory Operations

Use the lock to protect concurrent memory writes:

```go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/lookatitude/beluga-ai/memory"
	"github.com/lookatitude/beluga-ai/schema"
	"github.com/redis/go-redis/v9"
)

func main() {
	ctx := context.Background()

	// Create Redis client.
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer rdb.Close()

	// Create a memory instance (using any backend).
	mem, err := memory.New("inmemory", config.ProviderConfig{})
	if err != nil {
		log.Fatalf("Failed to create memory: %v", err)
	}

	// Create a lock for this session's memory operations.
	lock := distlock.New(rdb, "memory:session-123", 30*time.Second)

	// Acquire the lock before writing.
	lockCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err = lock.AcquireWithRetry(lockCtx, 3, 100*time.Millisecond)
	if err != nil {
		log.Fatalf("Failed to acquire lock: %v", err)
	}
	defer func() {
		if err := lock.Release(ctx); err != nil {
			log.Printf("Lock release warning: %v", err)
		}
	}()

	// Safely perform the memory write.
	err = mem.Save(ctx,
		schema.NewHumanMessage("Hello"),
		schema.NewAIMessage("Hi there!"),
	)
	if err != nil {
		log.Fatalf("Save failed: %v", err)
	}

	fmt.Println("Context saved successfully")
}
```

### Locked Memory Wrapper

Create a reusable wrapper that automatically handles locking:

```go
// LockedMemory wraps a memory.Memory with distributed locking.
type LockedMemory struct {
	mem     memory.Memory
	client  *redis.Client
	lockKey string
	lockTTL time.Duration
}

// NewLockedMemory creates a memory wrapper with distributed lock protection.
func NewLockedMemory(mem memory.Memory, client *redis.Client, lockKey string, ttl time.Duration) *LockedMemory {
	return &LockedMemory{
		mem:     mem,
		client:  client,
		lockKey: lockKey,
		lockTTL: ttl,
	}
}

// Save acquires the lock, writes the messages, and releases the lock.
func (m *LockedMemory) Save(ctx context.Context, input, output schema.Message) error {
	lock := distlock.New(m.client, m.lockKey, m.lockTTL)

	lockCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err := lock.AcquireWithRetry(lockCtx, 3, 100*time.Millisecond)
	if err != nil {
		return fmt.Errorf("lockedmemory: lock failed: %w", err)
	}
	defer func() {
		if releaseErr := lock.Release(ctx); releaseErr != nil {
			log.Printf("Lock release warning: %v", releaseErr)
		}
	}()

	return m.mem.Save(ctx, input, output)
}

// Load delegates directly (read operations may not need locking).
func (m *LockedMemory) Load(ctx context.Context, query string) ([]schema.Message, error) {
	return m.mem.Load(ctx, query)
}

// Search delegates directly.
func (m *LockedMemory) Search(ctx context.Context, query string, k int) ([]schema.Document, error) {
	return m.mem.Search(ctx, query, k)
}

// Clear acquires the lock before clearing.
func (m *LockedMemory) Clear(ctx context.Context) error {
	lock := distlock.New(m.client, m.lockKey, m.lockTTL)

	lockCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err := lock.AcquireWithRetry(lockCtx, 3, 100*time.Millisecond)
	if err != nil {
		return fmt.Errorf("lockedmemory: lock failed: %w", err)
	}
	defer func() {
		if releaseErr := lock.Release(ctx); releaseErr != nil {
			log.Printf("Lock release warning: %v", releaseErr)
		}
	}()

	return m.mem.Clear(ctx)
}
```

## Advanced Topics

### Lock Renewal

For long-running operations, extend the lock TTL before it expires:

```go
// Extend resets the lock TTL. Call this periodically during long operations.
func (l *Lock) Extend(ctx context.Context, ttl time.Duration) error {
	const script = `
		if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("pexpire", KEYS[1], ARGV[2])
		else
			return 0
		end
	`

	result, err := l.client.Eval(ctx, script, []string{l.key}, l.value, ttl.Milliseconds()).Result()
	if err != nil {
		return fmt.Errorf("distlock: extend failed: %w", err)
	}

	if result.(int64) == 0 {
		return fmt.Errorf("distlock: lock not held, cannot extend")
	}

	return nil
}
```

### Observability

Add OpenTelemetry tracing to lock operations:

```go
import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

tracer := otel.Tracer("beluga.memory.distlock")

func (l *Lock) AcquireTraced(ctx context.Context) (bool, error) {
	ctx, span := tracer.Start(ctx, "distlock.acquire",
		trace.WithAttributes(attribute.String("lock_key", l.key)),
	)
	defer span.End()

	acquired, err := l.Acquire(ctx)
	if err != nil {
		span.RecordError(err)
		return false, err
	}

	span.SetAttributes(attribute.Bool("acquired", acquired))
	return acquired, nil
}
```

### Redis Sentinel and Cluster

For high availability, configure the Redis client to use Sentinel or Cluster mode:

```go
// Redis Sentinel for automatic failover
rdb := redis.NewFailoverClient(&redis.FailoverOptions{
	MasterName:    "mymaster",
	SentinelAddrs: []string{"sentinel1:26379", "sentinel2:26379"},
})

// Redis Cluster for horizontal scaling
rdb := redis.NewClusterClient(&redis.ClusterOptions{
	Addrs: []string{"node1:6379", "node2:6379", "node3:6379"},
})
```

Note that distributed locks in Redis Cluster mode require all keys used by a lock to hash to the same slot. Use hash tags (e.g., `{session-123}:lock`) if you need multiple related keys.

## Configuration Reference

| Option | Description | Default | Required |
|--------|-------------|---------|----------|
| Lock key | Redis key for the lock | -- | Yes |
| TTL | Lock time-to-live | `30s` | No |
| Max retries | Maximum acquisition attempts | `3` | No |
| Retry delay | Delay between retry attempts | `100ms` | No |

## Troubleshooting

### Lock Not Released

If a lock is held after the owning process crashes, it will expire automatically after the TTL. To handle this gracefully:
- Always use `defer lock.Release(ctx)` immediately after acquisition.
- Set a TTL that balances between operation duration and deadlock recovery time.

### Lock Timeout

If operations frequently exceed the lock TTL:
1. Increase the TTL to accommodate the expected operation duration.
2. Implement lock renewal for long-running operations (see the Extend method above).
3. Break large operations into smaller locked sections.

### "distlock: lock not held by this instance"

This occurs when `Release` is called after the TTL has expired and another instance has acquired the lock. Ensure the lock TTL is long enough for the protected operation to complete.

## Related Resources

- [MongoDB Context Persistence](/docs/integrations/mongodb-persistence) -- Persistent memory storage
- [Memory System Guide](/docs/guides/memory) -- Full memory architecture documentation
- [Resilience Package](/docs/guides/resilience) -- Framework-level retry and circuit breaker patterns
