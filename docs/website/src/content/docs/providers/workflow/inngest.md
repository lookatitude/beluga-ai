---
title: Inngest
description: Workflow state store backed by Inngest's event-driven durable execution platform.
---

The Inngest provider implements the `workflow.WorkflowStore` interface using Inngest's HTTP API for persisting workflow state. Inngest is an event-driven durable execution platform that provides reliable state management, retries, and step functions. This provider stores and retrieves workflow state through the Inngest REST API.

## Installation

```bash
go get github.com/lookatitude/beluga-ai/workflow/providers/inngest
```

Start the Inngest dev server for local development:

```bash
npx inngest-cli@latest dev
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/lookatitude/beluga-ai/workflow"
    "github.com/lookatitude/beluga-ai/workflow/providers/inngest"
)

func main() {
    store, err := inngest.New(inngest.Config{
        BaseURL:  "http://localhost:8288",
        EventKey: "my-event-key",
    })
    if err != nil {
        log.Fatal(err)
    }

    executor := workflow.NewExecutor(
        workflow.WithStore(store),
    )

    ctx := context.Background()

    handle, err := executor.Execute(ctx, myWorkflow, workflow.WorkflowOptions{
        ID:    "order-process-1",
        Input: "process payment",
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
| `BaseURL` | `string` | `"http://localhost:8288"` | Inngest API base URL |
| `EventKey` | `string` | `""` | Event key for API authentication |
| `Client` | `inngest.HTTPClient` | `http.DefaultClient` | Custom HTTP client |

## HTTPClient Interface

The provider accepts a custom HTTP client for advanced connection configuration or testing:

```go
type HTTPClient interface {
    Do(req *http.Request) (*http.Response, error)
}
```

## Custom HTTP Client

```go
store, err := inngest.New(inngest.Config{
    BaseURL: "https://inngest.example.com",
    EventKey: os.Getenv("INNGEST_EVENT_KEY"),
    Client: &http.Client{
        Timeout: 30 * time.Second,
        Transport: &http.Transport{
            MaxIdleConns: 10,
        },
    },
})
if err != nil {
    log.Fatal(err)
}
```

## API Reference

### Save

```go
func (s *Store) Save(ctx context.Context, state workflow.WorkflowState) error
```

Serializes workflow state to JSON and sends it to the Inngest API via `PUT /v1/workflows/{workflowID}`. The state is also cached in-memory for listing operations. Returns an error if `WorkflowID` is empty or if the API returns a 4xx/5xx status.

### Load

```go
func (s *Store) Load(ctx context.Context, workflowID string) (*workflow.WorkflowState, error)
```

Retrieves workflow state from the Inngest API via `GET /v1/workflows/{workflowID}`. Returns `nil, nil` if the workflow is not found (404 response).

### List

```go
func (s *Store) List(_ context.Context, filter workflow.WorkflowFilter) ([]workflow.WorkflowState, error)
```

Returns workflows from the in-memory cache. Since the Inngest API does not natively support listing, this returns only workflows that have been saved through this store instance. Supports filtering by `Status` and limiting results with `Limit`.

### Delete

```go
func (s *Store) Delete(ctx context.Context, workflowID string) error
```

Removes workflow state via `DELETE /v1/workflows/{workflowID}` and clears the in-memory cache entry. Returns successfully if the workflow was not found (404 is treated as success).

## Authentication

When an `EventKey` is configured, it is sent as a Bearer token in the `Authorization` header on all API requests:

```
Authorization: Bearer <event-key>
```

For production Inngest deployments, set the event key via environment variable:

```go
store, err := inngest.New(inngest.Config{
    BaseURL:  "https://api.inngest.com",
    EventKey: os.Getenv("INNGEST_EVENT_KEY"),
})
```

## Error Handling

HTTP errors are reported with the status code for troubleshooting:

```go
err := store.Save(ctx, state)
if err != nil {
    // Error format: "inngest/save: status 500"
    log.Printf("Failed to save: %v", err)
}
```

## Listing Behavior

The `List` method returns workflows from an in-memory cache rather than querying the Inngest API. This means:

- Only workflows saved through this store instance are listed.
- The cache resets on process restart.
- `Load` always queries the Inngest API directly, providing accurate point-in-time state.
