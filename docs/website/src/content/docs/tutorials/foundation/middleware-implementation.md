---
title: Middleware Implementation Tutorial
description: "Build composable middleware for LLM calls in Go — logging, validation, and error recovery using Beluga AI's ChatModel wrapping pattern with ApplyMiddleware."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Beluga AI, Go, tutorial, middleware, ChatModel, composable, logging, error recovery, wrapping"
---

Middleware in Beluga AI wraps a `ChatModel` to add cross-cutting behavior — logging, validation, error handling, rate limiting — without modifying the underlying model. The pattern uses the `func(ChatModel) ChatModel` signature, which makes middleware composable: each layer receives the next handler and returns a new handler that includes the additional behavior. This is the same decorator pattern used in HTTP middleware (`func(http.Handler) http.Handler`), adapted for LLM interactions.

## What You Will Build

Three middleware components — logging, validation, and error recovery — composed into a single pipeline that wraps any `ChatModel` implementation.

## Prerequisites

- Understanding of the [Runnable interface](/tutorials/foundation/custom-runnable)
- Familiarity with Go interfaces and function types

## The Middleware Pattern

In Beluga AI v2, middleware is a function that takes a `ChatModel` and returns a new `ChatModel`:

```go
// Middleware wraps a ChatModel to add cross-cutting behaviour.
type Middleware func(ChatModel) ChatModel
```

Middleware is applied with `ApplyMiddleware`, which wraps the model in reverse order so that the first middleware in the list executes first (outermost). The right-to-left application order is necessary because wrapping is additive — wrapping model `M` with middleware `A` then `B` produces `B(A(M))`, where `B` is outermost. By iterating right-to-left, the first middleware listed becomes the outermost wrapper.

```go
model = llm.ApplyMiddleware(model,
    loggingMiddleware,    // executes first (outermost)
    validationMiddleware, // executes second
    retryMiddleware,      // executes last (innermost)
)
```

## Step 1: Logging Middleware

Create middleware that logs `Generate` and `Stream` calls using Go's standard `slog` logger. The middleware struct holds a reference to the `next` model in the chain and the logger instance. Each method calls the next model's corresponding method, wrapping it with logging before and after. This demonstrates the core middleware pattern: intercept, delegate, observe.

```go
package main

import (
    "context"
    "iter"
    "log/slog"
    "time"

    "github.com/lookatitude/beluga-ai/llm"
    "github.com/lookatitude/beluga-ai/schema"
)

func LoggingMiddleware(logger *slog.Logger) llm.Middleware {
    return func(next llm.ChatModel) llm.ChatModel {
        return &loggingModel{next: next, logger: logger}
    }
}

type loggingModel struct {
    next   llm.ChatModel
    logger *slog.Logger
}

func (m *loggingModel) Generate(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) (*schema.AIMessage, error) {
    start := time.Now()
    m.logger.InfoContext(ctx, "generate.start",
        "model", m.next.ModelID(),
        "messages", len(msgs),
    )

    resp, err := m.next.Generate(ctx, msgs, opts...)

    duration := time.Since(start)
    if err != nil {
        m.logger.ErrorContext(ctx, "generate.error",
            "model", m.next.ModelID(),
            "duration", duration,
            "error", err,
        )
        return nil, err
    }

    m.logger.InfoContext(ctx, "generate.done",
        "model", m.next.ModelID(),
        "duration", duration,
        "input_tokens", resp.Usage.InputTokens,
        "output_tokens", resp.Usage.OutputTokens,
    )
    return resp, nil
}

func (m *loggingModel) Stream(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) iter.Seq2[schema.StreamChunk, error] {
    m.logger.InfoContext(ctx, "stream.start", "model", m.next.ModelID())
    return m.next.Stream(ctx, msgs, opts...)
}

func (m *loggingModel) BindTools(tools []schema.ToolDefinition) llm.ChatModel {
    return &loggingModel{next: m.next.BindTools(tools), logger: m.logger}
}

func (m *loggingModel) ModelID() string { return m.next.ModelID() }
```

Beluga AI also provides a built-in `llm.WithLogging(logger)` middleware that follows this same pattern.

## Step 2: Validation Middleware

Middleware that rejects empty message lists before they reach the provider, saving API calls and cost. Validation middleware belongs at the outermost layer because rejecting invalid input early avoids unnecessary work in inner layers (logging, retrying, etc.). The `Stream` method returns a single-element error iterator rather than calling the underlying stream, since there is no valid stream to produce.

