---
title: "LLM API — ChatModel, Router, Middleware"
description: "LLM package API reference for Beluga AI. ChatModel interface, provider registry, composable middleware, hooks, structured output, and routing."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "LLM API, ChatModel, provider registry, middleware, structured output, Router, hooks, Beluga AI, Go, reference"
---

```go
import "github.com/lookatitude/beluga-ai/llm"
```

Package llm provides the LLM abstraction layer for the Beluga AI framework.

It defines the `ChatModel` interface that all LLM providers implement,
a provider registry for dynamic instantiation, composable middleware,
lifecycle hooks, structured output parsing, context window management,
tokenization, rate limiting, and multi-backend routing.

## ChatModel Interface

```go
type ChatModel interface {
    Generate(ctx context.Context, msgs []schema.Message, opts ...GenerateOption) (*schema.AIMessage, error)
    Stream(ctx context.Context, msgs []schema.Message, opts ...GenerateOption) iter.Seq2[schema.StreamChunk, error]
    BindTools(tools []schema.ToolDefinition) ChatModel
    ModelID() string
}
```

- `Generate` sends messages and returns a complete `*schema.AIMessage`.
- `Stream` sends messages and returns an `iter.Seq2[schema.StreamChunk, error]` iterator.
- `BindTools` returns a new `ChatModel` with the given tool definitions included in every request. The original model is not modified.
- `ModelID` returns the underlying model identifier (e.g. `"gpt-4o"`).

## Provider Registry

Providers register themselves via `init()` so that importing a provider
package is sufficient to make it available through the registry. The registry
accepts a `config.ProviderConfig` to configure the provider:

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/lookatitude/beluga-ai/config"
    "github.com/lookatitude/beluga-ai/llm"
    "github.com/lookatitude/beluga-ai/schema"

    _ "github.com/lookatitude/beluga-ai/llm/providers/openai"
)

func main() {
    ctx := context.Background()

    model, err := llm.New("openai", config.ProviderConfig{
        APIKey: "sk-...",
        Model:  "gpt-4o",
    })
    if err != nil {
        log.Fatal(err)
    }

    msgs := []schema.Message{schema.NewHumanMessage("What is 2+2?")}
    resp, err := model.Generate(ctx, msgs)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(resp.Text())
}
```

Use `Register` to add a custom provider factory, `New` to create a `ChatModel`
by name, and `List` to discover all registered providers:

```go
// Register a custom provider (call from init())
llm.Register("my-provider", func(cfg config.ProviderConfig) (llm.ChatModel, error) {
    return NewMyProvider(cfg)
})

providers := llm.List() // sorted provider names
```

## Streaming

Streaming uses `iter.Seq2` (Go 1.23+):

```go
for chunk, err := range model.Stream(ctx, msgs) {
    if err != nil {
        log.Fatal(err)
    }
    fmt.Print(chunk.Delta)
}
```

## Generate Options

`GenerateOption` functional options configure individual `Generate` and `Stream`
calls:

| Option | Type | Description |
|---|---|---|
| `WithTemperature(t float64)` | `GenerateOption` | Sampling temperature (0.0–2.0). |
| `WithMaxTokens(n int)` | `GenerateOption` | Maximum tokens to generate. |
| `WithTopP(p float64)` | `GenerateOption` | Nucleus sampling (0.0–1.0). |
| `WithStopSequences(seqs ...string)` | `GenerateOption` | Stop generation on these strings. |
| `WithResponseFormat(format ResponseFormat)` | `GenerateOption` | Output format (text, json_object, json_schema). |
| `WithToolChoice(choice ToolChoice)` | `GenerateOption` | `ToolChoiceAuto`, `ToolChoiceNone`, `ToolChoiceRequired`. |
| `WithSpecificTool(name string)` | `GenerateOption` | Force the model to call the named tool. |
| `WithMetadata(kv map[string]any)` | `GenerateOption` | Provider-specific options. |
| `WithReasoning(cfg ReasoningConfig)` | `GenerateOption` | Full reasoning configuration. |
| `WithReasoningEffort(effort ReasoningEffort)` | `GenerateOption` | Reasoning effort level. |
| `WithReasoningBudget(tokens int)` | `GenerateOption` | Reasoning token budget. |

## Reasoning Models

Beluga AI supports reasoning/chain-of-thought models such as OpenAI o-series
and Claude with extended thinking. Use `ReasoningConfig` and the associated
functional options to control reasoning behaviour:

```go
type ReasoningEffort string

