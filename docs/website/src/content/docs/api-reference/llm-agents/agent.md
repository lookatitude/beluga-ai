---
title: "Agent Package"
description: "Agent runtime, BaseAgent, Executor, Planner strategies, handoffs, and event bus"
---

```go
import "github.com/lookatitude/beluga-ai/agent"
```

Package agent provides the agent runtime for the Beluga AI framework.

It defines the core `Agent` interface, a composable `BaseAgent` implementation,
the `Executor` reasoning loop, pluggable `Planner` strategies, agent-to-agent
`Handoff` transfers, lifecycle `Hooks`, `Middleware`, an `EventBus` for async
messaging, and `AgentCard` for A2A protocol discovery.

## Agent Interface

The Agent interface is the primary abstraction for all agents:

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

Agents can be invoked synchronously with Invoke or streamed with Stream.
Stream returns an iter.Seq2[Event, error] following Go 1.23+ conventions.

## Creating an Agent

Use `New` with functional options to create a configured `BaseAgent`:

```go
a := agent.New("assistant",
    agent.WithLLM(model),
    agent.WithTools(tools),
    agent.WithPersona(agent.Persona{
        Role: "senior software engineer",
        Goal: "help users write clean, idiomatic Go code",
    }),
    agent.WithMaxIterations(5),
)

// Synchronous invocation
result, err := a.Invoke(ctx, "What is 2+2?")
if err != nil {
    log.Fatal(err)
}

// Streaming invocation
for event, err := range a.Stream(ctx, "Explain goroutines") {
    if err != nil {
        log.Fatal(err)
    }
    if event.Type == agent.EventText {
        fmt.Print(event.Text)
    }
}
```

## Planner Strategies

The Planner interface defines reasoning strategies. Planners decide what
actions the agent should take and can replan based on observations:

```go
type Planner interface {
    Plan(ctx context.Context, state PlannerState) ([]Action, error)
    Replan(ctx context.Context, state PlannerState) ([]Action, error)
}
```

Built-in planner strategies are registered via the planner registry:

- "react" — [ReActPlanner]: Reasoning + Acting. The default strategy.
  Sends conversation to the LLM and interprets tool calls or text responses.
- "reflexion" — [ReflexionPlanner]: Actor-Evaluator-Reflector loop.
  Generates a response, evaluates it, and reflects if below threshold.
- "self-discover" — [SelfDiscoverPlanner]: SELECT → ADAPT → IMPLEMENT.
  Discovers task-specific reasoning modules and composes them.
- "tree-of-thought" — [ToTPlanner]: BFS/DFS tree search over reasoning paths.
  Generates candidates, evaluates promise, and expands best branches.
- "graph-of-thought" — [GoTPlanner]: Arbitrary graph topology with merge,
  split, loop, and aggregate operations over a [ThoughtGraph].
- "lats" — [LATSPlanner]: Language Agent Tree Search using Monte Carlo
  Tree Search (MCTS) with UCT selection, expansion, and backpropagation.
- "moa" — [MoAPlanner]: Mixture of Agents with parallel LLM layers
  and a final aggregator for synthesis.

Select a planner by name or provide a custom implementation:

```go
// By name (via registry)
a := agent.New("solver", agent.WithPlannerName("tree-of-thought"))

// Direct planner instance
a := agent.New("solver", agent.WithPlanner(
    agent.NewToTPlanner(model,
        agent.WithBranchFactor(5),
        agent.WithMaxDepth(3),
    ),
))
```

## Executor Loop

The `Executor` runs the Plan → Act → Observe reasoning loop. It is
planner-agnostic: the same loop works for any planner strategy:

```go
executor := agent.NewExecutor(
    agent.WithExecutorPlanner(planner),
    agent.WithExecutorMaxIterations(10),
    agent.WithExecutorTimeout(5 * time.Minute),
)
```

## Handoffs

Handoffs enable agent-to-agent transfers. They are automatically converted
to tools (transfer_to_{id}) that appear in the LLM's tool list:

```go
researcher := agent.New("researcher", agent.WithLLM(model))
writer := agent.New("writer", agent.WithLLM(model))

manager := agent.New("manager",
    agent.WithLLM(model),
    agent.WithHandoffs([]agent.Handoff{
        agent.HandoffTo(researcher, "Hand off research tasks"),
        agent.HandoffTo(writer, "Hand off writing tasks"),
    }),
)
```

## Hooks

`Hooks` provide lifecycle callbacks invoked at various points during
execution. All fields are optional; nil hooks are skipped. Hooks are
composable via `ComposeHooks`:

```go
hooks := agent.Hooks{
    OnStart: func(ctx context.Context, input string) error {
        log.Printf("Agent started with: %s", input)
        return nil
    },
    OnToolCall: func(ctx context.Context, call agent.ToolCallInfo) error {
        log.Printf("Calling tool: %s", call.Name)
        return nil
    },
}
```

## Middleware

`Middleware` wraps an Agent to add cross-cutting concerns. Applied via
`ApplyMiddleware` in reverse order so the first middleware is outermost:

```go
wrapped := agent.ApplyMiddleware(myAgent, tracingMiddleware, loggingMiddleware)
```

## Event Bus

The `EventBus` interface enables agent-to-agent async messaging via
publish/subscribe. `InMemoryBus` provides an in-process implementation:

```go
bus := agent.NewInMemoryBus()
sub, err := bus.Subscribe(ctx, "updates", func(e agent.AgentEvent) {
    fmt.Printf("Event from %s: %v\n", e.SourceID, e.Payload)
})
```

## Planner Registry

Custom planners can be registered with `RegisterPlanner` and created
with `NewPlanner`. Use `ListPlanners` for discovery:

```go
agent.RegisterPlanner("custom", func(cfg agent.PlannerConfig) (agent.Planner, error) {
    return NewCustomPlanner(cfg.LLM), nil
})

planners := agent.ListPlanners() // ["custom", "graph-of-thought", "lats", "moa", "react", "reflexion", "self-discover", "tree-of-thought"]
```
