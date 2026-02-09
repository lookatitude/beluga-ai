---
title: LLM Package API
description: API documentation for the LLM abstraction layer.
---

```go
import "github.com/lookatitude/beluga-ai/llm"
```

Package llm provides the LLM abstraction layer with `ChatModel` interface, provider registry, middleware, hooks, structured output, context management, and routing.

## Quick Start

```go
import (
    "github.com/lookatitude/beluga-ai/llm"
    _ "github.com/lookatitude/beluga-ai/llm/providers/openai"
)

model, err := llm.New("openai", config.ProviderConfig{
    APIKey: os.Getenv("OPENAI_API_KEY"),
    Model:  "gpt-4o",
})

// Synchronous
resp, err := model.Generate(ctx, []schema.Message{
    schema.NewHumanMessage("Hello!"),
})

// Streaming
for chunk, err := range model.Stream(ctx, msgs) {
    if err != nil { break }
    fmt.Print(chunk.Delta)
}
```

## ChatModel Interface

```go
type ChatModel interface {
    Generate(ctx context.Context, msgs []schema.Message, opts ...GenerateOption) (*schema.AIMessage, error)
    Stream(ctx context.Context, msgs []schema.Message, opts ...GenerateOption) iter.Seq2[schema.StreamChunk, error]
    BindTools(tools []schema.ToolDefinition) ChatModel
    ModelID() string
}
```

## Generate Options

```go
resp, err := model.Generate(ctx, msgs,
    llm.WithTemperature(0.7),
    llm.WithMaxTokens(1000),
    llm.WithTopP(0.9),
    llm.WithStopSequences("END"),
    llm.WithToolChoice(llm.ToolChoiceRequired),
    llm.WithResponseFormat(llm.ResponseFormat{Type: "json_object"}),
)
```

## Tool Binding

```go
model = model.BindTools([]schema.ToolDefinition{
    {Name: "search", Description: "Search the web", InputSchema: ...},
    {Name: "calculate", Description: "Do math", InputSchema: ...},
})

resp, err := model.Generate(ctx, msgs)
if len(resp.ToolCalls) > 0 {
    // handle tool calls
}
```

## Middleware

Wrap models with cross-cutting concerns:

```go
model = llm.ApplyMiddleware(model,
    llm.WithLogging(logger),
    llm.WithFallback(backupModel),
    llm.WithProviderLimits(llm.ProviderLimits{
        RPM: 10000,
        TPM: 1000000,
    }),
)
```

## Hooks

Inject lifecycle callbacks:

```go
model = llm.WithHooks(model, llm.Hooks{
    BeforeGenerate: func(ctx context.Context, msgs []schema.Message) error {
        log.Printf("Generating with %d messages", len(msgs))
        return nil
    },
    AfterGenerate: func(ctx context.Context, resp *schema.AIMessage, err error) {
        if err == nil {
            log.Printf("Tokens: %d", resp.Usage.TotalTokens)
        }
    },
    OnStream: func(ctx context.Context, chunk schema.StreamChunk) {
        // observe streaming
    },
})
```

## Structured Output

Generate typed responses:

```go
type Response struct {
    Answer     string   `json:"answer" description:"The answer"`
    Confidence float64  `json:"confidence" min:"0" max:"1"`
    Sources    []string `json:"sources"`
}

structured := llm.NewStructured[Response](model,
    llm.WithMaxRetries(3),
)

result, err := structured.Generate(ctx, msgs)
fmt.Println(result.Answer, result.Confidence)
```

## Context Management

Fit messages within token budget:

```go
manager := llm.NewContextManager(
    llm.WithContextStrategy("sliding"), // or "truncate"
    llm.WithTokenizer(tokenizer),
    llm.WithKeepSystemMessages(true),
)

fitted, err := manager.Fit(ctx, msgs, 4096) // max 4096 tokens
```

## Router

Route requests across multiple models:

```go
router := llm.NewRouter(
    llm.WithModels(model1, model2, model3),
    llm.WithStrategy(&llm.RoundRobin{}),
)

// Automatically selects a model
resp, err := router.Generate(ctx, msgs)
```

### Failover Router

Automatic failover on retryable errors:

```go
failover := llm.NewFailoverRouter(primary, backup1, backup2)
resp, err := failover.Generate(ctx, msgs) // tries in order
```

## Provider Registry

```go
// List available providers
providers := llm.List() // ["openai", "anthropic", "google", ...]

// Create by name
model, err := llm.New("anthropic", cfg)
```

## Token Counting

```go
tokenizer := &llm.SimpleTokenizer{}
count := tokenizer.Count("Hello world")
totalTokens := tokenizer.CountMessages(msgs)
```

## See Also

- [Agent Package](./agent.md) for agent-model integration
- [Tool Package](./tool.md) for tool binding
- [Memory Package](./memory.md) for conversation history