const (
    ReasoningEffortLow    ReasoningEffort = "low"
    ReasoningEffortMedium ReasoningEffort = "medium"
    ReasoningEffortHigh   ReasoningEffort = "high"
)

type ReasoningConfig struct {
    Effort       ReasoningEffort
    BudgetTokens int
}
```

| Option | Type | Description |
|---|---|---|
| `WithReasoning(cfg ReasoningConfig)` | `GenerateOption` | Set the full reasoning configuration. |
| `WithReasoningEffort(effort ReasoningEffort)` | `GenerateOption` | Set reasoning effort level (creates config if nil). |
| `WithReasoningBudget(tokens int)` | `GenerateOption` | Set reasoning token budget (creates config if nil). |

Example:

```go
resp, err := model.Generate(ctx, msgs,
    llm.WithReasoningEffort(llm.ReasoningEffortHigh),
    llm.WithReasoningBudget(10000),
)
```

Reasoning tokens are tracked in `schema.Usage.ReasoningTokens`, and reasoning
content appears as `schema.ThinkingPart` in the response's content parts.
During streaming, reasoning deltas arrive in `schema.StreamChunk.ReasoningDelta`.

Use the `OnReasoning` hook to observe reasoning deltas as they stream:

```go
hooks := llm.Hooks{
    OnReasoning: func(ctx context.Context, delta string) {
        fmt.Print(delta) // stream reasoning to console
    },
}
```

## Middleware

`Middleware` has the signature `func(ChatModel) ChatModel`. Built-in middleware:

- `WithHooks(hooks Hooks) Middleware` — invokes lifecycle callbacks around Generate and Stream.
- `WithLogging(logger *slog.Logger) Middleware` — logs Generate and Stream calls.
- `WithFallback(fallback ChatModel) Middleware` — falls back to an alternative model on retryable errors.
- `WithProviderLimits(limits ProviderLimits) Middleware` — enforces RPM, TPM, and concurrency limits.

Apply middleware with `ApplyMiddleware`. The first middleware in the list
becomes the outermost wrapper and executes first:

```go
package main

import (
    "log/slog"
    "os"

    "github.com/lookatitude/beluga-ai/llm"
)

func applyMiddleware(model llm.ChatModel, backup llm.ChatModel) llm.ChatModel {
    logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
    return llm.ApplyMiddleware(model,
        llm.WithLogging(logger),
        llm.WithFallback(backup),
        llm.WithProviderLimits(llm.ProviderLimits{
            RPM:           60,
            MaxConcurrent: 5,
        }),
    )
}
```

## Hooks

`Hooks` provides optional callbacks invoked during LLM operations. All fields
are optional; nil hooks are skipped. Use `ComposeHooks` to merge multiple
`Hooks` values:

```go
type Hooks struct {
    BeforeGenerate func(ctx context.Context, msgs []schema.Message) error
    AfterGenerate  func(ctx context.Context, resp *schema.AIMessage, err error)
    OnStream       func(ctx context.Context, chunk schema.StreamChunk)
    OnToolCall     func(ctx context.Context, call schema.ToolCall)
    OnReasoning    func(ctx context.Context, delta string)
    OnError        func(ctx context.Context, err error) error
}
```

`BeforeGenerate` can abort the call by returning an error. `OnError` can
suppress an error by returning nil. Example:

```go
package main

import (
    "context"
    "log"

    "github.com/lookatitude/beluga-ai/llm"
    "github.com/lookatitude/beluga-ai/schema"
)

