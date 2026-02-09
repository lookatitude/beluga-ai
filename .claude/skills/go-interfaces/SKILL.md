---
name: go-interfaces
description: Designing Go interfaces with the registry, middleware, and hooks pattern used throughout Beluga AI v2. Use when creating new interfaces, implementing the extension contract, or adding hooks/middleware to any package.
---

# Go Interface Design for Beluga AI v2

## The Universal Extension Contract

Every extensible package exposes four interlocking mechanisms:

### 1. Extension Interface (small, 1-4 methods)
```go
// Keep interfaces SMALL. Compose larger surfaces.
type ChatModel interface {
    Generate(ctx context.Context, msgs []schema.Message, opts ...GenerateOption) (*schema.AIMessage, error)
    Stream(ctx context.Context, msgs []schema.Message, opts ...GenerateOption) iter.Seq2[schema.StreamChunk, error]
    BindTools(tools []tool.Tool) ChatModel
    ModelID() string
}
```

### 2. Hooks (lifecycle interception, all fields optional)
```go
type Hooks struct {
    // All fields are function pointers — nil means "skip this hook"
    BeforeGenerate func(ctx context.Context, msgs []schema.Message) (context.Context, []schema.Message, error)
    AfterGenerate  func(ctx context.Context, resp *schema.AIMessage, err error) (*schema.AIMessage, error)
    OnStream       func(ctx context.Context, event Event[schema.StreamChunk]) Event[schema.StreamChunk]
    OnError        func(ctx context.Context, err error) error
}

// ComposeHooks chains multiple hook sets. Each receives output of previous.
func ComposeHooks(hooks ...Hooks) Hooks {
    return Hooks{
        BeforeGenerate: func(ctx context.Context, msgs []schema.Message) (context.Context, []schema.Message, error) {
            var err error
            for _, h := range hooks {
                if h.BeforeGenerate != nil {
                    ctx, msgs, err = h.BeforeGenerate(ctx, msgs)
                    if err != nil {
                        return ctx, msgs, err
                    }
                }
            }
            return ctx, msgs, nil
        },
        // ... same pattern for each hook
    }
}
```

### 3. Middleware (composable decorators)
```go
// Pattern: func(T) T — wraps and returns the same interface
type Middleware func(ChatModel) ChatModel

func ApplyMiddleware(model ChatModel, mws ...Middleware) ChatModel {
    // Apply outside-in: last middleware wraps outermost
    for i := len(mws) - 1; i >= 0; i-- {
        model = mws[i](model)
    }
    return model
}

// Example middleware
func WithRetry(maxAttempts int, backoff time.Duration) Middleware {
    return func(next ChatModel) ChatModel {
        return &retryModel{inner: next, max: maxAttempts, backoff: backoff}
    }
}
```

### 4. Registry (Register/New/List)
See `go-framework` skill for full registry pattern.

## Interface Design Rules

1. **Accept interfaces, return structs** — functions return concrete types but accept interface params
2. **Small is beautiful** — if interface has >4 methods, split it
3. **Optional interfaces via type assertion**:
```go
// Core interface
type Retriever interface {
    Retrieve(ctx context.Context, query string, opts ...Option) ([]schema.Document, error)
}

// Optional capability — check at runtime
type BatchRetriever interface {
    Retriever
    RetrieveBatch(ctx context.Context, queries []string, opts ...Option) ([][]schema.Document, error)
}

// Usage:
if br, ok := retriever.(BatchRetriever); ok {
    results, err = br.RetrieveBatch(ctx, queries)
}
```

4. **Constructor returns the interface, not the struct** (for registries):
```go
func New(name string, cfg Config) (ChatModel, error) { ... }
```

5. **Embed for composition, not inheritance**:
```go
type BaseAgent struct {
    id      string
    persona Persona
    tools   []tool.Tool
    hooks   Hooks
}

// User embeds BaseAgent:
type MyAgent struct {
    agent.BaseAgent  // gets ID(), Persona(), Tools() for free
    // ... custom fields
}
```

## Hook Naming Convention

| Pattern | When | Signature |
|---------|------|-----------|
| `Before<Action>` | Before executing | `func(ctx, input) (input, error)` — can modify input |
| `After<Action>` | After executing | `func(ctx, output, error) (output, error)` — can modify output |
| `On<Event>` | When event occurs | `func(ctx, data)` — observe only, or `func(ctx, data) (data, error)` — modify |

## Middleware vs Hooks

- **Middleware**: Wraps the entire interface call. Good for: retry, rate-limit, cache, logging, tracing.
- **Hooks**: Fire at specific lifecycle points. Good for: audit, cost tracking, modification, validation.
- Use both together. Middleware applies first (outermost), hooks fire within the execution.
