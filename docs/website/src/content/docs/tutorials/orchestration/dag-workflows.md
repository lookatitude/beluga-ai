---
title: DAG-Based Workflow Orchestration
description: Build directed acyclic graph workflows with conditional branching, parallel execution, and state management using the orchestration package.
---

Complex AI workflows rarely follow a straight line. They branch on conditions, fan out to parallel workers, and converge results. The `orchestration` package provides a `Graph` type for building directed acyclic graphs (DAGs) of `core.Runnable` nodes connected by conditional edges. Unlike event-driven architectures where the execution path emerges at runtime, DAGs define the full topology upfront -- every possible path through the graph is visible in the code. This makes them easier to test, debug, and reason about for deterministic workflows where the branching logic is known at build time.

## What You Will Build

A multi-step data processing pipeline that loads data, branches to different analyzers based on conditions, runs parallel analysis, and aggregates results. Along the way you will use `Graph`, `Edge`, `ScatterGather`, and `Chain`.

## Prerequisites

- Familiarity with the `core.Runnable` interface
- Basic understanding of the `orchestration` package

## Core Concepts

### Graph Structure

A `Graph` is composed of named nodes (each a `core.Runnable`) connected by directed `Edge` values. Traversal starts at a configured entry node and follows matching edges until a terminal node (one with no outgoing edges) is reached. The graph itself implements `core.Runnable`, which means it can be embedded as a node inside another graph -- enabling hierarchical composition of complex workflows.

```go
import "github.com/lookatitude/beluga-ai/orchestration"

// Edge has From, To, and an optional Condition function.
// If Condition is nil, the edge is unconditional (always taken).
edge := orchestration.Edge{
    From:      "loader",
    To:        "analyzer",
    Condition: nil, // unconditional
}
```

### Conditional Branching

When multiple edges leave a node, the first edge whose `Condition` returns `true` is taken. This enables if/else routing through the graph. Conditions are evaluated against the output of the source node, so branching decisions are data-driven rather than hardcoded.

## Step 1: Define Runnable Nodes

Each node in the graph is a `core.Runnable` -- the same interface used throughout the framework for composable processing steps. The `runnableFunc` wrapper adapts a plain function into a `Runnable`, implementing both `Invoke` (synchronous) and `Stream` (iterator-based) methods. The streaming implementation produces a single result, which is sufficient for most graph nodes; nodes that need true streaming (such as LLM generation) would implement a full streaming producer.

```go
package main

import (
    "context"
    "fmt"
    "iter"
    "strings"

    "github.com/lookatitude/beluga-ai/core"
    "github.com/lookatitude/beluga-ai/orchestration"
)

// runnableFunc wraps a function as a core.Runnable.
type runnableFunc struct {
    fn func(ctx context.Context, input any) (any, error)
}

func (r *runnableFunc) Invoke(ctx context.Context, input any, _ ...core.Option) (any, error) {
    return r.fn(ctx, input)
}

func (r *runnableFunc) Stream(ctx context.Context, input any, opts ...core.Option) iter.Seq2[any, error] {
    return func(yield func(any, error) bool) {
        result, err := r.Invoke(ctx, input, opts...)
        yield(result, err)
    }
}

func wrap(fn func(ctx context.Context, input any) (any, error)) core.Runnable {
    return &runnableFunc{fn: fn}
}
```

## Step 2: Build the Graph

Graph construction follows a builder pattern: add nodes, add edges (with optional conditions), and set the entry point. The conditional edges create branching logic -- here, positive sentiment routes to a summary node while other sentiments route to a detailed analysis node. Each edge's condition function receives the output of the source node, enabling data-driven routing decisions.

The error handling on every `AddNode` and `AddEdge` call is important: the graph validates that node names are unique and that edges reference existing nodes, catching configuration errors at build time rather than at runtime.

```go
func buildPipeline() (*orchestration.Graph, error) {
    g := orchestration.NewGraph()

    // Add nodes.
    if err := g.AddNode("loader", wrap(func(ctx context.Context, input any) (any, error) {
        query := input.(string)
        return map[string]any{"query": query, "data": "loaded content for: " + query}, nil
    })); err != nil {
        return nil, err
    }

    if err := g.AddNode("sentiment", wrap(func(ctx context.Context, input any) (any, error) {
        data := input.(map[string]any)
        data["sentiment"] = "positive"
        return data, nil
    })); err != nil {
        return nil, err
    }

    if err := g.AddNode("summary", wrap(func(ctx context.Context, input any) (any, error) {
        data := input.(map[string]any)
        data["summary"] = "Summary of: " + data["data"].(string)
        return data, nil
    })); err != nil {
        return nil, err
    }

    if err := g.AddNode("detailed", wrap(func(ctx context.Context, input any) (any, error) {
        data := input.(map[string]any)
        data["report"] = "Detailed analysis of: " + data["data"].(string)
        return data, nil
    })); err != nil {
        return nil, err
    }

    // Add edges with conditional branching.
    if err := g.AddEdge(orchestration.Edge{From: "loader", To: "sentiment"}); err != nil {
        return nil, err
    }

    // Branch: positive sentiment goes to summary, otherwise to detailed analysis.
    if err := g.AddEdge(orchestration.Edge{
        From: "sentiment",
        To:   "summary",
        Condition: func(v any) bool {
            data := v.(map[string]any)
            return data["sentiment"] == "positive"
        },
    }); err != nil {
        return nil, err
    }

    if err := g.AddEdge(orchestration.Edge{
        From: "sentiment",
        To:   "detailed",
        Condition: func(v any) bool {
            data := v.(map[string]any)
            return data["sentiment"] != "positive"
        },
    }); err != nil {
        return nil, err
    }

    // Set entry point.
    if err := g.SetEntry("loader"); err != nil {
        return nil, err
    }

    return g, nil
}
```

