---
title: "Agent API — Runtime, Planners, Handoffs"
description: "Agent package API reference for Beluga AI. Agent interface, BaseAgent, Executor loop, planner strategies (ReAct, ToT, LATS), and handoffs."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "agent API, Agent interface, Planner, Executor, handoffs, ReAct, BaseAgent, EventBus, Beluga AI, Go, reference"
---

```go
import "github.com/lookatitude/beluga-ai/agent"
```

Package agent provides the agent runtime for the Beluga AI framework.

It defines the core `Agent` interface, a composable `BaseAgent` implementation,
the `Executor` reasoning loop, pluggable `Planner` strategies, agent-to-agent
`Handoff` transfers, lifecycle `Hooks`, `Middleware`, and an `EventBus` for
async messaging.

## Agent Interface

```go
type Agent interface {
    ID() string
    Persona() Persona
    Tools() []tool.Tool
    Children() []Agent
    Invoke(ctx context.Context, input string, opts ...Option) (string, error)
    Stream(ctx context.Context, input string, opts ...Option) iter.Seq2[Event, error]
}
```

Agents are invoked synchronously with `Invoke` or streamed with `Stream`.
`Stream` returns an `iter.Seq2[Event, error]` following Go 1.23+ conventions.

## Creating an Agent

Use `New` with functional options to create a configured `BaseAgent`:

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/lookatitude/beluga-ai/agent"
    "github.com/lookatitude/beluga-ai/llm"
    "github.com/lookatitude/beluga-ai/tool"
)

func main() {
    ctx := context.Background()

    var model llm.ChatModel // initialize via llm.New(...)
    var tools []tool.Tool   // build with tool.NewFuncTool(...)

    a := agent.New("assistant",
        agent.WithLLM(model),
        agent.WithTools(tools),
        agent.WithPersona(agent.Persona{
            Role: "senior software engineer",
            Goal: "help users write clean, idiomatic Go code",
        }),
        agent.WithMaxIterations(10),
    )

    // Synchronous invocation
    result, err := a.Invoke(ctx, "What is 2+2?")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(result)

    // Streaming invocation
    for event, err := range a.Stream(ctx, "Explain goroutines") {
        if err != nil {
            log.Fatal(err)
        }
        if event.Type == agent.EventText {
            fmt.Print(event.Text)
        }
    }
}
```

## Agent Options

`New` accepts `Option` functional options:

| Option | Description |
|---|---|
| `WithLLM(model llm.ChatModel)` | Language model for reasoning. |
| `WithTools(tools []tool.Tool)` | Tools available to the agent. |
| `WithPersona(p Persona)` | Role, Goal, Backstory, and Traits. |
| `WithMaxIterations(n int)` | Max reasoning loop iterations (default: 10). |
| `WithTimeout(d time.Duration)` | Max execution duration (default: 5 minutes). |
| `WithHandoffs(handoffs []Handoff)` | Agent-to-agent transfer targets. |
| `WithPlanner(p Planner)` | Custom planner instance (bypasses registry). |
| `WithPlannerName(name string)` | Select planner by registry name (default: `"react"`). |
| `WithHooks(h Hooks)` | Lifecycle hooks. |
| `WithMemory(m Memory)` | Memory backend for conversation persistence. |
| `WithChildren(children []Agent)` | Child agents for orchestration. |
| `WithMetadata(meta map[string]any)` | Arbitrary metadata. |

## Persona

`Persona` defines agent identity using the Role-Goal-Backstory framework.
It is converted to a system message at invocation time:

```go
persona := agent.Persona{
    Role:      "data analyst",
    Goal:      "provide clear insights from raw data",
    Backstory: "10 years of experience in business intelligence",
    Traits:    []string{"concise", "data-driven"},
}
```

`Persona.ToSystemMessage()` returns a `*schema.SystemMessage`. It returns nil
if all fields are empty.

## Events

`Stream` emits `Event` values. Each event has a `Type` that determines which
fields are populated:

| `EventType` | Constant | Populated fields |
|---|---|---|
| `"text"` | `EventText` | `Text`, `AgentID` |
| `"tool_call"` | `EventToolCall` | `ToolCall`, `AgentID` |
| `"tool_result"` | `EventToolResult` | `ToolResult`, `AgentID` |
| `"handoff"` | `EventHandoff` | `AgentID`, `Metadata` |
| `"done"` | `EventDone` | `AgentID` |
| `"error"` | `EventError` | `Text` (error message), `AgentID` |

```go
for event, err := range a.Stream(ctx, input) {
    if err != nil {
        log.Fatal(err)
    }
    switch event.Type {
    case agent.EventText:
        fmt.Print(event.Text)
    case agent.EventToolCall:
        fmt.Printf("calling tool: %s\n", event.ToolCall.Name)
    case agent.EventDone:
        fmt.Println("\ndone")
    }
}
```

## Planner Interface

The `Planner` interface defines reasoning strategies. Planners decide what
actions the agent takes and can replan based on observations:

```go
type Planner interface {
    Plan(ctx context.Context, state PlannerState) ([]Action, error)
    Replan(ctx context.Context, state PlannerState) ([]Action, error)
}
```

`PlannerState` carries the original input, conversation history, available
tools, previous observations, iteration count, and planner-specific metadata.

`Action` has a `Type` (`ActionTool`, `ActionRespond`, `ActionFinish`,
`ActionHandoff`) and type-specific fields.

## Built-in Planners

Built-in planner strategies are available by name via the planner registry:

| Name | Description |
|---|---|
| `"react"` | Reasoning + Acting. Default strategy. Interprets tool calls or text responses from the LLM. |
| `"reflexion"` | Actor-Evaluator-Reflector loop. Generates, evaluates, and reflects when below threshold. |
| `"self-discover"` | SELECT → ADAPT → IMPLEMENT. Discovers task-specific reasoning modules. |
| `"tree-of-thought"` | BFS/DFS tree search over reasoning paths. Evaluates candidate branches. |
| `"graph-of-thought"` | Arbitrary graph topology with merge, split, loop, and aggregate operations. |
| `"lats"` | Language Agent Tree Search using Monte Carlo Tree Search (MCTS) with UCT selection. |
| `"moa"` | Mixture of Agents with parallel LLM layers and a final aggregator. |

Select a planner by name or provide a custom implementation:

```go
// By registry name
a := agent.New("solver", agent.WithLLM(model), agent.WithPlannerName("tree-of-thought"))

