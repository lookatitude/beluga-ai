# Pattern: Functional Options

## What it is

Constructors accept variadic `Option` values rather than a configuration struct. Each option is a function `func(*config)` that sets a field on an internal config. Users call `NewX(WithA(1), WithB(2), WithC(3))` in any order.

## Why we use it

- **Grows without breaking the API.** Adding `WithD(v)` tomorrow doesn't break any caller.
- **Sensible defaults.** The zero-argument case (`NewX()`) must work. Options override defaults.
- **Discoverable.** IDE autocomplete shows the available `With*` functions. Self-documenting.
- **Order-independent.** Callers don't need to remember positional arguments.
- **Optional fields.** No `nil` checks on a config struct. If a `WithX` wasn't called, the default is used.

**Alternatives considered:**
- **Config struct.** Works but requires users to populate every field they care about, and adding fields is a breaking change to the struct.
- **Builder pattern.** Feels Java-y and requires a separate type. Functional options are the Go-idiomatic answer.
- **Keyword arguments (not in Go).** Would be ideal but the language doesn't have them.

## How it works

From `agent/option.go` and `agent/base.go` (trimmed to four of the fields):

```go
// agentConfig holds the configuration applied by options.
type agentConfig struct {
    llm           llm.ChatModel
    tools         []tool.Tool
    persona       Persona
    maxIterations int
    // ... timeout, hooks, handoffs, memory, planner, children, metadata, contract
}

// Option configures an agent.
type Option func(*agentConfig)

func defaultConfig() agentConfig {
    return agentConfig{
        maxIterations: 10,
        timeout:       5 * time.Minute,
        plannerName:   "react",
    }
}

// WithLLM sets the language model for the agent.
func WithLLM(model llm.ChatModel) Option {
    return func(c *agentConfig) { c.llm = model }
}

// WithPersona sets the agent's persona.
func WithPersona(p Persona) Option {
    return func(c *agentConfig) { c.persona = p }
}

// New creates a new BaseAgent with the given ID and options.
func New(id string, opts ...Option) *BaseAgent {
    cfg := defaultConfig()
    for _, opt := range opts {
        opt(&cfg)
    }
    return &BaseAgent{id: id, config: cfg}
}
```

The required value (the ID) is a positional argument; everything else is an option with a default.

User code:

```go
a := agent.New("research-assistant",
    agent.WithLLM(model),
    agent.WithTools(tools),
    agent.WithPlannerName("reflexion"),
    agent.WithMaxIterations(5),
)
```

## Where it's used

Most constructors in Beluga:

- `agent.New` (`agent.With*`)
- `runtime.NewRunner` (`runtime.With*`), `runtime.NewTeam`
- `hitl.NewManager` (`hitl.WithNotifier`, `hitl.WithTimeout`)
- `workflow.NewExecutor` (`workflow.WithStore`, `workflow.WithExecutorHooks`)
- `guard.NewPipeline` (`guard.Input`, `guard.Output`, `guard.Tool`)
- `cost.NewInMemoryTracker`, `audit.NewInMemoryStore`

Registry constructors (`llm.New`, `memory.New`, `rag/vectorstore.New`, ...) take a `config.ProviderConfig` struct instead, because the config is often loaded from a file.

## Common mistakes

- **Constructor that doesn't work with zero arguments.** If `NewX()` panics, the pattern is broken. Required values should be top-level positional arguments, with `With*` options only for the optional ones (e.g. `agent.New(id string, opts ...Option)`).
- **Silent option conflicts.** If `WithPlanner(a)` and `WithPlanner(b)` are both passed, the last wins without a warning. Document this or return an error.
- **Exposing the internal config struct.** Keep `config` unexported. If users can import it, you've lost the "grows without breaking" benefit.
- **Options that mutate shared state.** An `Option` should set fields on the local config, not touch package-level variables.
- **Options that need a context.** Options are evaluated synchronously during construction. If your option needs a context or a network call, make it a separate method call after construction, not an option.

## Example: implementing your own

A simple HTTP client with options:

```go
package myhttp

import (
    "net/http"
    "time"
)

type config struct {
    timeout    time.Duration
    userAgent  string
    maxRetries int
    headers    http.Header
}

type Option func(*config)

func WithTimeout(d time.Duration) Option {
    return func(c *config) { c.timeout = d }
}

func WithUserAgent(ua string) Option {
    return func(c *config) { c.userAgent = ua }
}

func WithMaxRetries(n int) Option {
    return func(c *config) { c.maxRetries = n }
}

func WithHeader(k, v string) Option {
    return func(c *config) {
        if c.headers == nil {
            c.headers = make(http.Header)
        }
        c.headers.Set(k, v)
    }
}

type Client struct {
    http       *http.Client
    userAgent  string
    maxRetries int
    headers    http.Header
}

func New(opts ...Option) *Client {
    // sensible defaults
    c := config{
        timeout:    30 * time.Second,
        userAgent:  "beluga-myhttp/1.0",
        maxRetries: 3,
    }
    for _, opt := range opts {
        opt(&c)
    }
    return &Client{
        http:       &http.Client{Timeout: c.timeout},
        userAgent:  c.userAgent,
        maxRetries: c.maxRetries,
        headers:    c.headers,
    }
}
```

Usage:

```go
client := myhttp.New()                           // all defaults
client := myhttp.New(myhttp.WithTimeout(5 * time.Second))
client := myhttp.New(
    myhttp.WithTimeout(60 * time.Second),
    myhttp.WithHeader("Authorization", "Bearer " + token),
    myhttp.WithMaxRetries(5),
)
```

Adding `WithProxy(url)` tomorrow is non-breaking — zero existing callers need to change.

## Related

- [03 — Extensibility Patterns](../architecture/03-extensibility-patterns.md) — functional options sit inside the registry/factory pattern.
- [Go blog: Self-referential functions and the design of options](https://commandcenter.blogspot.com/2014/01/self-referential-functions-and-design.html) — Rob Pike's original write-up.
