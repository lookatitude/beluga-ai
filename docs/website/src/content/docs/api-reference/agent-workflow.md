---
title: "Agent Workflows"
description: "Sequential, Parallel, and Loop workflow agents for multi-agent orchestration"
---

```go
import "github.com/lookatitude/beluga-ai/agent/workflow"
```

Package workflow provides deterministic workflow agents that orchestrate
child agents without LLM reasoning. These agents compose child agents in
common execution patterns: sequential pipelines, parallel fan-out, and
iterative loops.

Workflow agents implement the [agent.Agent] interface but do not use tools
or planners directly. Instead, they delegate to child agents and manage
the flow of data between them.

## Sequential Agent

`SequentialAgent` runs child agents in sequence, passing the output of each
as the input to the next. This is useful for multi-step pipelines where
each stage transforms or enriches the result:

```go
researcher := agent.New("researcher", agent.WithLLM(model))
writer := agent.New("writer", agent.WithLLM(model))
editor := agent.New("editor", agent.WithLLM(model))

pipeline := workflow.NewSequentialAgent("content-pipeline",
    []agent.Agent{researcher, writer, editor},
)

result, err := pipeline.Invoke(ctx, "Write about Go concurrency")
```

## Parallel Agent

`ParallelAgent` runs child agents concurrently and collects all results.
Each child receives the same input. Results are concatenated in order:

```go
analyzer := agent.New("analyzer", agent.WithLLM(model))
reviewer := agent.New("reviewer", agent.WithLLM(model))

parallel := workflow.NewParallelAgent("review-team",
    []agent.Agent{analyzer, reviewer},
)

result, err := parallel.Invoke(ctx, "Review this code")
```

## Loop Agent

`LoopAgent` runs a single child agent repeatedly until a condition is met
or the maximum number of iterations is reached. Configure with
`WithLoopMaxIterations` and `WithLoopCondition`:

```go
refiner := agent.New("refiner", agent.WithLLM(model))

loop := workflow.NewLoopAgent("iterative-refiner", refiner,
    workflow.WithLoopMaxIterations(3),
    workflow.WithLoopCondition(func(iteration int, result string) bool {
        return strings.Contains(result, "DONE")
    }),
)

result, err := loop.Invoke(ctx, "Refine this draft")
```

## Streaming

All workflow agents support streaming via the Stream method, which yields
events from each child agent as they execute. Events include text chunks,
tool calls, and completion markers.
