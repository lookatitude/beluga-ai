---
title: "In-Memory Cache Provider"
description: "In-memory LRU cache for LLM response caching in Beluga AI. Thread-safe cache with TTL, size limits, and zero external dependencies in Go."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "in-memory cache, LRU cache, LLM caching, response cache, TTL, thread-safe, Go, Beluga AI"
---

The in-memory provider implements an LRU (Least Recently Used) cache with TTL-based expiration. It uses a doubly-linked list combined with a hash map for O(1) get, set, and eviction operations. Entries expire lazily on access.

Choose the In-Memory cache for development, testing, and single-process deployments where local memoization is sufficient. It provides O(1) operations with LRU eviction and per-entry TTL, and can be combined with `SemanticCache` for embedding-based lookups. No external dependencies are required.

## Installation

```bash
go get github.com/lookatitude/beluga-ai/cache/providers/inmemory
```

## Configuration

| Field     | Required | Default | Description                              |
|-----------|----------|---------|------------------------------------------|
| `TTL`     | No       | `0`     | Default time-to-live for entries         |
| `MaxSize` | No       | `0`     | Maximum entries before LRU eviction (0 = unlimited) |

## Basic Usage

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/lookatitude/beluga-ai/cache"
    _ "github.com/lookatitude/beluga-ai/cache/providers/inmemory"
)

func main() {
    c, err := cache.New("inmemory", cache.Config{
        TTL:     5 * time.Minute,
        MaxSize: 1000,
    })
    if err != nil {
        log.Fatal(err)
    }

    ctx := context.Background()

    // Store a value
    err = c.Set(ctx, "user:123", map[string]string{"name": "Alice"}, 0)
    if err != nil {
        log.Fatal(err)
    }

    // Retrieve the value
    val, found, err := c.Get(ctx, "user:123")
    if err != nil {
        log.Fatal(err)
    }
    if found {
        fmt.Printf("Found: %v\n", val)
    }
}
```

## Advanced Features

### LRU Eviction

When `MaxSize` is set, the cache evicts the least recently used entry when capacity is exceeded. Accessing or updating an entry promotes it to the front of the LRU list:

```go
c, err := cache.New("inmemory", cache.Config{
    MaxSize: 100, // Keep at most 100 entries
})
if err != nil {
    log.Fatal(err)
}

// After 100 entries, the least recently accessed entry is evicted
for i := 0; i < 200; i++ {
    key := fmt.Sprintf("key:%d", i)
    err := c.Set(ctx, key, i, 0)
    if err != nil {
        log.Fatal(err)
    }
}
```

### TTL Per Entry

Override the default TTL for individual entries:

```go
// Use default TTL (from Config)
err := c.Set(ctx, "default-ttl", "value", 0)

// Custom TTL for this entry
err = c.Set(ctx, "short-lived", "value", 30*time.Second)

// Never expire this entry
err = c.Set(ctx, "permanent", "value", -1)
```

### Lazy Expiration

Expired entries are removed lazily when accessed. A `Get` on an expired key returns `(nil, false, nil)` and removes the entry:

```go
err := c.Set(ctx, "temp", "value", 1*time.Second)
if err != nil {
    log.Fatal(err)
}

time.Sleep(2 * time.Second)

val, found, err := c.Get(ctx, "temp")
// found == false, val == nil (entry expired and was removed)
```

### Bulk Operations

Clear all entries from the cache:

```go
err := c.Clear(ctx)
if err != nil {
    log.Fatal(err)
}
```

Delete a specific entry:

```go
err := c.Delete(ctx, "user:123")
if err != nil {
    log.Fatal(err)
}
```

### Cache Size

The in-memory provider exposes a `Len()` method for monitoring:

```go
import "github.com/lookatitude/beluga-ai/cache/providers/inmemory"

c := inmemory.New(cache.Config{MaxSize: 1000})
// ... add entries ...
fmt.Printf("Cache entries: %d\n", c.Len())
```

## Direct Construction

```go
import "github.com/lookatitude/beluga-ai/cache/providers/inmemory"

c := inmemory.New(cache.Config{
    TTL:     10 * time.Minute,
    MaxSize: 500,
})
```

## Error Handling

The in-memory cache returns `nil` errors for all operations under normal conditions. The `Get` method uses a three-value return to distinguish between missing keys and errors:

```go
val, found, err := c.Get(ctx, "key")
if err != nil {
    log.Printf("cache error: %v", err)
}
if !found {
    // Key does not exist or has expired
}
```

## Thread Safety

The in-memory cache is safe for concurrent use. All operations are protected by a `sync.Mutex`, making it suitable for use across goroutines without additional synchronization.

## Semantic Cache Layer

Combine the In-Memory provider with `SemanticCache` for embedding-based lookups:

```go
import (
    "github.com/lookatitude/beluga-ai/cache"
    "github.com/lookatitude/beluga-ai/cache/providers/inmemory"
)

c := inmemory.New(cache.Config{
    TTL:     10 * time.Minute,
    MaxSize: 5000,
})

sc := cache.NewSemanticCache(c, 0.90)

// Store by embedding
embedding := []float32{0.1, 0.2, 0.3}
err := sc.SetSemantic(ctx, embedding, "cached LLM response")
if err != nil {
    log.Fatal(err)
}

// Retrieve by embedding
val, found, err := sc.GetSemantic(ctx, embedding, 0) // 0 = use default threshold
if err != nil {
    log.Fatal(err)
}
```

## Limitations

- State is lost when the process exits. Not suitable for production use where durability is required.
- Expired entries are removed lazily (on access), not proactively. The `Len()` count may include expired entries.
- No built-in size limit on individual values. Large values consume proportional memory.
- LRU eviction is by entry count, not by memory usage.

## When to Use

The In-Memory cache is the right choice for:

- **Development and testing** where a cache backend is needed without external dependencies
- **Single-process services** where network latency to an external cache is unacceptable
- **LLM response caching** where repeated prompts benefit from local memoization
- **CI pipelines** where no external services are available
