# Redis Distributed Locking

Welcome, colleague! In this integration guide, we're going to implement distributed locking with Redis for Beluga AI's memory operations. This ensures thread-safe memory operations in distributed systems.

## What you will build

You will create a distributed locking mechanism using Redis to coordinate memory operations across multiple instances, preventing race conditions and ensuring data consistency.

## Learning Objectives

- ✅ Implement Redis distributed locks
- ✅ Use locks for memory operations
- ✅ Handle lock timeouts and failures
- ✅ Understand distributed locking patterns

## Prerequisites

- Go 1.24 or later installed
- Beluga AI Framework installed
- Redis server
- Redis Go client

## Step 1: Setup and Installation

Install Redis client:
bash
```bash
go get github.com/redis/go-redis/v9
```

Start Redis:
redis-server
```

## Step 2: Create Distributed Lock

Create a Redis-based distributed lock:
```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/redis/go-redis/v9"
)

type DistributedLock struct {
    client *redis.Client
    key    string
    value  string
    ttl    time.Duration
}

func NewDistributedLock(client *redis.Client, key string, ttl time.Duration) *DistributedLock {
    return &DistributedLock{
        client: client,
        key:    key,
        value:  generateLockValue(),
        ttl:    ttl,
    }
}

func generateLockValue() string {
    return fmt.Sprintf("%d-%d", time.Now().UnixNano(), 12345) // Use UUID in production
}

func (l *DistributedLock) Acquire(ctx context.Context) (bool, error) {
    result, err := l.client.SetNX(ctx, l.key, l.value, l.ttl).Result()
    if err != nil {
        return false, fmt.Errorf("failed to acquire lock: %w", err)
    }
    return result, nil
}

func (l *DistributedLock) Release(ctx context.Context) error {
    // Use Lua script for atomic release
    script := `
        if redis.call("get", KEYS[1]) == ARGV[1] then
            return redis.call("del", KEYS[1])
        else
            return 0
        end
    `
    
    result, err := l.client.Eval(ctx, script, []string{l.key}, l.value).Result()
    if err != nil {
        return fmt.Errorf("failed to release lock: %w", err)
    }
    
    if result.(int64) == 0 {
        return fmt.Errorf("lock not held by this instance")
    }
    
    return nil
}
```

## Step 3: Use Lock with Memory Operations

Wrap memory operations with locks:
```go
type LockedMemory struct {
    memory iface.Memory
    lock   *DistributedLock
}

func NewLockedMemory(memory iface.Memory, lock *DistributedLock) *LockedMemory {
    return &LockedMemory{
        memory: memory,
        lock:   lock,
    }
}

func (m *LockedMemory) SaveContext(ctx context.Context, inputs, outputs map[string]any) error {
    // Acquire lock
    acquired, err := m.lock.Acquire(ctx)
    if err != nil {
        return fmt.Errorf("failed to acquire lock: %w", err)
    }
    if !acquired {
        return fmt.Errorf("failed to acquire lock: lock held by another instance")
    }
    defer m.lock.Release(ctx)
    
    // Perform operation
    return m.memory.SaveContext(ctx, inputs, outputs)
}
```

## Step 4: Lock with Retry

Implement retry logic:
```go
func (l *DistributedLock) AcquireWithRetry(ctx context.Context, maxRetries int, retryDelay time.Duration) error {
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
    
    return fmt.Errorf("failed to acquire lock after %d retries", maxRetries)
}
```

## Step 5: Complete Integration

Here's a complete, production-ready example:
```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/lookatitude/beluga-ai/pkg/memory/iface"
    "github.com/redis/go-redis/v9"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"
)

type ProductionDistributedLock struct {
    client *redis.Client
    key    string
    value  string
    ttl    time.Duration
    tracer trace.Tracer
}

func NewProductionDistributedLock(client *redis.Client, key string, ttl time.Duration) *ProductionDistributedLock {
    return &ProductionDistributedLock{
        client: client,
        key:    key,
        value:  fmt.Sprintf("%d", time.Now().UnixNano()),
        ttl:    ttl,
        tracer: otel.Tracer("beluga.memory.redis_lock"),
    }
}

