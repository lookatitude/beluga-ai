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

```go
// agent/base.go — conceptual
package agent

type config struct {
    id        string
    persona   Persona
    llm       llm.Model
    memory    memory.Memory
    planner   Planner
    hooks     Hooks
    tools     []Tool
    handoffs  []Agent
}

type Option func(*config)

func WithID(id string) Option {
    return func(c *config) { c.id = id }
}

func WithPersona(p Persona) Option {
    return func(c *config) { c.persona = p }
}

func WithLLM(m llm.Model) Option {
    return func(c *config) { c.llm = m }
}

func WithMemory(m memory.Memory) Option {
    return func(c *config) { c.memory = m }
}

func NewLLMAgent(opts ...Option) *LLMAgent {
    // defaults
    c := config{
        id:      generateID(),
        persona: Persona{Role: "assistant"},
        hooks:   Hooks{},
    }
    // apply overrides
    for _, opt := range opts {
        opt(&c)
    }
    // validate
    if c.llm == nil {
        panic("agent: WithLLM is required") // or return error from a NewX that returns error
    }
    return &LLMAgent{
        BaseAgent: BaseAgent{
            id:      c.id,
            persona: c.persona,
            hooks:   c.hooks,
        },
        llm:     c.llm,
        memory:  c.memory,
        planner: c.planner,
    }
}
```

User code:

```go
a := agent.NewLLMAgent(
    agent.WithID("research-assistant"),
    agent.WithLLM(model),
    agent.WithMemory(mem),
    agent.WithPlanner(planner.ReAct()),
)
```

## Where it's used

Every constructor in Beluga:

- `agent.NewLLMAgent`, `agent.NewSequentialAgent`, etc.
- `runtime.NewRunner`
- `llm.New` (via `Config` struct that itself uses options inside providers)
- `tool.New*`
- `memory.NewComposite`
- `rag.NewRetriever`
- `guard.New*`

## Common mistakes

- **Constructor that doesn't work with zero arguments.** If `NewX()` panics, the pattern is broken. Required values should be top-level positional arguments, with `With*` options only for the optional ones. (e.g., `NewLLMAgent(model llm.Model, opts ...Option)`.)
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
