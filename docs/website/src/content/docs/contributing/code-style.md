---
title: Code Style Guide
description: Coding conventions and style guidelines for Beluga AI
---

Beluga AI follows idiomatic Go conventions with a set of project-specific patterns that ensure consistency across the codebase. This guide covers everything you need to know to write code that fits in.

## General Go Conventions

- Run **`gofmt`** and **`goimports`** on all code. The CI linter enforces this.
- Follow the conventions in [Effective Go](https://go.dev/doc/effective_go) and the [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments).
- Use `make fmt` to format the entire codebase before committing.

## Interfaces First

Define the interface, then write implementations. Keep interfaces small — ideally 1 to 4 methods.

```go
// ChatModel is the core LLM abstraction.
type ChatModel interface {
    Generate(ctx context.Context, messages []schema.Message, opts ...Option) (*schema.Message, error)
    Stream(ctx context.Context, messages []schema.Message, opts ...Option) iter.Seq2[schema.Event, error]
}
```

Larger interfaces should be composed from smaller ones when possible.

## Functional Options

Use the `WithX()` pattern for configuration instead of config structs or builders:

```go
// Option configures a ChatModel call.
type Option func(*options)

func WithTemperature(t float64) Option {
    return func(o *options) { o.temperature = &t }
}

func WithMaxTokens(n int) Option {
    return func(o *options) { o.maxTokens = &n }
}

// Usage
response, err := model.Generate(ctx, messages, WithTemperature(0.7), WithMaxTokens(1024))
```

## Error Handling

- Always return `(T, error)`. Never panic for recoverable errors.
- Use typed errors from `core/errors.go` with `ErrorCode`:

```go
return nil, core.NewError(core.ErrCodeInvalidInput, "temperature must be between 0 and 2")
```

- Always check `IsRetryable()` for LLM and tool errors before retrying:

```go
if err != nil && core.IsRetryable(err) {
    // safe to retry
}
```

## Context Propagation

Every public function's first parameter **must** be `context.Context`. No exceptions:

```go
// Good
func (a *Agent) Run(ctx context.Context, input string) (string, error)

// Bad — missing context
func (a *Agent) Run(input string) (string, error)
```

## Naming Conventions

| Pattern | Convention | Example |
|---|---|---|
| Constructors | `New()` or `NewXxx()` | `NewAgent(opts...)` |
| Registry | `Register()` + `New()` + `List()` | `llm.Register("openai", factory)` |
| Options | `WithXxx()` | `WithTemperature(0.7)` |
| Getters | No `Get` prefix | `agent.Name()` not `agent.GetName()` |
| Interfaces | `-er` suffix when idiomatic | `Embedder`, `Retriever`, `Splitter` |

## No Global State

No mutable global state beyond `init()` registrations. Registry mutations happen **only** in `init()`:

```go
func init() {
    llm.Register("openai", func(cfg llm.ProviderConfig) (llm.ChatModel, error) {
        return New(cfg)
    })
}
```

## Embedding Over Inheritance

Compose behavior via struct embedding, not deep interface hierarchies:

```go
type MyAgent struct {
    agent.BaseAgent // embed base functionality
    // add custom fields
}
```

## Streaming with iter.Seq2

Beluga uses `iter.Seq2[T, error]` (Go 1.23+) for all streaming. **Never** use channels for streaming:

```go
// Good — iter.Seq2
func (a *Agent) Stream(ctx context.Context, input string) iter.Seq2[schema.Event, error] {
    return func(yield func(schema.Event, error) bool) {
        for _, event := range events {
            if !yield(event, nil) {
                return
            }
        }
    }
}

// Consumers use range
for event, err := range agent.Stream(ctx, input) {
    if err != nil {
        break
    }
    // handle event
}
```

Use `iter.Pull()` when pull-based semantics are needed.

## Registry Pattern

Every extensible package follows this exact pattern:

```go
var registry = make(map[string]Factory)

func Register(name string, f Factory) { registry[name] = f }  // called in init()
func New(name string, cfg Config) (Interface, error) { /* factory lookup */ }
func List() []string { /* return registered names */ }
```

## Middleware Pattern

Middleware wraps an interface to add behavior. The signature is always `func(T) T`:

```go
type Middleware func(ChatModel) ChatModel

func ApplyMiddleware(model ChatModel, mws ...Middleware) ChatModel {
    for i := len(mws) - 1; i >= 0; i-- {
        model = mws[i](model)
    }
    return model
}
```

## Hooks Pattern

Hooks provide lifecycle callbacks. All fields are optional — `nil` hooks are skipped:

```go
type Hooks struct {
    OnStart func(ctx context.Context, input any) error
    OnEnd   func(ctx context.Context, result any, err error)
    OnError func(ctx context.Context, err error) error
}

func ComposeHooks(hooks ...Hooks) Hooks { /* merge multiple hook sets */ }
```

## Documentation

Every exported type and function **must** have a doc comment:

```go
// Agent represents an AI agent that can process inputs and produce outputs.
// It supports streaming via iter.Seq2 and can be composed with middleware.
type Agent struct { ... }

// Run executes the agent synchronously and returns the final result.
func (a *Agent) Run(ctx context.Context, input string) (string, error) { ... }
```

Include a usage example in the package-level doc comment (`doc.go`).

## Commit Message Format

Beluga AI uses [Conventional Commits](https://www.conventionalcommits.org/) for all commit messages. This format enables automatic changelog generation via git-cliff.

### Structure

```
<type>(<scope>): <description>

[optional body]

[optional footer(s)]
```

### Types

| Type | Description |
|---|---|
| `feat` | A new feature |
| `fix` | A bug fix |
| `docs` | Documentation changes only |
| `chore` | Maintenance tasks, dependency updates |
| `test` | Adding or updating tests |
| `refactor` | Code change that neither fixes a bug nor adds a feature |
| `perf` | Performance improvement |
| `ci` | CI/CD configuration changes |

### Examples

```
feat(llm): add streaming support for Anthropic provider
fix(agent): prevent nil pointer on empty tool response
docs(rag): add hybrid search configuration guide
test(memory): add table-driven tests for recall store
refactor(core): simplify error wrapping utilities
perf(voice): reduce frame allocation in STT pipeline
ci: add fuzz testing to PR workflow
chore: update golangci-lint to v1.62
```

The scope in parentheses is optional but encouraged when the change is specific to a package.
