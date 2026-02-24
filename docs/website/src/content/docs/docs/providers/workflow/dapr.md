---
title: "Dapr Workflow Provider"
description: "Workflow state store with Dapr state management in Beluga AI. Cloud-native state persistence with pluggable backends for durable execution in Go."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Dapr, workflow state, cloud-native, state management, durable execution, microservices, Go, Beluga AI"
---

The Dapr provider implements the `workflow.WorkflowStore` interface using Dapr's state management API. It persists workflow state as JSON through Dapr's pluggable state store abstraction, enabling deployment across any Dapr-supported backend (Redis, PostgreSQL, CosmosDB, DynamoDB, and others) without code changes.

Choose Dapr when you are already running a Dapr sidecar and want to reuse your existing state store configuration for workflow persistence. Dapr's pluggable architecture means you can switch between Redis, PostgreSQL, CosmosDB, or DynamoDB without code changes. For full workflow orchestration with replay and signals, consider [Temporal](/providers/workflow/temporal). For lightweight key-value persistence, consider [NATS](/providers/workflow/nats).

## Installation

```bash
go get github.com/lookatitude/beluga-ai/workflow/providers/dapr
```

Ensure a Dapr sidecar is running with a configured state store component.

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/lookatitude/beluga-ai/workflow"
    "github.com/lookatitude/beluga-ai/workflow/providers/dapr"
)

func main() {
    store, err := dapr.New(dapr.Config{
        Client:    daprClient, // your Dapr StateClient implementation
        StoreName: "statestore",
    })
    if err != nil {
        log.Fatal(err)
    }

    executor := workflow.NewExecutor(
        workflow.WithStore(store),
    )

    ctx := context.Background()

    handle, err := executor.Execute(ctx, myWorkflow, workflow.WorkflowOptions{
        ID:    "order-789",
        Input: "process order",
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
| `Client` | `dapr.StateClient` | (required) | Dapr state client for state operations |
| `StoreName` | `string` | `"statestore"` | Dapr state store component name |

## StateClient Interface

The provider defines a `StateClient` interface matching the subset of the Dapr SDK used for state operations:

```go
type StateClient interface {
    SaveState(ctx context.Context, storeName, key string, data []byte, meta map[string]string, so ...any) error
    GetState(ctx context.Context, storeName, key string, meta map[string]string) (*StateItem, error)
    DeleteState(ctx context.Context, storeName, key string, meta map[string]string) error
}
```

The `StateItem` type returned by `GetState`:

```go
type StateItem struct {
    Key   string
    Value []byte
    Etag  string
}
```

## Direct Construction

Use `New` for validated construction with a `Config` struct:

```go
store, err := dapr.New(dapr.Config{
    Client:    daprClient,
    StoreName: "my-state-store",
})
if err != nil {
    log.Fatal(err)
}
```

Use `NewWithClient` for testing with mock implementations:

```go
store := dapr.NewWithClient(mockClient, "statestore")
```

## API Reference

### Save

```go
func (s *Store) Save(ctx context.Context, state workflow.WorkflowState) error
```

Serializes the workflow state to JSON and persists it via `SaveState`. The workflow ID serves as the state key. Returns an error if `WorkflowID` is empty or if the Dapr call fails.

### Load

```go
func (s *Store) Load(ctx context.Context, workflowID string) (*workflow.WorkflowState, error)
```

Retrieves and deserializes workflow state by ID. Returns `nil, nil` if the key does not exist (empty value from Dapr).

### List

```go
func (s *Store) List(ctx context.Context, filter workflow.WorkflowFilter) ([]workflow.WorkflowState, error)
```

Returns workflows matching the filter. The store maintains an in-memory index of known workflow IDs and loads each from Dapr individually. Supports filtering by `Status` and limiting results with `Limit`.

### Delete

```go
func (s *Store) Delete(ctx context.Context, workflowID string) error
```

Removes the workflow state from the Dapr state store and the in-memory index.

## Listing Behavior

The Dapr state management API does not natively support listing all keys. The provider maintains an in-memory index of workflow IDs that have been saved through this store instance. This means:

- Workflows saved through a different instance or directly via the Dapr API are not discoverable through `List`.
- The index is rebuilt from scratch on process restart.
- For comprehensive listing, consider using a provider with native key enumeration (such as NATS or Kafka).

## Error Handling

All Dapr API errors are wrapped with a descriptive prefix for traceability:

```go
state, err := store.Load(ctx, "wf-123")
if err != nil {
    // Error format: "dapr/load: <underlying error>"
    log.Printf("Failed to load workflow: %v", err)
}
```

## Supported Dapr State Stores

Any Dapr state store component works with this provider. Common choices include:

| Dapr Component | Backend |
|---|---|
| `state.redis` | Redis |
| `state.postgresql` | PostgreSQL |
| `state.cosmosdb` | Azure Cosmos DB |
| `state.dynamodb` | AWS DynamoDB |
| `state.mongodb` | MongoDB |
| `state.in-memory` | In-process (development) |

Configure the state store component in your Dapr configuration (typically `components/statestore.yaml`).
