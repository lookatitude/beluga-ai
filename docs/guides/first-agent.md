# First Agent — 20 minutes from zero to streaming output

**You will build:** a single-LLM agent with a typed calculator tool that streams its answer back one chunk at a time.
**Prerequisites:** Go 1.23+, an API key for an LLM provider registered in this build (OpenAI, Anthropic, Ollama, etc.).
**Related:** [DOC-02 Core Primitives](../architecture/02-core-primitives.md), [DOC-05 Agent Anatomy](../architecture/05-agent-anatomy.md), [Provider Template](../patterns/provider-template.md).

Every code block below has been compile-verified against the current tree using a `replace` directive against this repository. You can paste the blocks into a fresh module and run them unchanged.

## 1. Install

```bash
mkdir first-agent && cd first-agent
go mod init example.com/first-agent
go get github.com/lookatitude/beluga-ai@latest
```

Beluga requires Go 1.23 or newer (streaming uses `iter.Seq2`, introduced in the 1.23 stdlib).

## 2. Pick a provider and set credentials

Provider packages register themselves in `init()` — you opt in by *blank-importing* the subpackage you want. This guide uses OpenAI; swap in any of `anthropic`, `ollama`, `google`, `bedrock`, etc. by changing two lines.

```bash
export OPENAI_API_KEY=sk-...
```

The provider is wired up by a single blank import:

```go
import _ "github.com/lookatitude/beluga-ai/llm/providers/openai"
```

That import triggers the provider's `init()` which calls `llm.Register("openai", ...)`. After that, `llm.New("openai", cfg)` will resolve. See [`llm/providers/openai/openai.go`](../../llm/providers/openai/openai.go) and [`llm/registry.go`](../../llm/registry.go) for the exact registration pattern.

## 3. Your first agent

One file. It builds an LLM, constructs a `BaseAgent` with a persona, and collects the answer via `Invoke`:

```go
// main.go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/lookatitude/beluga-ai/agent"
	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/llm"

	// Register the OpenAI provider via init().
	_ "github.com/lookatitude/beluga-ai/llm/providers/openai"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// 1. Build the LLM from a ProviderConfig struct (not a map).
	model, err := llm.New("openai", config.ProviderConfig{
		Provider: "openai",
		APIKey:   os.Getenv("OPENAI_API_KEY"),
		Model:    "gpt-4o-mini",
	})
	if err != nil {
		log.Fatalf("llm.New: %v", err)
	}

	// 2. Build the agent using functional options.
	a := agent.New("math-tutor",
		agent.WithLLM(model),
		agent.WithPersona(agent.Persona{
			Role:      "patient math tutor",
			Goal:      "answer arithmetic questions clearly",
			Backstory: "You explain each step before giving the final answer.",
			Traits:    []string{"concise", "accurate"},
		}),
	)

	// 3. Invoke the agent synchronously — this streams internally and
	//    returns the concatenated text.
	answer, err := a.Invoke(ctx, "What is 17 times 42, minus 19?")
	if err != nil {
		log.Fatalf("agent.Invoke: %v", err)
	}

	fmt.Println(answer)
}
```

Key API facts (verify in source):

- `agent.New(id string, opts ...Option) *BaseAgent` lives at [`agent/base.go:23`](../../agent/base.go). There is no `agent.NewLLMAgent`.
- `llm.New` takes a `config.ProviderConfig` struct, defined at [`config/provider.go:19`](../../config/provider.go). It is not a `map[string]any`.
- `Persona` fields are `Role`, `Goal`, `Backstory`, `Traits` — see [`agent/persona.go:13`](../../agent/persona.go).

Run it:

```bash
go run .
```

## 4. Add a tool

Tools in Beluga are typed. `tool.NewFuncTool[I any]` wraps a Go function whose input is a struct with `json` and `description` tags — the JSON Schema is generated automatically. See [`tool/functool.go:44`](../../tool/functool.go).

Add a calculator tool in the same module. This example uses a trivial evaluator so the guide stays self-contained; in real code use `go/parser` or a proper expression library.

```go
// calculator.go
package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/lookatitude/beluga-ai/core"
	"github.com/lookatitude/beluga-ai/tool"
)

// CalculatorInput is the typed input schema for the calculator tool.
// Struct tags drive the generated JSON Schema.
type CalculatorInput struct {
	A  float64 `json:"a" description:"Left operand" required:"true"`
	Op string  `json:"op" description:"Operator: one of + - * /" required:"true"`
	B  float64 `json:"b" description:"Right operand" required:"true"`
}

// NewCalculatorTool returns a tool that evaluates a single binary operation.
func NewCalculatorTool() tool.Tool {
	return tool.NewFuncTool(
		"calculator",
		"Evaluate a single binary arithmetic operation (a op b).",
		func(ctx context.Context, in CalculatorInput) (*tool.Result, error) {
			var out float64
			switch strings.TrimSpace(in.Op) {
			case "+":
				out = in.A + in.B
			case "-":
				out = in.A - in.B
			case "*":
				out = in.A * in.B
			case "/":
				if in.B == 0 {
					return nil, core.Errorf(core.ErrInvalidInput, "calculator: divide by zero")
				}
				out = in.A / in.B
			default:
				return nil, core.Errorf(core.ErrInvalidInput, "calculator: unknown operator %q", in.Op)
			}
			return tool.TextResult(strconv.FormatFloat(out, 'f', -1, 64)), nil
		},
	)
}

// stringMustNotBeEmpty is unused here but kept as a reminder that tool
// results are text by default — use schema.ContentPart for multimodal.
var _ = fmt.Sprintf
```

