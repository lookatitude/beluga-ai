---
title: Agent Package API
description: API documentation for the agent runtime and planners.
---

```go
import "github.com/lookatitude/beluga-ai/agent"
```

Package agent provides the agent runtime with `Agent` interface, `BaseAgent` implementation, `Executor` reasoning loop, pluggable `Planner` implementations, handoffs, and lifecycle hooks.

## Quick Start

```go
a := agent.New("assistant",
    agent.WithLLM(model),
    agent.WithTools(tools),
    agent.WithPersona(agent.Persona{
        Role: "helpful assistant",
        Goal: "help users write clean code",
    }),
)

// Synchronous
result, err := a.Invoke(ctx, "What is 2+2?")

// Streaming
for event, err := range a.Stream(ctx, "Explain Go") {
    if err != nil { break }
    fmt.Println(event.Text)
}
```

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

## BaseAgent

Default agent implementation:

```go
agent := agent.New("myagent",
    agent.WithLLM(model),
    agent.WithTools(searchTool, calcTool),
    agent.WithPersona(agent.Persona{
        Role:      "senior engineer",
        Goal:      "write clean, idiomatic Go code",
        Backstory: "10 years Go experience",
        Traits:    []string{"concise", "precise"},
    }),
    agent.WithPlannerName("react"), // or "reflexion", "tot", "got", etc.
    agent.WithMaxIterations(10),
    agent.WithTimeout(2*time.Minute),
)
```

## Planners

### ReAct (default)

Reasoning + Acting pattern:

```go
planner := agent.NewReActPlanner(model)
```

### Reflexion

Self-reflection with Actor-Evaluator-Reflector:

```go
planner := agent.NewReflexionPlanner(model,
    agent.WithThreshold(0.7),
    agent.WithMaxReflections(3),
)
```

### Tree of Thought (ToT)

Explore multiple reasoning paths:

```go
planner := agent.NewToTPlanner(model,
    agent.WithMaxDepth(3),
    agent.WithBranchFactor(3),
    agent.WithSearchStrategy(agent.StrategyBFS),
)
```

### Graph of Thought (GoT)

Arbitrary graph topology with merge/split/loop:

```go
planner := agent.NewGoTPlanner(model,
    agent.WithController(controller),
    agent.WithMaxOperations(20),
)
```

### LATS

Language Agent Tree Search with MCTS:

```go
planner := agent.NewLATSPlanner(model,
    agent.WithLATSMaxDepth(5),
    agent.WithExpansionWidth(3),
    agent.WithExplorationConstant(1.4),
)
```

### Mixture of Agents

Multiple LLMs in parallel layers:

```go
planner := agent.NewMoAPlanner(defaultModel,
    agent.WithLayers([][]llm.ChatModel{
        {model1, model2, model3}, // layer 1
        {model4, model5},          // layer 2
    }),
    agent.WithAggregator(aggregatorModel),
)
```

### Self-Discover

SELECT → ADAPT → IMPLEMENT reasoning structure:

```go
planner := agent.NewSelfDiscoverPlanner(model,
    agent.WithReasoningModules(agent.DefaultReasoningModules),
)
```

## Handoffs

Transfer control between agents:

```go
specialist := agent.New("specialist", ...)

generalist := agent.New("generalist",
    agent.WithHandoffs([]agent.Handoff{
        agent.HandoffTo(specialist, "For complex technical questions"),
    }),
)

// The LLM can now call transfer_to_specialist tool
```

## Hooks

Inject lifecycle callbacks:

```go
a := agent.New("myagent",
    agent.WithHooks(agent.Hooks{
        OnStart: func(ctx context.Context, input string) error {
            log.Printf("Starting: %s", input)
            return nil
        },
        BeforePlan: func(ctx context.Context, state agent.PlannerState) error {
            log.Printf("Planning iteration %d", state.Iteration)
            return nil
        },
        OnToolCall: func(ctx context.Context, call agent.ToolCallInfo) error {
            log.Printf("Calling tool: %s", call.Name)
            return nil
        },
        OnEnd: func(ctx context.Context, result string, err error) {
            log.Printf("Finished: %v", err)
        },
    }),
)
```

## Middleware

Wrap agents with cross-cutting concerns:

```go
wrapped := agent.ApplyMiddleware(a,
    loggingMiddleware,
    tracingMiddleware,
)
```

## Event Bus

Inter-agent messaging:

```go
bus := agent.NewInMemoryBus()

sub, err := bus.Subscribe(ctx, "alerts", func(event agent.AgentEvent) {
    log.Printf("Alert: %+v", event)
})
defer sub.Unsubscribe()

bus.Publish(ctx, "alerts", agent.AgentEvent{
    Type:     "tool_failed",
    SourceID: "agent-1",
    Payload:  errorDetails,
})
```

## Agent Cards (A2A)

Describe agent capabilities for discovery:

```go
card := agent.BuildCard(a)
// card.Name, card.Description, card.Skills, card.Protocols
```

## See Also

- [LLM Package](./llm.md) for model integration
- [Tool Package](./tool.md) for tool binding
- [Memory Package](./memory.md) for conversation history
