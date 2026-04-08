---
title: "Core API — Streams, Runnable, Errors"
description: "Core package API reference for Beluga AI. Runnable interface, event streaming with iter.Seq2, batch processing, lifecycle management, and structured errors."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "core API, Runnable, Stream, iter.Seq2, lifecycle, errors, batch processing, Beluga AI, Go, reference"
---

```go
import "github.com/lookatitude/beluga-ai/core"
```

Package core provides the foundational primitives for the Beluga AI framework.

It defines the universal execution model, typed event streaming, batch
processing, lifecycle management, multi-tenancy, context propagation,
functional options, and structured error handling that all other packages
build upon.

## Runnable Interface

`Runnable` is the universal execution interface. Every component that processes
input — LLMs, tools, agents, pipelines — implements `Runnable`. It supports
both synchronous invocation and streaming:

```go
type Runnable interface {
    Invoke(ctx context.Context, input any, opts ...Option) (any, error)
    Stream(ctx context.Context, input any, opts ...Option) iter.Seq2[any, error]
}
```

Runnables compose via `Pipe` (sequential) and `Parallel` (concurrent):

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/lookatitude/beluga-ai/core"
)

func main() {
    ctx := context.Background()

    pipeline := core.Pipe(tokenizer, llmRunnable)
    result, err := pipeline.Invoke(ctx, "Hello, world!")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(result)

    fanOut := core.Parallel(agent1, agent2, agent3)
    results, err := fanOut.Invoke(ctx, input)
    if err != nil {
        log.Fatal(err)
    }
    // results is []any with one entry per Runnable
    fmt.Println(results)
}
```

`Pipe` invokes `a` synchronously, then passes its output as input to `b`.
`Parallel` fans out to all Runnables concurrently and returns all results as
`[]any`. If any Runnable returns an error, the first error is returned and
remaining results may be incomplete.

## Streaming

`Stream` is a pull-based iterator built on Go 1.23+ `iter.Seq2`. Consumers
range over the returned sequence:

```go
stream := r.Stream(ctx, input)
for val, err := range stream {
    if err != nil {
        break
    }
    fmt.Println(val)
}
```

## Batch Processing

`BatchInvoke` executes a function over multiple inputs concurrently. It is
generic over the input type `I` and output type `O`, and returns a
`[]BatchResult[O]` with one entry per input at the corresponding index:

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/lookatitude/beluga-ai/core"
)

func embed(ctx context.Context, text string) ([]float32, error) {
    // call embedding model
    return nil, nil
}

func main() {
    ctx := context.Background()
    docs := []string{"doc one", "doc two", "doc three"}

    results := core.BatchInvoke(ctx, embed, docs, core.BatchOptions{
        MaxConcurrency: 10,
        Timeout:        5 * time.Second,
    })

    for i, r := range results {
        if r.Err != nil {
            log.Printf("item %d failed: %v", i, r.Err)
            continue
        }
        fmt.Printf("item %d: %d dimensions\n", i, len(r.Value))
    }
}
```

`BatchOptions` fields:

| Field | Type | Description |
|---|---|---|
| `MaxConcurrency` | `int` | Max concurrent executions. 0 = unlimited. |
| `BatchSize` | `int` | Informational; BatchInvoke calls fn once per item. |
| `Timeout` | `time.Duration` | Per-item timeout. 0 = no per-item timeout. |
| `RetryPolicy` | `*RetryPolicy` | Optional retry configuration per item. |

`BatchResult[O]` carries `Value O` and `Err error` for each input.

## Lifecycle Management

The `Lifecycle` interface provides `Start`/`Stop`/`Health` semantics for
components that require explicit initialization and graceful shutdown.
`App` manages a set of `Lifecycle` components, starting them in registration
order and stopping them in reverse:

```go
app := core.NewApp()
app.Register(dbPool, cacheLayer, httpServer)
if err := app.Start(ctx); err != nil {
    log.Fatal(err)
}
defer app.Shutdown(ctx)
```

## Multi-Tenancy

`WithTenant` and `GetTenant` store and retrieve a `TenantID` from context,
enabling tenant-scoped data isolation across all framework operations.
`TenantID` is a named string type:

```go
package main

import (
    "context"
    "fmt"

    "github.com/lookatitude/beluga-ai/core"
)

func main() {
    ctx := core.WithTenant(context.Background(), core.TenantID("acme-corp"))
    tenantID := core.GetTenant(ctx)
    fmt.Println(tenantID) // "acme-corp"
}
```

## Context Helpers

`WithSessionID`, `GetSessionID`, `WithRequestID`, and `GetRequestID`
propagate session and request identifiers through context for correlation
across distributed traces and logs:

```go
ctx = core.WithSessionID(ctx, "session-abc")
ctx = core.WithRequestID(ctx, "req-xyz")

sessionID := core.GetSessionID(ctx) // "session-abc"
requestID := core.GetRequestID(ctx) // "req-xyz"
```

## Structured Errors

`Error` carries an operation name, `ErrorCode`, human-readable message,
and optional wrapped cause:

```go
type Error struct {
    Op      string
    Code    ErrorCode
    Message string
    Err     error
}
```

Create errors with `NewError`:

```go
err := core.NewError("llm.generate", core.ErrRateLimit, "quota exceeded", cause)
```

Error codes and their retryability:

| Code | Constant | Retryable |
|---|---|---|
| `rate_limit` | `ErrRateLimit` | yes |
| `timeout` | `ErrTimeout` | yes |
| `provider_unavailable` | `ErrProviderDown` | yes |
| `auth_error` | `ErrAuth` | no |
| `invalid_input` | `ErrInvalidInput` | no |
| `tool_failed` | `ErrToolFailed` | no |
| `guard_blocked` | `ErrGuardBlocked` | no |
| `budget_exhausted` | `ErrBudgetExhausted` | no |
| `not_found` | `ErrNotFound` | no |

Use `IsRetryable` to decide whether to retry:

```go
if core.IsRetryable(err) {
    // safe to retry
}
```

`Error` implements `errors.Is` by matching on `Code`, and `errors.As` via
`Unwrap`, so standard Go error chain traversal works as expected.

## Functional Options

The `Option` interface and `OptionFunc` adapter implement the functional
options pattern used throughout the framework for configuration.

## Related

- `docs/concepts.md` — Design decisions
- `docs/packages.md` — Package layout
- [`llm`](/docs/api-reference/llm-agents/llm) — ChatModel built on Runnable
- [`agent`](/docs/api-reference/llm-agents/agent) — Agent built on Runnable
