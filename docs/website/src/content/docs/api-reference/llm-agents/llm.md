---
title: "LLM Package"
description: "ChatModel interface, provider registry, middleware, hooks, structured output, routing"
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

The core abstraction is `ChatModel`, which every provider implements:

- Generate sends messages and returns a complete [schema.AIMessage].
- Stream sends messages and returns an [iter.Seq2] of [schema.StreamChunk] values.
- BindTools returns a new ChatModel with the given tool definitions included in every request.
- ModelID returns the underlying model identifier (e.g. "gpt-4o").

## Provider Registry

Providers register themselves via init() so that importing a provider
package is sufficient to make it available through the registry:

```go
import _ "github.com/lookatitude/beluga-ai/llm/providers/openai"

model, err := llm.New("openai", cfg)
```

Use `Register` to add a provider factory, `New` to create a ChatModel by
name, and `List` to discover all registered providers.

## Middleware

`Middleware` wraps a ChatModel to add cross-cutting concerns. Built-in
middleware includes logging, fallback, hooks, and rate limiting:

```go
model = llm.ApplyMiddleware(model,
    llm.WithLogging(logger),
    llm.WithFallback(backup),
    llm.WithHooks(hooks),
    llm.WithProviderLimits(limits),
)
```

Middleware is applied right-to-left: the first middleware in the list
becomes the outermost wrapper and executes first.

## Hooks

`Hooks` provides optional callbacks invoked during LLM operations:
BeforeGenerate, AfterGenerate, OnStream, OnToolCall, and OnError.
All fields are optional; nil hooks are skipped. Use `ComposeHooks`
to merge multiple Hooks into one.

## Structured Output

`StructuredOutput` wraps a ChatModel to produce typed Go values.
It generates a JSON Schema from the type parameter, instructs the model
to respond in JSON, parses the response, and retries on parse failures:

```go
type Sentiment struct {
    Label string  `json:"label"`
    Score float64 `json:"score"`
}
so := llm.NewStructured[Sentiment](model)
result, err := so.Generate(ctx, msgs)
```

## Context Management

`ContextManager` fits a message sequence within a token budget.
Two strategies are provided: "truncate" (drops oldest non-system messages)
and "sliding" (keeps the most recent messages that fit). Use
`NewContextManager` with options to configure:

```go
cm := llm.NewContextManager(
    llm.WithContextStrategy("sliding"),
    llm.WithTokenizer(tokenizer),
    llm.WithKeepSystemMessages(true),
)
fitted, err := cm.Fit(ctx, msgs, 4096)
```

## Tokenizer

`Tokenizer` provides token counting and encoding/decoding.
`SimpleTokenizer` is a built-in word-based approximation (1 token per
4 characters) suitable for budget estimation when a model-specific
tokenizer is unavailable.

## Routing

`Router` implements ChatModel by delegating to one of several backend
models chosen by a pluggable `RouterStrategy`. Built-in strategies include
`RoundRobin` and `FailoverChain`. For automatic retry across models, use
`FailoverRouter`:

```go
r := llm.NewRouter(
    llm.WithModels(modelA, modelB),
    llm.WithStrategy(&llm.RoundRobin{}),
)
```

## Rate Limiting

`WithProviderLimits` returns middleware that enforces requests-per-minute,
tokens-per-minute, and concurrency limits per provider.

## Generate Options

`GenerateOption` functional options configure individual Generate/Stream
calls: temperature, max tokens, top-p, stop sequences, response format,
tool choice, and provider-specific metadata.

## Streaming

Streaming uses iter.Seq2 (Go 1.23+):

```go
for chunk, err := range model.Stream(ctx, msgs) {
    if err != nil { break }
    fmt.Print(chunk.Delta)
}
```
