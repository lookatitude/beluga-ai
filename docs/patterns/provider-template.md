# Pattern: Provider Template

## What it is

The full end-to-end template for implementing a new provider in any Beluga package — LLM, embedding, vector store, memory store, voice transport, tool, etc. Once you know the template, every provider in every package looks the same.

## Why we use it

A consistent shape means a reviewer can understand a new provider in 30 seconds. A developer can copy an existing provider as a starting point. Tests can be written against a conformance suite. Documentation can point at one template and all providers follow it.

## How it works

The template has five parts:

1. **Interface implementation** — the provider type implements the package interface.
2. **Factory function** — for `llm`, `rag/embedding`, `rag/vectorstore` and `memory` this is `func(cfg config.ProviderConfig) (Interface, error)`.
3. **`init()` registration** — calls `package.Register("name", factory)`.
4. **Test file** — table-driven tests that hit the conformance suite.
5. **Documentation** — godoc comments on every exported symbol.

## Template: minimal LLM provider

Let's implement a fictional `echo` LLM provider, in your own module, that returns a prefixed version of the last message. (The framework's own offline test provider is `llm/providers/mock`.)

```go
// providers/echo/echo.go
//
// Package echo is a trivial LLM provider that echoes input with a prefix.
// Useful for testing and demonstrations.
package echo

import (
    "context"
    "iter"

    "github.com/lookatitude/beluga-ai/v2/config"
    "github.com/lookatitude/beluga-ai/v2/core"
    "github.com/lookatitude/beluga-ai/v2/llm"
    "github.com/lookatitude/beluga-ai/v2/schema"
)

// compile-time interface check
var _ llm.ChatModel = (*Model)(nil)

// Model echoes the last input message back with a configurable prefix.
type Model struct {
    prefix string
}

// Generate returns a single AI message.
func (m *Model) Generate(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) (*schema.AIMessage, error) {
    if err := ctx.Err(); err != nil {
        return nil, core.Errorf(core.ErrTimeout, "echo.Generate: %w", err)
    }
    if len(msgs) == 0 {
        return nil, core.Errorf(core.ErrInvalidInput, "echo: empty messages")
    }
    var text string
    for _, p := range msgs[len(msgs)-1].GetContent() {
        if tp, ok := p.(schema.TextPart); ok {
            text += tp.Text
        }
    }
    return schema.NewAIMessage(m.prefix + " " + text), nil
}

// Stream yields the same output as a single chunk.
func (m *Model) Stream(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) iter.Seq2[schema.StreamChunk, error] {
    return func(yield func(schema.StreamChunk, error) bool) {
        resp, err := m.Generate(ctx, msgs, opts...)
        if err != nil {
            yield(schema.StreamChunk{}, err)
            return
        }
        yield(schema.StreamChunk{Delta: resp.Text(), FinishReason: "stop"}, nil)
    }
}

// BindTools returns the model unchanged; echo ignores tools.
func (m *Model) BindTools(_ []schema.ToolDefinition) llm.ChatModel { return m }

// ModelID identifies the model.
func (m *Model) ModelID() string { return "echo" }

// New builds a Model. The prefix comes from Options["prefix"].
func New(cfg config.ProviderConfig) (llm.ChatModel, error) {
    prefix, ok := config.GetOption[string](cfg, "prefix")
    if !ok {
        prefix = "echo>"
    }
    return &Model{prefix: prefix}, nil
}

// init registers this provider under the name "echo".
func init() {
    llm.Register("echo", New)
}
```

And the test file:

```go
// providers/echo/echo_test.go
package echo

import (
    "context"
    "slices"
    "testing"

    "github.com/lookatitude/beluga-ai/v2/llm"
    "github.com/lookatitude/beluga-ai/v2/schema"
)

func TestModel_Generate(t *testing.T) {
    tests := []struct {
        name    string
        input   []schema.Message
        prefix  string
        want    string
        wantErr bool
    }{
        {
            name:   "simple echo",
            input:  []schema.Message{schema.NewHumanMessage("hello")},
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
            m := &Model{prefix: tt.prefix}
            resp, err := m.Generate(context.Background(), tt.input)
            if (err != nil) != tt.wantErr {
                t.Fatalf("err=%v wantErr=%v", err, tt.wantErr)
            }
            if err == nil && resp.Text() != tt.want {
                t.Errorf("text=%q want=%q", resp.Text(), tt.want)
            }
        })
    }
}

// TestRegistration validates that init() registered the provider.
func TestRegistration(t *testing.T) {
    if !slices.Contains(llm.List(), "echo") {
        t.Fatal("echo provider not in llm.List()")
    }
}
```

User code consumes it:

```go
import _ "example.com/myrepo/providers/echo"

m, err := llm.New("echo", config.ProviderConfig{
    Provider: "echo",
    Options:  map[string]any{"prefix": "bot>"},
})
```

## Template checklist

Use this when reviewing a new provider PR:

- [ ] Compile-time interface check: `var _ Interface = (*Impl)(nil)`
- [ ] `init()` registers in the correct registry
- [ ] Factory function returns a typed error on bad config
- [ ] Errors use `core.Errorf` with appropriate `ErrorCode`
- [ ] `context.Context` is the first parameter of every public method
- [ ] `context.Context` cancellation respected in any blocking call
- [ ] `Stream` returns an `iter.Seq2` that yields a final error on failure and stops when `yield` returns false
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
- `guard/providers/*/` — guard providers

All use the same five-part template with the same checklist.

## Common mistakes

- **Missing compile-time check.** Means a method rename in the interface compiles fine but breaks at runtime.
- **Panicking on config errors.** Use `error` returns. Panics are for programming errors only.
- **Forgetting to wire `context.Context` through.** Providers that ignore the context hang forever on cancellation.
- **Returning `fmt.Errorf` instead of `core.Errorf`.** Loses the `ErrorCode` — middleware can't decide whether to retry.
- **Skipping the registration test.** If you forget to import the provider subpackage, the provider is silently absent. A test catches this.

## Related

- [03 — Extensibility Patterns](../architecture/03-extensibility-patterns.md)
- [`patterns/registry-factory.md`](./registry-factory.md) — the registry side.
- [`patterns/error-handling.md`](./error-handling.md) — `core.Errorf` semantics.
- [Custom Provider guide](../guides/custom-provider.md) — a full walk-through of implementing a real provider.
