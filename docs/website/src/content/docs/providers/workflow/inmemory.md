---
title: In-Memory
description: In-process workflow state store for development and testing.
---

The In-Memory provider implements the `workflow.WorkflowStore` interface using a Go map protected by a read-write mutex. It stores workflow state entirely in process memory, making it ideal for development, testing, and prototyping. State does not persist across process restarts.

## Installation

```bash
go get github.com/lookatitude/beluga-ai/workflow/providers/inmemory
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/lookatitude/beluga-ai/workflow"
    "github.com/lookatitude/beluga-ai/workflow/providers/inmemory"
)

func main() {
    store := inmemory.New()

    executor := workflow.NewExecutor(
        workflow.WithStore(store),
    )

    ctx := context.Background()

    handle, err := executor.Execute(ctx, myWorkflow, workflow.WorkflowOptions{
        ID:    "test-wf-1",
        Input: "hello",
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
    return fmt.Sprintf("processed: %v", input), nil
}
```

## Configuration

The In-Memory store requires no configuration. Call `inmemory.New()` to create a ready-to-use instance.

## API Reference

### New

```go
func New() *Store
```

Creates a new in-memory workflow store with an empty state map.

### Save

```go
func (s *Store) Save(_ context.Context, state workflow.WorkflowState) error
```

Persists the workflow state in the in-memory map. Returns an error if `WorkflowID` is empty. Overwrites any existing state for the same workflow ID.

### Load

```go
func (s *Store) Load(_ context.Context, workflowID string) (*workflow.WorkflowState, error)
```

Retrieves workflow state by ID. Returns `nil, nil` if the workflow ID is not found.

### List

```go
func (s *Store) List(_ context.Context, filter workflow.WorkflowFilter) ([]workflow.WorkflowState, error)
```

Returns all workflows matching the filter criteria. Supports filtering by `Status` and limiting results with `Limit`.

### Delete

```go
func (s *Store) Delete(_ context.Context, workflowID string) error
```

Removes the workflow state for the given ID. Deleting a non-existent ID is a no-op.

## Concurrency

All operations are protected by a `sync.RWMutex`. Multiple goroutines can safely read and write to the store concurrently. Read operations (`Load`, `List`) acquire a read lock, while write operations (`Save`, `Delete`) acquire an exclusive write lock.

## Listing and Filtering

```go
// List all running workflows
running, err := store.List(ctx, workflow.WorkflowFilter{
    Status: workflow.StatusRunning,
})
if err != nil {
    log.Fatal(err)
}

// List up to 10 completed workflows
completed, err := store.List(ctx, workflow.WorkflowFilter{
    Status: workflow.StatusCompleted,
    Limit:  10,
})
if err != nil {
    log.Fatal(err)
}

// List all workflows (no filter)
all, err := store.List(ctx, workflow.WorkflowFilter{})
if err != nil {
    log.Fatal(err)
}
```

## Limitations

- State is lost when the process exits. Do not use in production where durability is required.
- Listing returns workflows in non-deterministic map iteration order.
- No built-in size limits. Long-running processes with many workflows may consume significant memory.

## When to Use

The In-Memory store is the right choice for:

- **Unit and integration tests** where workflow execution needs a state backend
- **Development** when iterating on workflow logic without external dependencies
- **Prototyping** new workflows before selecting a production store
- **CI pipelines** where no external services are available