```go
func ValidationMiddleware() llm.Middleware {
    return func(next llm.ChatModel) llm.ChatModel {
        return &validationModel{next: next}
    }
}

type validationModel struct {
    next llm.ChatModel
}

func (m *validationModel) Generate(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) (*schema.AIMessage, error) {
    if len(msgs) == 0 {
        return nil, fmt.Errorf("validation: message list cannot be empty")
    }
    return m.next.Generate(ctx, msgs, opts...)
}

func (m *validationModel) Stream(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) iter.Seq2[schema.StreamChunk, error] {
    if len(msgs) == 0 {
        return func(yield func(schema.StreamChunk, error) bool) {
            yield(schema.StreamChunk{}, fmt.Errorf("validation: message list cannot be empty"))
        }
    }
    return m.next.Stream(ctx, msgs, opts...)
}

func (m *validationModel) BindTools(tools []schema.ToolDefinition) llm.ChatModel {
    return &validationModel{next: m.next.BindTools(tools)}
}

func (m *validationModel) ModelID() string { return m.next.ModelID() }
```

## Step 3: Applying Middleware

Compose multiple middleware layers using `llm.ApplyMiddleware`. The order matters: `ApplyMiddleware` applies right-to-left so the first middleware in the list becomes the outermost wrapper. In the example below, validation runs first (rejecting bad input before it reaches the logger), then logging wraps the validated call.

```go
import "log/slog"

func main() {
    logger := slog.Default()

    // Create a base model from the registry
    base, err := llm.New("openai", cfg)
    if err != nil {
        slog.Error("failed to create model", "error", err)
        return
    }

    // Apply middleware — first in list is outermost (executes first)
    model := llm.ApplyMiddleware(base,
        ValidationMiddleware(),
        LoggingMiddleware(logger),
    )

    // Use the wrapped model normally
    ctx := context.Background()
    msgs := []schema.Message{
        schema.NewHumanMessage("What is Go?"),
    }

    resp, err := model.Generate(ctx, msgs)
    if err != nil {
        slog.Error("generate failed", "error", err)
        return
    }
    fmt.Println(resp.Text())
}
```

## Built-in Middleware

Beluga AI v2 ships with several middleware out of the box:

| Middleware | Description |
|:---|:---|
| `llm.WithLogging(logger)` | Logs Generate/Stream calls with slog |
| `llm.WithHooks(hooks)` | Invokes lifecycle hooks (BeforeGenerate, AfterGenerate, OnError, OnToolCall, OnStream) |
| `llm.WithFallback(backup)` | Falls back to another model on retryable errors |

## Using Hooks Middleware

The hooks middleware provides fine-grained lifecycle callbacks without requiring a full middleware implementation. Hooks are optional function fields — any `nil` hook is simply skipped, which eliminates the need for no-op implementations. The `OnError` hook can return `nil` to suppress an error, or return a different error to transform it.

```go
hooks := llm.Hooks{
    BeforeGenerate: func(ctx context.Context, msgs []schema.Message) error {
        slog.InfoContext(ctx, "about to generate", "count", len(msgs))
        return nil
    },
    AfterGenerate: func(ctx context.Context, resp *schema.AIMessage, err error) {
        if resp != nil {
            slog.InfoContext(ctx, "generated",
                "tokens", resp.Usage.TotalTokens,
            )
        }
    },
    OnError: func(ctx context.Context, err error) error {
        slog.ErrorContext(ctx, "model error", "error", err)
        return err // return nil to suppress the error
    },
}

model = llm.ApplyMiddleware(base, llm.WithHooks(hooks))
```

Compose multiple hooks with `llm.ComposeHooks`. The composed hooks execute in order — each `BeforeGenerate` runs in sequence, and `OnError` short-circuits if any hook returns a non-nil error.

```go
combined := llm.ComposeHooks(loggingHooks, metricsHooks, auditHooks)
model = llm.ApplyMiddleware(base, llm.WithHooks(combined))
```

## Middleware Execution Order

`ApplyMiddleware` applies middleware right-to-left so the first middleware in the list is outermost:

```go
// Execution order: Validation -> Logging -> Base Model
model = llm.ApplyMiddleware(base,
    ValidationMiddleware(),  // outermost — executes first
    LoggingMiddleware(logger), // innermost — executes second
)
```

## Next Steps

- [Custom Runnable](/tutorials/foundation/custom-runnable) — Build composable components
- [OpenTelemetry Tracing](/tutorials/foundation/otel-tracing) — Distributed tracing for LLM calls
