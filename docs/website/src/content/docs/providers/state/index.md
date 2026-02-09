---
title: State Providers
description: Overview of shared agent state providers available in Beluga AI v2.
---

Beluga AI v2 provides a shared state system for agent coordination through the `state.Store` interface. Stores support key-value operations with change notifications via `Watch`, enabling reactive patterns where agents can observe and respond to state changes in real time. Keys can be scoped by agent, session, or global visibility.

## Interface

```go
type Store interface {
    Get(ctx context.Context, key string) (any, error)
    Set(ctx context.Context, key string, value any) error
    Delete(ctx context.Context, key string) error
    Watch(ctx context.Context, key string) (<-chan StateChange, error)
    Close() error
}
```

## Key Concepts

### State Changes

Every mutation to the store produces a `StateChange` that is delivered to watchers:

```go
type StateChange struct {
    Key      string   // the affected key
    OldValue any      // previous value (nil if new)
    Value    any      // new value (nil for deletes)
    Op       ChangeOp // "set" or "delete"
}
```

### Scoped Keys

Keys can be scoped to control visibility across the agent system:

```go
type Scope string

const (
    ScopeAgent   Scope = "agent"   // visible only to a single agent
    ScopeSession Scope = "session" // visible within the current session
    ScopeGlobal  Scope = "global"  // visible across all agents and sessions
)

// Create a scoped key: "agent:counter"
key := state.ScopedKey(state.ScopeAgent, "counter")
```

Scoped keys are a naming convention using the format `scope:key`. The store does not enforce visibility rules; scoping is implemented by convention at the application layer.

## Registry Usage

```go
import (
    "context"
    "log"

    "github.com/lookatitude/beluga-ai/state"

    // Register the provider via blank import
    _ "github.com/lookatitude/beluga-ai/state/providers/inmemory"
)

func main() {
    store, err := state.New("inmemory", state.Config{})
    if err != nil {
        log.Fatal(err)
    }
    defer store.Close()

    ctx := context.Background()

    err = store.Set(ctx, state.ScopedKey(state.ScopeGlobal, "count"), 0)
    if err != nil {
        log.Fatal(err)
    }

    val, err := store.Get(ctx, state.ScopedKey(state.ScopeGlobal, "count"))
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("count = %v", val)
}
```

## Available Providers

| Provider | Registry Name | Type | Best For |
|---|---|---|---|
| [In-Memory](/providers/state/inmemory) | `inmemory` | In-process map | Development, testing, single-process agents |

## Watching for Changes

The `Watch` method enables reactive patterns by delivering `StateChange` notifications:

```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

ch, err := store.Watch(ctx, state.ScopedKey(state.ScopeGlobal, "status"))
if err != nil {
    log.Fatal(err)
}

// Observe changes in a goroutine
go func() {
    for change := range ch {
        log.Printf("Key %s changed: %v -> %v (op: %s)",
            change.Key, change.OldValue, change.Value, change.Op)
    }
}()

// Trigger a change
err = store.Set(ctx, state.ScopedKey(state.ScopeGlobal, "status"), "active")
if err != nil {
    log.Fatal(err)
}
```

The watcher channel is closed when:
- The context passed to `Watch` is canceled
- The store is closed via `Close()`

## Hooks

Hooks provide lifecycle callbacks around store operations:

```go
hooks := state.Hooks{
    BeforeGet: func(ctx context.Context, key string) error {
        log.Printf("Getting key: %s", key)
        return nil
    },
    AfterGet: func(ctx context.Context, key string, value any, err error) {
        log.Printf("Got key %s = %v", key, value)
    },
    BeforeSet: func(ctx context.Context, key string, value any) error {
        log.Printf("Setting key %s = %v", key, value)
        return nil
    },
    AfterSet: func(ctx context.Context, key string, value any, err error) {
        log.Printf("Set key %s = %v (err: %v)", key, value, err)
    },
    OnDelete: func(ctx context.Context, key string) error {
        log.Printf("Deleting key: %s", key)
        return nil
    },
    OnWatch: func(ctx context.Context, key string) error {
        log.Printf("Watching key: %s", key)
        return nil
    },
    OnError: func(ctx context.Context, err error) error {
        log.Printf("Error: %v", err)
        return err
    },
}

// Compose multiple hook sets
combined := state.ComposeHooks(loggingHooks, metricsHooks)
```

`Before*` hooks and `OnDelete`/`OnWatch` can abort the operation by returning an error. `OnError` can suppress errors by returning `nil`, or replace the error with a different one. In composed hooks, the first error returned short-circuits remaining hooks.

## Middleware

Middleware wraps a `Store` to add cross-cutting behavior:

```go
type Middleware func(Store) Store

// Apply middleware (first in list is outermost)
store = state.ApplyMiddleware(store, loggingMW, metricsMW)

// Use the built-in WithHooks middleware
store = state.ApplyMiddleware(store,
    state.WithHooks(hooks),
)
```

## Provider Discovery

List all registered state providers at runtime:

```go
names := state.List()
// Returns sorted list: ["inmemory"]
```
