---
title: "In-Memory State Provider"
description: "In-process state store for agent coordination in Beluga AI. Thread-safe key-value state with Watch notifications for development and testing in Go."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "in-memory state, agent state, key-value store, Watch notifications, development, testing, Go, Beluga AI"
---

The In-Memory provider implements the `state.Store` interface using a Go map with read-write mutex protection. It supports the full `Store` contract including `Watch` for real-time change notifications. State does not persist across process restarts.

Choose the In-Memory state store for development, testing, and single-process agent deployments. It supports the full `Store` contract including `Watch` for reactive change notifications, scoped keys, and middleware hooks. No external dependencies are required.

## Installation

```bash
go get github.com/lookatitude/beluga-ai/state/providers/inmemory
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/lookatitude/beluga-ai/state"
    _ "github.com/lookatitude/beluga-ai/state/providers/inmemory"
)

func main() {
    store, err := state.New("inmemory", state.Config{})
    if err != nil {
        log.Fatal(err)
    }
    defer store.Close()

    ctx := context.Background()

    err = store.Set(ctx, state.ScopedKey(state.ScopeAgent, "counter"), 42)
    if err != nil {
        log.Fatal(err)
    }

    val, err := store.Get(ctx, state.ScopedKey(state.ScopeAgent, "counter"))
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println("counter:", val) // counter: 42
}
```

## Configuration

The In-Memory store requires no configuration. It registers via `init()` under the name `"inmemory"` and can be created through the registry or directly.

### Via Registry

```go
store, err := state.New("inmemory", state.Config{})
```

### Direct Construction

```go
import "github.com/lookatitude/beluga-ai/state/providers/inmemory"

store := inmemory.New()
defer store.Close()
```

## API Reference

### Get

```go
func (s *Store) Get(ctx context.Context, key string) (any, error)
```

Retrieves the value for the given key. Returns `nil, nil` if the key does not exist. Returns an error if the context is canceled or the store is closed.

### Set

```go
func (s *Store) Set(ctx context.Context, key string, value any) error
```

Stores a value under the given key. Overwrites any existing value. Broadcasts a `StateChange` with `Op: OpSet` to all watchers of this key, including the `OldValue` if one existed.

### Delete

```go
func (s *Store) Delete(ctx context.Context, key string) error
```

Removes the given key. Deleting a non-existent key is a no-op (returns `nil`). Broadcasts a `StateChange` with `Op: OpDelete` to all watchers if the key existed.

### Watch

```go
func (s *Store) Watch(ctx context.Context, key string) (<-chan state.StateChange, error)
```

Returns a buffered channel (capacity 16) that receives `StateChange` notifications for the given key. Multiple watchers can observe the same key. The channel is closed when:

- The context is canceled
- The store is closed via `Close()`

If a watcher's channel is full, change notifications are dropped silently to prevent blocking writers.

### Close

```go
func (s *Store) Close() error
```

Closes the store, marks it as closed, and closes all active watcher channels. Subsequent operations return an error. Calling `Close` multiple times is safe and returns `nil`.

## Watching for Changes

```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

// Set up a watcher before making changes
ch, err := store.Watch(ctx, "agent:status")
if err != nil {
    log.Fatal(err)
}

// Observe changes
go func() {
    for change := range ch {
        switch change.Op {
        case state.OpSet:
            log.Printf("Key %s set to %v (was %v)", change.Key, change.Value, change.OldValue)
        case state.OpDelete:
            log.Printf("Key %s deleted (was %v)", change.Key, change.OldValue)
        }
    }
    log.Println("Watcher closed")
}()

// Trigger changes
err = store.Set(ctx, "agent:status", "active")
err = store.Set(ctx, "agent:status", "idle")
err = store.Delete(ctx, "agent:status")
```

## Using Scoped Keys

```go
// Agent-scoped: visible only within a single agent
agentKey := state.ScopedKey(state.ScopeAgent, "progress")
err := store.Set(ctx, agentKey, 0.75) // key: "agent:progress"

// Session-scoped: visible within the current session
sessionKey := state.ScopedKey(state.ScopeSession, "user_id")
err = store.Set(ctx, sessionKey, "user-123") // key: "session:user_id"

// Global-scoped: visible across all agents and sessions
globalKey := state.ScopedKey(state.ScopeGlobal, "active_agents")
err = store.Set(ctx, globalKey, 5) // key: "global:active_agents"
```

## Concurrency

All operations are protected by a `sync.RWMutex`:

- `Get` acquires a read lock
- `Set`, `Delete`, and `Watch` acquire an exclusive write lock
- Change notifications are broadcast synchronously within the write lock

The store is safe for concurrent use from multiple goroutines.

## With Hooks and Middleware

```go
import "github.com/lookatitude/beluga-ai/state"

store := inmemory.New()

// Add observability via hooks
observed := state.ApplyMiddleware(store,
    state.WithHooks(state.Hooks{
        AfterSet: func(ctx context.Context, key string, value any, err error) {
            log.Printf("State set: %s = %v", key, value)
        },
        OnError: func(ctx context.Context, err error) error {
            log.Printf("State error: %v", err)
            return err
        },
    }),
)

err := observed.Set(ctx, "key", "value")
```

## Limitations

- State is lost when the process exits. Do not use where durability is required.
- Watcher channels have a buffer of 16. Slow consumers may miss notifications.
- No built-in size limits. Storing large values or many keys may consume significant memory.
- The store does not enforce scope-based access control; scoping is by naming convention.

## When to Use

The In-Memory state store is the right choice for:

- **Unit and integration tests** requiring a state backend
- **Development** when iterating on agent coordination logic
- **Single-process deployments** where state does not need to survive restarts
- **Prototyping** multi-agent systems before adding distributed state