Wire the tool into the agent by adding one option to the `agent.New` call in `main.go`:

```go
	a := agent.New("math-tutor",
		agent.WithLLM(model),
		agent.WithPersona(agent.Persona{
			Role:      "patient math tutor",
			Goal:      "answer arithmetic questions clearly",
			Backstory: "You explain each step before giving the final answer.",
			Traits:    []string{"concise", "accurate"},
		}),
		agent.WithTools([]tool.Tool{NewCalculatorTool()}),
	)
```

Add `"github.com/lookatitude/beluga-ai/tool"` to the imports of `main.go`. The default planner is `react`, so the LLM will see the tool's schema, decide to call it, receive the observation, and continue.

## 5. Stream events

`Invoke` is a thin wrapper that collects `Stream` and returns the final text. For real-time UX you want the stream directly. `BaseAgent.Stream` returns `iter.Seq2[agent.Event, error]` — you consume it with a standard `for ... range` loop.

```go
// stream_main.go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/lookatitude/beluga-ai/agent"
	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/llm"
	"github.com/lookatitude/beluga-ai/tool"

	_ "github.com/lookatitude/beluga-ai/llm/providers/openai"
)

func RunStreaming() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	model, err := llm.New("openai", config.ProviderConfig{
		Provider: "openai",
		APIKey:   os.Getenv("OPENAI_API_KEY"),
		Model:    "gpt-4o-mini",
	})
	if err != nil {
		log.Fatalf("llm.New: %v", err)
	}

	a := agent.New("math-tutor",
		agent.WithLLM(model),
		agent.WithPersona(agent.Persona{
			Role: "patient math tutor",
			Goal: "answer arithmetic questions clearly",
		}),
		agent.WithTools([]tool.Tool{NewCalculatorTool()}),
	)

	// Stream returns iter.Seq2[agent.Event, error]. Range over it directly.
	for event, err := range a.Stream(ctx, "What is 17 times 42, minus 19?") {
		if err != nil {
			log.Fatalf("stream error: %v", err)
		}
		switch event.Type {
		case agent.EventText:
			fmt.Print(event.Text)
		case agent.EventToolCall:
			if event.ToolCall != nil {
				fmt.Printf("\n[tool call: %s]\n", event.ToolCall.Name)
			}
		case agent.EventToolResult:
			fmt.Printf("[tool result received]\n")
		case agent.EventDone:
			fmt.Println()
		case agent.EventError:
			log.Fatalf("agent error: %s", event.Text)
		}
	}
}
```

The loop variables are `(agent.Event, error)` — `err` is the second range value. That is the whole streaming API. There is no `stream.Range`, no `stream.Close()`, no channel to drain. See [`agent/base.go:87`](../../agent/base.go) for the `Stream` signature and [`agent/agent.go:71`](../../agent/agent.go) for the `Event` type.

To wire this as the main entry, replace `main()` in `main.go` with a call to `RunStreaming()`, or move `RunStreaming` into `main` directly.

## 6. What just happened?

You composed three layers of the framework: Layer 3 (`llm`, `tool`) provided the capabilities, Layer 1 (`core`, `config`) provided the typed primitives, and Layer 6 (`agent`) tied them together with the default ReAct planner — see [DOC-02 Core Primitives](../architecture/02-core-primitives.md) for `iter.Seq2` streaming and [DOC-05 Agent Anatomy](../architecture/05-agent-anatomy.md) for the persona → planner → executor pipeline your `agent.New` call set up. The OpenAI provider registered itself in `init()` when you blank-imported it, which is why `llm.New("openai", ...)` resolved without any configuration files.

## 7. Next steps

- [Custom Provider guide](./custom-provider.md) — plug in your own LLM, embedder, or tool using the same registry pattern.
- [Multi-agent team guide](./multi-agent-team.md) — compose several agents via supervisor, handoff, or scatter-gather.
- [Deploy on Docker guide](./deploy-docker.md) — package this agent behind an HTTP server for real use.

## Common mistakes

- **`provider "openai" not found`** — you forgot the blank import `_ "github.com/lookatitude/beluga-ai/llm/providers/openai"`. Without it, `init()` never runs and the registry is empty.
- **Passing a `map[string]any` to `llm.New`** — `llm.New` takes `config.ProviderConfig`, a struct. Use the named fields (`Provider`, `APIKey`, `Model`).
- **Calling `agent.NewLLMAgent`** — that constructor does not exist. Use `agent.New(id, opts...)`.
- **Using a `for _, ev := range stream.Range` loop** — the stream is `iter.Seq2[Event, error]`. Range over it with two variables: `for event, err := range a.Stream(ctx, input)`.