## Step 3: Execute the Graph

Since the graph implements `core.Runnable`, execution uses the same `Invoke` method as any other runnable component. The graph traverses from the entry node, evaluating edge conditions at each step, until it reaches a terminal node. The output of the terminal node becomes the output of the entire graph.

```go
func main() {
    g, err := buildPipeline()
    if err != nil {
        fmt.Printf("build error: %v\n", err)
        return
    }

    ctx := context.Background()
    result, err := g.Invoke(ctx, "quarterly revenue data")
    if err != nil {
        fmt.Printf("execution error: %v\n", err)
        return
    }

    data := result.(map[string]any)
    fmt.Printf("Result: %v\n", data)
}
```

## Step 4: Parallel Execution with ScatterGather

For fan-out/fan-in patterns where multiple independent analyses need to run concurrently, `ScatterGather` distributes the same input to multiple workers and collects their results. The aggregator function receives all worker results and combines them into a single output. This is more efficient than sequential execution because all workers run in parallel goroutines, and it provides cleaner composition than manually managing goroutines and WaitGroups.

```go
func buildParallelPipeline() core.Runnable {
    metricsWorker := wrap(func(ctx context.Context, input any) (any, error) {
        return "metrics: revenue up 15%", nil
    })

    sentimentWorker := wrap(func(ctx context.Context, input any) (any, error) {
        return "sentiment: generally positive", nil
    })

    aggregator := func(results []any) (any, error) {
        var parts []string
        for _, r := range results {
            parts = append(parts, r.(string))
        }
        return strings.Join(parts, " | "), nil
    }

    return orchestration.NewScatterGather(aggregator, metricsWorker, sentimentWorker)
}
```

Execute the scatter-gather pipeline:

```go
func main() {
    sg := buildParallelPipeline()

    ctx := context.Background()
    result, err := sg.Invoke(ctx, "analyze Q4 data")
    if err != nil {
        fmt.Printf("error: %v\n", err)
        return
    }

    fmt.Println(result)
    // Output: metrics: revenue up 15% | sentiment: generally positive
}
```

## Step 5: Composing Chains with Graphs

`Chain` creates a sequential pipeline of runnables where each step's output becomes the next step's input. Since chains implement `core.Runnable`, they can be embedded as nodes inside a graph -- this enables a powerful composition model where a graph node performs multiple sequential transformations internally. Here, the preprocessing chain normalizes the input before the analysis node processes it.

```go
func buildComposedPipeline() (*orchestration.Graph, error) {
    preprocessChain := orchestration.Chain(
        wrap(func(ctx context.Context, input any) (any, error) {
            return strings.ToLower(input.(string)), nil
        }),
        wrap(func(ctx context.Context, input any) (any, error) {
            return strings.TrimSpace(input.(string)), nil
        }),
    )

    g := orchestration.NewGraph()
    if err := g.AddNode("preprocess", preprocessChain); err != nil {
        return nil, err
    }
    if err := g.AddNode("analyze", wrap(func(ctx context.Context, input any) (any, error) {
        return "analyzed: " + input.(string), nil
    })); err != nil {
        return nil, err
    }

    if err := g.AddEdge(orchestration.Edge{From: "preprocess", To: "analyze"}); err != nil {
        return nil, err
    }
    if err := g.SetEntry("preprocess"); err != nil {
        return nil, err
    }

    return g, nil
}
```

## Streaming Graph Traversal

The graph supports streaming via `iter.Seq2`, consistent with Beluga AI's streaming-first design. When streaming, the terminal node (the last node with no outgoing edges) produces an iterator rather than a single result, enabling real-time output for LLM-backed nodes within the graph.

```go
ctx := context.Background()
for val, err := range g.Stream(ctx, "input data") {
    if err != nil {
        fmt.Printf("stream error: %v\n", err)
        break
    }
    fmt.Printf("chunk: %v\n", val)
}
```

## Verification

1. Build a graph with the pattern: `Start -> A -> (B or C) -> End`.
2. Run the graph with different inputs to trigger both branches.
3. Verify that the correct branch executed by inspecting the result.
4. Build a `ScatterGather` and confirm all workers execute in parallel.

## Next Steps

- [Temporal Workflows](/tutorials/orchestration/temporal-workflows) -- Durable, long-running workflows that survive process restarts
- [Event-Driven Message Bus](/tutorials/orchestration/message-bus) -- Asynchronous event-driven agent architectures
