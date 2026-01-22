# Building DAG-based Agents

<!--
Persona: Pair Programmer Colleague
- Treat the reader as a competent peer
- Be helpful, direct, and slightly informal but professional
- Focus on getting results quickly
- Keep it concise and functional
- Provide runnable code examples
-->

## What you will build
In this tutorial, we'll build agents with complex, non-linear workflows using Directed Acyclic Graphs (DAGs). You'll learn how to define nodes, edges, and implement conditional branching and parallel execution.

## Learning Objectives
- ✅ Understand Graph orchestration
- ✅ Define Nodes and Edges
- ✅ Implement conditional branching
- ✅ Parallel execution

## Introduction
Welcome, colleague! Linear chains (`A -> B -> C`) are fine for simple tasks, but real-world AI workflows are messy. They need branching, loops, and parallel steps. Let's look at how to use graphs to orchestrate complex agent behavior.

## Prerequisites

- [Orchestration Basics](../../getting-started/06-orchestration-basics.md)

## Why Graphs?

Chains (`A -> B -> C`) are limited. Real workflows have:
- **Branching**: If A is true, go to B, else C.
- **Parallelism**: Run B and C at the same time, then merge in D.
- **Cycles**: (Technically not DAG, but Graphs support loops) Retry B until valid.

## Step 1: The Orchestrator
```text
import "github.com/lookatitude/beluga-ai/pkg/orchestration"
```

graph, _ := orchestration.NewOrchestrator(config).CreateGraph()
```

## Step 2: Define Nodes

Nodes are just `Runnables` (functions, chains, agents).
mermaid
```mermaid
graph.AddNode("loader", loadDataFunc)
graph.AddNode("analyzer", analyzeFunc)
graph.AddNode("summarizer", summarizeFunc)
graph.AddNode("emailer", emailFunc)
```

## Step 3: Define Edges (Flow)
```mermaid
// Sequential
graph.AddEdge("loader", "analyzer")

// Branching (Conditional Edge)
// Note: This API is conceptual; check specific graph implementation
graph.AddConditionalEdge("analyzer", func(input any) string {
    if input.(Analysis).IsImportant {
        return "emailer"
    }
    return "summarizer"
```
})

```mermaid
graph.AddEdge("summarizer", "emailer")
```

## Step 4: Parallelism

If multiple nodes depend on the same parent, they run in parallel (if supported by executor).
mermaid
```mermaid
graph.AddEdge("loader", "analyzer_metrics")
graph.AddEdge("loader", "analyzer_sentiment")
// Both run after loader
```

## Step 5: State Management

Graphs typically pass a state object around.
```go
type State struct {
    Input    string
    Data     []byte
    Analysis Result
    Summary  string
}
```

Each node accepts `State` and returns `State` (or partial update).

## Verification

1. Construct a flow: `Start -> A -> (B, C) -> D -> End`.
2. Run it.
3. Verify B and C executed (add logs/delays).
4. Verify D received data from both.

## Next Steps

- **[Multi-Agent Orchestration](./agents-multi-agent-orchestration.md)** - Nodes can be Agents!
- **[Temporal Workflows](./orchestration-temporal-workflows.md)** - Durable graphs
