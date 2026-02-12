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
scatter-gather, supervisors, and blackboard architectures.

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

## Graph

`Graph` is a directed graph of named [core.Runnable] nodes connected by
conditional `Edge` values. Traversal starts at the entry node and follows
matching edges until a terminal node is reached:

```go
g := orchestration.NewGraph()
g.AddNode("classify", classifier)
g.AddNode("math", mathAgent)
g.AddNode("code", codeAgent)
g.AddEdge(orchestration.Edge{From: "classify", To: "math", Condition: isMath})
g.AddEdge(orchestration.Edge{From: "classify", To: "code", Condition: isCode})
g.SetEntry("classify")
result, err := g.Invoke(ctx, input)
```

## Router

`Router` dispatches input to named routes based on a `ClassifierFunc`.
An optional fallback handler catches unrecognized routes:

```go
router := orchestration.NewRouter(classifier).
    AddRoute("math", mathAgent).
    AddRoute("code", codeAgent).
    SetFallback(generalAgent)
result, err := router.Invoke(ctx, input)
```

## Scatter-Gather

`ScatterGather` fans out input to multiple workers concurrently and
aggregates their results via an `AggregatorFunc`:

```go
sg := orchestration.NewScatterGather(aggregator, worker1, worker2, worker3)
sg.WithTimeout(30 * time.Second)
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

## Blackboard

`Blackboard` implements the blackboard architecture pattern: multiple
agents collaborate by reading from and writing to a shared board. Each
round, every agent sees the current board state and produces output.
Execution continues until a `TerminationFunc` returns true or maxRounds
is reached:

```go
bb := orchestration.NewBlackboard(terminationFn, agents...).
    WithMaxRounds(20)
result, err := bb.Invoke(ctx, input)
```

## Middleware and Hooks

`Middleware` wraps a [core.Runnable] to add cross-cutting behavior.
`ApplyMiddleware` composes middlewares so the first in the list executes
first (outermost wrapper).

`Hooks` provides optional callbacks at step boundaries (BeforeStep,
AfterStep), branch transitions (OnBranch), and error handling (OnError).
`ComposeHooks` merges multiple Hooks values, calling callbacks in order.
