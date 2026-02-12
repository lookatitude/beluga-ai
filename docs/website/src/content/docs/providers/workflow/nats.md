---
title: "NATS Workflow Provider"
description: "Workflow state store with NATS JetStream KV in Beluga AI. Distributed state persistence with low latency for durable execution in Go."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "NATS, JetStream, workflow state, distributed, key-value, durable execution, Go, Beluga AI"
---

The NATS provider implements the `workflow.WorkflowStore` interface using NATS JetStream's Key-Value (KV) store. Each workflow state is stored as a JSON-encoded value in a JetStream KV bucket, with the workflow ID as the key. NATS JetStream provides durable, replicated storage with built-in key enumeration, making it a lightweight alternative to heavier persistence backends.

Choose NATS when you want a lightweight, durable key-value store with native key enumeration for workflow state. Unlike Kafka or Dapr, NATS `List` discovers all workflows in the bucket (including those saved by other instances) without requiring an in-memory index. NATS JetStream is simple to operate and provides replicated storage with low operational overhead. For full orchestration with signals and replay, consider [Temporal](/providers/workflow/temporal).

## Installation

```bash
go get github.com/lookatitude/beluga-ai/workflow/providers/nats
```

Start a NATS server with JetStream enabled:

```bash
docker run -p 4222:4222 nats -js
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/lookatitude/beluga-ai/workflow"
    natsstore "github.com/lookatitude/beluga-ai/workflow/providers/nats"
)

func main() {
    store, err := natsstore.New(natsstore.Config{
        URL:    "nats://localhost:4222",
        Bucket: "workflows",
    })
    if err != nil {
        log.Fatal(err)
    }
    defer store.Close()

    executor := workflow.NewExecutor(
        workflow.WithStore(store),
    )

    ctx := context.Background()

    handle, err := executor.Execute(ctx, myWorkflow, workflow.WorkflowOptions{
        ID:    "data-pipeline-1",
        Input: "start",
    })
    if err != nil {
        log.Fatal(err)
    }

    result, err := handle.Result(ctx)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println("Result:", result)
}

func myWorkflow(ctx workflow.WorkflowContext, input any) (any, error) {
    return fmt.Sprintf("completed: %v", input), nil
}
```

## Configuration

| Parameter | Type | Default | Description |
|---|---|---|---|
| `URL` | `string` | `nats.DefaultURL` | NATS server URL |
| `Bucket` | `string` | `"beluga_workflows"` | JetStream KV bucket name |
| `Conn` | `*nats.Conn` | `nil` | Optional pre-existing NATS connection |

## Pre-existing Connection

When providing an existing NATS connection, the store does not own the connection and will not close it:

```go
import "github.com/nats-io/nats.go"

conn, err := nats.Connect("nats://my-cluster:4222",
    nats.UserInfo("user", "password"),
    nats.MaxReconnects(10),
)
if err != nil {
    log.Fatal(err)
}
defer conn.Close()

store, err := natsstore.New(natsstore.Config{
    Conn:   conn,
    Bucket: "my-workflows",
})
if err != nil {
    log.Fatal(err)
}
// store.Close() is a no-op since we don't own the connection
```

## API Reference

### New

```go
func New(cfg Config) (*Store, error)
```

Creates a new NATS workflow store. If no `Conn` is provided, connects to the URL (defaulting to `nats://localhost:4222`). Creates or updates the JetStream KV bucket automatically.

### Save

```go
func (s *Store) Save(ctx context.Context, state workflow.WorkflowState) error
```

Serializes the workflow state to JSON and stores it in the KV bucket using `Put`. Returns an error if `WorkflowID` is empty or if the KV operation fails.

### Load

```go
func (s *Store) Load(ctx context.Context, workflowID string) (*workflow.WorkflowState, error)
```

Retrieves and deserializes workflow state by key. Returns `nil, nil` if the key is not found.

### List

```go
func (s *Store) List(ctx context.Context, filter workflow.WorkflowFilter) ([]workflow.WorkflowState, error)
```

Enumerates all keys in the KV bucket, loads each workflow state, and applies filtering. Supports filtering by `Status` and limiting results with `Limit`. Returns `nil, nil` if no keys exist in the bucket.

### Delete

```go
func (s *Store) Delete(ctx context.Context, workflowID string) error
```

Removes the workflow state from the KV bucket. Deleting a non-existent key is a no-op (returns `nil`).

### Close

```go
func (s *Store) Close()
```

Closes the underlying NATS connection if the store created it. If a pre-existing `Conn` was provided, `Close` is a no-op.

## Key-Value Bucket

The store uses JetStream's `CreateOrUpdateKeyValue` on initialization to ensure the KV bucket exists. If the bucket already exists, it is reused without modification.

The KV bucket provides:
- Durable, replicated storage across NATS cluster nodes
- Native key enumeration (used by `List`)
- Atomic put and delete operations
- Built-in TTL support (configurable at bucket level)

## Listing Behavior

Unlike providers that maintain an in-memory index, the NATS provider uses native key enumeration via `Keys()`. This means:

- `List` discovers all workflows in the bucket, including those saved by other store instances.
- No in-memory cache is needed; state is always read from NATS.
- `List` performs N+1 operations (one `Keys` call plus one `Get` per key), which may be slower for large numbers of workflows.

## Error Handling

Errors are wrapped with descriptive prefixes. Not-found errors are handled gracefully:

```go
state, err := store.Load(ctx, "wf-123")
if err != nil {
    // Error format: "nats/load: get: <underlying error>"
    log.Printf("Failed to load: %v", err)
}

// state is nil if not found (no error)
if state == nil {
    log.Println("Workflow not found")
}
```

## Resource Cleanup

Close the store to release the NATS connection when finished:

```go
store, err := natsstore.New(cfg)
if err != nil {
    log.Fatal(err)
}
defer store.Close()
```

## Dependencies

This provider depends on the NATS Go client and JetStream library:

- `github.com/nats-io/nats.go`
- `github.com/nats-io/nats.go/jetstream`
