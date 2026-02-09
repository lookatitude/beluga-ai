---
title: "Cache Providers"
description: "Overview of all supported cache providers in Beluga AI."
---

Beluga AI provides a unified `cache.Cache` interface for key-value storage with TTL support. The cache system supports both exact-match and semantic (embedding-based) lookups, making it suitable for LLM response caching, prompt caching, and general application caching.

## How It Works

All cache providers implement the same interface:

```go
type Cache interface {
    Get(ctx context.Context, key string) (any, bool, error)
    Set(ctx context.Context, key string, value any, ttl time.Duration) error
    Delete(ctx context.Context, key string) error
    Clear(ctx context.Context) error
}
```

You can instantiate any provider two ways:

**Via the registry** (recommended for dynamic configuration):

```go
import (
    "time"

    "github.com/lookatitude/beluga-ai/cache"
    _ "github.com/lookatitude/beluga-ai/cache/providers/inmemory"
)

c, err := cache.New("inmemory", cache.Config{
    TTL:     5 * time.Minute,
    MaxSize: 1000,
})
```

**Via direct construction** (for compile-time type safety):

```go
import "github.com/lookatitude/beluga-ai/cache/providers/inmemory"

c := inmemory.New(cache.Config{
    TTL:     5 * time.Minute,
    MaxSize: 1000,
})
```

## Configuration

All providers accept `cache.Config`:

| Field     | Type              | Description                                 |
|-----------|-------------------|---------------------------------------------|
| `TTL`     | `time.Duration`   | Default time-to-live for entries             |
| `MaxSize` | `int`             | Maximum number of entries (0 = unlimited)    |
| `Options` | `map[string]any`  | Provider-specific configuration              |

### TTL Behavior

| TTL Value  | Behavior                           |
|------------|-------------------------------------|
| `> 0`      | Entry expires after the given duration |
| `0`        | Uses the cache's default TTL        |
| `< 0`      | Entry never expires                 |

## Semantic Cache

The `cache.SemanticCache` wraps any `Cache` to provide similarity-based lookups using embedding vectors:

```go
import "github.com/lookatitude/beluga-ai/cache"

c, err := cache.New("inmemory", cache.Config{TTL: 10 * time.Minute})
if err != nil {
    log.Fatal(err)
}

sc := cache.NewSemanticCache(c, 0.95) // 0.95 cosine similarity threshold

// Store with embedding vector
err = sc.SetSemantic(ctx, embedding, cachedResponse)

// Retrieve by similarity
val, found, err := sc.GetSemantic(ctx, queryEmbedding, 0)
```

The threshold parameter controls minimum cosine similarity required for a match:
- `0.95` — Requires very high similarity (near-exact match)
- `0.80` — More permissive, allows paraphrased queries to hit cache

## Available Providers

| Provider | Registry Name | Description |
|----------|---------------|-------------|
| [In-Memory](/providers/cache/inmemory) | `inmemory` | LRU cache with TTL-based expiration |

## Provider Discovery

List all registered cache providers at runtime:

```go
for _, name := range cache.List() {
    fmt.Println(name)
}
```

## Choosing a Provider

| Use Case | Recommended Provider | Reason |
|----------|---------------------|--------|
| Development and testing | `inmemory` | Zero dependencies, fast setup |
| Single-process production | `inmemory` | Low-latency, no network overhead |
| Multi-process / distributed | Custom provider | Implement `cache.Cache` with Redis, Memcached, etc. |