func (l *ProductionDistributedLock) Acquire(ctx context.Context) (bool, error) {
    ctx, span := l.tracer.Start(ctx, "lock.acquire",
        trace.WithAttributes(attribute.String("lock_key", l.key)),
    )
    defer span.End()
    
    result, err := l.client.SetNX(ctx, l.key, l.value, l.ttl).Result()
    if err != nil {
        span.RecordError(err)
        return false, fmt.Errorf("failed to acquire: %w", err)
    }
    
    span.SetAttributes(attribute.Bool("acquired", result))
    return result, nil
}

func (l *ProductionDistributedLock) Release(ctx context.Context) error {
    ctx, span := l.tracer.Start(ctx, "lock.release")
    defer span.End()
    
    script := `
        if redis.call("get", KEYS[1]) == ARGV[1] then
            return redis.call("del", KEYS[1])
        else
            return 0
        end
    `
    
    result, err := l.client.Eval(ctx, script, []string{l.key}, l.value).Result()
    if err != nil {
        span.RecordError(err)
        return fmt.Errorf("failed to release: %w", err)
    }
    
    if result.(int64) == 0 {
        err := fmt.Errorf("lock not held")
        span.RecordError(err)
        return err
    }
    
    return nil
}

type LockedMemoryWrapper struct {
    memory iface.Memory
    lock   *ProductionDistributedLock
}

func NewLockedMemoryWrapper(memory iface.Memory, lock *ProductionDistributedLock) *LockedMemoryWrapper {
    return &LockedMemoryWrapper{
        memory: memory,
        lock:   lock,
    }
}

func (m *LockedMemoryWrapper) SaveContext(ctx context.Context, inputs, outputs map[string]any) error {
    // Acquire lock with timeout
    lockCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()
    
    acquired, err := m.lock.Acquire(lockCtx)
    if err != nil {
        return fmt.Errorf("lock acquisition failed: %w", err)
    }
    if !acquired {
        return fmt.Errorf("lock held by another instance")
    }
    defer m.lock.Release(ctx)
    
    // Perform operation
    return m.memory.SaveContext(ctx, inputs, outputs)
}

func main() {
    ctx := context.Background()
    
    // Create Redis client
    rdb := redis.NewClient(&redis.Options{
        Addr: "localhost:6379",
    })
    defer rdb.Close()
    
    // Create lock
    lock := NewProductionDistributedLock(rdb, "memory:session-123", 30*time.Second)
    
    // Create memory (example - use your actual memory implementation)
    // memory := ...
    // lockedMemory := NewLockedMemoryWrapper(memory, lock)

    
    // Use locked memory
    // lockedMemory.SaveContext(ctx, inputs, outputs)
}
```

## Configuration Options

| Option | Description | Default | Required |
|--------|-------------|---------|----------|
| `LockKey` | Redis key for lock | - | Yes |
| `TTL` | Lock time-to-live | `30s` | No |
| `RetryCount` | Maximum retry attempts | `3` | No |
| `RetryDelay` | Delay between retries | `100ms` | No |

## Common Issues

### "Lock not released"

**Problem**: Lock held after operation completes.

**Solution**: Always use defer:defer lock.Release(ctx)
```

### "Lock timeout"

**Problem**: Operation takes longer than lock TTL.

**Solution**: Increase TTL or implement lock renewal:lock.ttl = 60 * time.Second
```

## Production Considerations

When using distributed locks in production:

- **Lock TTL**: Set appropriate TTL to prevent deadlocks
- **Lock renewal**: Implement renewal for long operations
- **Error handling**: Handle lock failures gracefully
- **Monitoring**: Track lock acquisition/release
- **Deadlock prevention**: Use timeouts and retries

## Next Steps

Congratulations! You've implemented Redis distributed locking. Next, learn how to:

- **[MongoDB Context Persistence](./mongodb-context-persistence.md)** - MongoDB integration
- **[Memory Package Documentation](../../api/packages/memory.md)** - Deep dive into memory package
- **[Memory Tutorial](../../getting-started/05-memory-management.md)** - Memory patterns

---

**Ready for more?** Check out the [Integrations Index](./README.md) for more integration guides!