// Direct instance
a := agent.New("solver",
    agent.WithLLM(model),
    agent.WithPlanner(myCustomPlanner),
)
```

## Planner Registry

Custom planners register via `RegisterPlanner` and are created via `NewPlanner`.
Use `ListPlanners` for discovery:

```go
package main

import (
    "github.com/lookatitude/beluga-ai/agent"
)

func init() {
    agent.RegisterPlanner("custom", func(cfg agent.PlannerConfig) (agent.Planner, error) {
        return NewCustomPlanner(cfg.LLM), nil
    })
}

func listAvailable() []string {
    return agent.ListPlanners()
    // ["custom", "graph-of-thought", "lats", "moa", "react", "reflexion", "self-discover", "tree-of-thought"]
}
```

`PlannerConfig` carries the `LLM llm.ChatModel` and an `Extra map[string]any`
for planner-specific configuration.

## Handoffs

Handoffs enable agent-to-agent transfers. They are automatically converted
to tools named `transfer_to_{id}` that appear in the LLM's tool list:

```go
package main

import (
    "github.com/lookatitude/beluga-ai/agent"
    "github.com/lookatitude/beluga-ai/llm"
)

func buildTeam(model llm.ChatModel) agent.Agent {
    researcher := agent.New("researcher", agent.WithLLM(model))
    writer := agent.New("writer", agent.WithLLM(model))

    return agent.New("manager",
        agent.WithLLM(model),
        agent.WithHandoffs([]agent.Handoff{
            agent.HandoffTo(researcher, "Hand off research tasks"),
            agent.HandoffTo(writer, "Hand off writing tasks"),
        }),
    )
}
```

`HandoffTo(target Agent, description string) Handoff` is the convenience
constructor. The full `Handoff` struct also supports `InputFilter`,
`OnHandoff`, and `IsEnabled` for conditional and filtered handoffs.

`HandoffsToTools(handoffs []Handoff) []tool.Tool` converts handoffs to
`tool.Tool` instances. This is called automatically by `BaseAgent`.

## Hooks

`Hooks` provides lifecycle callbacks invoked during execution. All fields are
optional; nil hooks are skipped. Compose multiple `Hooks` values with
`ComposeHooks`:

```go
hooks := agent.Hooks{
    OnStart: func(ctx context.Context, input string) error {
        log.Printf("agent started: %s", input)
        return nil
    },
    OnToolCall: func(ctx context.Context, call agent.ToolCallInfo) error {
        log.Printf("calling tool: %s", call.Name)
        return nil
    },
    OnEnd: func(ctx context.Context, result string, err error) {
        log.Printf("agent finished: err=%v", err)
    },
}

a := agent.New("assistant", agent.WithLLM(model), agent.WithHooks(hooks))
```

Available hook fields: `OnStart`, `OnEnd`, `OnError`, `BeforePlan`, `AfterPlan`,
`BeforeAct`, `AfterAct`, `OnToolCall`, `OnToolResult`, `OnIteration`,
`OnHandoff`, `BeforeGenerate`, `AfterGenerate`.

## Middleware

`Middleware` wraps an `Agent` to add cross-cutting concerns. Applied via
`ApplyMiddleware` — the first middleware in the list is the outermost wrapper
and executes first:

```go
wrapped := agent.ApplyMiddleware(myAgent, tracingMiddleware, loggingMiddleware)
```

`Middleware` has the signature `func(Agent) Agent`.

## Event Bus

`EventBus` enables agent-to-agent async messaging via publish/subscribe.
`InMemoryBus` provides an in-process implementation:

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/lookatitude/beluga-ai/agent"
)

func main() {
    ctx := context.Background()
    bus := agent.NewInMemoryBus()

    sub, err := bus.Subscribe(ctx, "updates", func(e agent.AgentEvent) {
        fmt.Printf("event from %s: %v\n", e.SourceID, e.Payload)
    })
    if err != nil {
        log.Fatal(err)
    }
    defer sub.Unsubscribe()

    err = bus.Publish(ctx, "updates", agent.AgentEvent{
        Type:     "status",
        SourceID: "agent-1",
        Payload:  "processing complete",
    })
    if err != nil {
        log.Fatal(err)
    }
}
```

`Subscribe` returns a `Subscription` with an `Unsubscribe() error` method.

## Related

- [`llm`](/docs/api-reference/llm-agents/llm) — ChatModel interface agents use for reasoning
- [`tool`](/docs/api-reference/llm-agents/tool) — Tool interface and FuncTool
- [`core`](/docs/api-reference/foundation/core) — Runnable, errors, context helpers
- `docs/concepts.md` — Architecture decisions
