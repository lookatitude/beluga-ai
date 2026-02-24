---
title: "Kafka Workflow Provider"
description: "Workflow state store with Kafka compacted topics in Beluga AI. Event-driven state persistence with high throughput for durable execution in Go."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Kafka, workflow state, event-driven, compacted topics, durable execution, state persistence, Go, Beluga AI"
---

The Kafka provider implements the `workflow.WorkflowStore` interface using Kafka compacted topics. Workflow state is serialized to JSON and written as messages to a Kafka topic, with the workflow ID as the message key. Kafka's log compaction ensures the latest state for each workflow is retained indefinitely. This provider is well-suited for event-driven architectures where Kafka is already part of the infrastructure.

Choose Kafka when Kafka is already part of your infrastructure and you want workflow state persistence that integrates naturally with your event-driven architecture. Compacted topics provide durable, indefinite state retention with built-in replication. For native key enumeration and simpler operations, consider [NATS](/docs/providers/workflow/nats). For full orchestration, consider [Temporal](/docs/providers/workflow/temporal).

## Installation

```bash
go get github.com/lookatitude/beluga-ai/workflow/providers/kafka
```

Ensure a Kafka cluster is available. For local development:

```bash
docker run -p 9092:9092 apache/kafka
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/lookatitude/beluga-ai/workflow"
    "github.com/lookatitude/beluga-ai/workflow/providers/kafka"
)

func main() {
    store, err := kafka.New(kafka.Config{
        Brokers: []string{"localhost:9092"},
        Topic:   "beluga-workflows",
        Writer:  kafkaWriter, // your kafka.Writer implementation
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
        ID:    "pipeline-001",
        Input: "start processing",
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
| `Brokers` | `[]string` | (none) | Kafka broker addresses |
| `Topic` | `string` | `"beluga-workflows"` | Kafka topic for workflow state |
| `Writer` | `kafka.Writer` | (required) | Kafka writer for producing messages |
| `Reader` | `kafka.Reader` | `nil` | Optional Kafka reader for consuming state |

## Writer and Reader Interfaces

The provider defines its own `Writer` and `Reader` interfaces to decouple from specific Kafka client libraries:

```go
type Writer interface {
    WriteMessages(ctx context.Context, msgs ...Message) error
    Close() error
}

type Reader interface {
    ReadMessage(ctx context.Context) (Message, error)
    Close() error
}

type Message struct {
    Key   []byte
    Value []byte
}
```

These interfaces are compatible with popular Go Kafka clients such as `segmentio/kafka-go`.

## Direct Construction

Use `NewWithWriterReader` for testing with mock implementations:

```go
store := kafka.NewWithWriterReader(mockWriter, mockReader)
```

## API Reference

### Save

```go
func (s *Store) Save(ctx context.Context, state workflow.WorkflowState) error
```

Serializes the workflow state to JSON and writes it to the Kafka topic with the workflow ID as the message key. The state is also cached in-memory. Returns an error if `WorkflowID` is empty or if the write fails.

### Load

```go
func (s *Store) Load(_ context.Context, workflowID string) (*workflow.WorkflowState, error)
```

Retrieves workflow state from the in-memory cache by ID. Returns `nil, nil` if the workflow ID is not found. This is a read-lock-only operation with no Kafka round-trip.

### List

```go
func (s *Store) List(_ context.Context, filter workflow.WorkflowFilter) ([]workflow.WorkflowState, error)
```

Returns workflows from the in-memory cache matching the filter. Supports filtering by `Status` and limiting results with `Limit`.

### Delete

```go
func (s *Store) Delete(ctx context.Context, workflowID string) error
```

Writes a tombstone message (nil value) to the Kafka topic for the given workflow ID. Kafka log compaction removes the workflow state after the tombstone is processed. The in-memory cache entry is removed immediately.

### Close

```go
func (s *Store) Close() error
```

Closes the underlying Kafka writer and reader. Always call `Close` when the store is no longer needed to release Kafka connections.

## Log Compaction

This provider is designed for use with Kafka's log compaction feature. Configure the topic with:

```
cleanup.policy=compact
```

With compaction enabled:
- The latest state for each workflow ID is retained indefinitely.
- Tombstone messages (produced by `Delete`) trigger removal during compaction.
- Old state versions are removed by the Kafka broker during log cleaning.

## In-Memory Cache

The store maintains an in-memory cache of all workflow states. This means:

- `Load` and `List` operations do not require Kafka round-trips.
- The cache is populated from `Save` calls. On process restart, the cache is empty until workflows are saved again.
- For cache recovery on startup, configure a `Reader` and consume the compacted topic to rebuild state.

## Error Handling

All Kafka errors are wrapped with a descriptive prefix:

```go
err := store.Save(ctx, state)
if err != nil {
    // Error format: "kafka/save: write: <underlying error>"
    log.Printf("Failed to save: %v", err)
}
```

## Resource Cleanup

Always defer `Close` to release Kafka connections:

```go
store, err := kafka.New(cfg)
if err != nil {
    log.Fatal(err)
}
defer store.Close()
```