func loggingHooks() llm.Hooks {
    return llm.Hooks{
        BeforeGenerate: func(ctx context.Context, msgs []schema.Message) error {
            log.Printf("generating with %d messages", len(msgs))
            return nil
        },
        OnError: func(ctx context.Context, err error) error {
            log.Printf("llm error: %v", err)
            return err
        },
    }
}
```

## Structured Output

`StructuredOutput[T]` wraps a `ChatModel` to produce typed Go values. It
generates a JSON Schema from `T`, instructs the model to respond in JSON,
parses the response, and retries on parse failures (default: 2 retries):

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/lookatitude/beluga-ai/llm"
    "github.com/lookatitude/beluga-ai/schema"
)

type Sentiment struct {
    Label string  `json:"label"`
    Score float64 `json:"score"`
}

func analyzeSentiment(ctx context.Context, model llm.ChatModel, text string) (Sentiment, error) {
    so := llm.NewStructured[Sentiment](model, llm.WithMaxRetries(3))
    msgs := []schema.Message{
        schema.NewHumanMessage("Analyze the sentiment of: " + text),
    }
    return so.Generate(ctx, msgs)
}

func main() {
    ctx := context.Background()
    // model := ... (create a ChatModel)
    result, err := analyzeSentiment(ctx, model, "I love this product!")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Label: %s, Score: %.2f\n", result.Label, result.Score)
}
```

`NewStructured[T]` accepts optional `StructuredOption` values. The only
built-in option is `WithMaxRetries(n int)`.

## Context Management

`ContextManager` fits a message sequence within a token budget. Two strategies
are built in: `"truncate"` (drops oldest non-system messages) and `"sliding"`
(keeps the most recent messages that fit):

```go
package main

import (
    "context"
    "log"

    "github.com/lookatitude/beluga-ai/llm"
    "github.com/lookatitude/beluga-ai/schema"
)

func fitMessages(ctx context.Context, msgs []schema.Message) ([]schema.Message, error) {
    cm := llm.NewContextManager(
        llm.WithContextStrategy("sliding"),
        llm.WithKeepSystemMessages(true),
    )
    fitted, err := cm.Fit(ctx, msgs, 4096)
    if err != nil {
        return nil, err
    }
    return fitted, nil
}
```

`NewContextManager` options:

| Option | Default | Description |
|---|---|---|
| `WithContextStrategy(name string)` | `"truncate"` | Strategy: `"truncate"` or `"sliding"`. |
| `WithTokenizer(t Tokenizer)` | `SimpleTokenizer` | Tokenizer for counting tokens. |
| `WithKeepSystemMessages(keep bool)` | `true` | Never remove system messages. |

`ContextManager` is an interface with a single method:

```go
type ContextManager interface {
    Fit(ctx context.Context, msgs []schema.Message, budget int) ([]schema.Message, error)
}
```

## Tokenizer

`Tokenizer` provides token counting. `SimpleTokenizer` is a built-in
approximation (1 token per 4 characters) suitable when a model-specific
tokenizer is unavailable.

## Routing

`Router` implements `ChatModel` by delegating to one of several backend models
chosen by a pluggable `ModelSelector`. Built-in strategies:

- `RoundRobin` — selects models in round-robin order.
- `FailoverChain` — always returns the first model (use `FailoverRouter` for actual failover).

`FailoverRouter` retries across models on retryable errors:

```go
package main

import (
    "log"

    "github.com/lookatitude/beluga-ai/llm"
)

func makeRouter(modelA, modelB llm.ChatModel) llm.ChatModel {
    return llm.NewRouter(
        llm.WithModels(modelA, modelB),
        llm.WithStrategy(&llm.RoundRobin{}),
    )
}

func makeFailover(primary, backup llm.ChatModel) llm.ChatModel {
    return llm.NewFailoverRouter(primary, backup)
}
```

`NewRouter` returns a `*Router`. If no strategy is set, `RoundRobin` is used.
`NewFailoverRouter` returns a `*FailoverRouter` that tries models in order,
falling back on retryable errors.

## Rate Limiting

`WithProviderLimits` returns middleware that enforces per-provider limits:

```go
type ProviderLimits struct {
    RPM             int           // requests per minute
    TPM             int           // tokens per minute
    MaxConcurrent   int           // max concurrent requests
    CooldownOnRetry time.Duration // wait before retry after hitting limit
}
```

## Related

- [`core`](/docs/reference/api/foundation/core) — Runnable, BatchInvoke, errors
- [`agent`](/docs/reference/api/llm-agents/agent) — Agent runtime using ChatModel
- [`tool`](/docs/reference/api/llm-agents/tool) — Tool interface and BindTools
- `docs/providers.md` — Full provider list and extension guide
