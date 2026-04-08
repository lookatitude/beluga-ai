---
title: "Orchestration API — Chain, Graph, Router"
description: "Orchestration package API for Beluga AI. Chain, Graph, Router, ScatterGather, Supervisor, and Blackboard workflow composition patterns."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "orchestration API, Chain, Graph, Router, Supervisor, ScatterGather, Blackboard, workflow, Beluga AI, Go, reference"
---

```go
import "github.com/lookatitude/beluga-ai/orchestration"
```

Package orchestration provides workflow composition patterns for the
Beluga AI framework including chains, directed graphs, routers,
scatter-gather, supervisors, blackboard architectures, agent pipelines,
and handoff orchestration.

All patterns implement [core.Runnable], allowing seamless composition
with the rest of the framework. `Hooks` and `Middleware` provide
extensibility for logging, tracing, and custom cross-cutting concerns.

## Chain

`Chain` composes steps sequentially: the output of step N becomes the
input of step N+1. For streaming, all steps except the last are invoked
synchronously and the last step is streamed:

```go
pipeline := orchestration.Chain(tokenizer, llm, formatter)
result, err := pipeline.Invoke(ctx, input)
```

## Pipeline

`Pipeline` executes a sequence of `agent.Agent` values where the text
output of agent N becomes the string input of agent N+1. It satisfies
`OrchestrationPattern`:

```go
p := orchestration.NewPipeline(researchAgent, writerAgent, reviewerAgent)
result, err := p.Invoke(ctx, "Write a report on Go concurrency")
```

For streaming, leading agents are invoked synchronously and the final
agent is streamed. Events from the final stage are yielded to the caller.

## HandoffOrchestrator

`HandoffOrchestrator` manages peer-to-peer agent transfers. When an
agent emits an `agent.EventHandoff` event (triggered by a
`transfer_to_{name}` tool call), the orchestrator routes control to the
target agent and continues execution. It satisfies `OrchestrationPattern`:

```go
h := orchestration.NewHandoffOrchestrator(triageAgent, billingAgent, supportAgent).
    WithMaxHops(5).
    WithEntry(triageAgent.ID())

result, err := h.Invoke(ctx, "I have a billing question")
```

The first agent in the constructor is used as the entry point by default.
`WithEntry` overrides this. `WithMaxHops` limits the total number of
agent-to-agent transfers (default 10).

## Graph

`Graph` is a directed graph of named [core.Runnable] nodes connected by
conditional `Edge` values. Traversal starts at the entry node and follows
matching edges until a terminal node is reached.

`AddNode`, `AddEdge`, and `SetEntry` all return errors and must be
checked:

```go
import (
    "context"
    "fmt"

    "github.com/lookatitude/beluga-ai/orchestration"
)

g := orchestration.NewGraph()
if err := g.AddNode("classify", classifier); err != nil {
    return fmt.Errorf("add node: %w", err)
}
if err := g.AddNode("math", mathAgent); err != nil {
    return fmt.Errorf("add node: %w", err)
}
if err := g.AddNode("code", codeAgent); err != nil {
    return fmt.Errorf("add node: %w", err)
}
if err := g.AddEdge(orchestration.Edge{From: "classify", To: "math", Condition: isMath}); err != nil {
    return fmt.Errorf("add edge: %w", err)
}
if err := g.AddEdge(orchestration.Edge{From: "classify", To: "code", Condition: isCode}); err != nil {
    return fmt.Errorf("add edge: %w", err)
}
if err := g.SetEntry("classify"); err != nil {
    return fmt.Errorf("set entry: %w", err)
}
result, err := g.Invoke(ctx, input)
```

An `Edge.Condition` of nil is unconditional (always taken). For multiple
matching edges from a node, the first match wins. Traversal depth is
capped at 100 to prevent infinite loops.

## Router

`Router` dispatches input to named routes based on a `ClassifierFunc`.
An optional fallback handler catches unrecognized routes:

```go
classifier := func(ctx context.Context, input any) (string, error) {
    // return route name based on input
    return "math", nil
}

router := orchestration.NewRouter(classifier).
    AddRoute("math", mathAgent).
    AddRoute("code", codeAgent).
    SetFallback(generalAgent)
result, err := router.Invoke(ctx, input)
```

## Scatter-Gather

`ScatterGather` fans out input to multiple workers concurrently and
aggregates their results via an `AggregatorFunc`. The aggregator is a
function, not a Runnable:

```go
aggregator := func(results []any) (any, error) {
    // combine results from all workers
    return strings.Join(toStrings(results), "\n"), nil
}

sg := orchestration.NewScatterGather(aggregator, worker1, worker2, worker3).
    WithTimeout(30 * time.Second)
result, err := sg.Invoke(ctx, input)
```

## Supervisor

`Supervisor` orchestrates multiple agents by delegating work using a
`StrategyFunc`. Built-in strategies include `DelegateBySkill` (keyword
matching against agent personas), `RoundRobin`, and `LoadBalanced`:

```go
sup := orchestration.NewSupervisor(orchestration.RoundRobin(), agents...).
    WithMaxRounds(5)
result, err := sup.Invoke(ctx, input)
```

`StrategyFunc` returns `nil` to signal that execution should stop.
`NewSupervisor` defaults to 1 round; use `WithMaxRounds` to increase it.

## Blackboard

`Blackboard` implements the blackboard architecture pattern: multiple
agents collaborate by reading from and writing to a shared board. Each
round, every agent sees the current board state and produces output.
Execution continues until a `TerminationFunc` returns true or maxRounds
is reached:

```go
terminationFn := func(board map[string]any) bool {
    _, done := board["conclusion"]
    return done
}

bb := orchestration.NewBlackboard(terminationFn, agents...).
    WithMaxRounds(20)
result, err := bb.Invoke(ctx, input)
```

`Set` and `Get` allow direct board manipulation outside the execution loop.

## OrchestrationPattern Interface

All patterns that participate in the pattern registry implement this interface:

```go
type OrchestrationPattern interface {
    core.Runnable
    Name() string
}
```

`Pipeline` and `HandoffOrchestrator` satisfy this interface.

## Middleware and Hooks

`Middleware` wraps a [core.Runnable] to add cross-cutting behavior.
`ApplyMiddleware` composes middlewares so the first in the list executes
first (outermost wrapper).

`Hooks` provides optional callbacks at step boundaries (BeforeStep,
AfterStep), branch transitions (OnBranch), and error handling (OnError).
`ComposeHooks` merges multiple Hooks values, calling callbacks in order.
