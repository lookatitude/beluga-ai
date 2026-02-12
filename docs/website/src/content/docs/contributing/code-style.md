---
title: "Code Style — Beluga AI"
description: "Coding conventions and style guidelines for the Beluga AI Go framework. Interfaces, functional options, error handling, and naming patterns."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Beluga AI code style, Go conventions, functional options, error handling Go, registry pattern, middleware pattern, hooks pattern"
---

Beluga AI follows idiomatic Go conventions with a set of project-specific patterns that ensure consistency across the codebase. With 157 packages and 100+ providers, consistency is not optional — it's what makes the framework learnable. When every package uses the same registry pattern, the same middleware signature, and the same hooks structure, developers can navigate unfamiliar code with confidence. This guide covers everything you need to know to write code that fits in.

## General Go Conventions

- Run **`gofmt`** and **`goimports`** on all code. The CI linter enforces this.
- Follow the conventions in [Effective Go](https://go.dev/doc/effective_go) and the [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments).
- Use `make fmt` to format the entire codebase before committing.

## Interfaces First

Define the interface, then write implementations. Keep interfaces small — ideally 1 to 4 methods. The "small interface" constraint exists because every interface in Beluga has a corresponding mock in `internal/testutil/`, and every provider must implement the full interface. Smaller interfaces mean less boilerplate for mocks, less work for provider authors, and clearer contracts for users. When an interface grows beyond 4 methods, it's a signal to split it or use type assertions for optional capabilities.

```go
// ChatModel is the core LLM abstraction.
type ChatModel interface {
    Generate(ctx context.Context, messages []schema.Message, opts ...Option) (*schema.Message, error)
    Stream(ctx context.Context, messages []schema.Message, opts ...Option) iter.Seq2[schema.Event, error]
}
```

Larger interfaces should be composed from smaller ones when possible.

## Functional Options

Use the `WithX()` pattern for configuration instead of config structs or builders. Functional options provide backward-compatible API evolution: adding a new `WithTimeout()` option doesn't change any existing function signatures, and options compose naturally when passed as variadic arguments:

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

Beluga's error model uses typed errors with retry semantics because agentic systems cross multiple failure boundaries (provider APIs, tool execution, guard validation). Without a unified error model, generic retry middleware cannot determine whether an error from an arbitrary provider is safe to retry.

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

Every public function's first parameter **must** be `context.Context`. No exceptions. This enables cancellation propagation from HTTP handlers through agent execution to LLM calls and tool execution. It also carries OpenTelemetry spans for tracing and tenant isolation data for multi-tenant deployments. Omitting context from a public function breaks the entire observability and cancellation chain:

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

No mutable global state beyond `init()` registrations. Registry mutations happen **only** in `init()`. This constraint prevents race conditions in concurrent programs and ensures that the set of registered providers is deterministic — it depends on import statements, not on execution order:

```go
func init() {
    llm.Register("openai", func(cfg llm.ProviderConfig) (llm.ChatModel, error) {
        return New(cfg)
    })
}
```

## Embedding Over Inheritance

Go has no inheritance, and Beluga embraces that by using struct embedding for code reuse. `BaseAgent` provides default implementations for common agent operations, and custom agents embed it to get those defaults while overriding only the methods they need to customize:

```go
type MyAgent struct {
    agent.BaseAgent // embed base functionality
    // add custom fields
}
```

## Streaming with iter.Seq2

Beluga uses `iter.Seq2[T, error]` (Go 1.23+) for all public streaming APIs. Channels are reserved for internal goroutine communication only (voice frame processors, background workers). The `iter.Seq2` approach was chosen because it requires no goroutine per stream, provides natural backpressure via the `yield` return value, and composes cleanly with utility functions like `MapStream` and `FilterStream`. **Never** use channels for streaming in public APIs:

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

Every extensible package follows this exact pattern. There are 19 registries in the framework, and they all use the same three-function contract. This consistency means that understanding the LLM registry immediately teaches you how the embedding, vectorstore, voice, and workflow registries work:

```go
var registry = make(map[string]Factory)

func Register(name string, f Factory) { registry[name] = f }  // called in init()
func New(name string, cfg Config) (Interface, error) { /* factory lookup */ }
func List() []string { /* return registered names */ }
```

## Middleware Pattern

Middleware wraps an interface to add cross-cutting behavior without modifying the implementation. The signature is always `func(T) T`, which means middleware composes naturally — a retry middleware wrapping a cache middleware wrapping a rate limiter all satisfy the same interface. Note that `ApplyMiddleware` applies right-to-left: the last middleware in the list becomes the outermost wrapper in the call chain:

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

Hooks provide lifecycle callbacks for observation and modification without wrapping the entire interface. Unlike middleware (which intercepts the full call), hooks fire at specific points in the execution lifecycle. All fields are optional — `nil` hooks are skipped, so you only implement the callbacks you need. This struct-with-optional-fields design avoids the boilerplate of interface implementations with stub methods:

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

Beluga AI uses [Conventional Commits](https://www.conventionalcommits.org/) for all commit messages. This format is not just a style preference — it enables automatic changelog generation via git-cliff during the release process. Each commit message becomes a changelog entry, grouped by type (`feat`, `fix`, `perf`), so clear and descriptive messages directly improve the quality of release notes.

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
