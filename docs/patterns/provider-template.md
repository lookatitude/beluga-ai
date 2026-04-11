# Pattern: Provider Template

## What it is

The full end-to-end template for implementing a new provider in any Beluga package — LLM, embedding, vector store, memory store, voice transport, tool, etc. Once you know the template, every provider in every package looks the same.

## Why we use it

A consistent shape means a reviewer can understand a new provider in 30 seconds. A developer can copy an existing provider as a starting point. Tests can be written against a conformance suite. Documentation can point at one template and all providers follow it.

## How it works

The template has five parts:

1. **Interface implementation** — the provider type implements the package interface.
2. **Factory function** — a `func(cfg Config) (Interface, error)` closure.
3. **`init()` registration** — calls `package.Register("name", factory)`.
4. **Test file** — table-driven tests that hit the conformance suite.
5. **Documentation** — godoc comments on every exported symbol.

## Template: minimal LLM provider

Let's implement a fictional `echo` LLM provider that returns a prefixed version of the input.

```go
// llm/providers/echo/echo.go
//
// Package echo is a trivial LLM provider that echoes input with a prefix.
// Useful for testing and demonstrations.
package echo

import (
    "context"
    "fmt"

    "github.com/lookatitude/beluga-ai/core"
    "github.com/lookatitude/beluga-ai/llm"
    "github.com/lookatitude/beluga-ai/schema"
)

// compile-time interface check
var _ llm.Provider = (*Provider)(nil)

// Provider echoes input messages back with a configurable prefix.
type Provider struct {
    prefix string
}

// Generate returns a single-message response.
func (p *Provider) Generate(ctx context.Context, req llm.Request) (*llm.Response, error) {
    if err := ctx.Err(); err != nil {
        return nil, core.Errorf(core.ErrTimeout, "echo.Generate: %w", err)
    }
    if len(req.Messages) == 0 {
        return nil, core.Errorf(core.ErrInvalidInput, "echo: empty messages")
    }
    last := req.Messages[len(req.Messages)-1]
    out := fmt.Sprintf("%s %s", p.prefix, last.Content)
    return &llm.Response{
        Message: schema.Message{
            Role:    "assistant",
            Content: out,
        },
    }, nil
}

// Stream streams a single chunk with the prefixed output.
func (p *Provider) Stream(ctx context.Context, req llm.Request) (*core.Stream[core.Event[llm.Chunk]], error) {
    resp, err := p.Generate(ctx, req)
    if err != nil {
        return nil, err
    }
    return core.NewStream("echo.stream", func(yield func(int, core.Event[llm.Chunk]) bool) {
        yield(0, core.Event[llm.Chunk]{
            Type:    core.EventData,
            Payload: llm.Chunk{Text: resp.Message.Content},
        })
        yield(1, core.Event[llm.Chunk]{Type: core.EventDone})
    }), nil
}

// init registers this provider under the name "echo".
func init() {
    if err := llm.Register("echo", newFactory()); err != nil {
        panic(err) // duplicate registration is a programming error
    }
}

func newFactory() llm.Factory {
    return func(cfg llm.Config) (llm.Provider, error) {
        prefix, _ := cfg.GetString("prefix", "echo>")
        return &Provider{prefix: prefix}, nil
    }
}
```

And the test file:

```go
// llm/providers/echo/echo_test.go
package echo

import (
    "context"
    "testing"

    "github.com/lookatitude/beluga-ai/llm"
    "github.com/lookatitude/beluga-ai/schema"
)

func TestProvider_Generate(t *testing.T) {
    tests := []struct {
        name    string
        input   []schema.Message
        prefix  string
        want    string
        wantErr bool
    }{
        {
            name:   "simple echo",
            input:  []schema.Message{{Role: "user", Content: "hello"}},
            prefix: "bot>",
            want:   "bot> hello",
        },
        {
            name:    "empty input",
            input:   nil,
            prefix:  "bot>",
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            p := &Provider{prefix: tt.prefix}
            resp, err := p.Generate(context.Background(), llm.Request{Messages: tt.input})
            if (err != nil) != tt.wantErr {
                t.Fatalf("err=%v wantErr=%v", err, tt.wantErr)
            }
            if err == nil && resp.Message.Content != tt.want {
                t.Errorf("content=%q want=%q", resp.Message.Content, tt.want)
            }
        })
    }
}

// TestRegistration validates that init() registered the provider.
func TestRegistration(t *testing.T) {
    found := false
    for _, name := range llm.List() {
        if name == "echo" {
            found = true
            break
        }
    }
    if !found {
        t.Fatal("echo provider not in llm.List()")
    }
}
```

User code consumes it:

```go
import _ "github.com/lookatitude/beluga-ai/llm/providers/echo"

p, err := llm.New("echo", llm.Config{"prefix": "bot>"})
```

## Template checklist

Use this when reviewing a new provider PR:

- [ ] Compile-time interface check: `var _ Interface = (*Impl)(nil)`
- [ ] `init()` registers in the correct registry
- [ ] Factory function returns a typed error on bad config
- [ ] Errors use `core.Errorf` with appropriate `ErrorCode`
- [ ] `context.Context` is the first parameter of every public method
- [ ] `context.Context` cancellation respected in any blocking call
- [ ] `Stream` properly yields `EventDone` or `EventError` at termination
- [ ] Godoc on every exported type and function
- [ ] Table-driven test with happy path + error paths
- [ ] Test of `init()` registration
- [ ] No global mutable state outside the registry
- [ ] OTel span opened at the public method boundary (or middleware handles it)

## Where it's used

Every provider in Beluga follows this template. See:

- `llm/providers/*/` — LLM providers
- `rag/embedding/providers/*/` — embedders
- `rag/vectorstore/providers/*/` — vector stores
- `memory/stores/*/` — memory stores
- `voice/stt/providers/*/` — STT providers
- `voice/tts/providers/*/` — TTS providers
- `tool/builtin/*/` — built-in tools
- `guard/providers/*/` — guard providers

All use the same five-part template with the same checklist.

## Common mistakes

- **Missing compile-time check.** Means a method rename in the interface compiles fine but breaks at runtime.
- **Panicking on config errors.** Use `error` returns. Panic is only for programming errors (duplicate registration).
- **Forgetting to wire `context.Context` through.** Providers that ignore the context hang forever on cancellation.
- **Returning `fmt.Errorf` instead of `core.Errorf`.** Loses the `ErrorCode` — middleware can't decide whether to retry.
- **Skipping the registration test.** If you forget to import the provider subpackage, the provider is silently absent. A test catches this.

## Related

- [03 — Extensibility Patterns](../architecture/03-extensibility-patterns.md)
- [`patterns/registry-factory.md`](./registry-factory.md) — the registry side.
- [`patterns/error-handling.md`](./error-handling.md) — `core.Errorf` semantics.
- [Custom Provider guide](../guides/custom-provider.md) — a full walk-through of implementing a real provider.
