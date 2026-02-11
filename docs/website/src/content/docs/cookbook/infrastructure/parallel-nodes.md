---
title: "Parallel Node Execution in Graphs"
description: "Execute independent graph nodes in parallel with dependency resolution and concurrency control."
---

# Parallel Node Execution in Graphs

## Problem

Orchestration graphs often contain nodes that are independent of each other. For example, a research agent might need to query three different data sources, generate embeddings for a document, and validate user permissions. These operations have no dependencies and can execute simultaneously, yet sequential execution forces them to run one after another.

The cost of sequential execution grows linearly with the number of independent nodes. If you have five independent nodes that each take one second, sequential execution takes five seconds. Parallel execution takes one second. In production systems with dozens of nodes, this difference becomes critical for user experience and system throughput.

The challenge is identifying which nodes can run in parallel while respecting dependencies. If node C depends on outputs from nodes A and B, C must wait for both to complete. But if nodes A and B are independent, they can run concurrently. You need dependency resolution that determines execution order while maximizing parallelism.

Additionally, unbounded parallelism can overwhelm resources. Running 100 nodes simultaneously might exceed connection pool limits, exhaust memory, or trigger rate limits. You need concurrency control that limits simultaneous execution while still achieving significant speedup.

## Solution

Parallel graph execution solves this through topological execution with dependency tracking. The executor builds a dependency graph that maps each node to its prerequisites. It then repeatedly identifies nodes that are ready to execute (all dependencies satisfied) and runs them concurrently.

The ready node identification algorithm is key to this design. On each iteration, scan all nodes to find those whose dependencies are all marked as executed. These nodes form a wave of parallel execution. Launch them concurrently, wait for all to complete, then repeat. This pattern continues until all nodes have executed or a failure occurs.

Semaphore-based concurrency control limits simultaneous execution without forcing sequential processing. The semaphore acts as a token pool. Before executing a node, acquire a token; after execution, release it. If all tokens are in use, the goroutine blocks until one becomes available. This provides backpressure without complicated scheduling logic.

Per-node result storage enables dependent nodes to access parent outputs. As each node completes, its result is stored in a map keyed by node ID. When a dependent node executes, it can retrieve inputs from this map. This design decouples result passing from execution order, simplifying the implementation.

The critical insight is that even with dependencies, most graphs have significant parallelizable sections. Identifying and exploiting this parallelism dramatically reduces total execution time without changing graph semantics.

Error handling is fail-fast by design. If any node in a wave fails, the entire execution terminates immediately. This prevents wasted work executing nodes whose outputs will be discarded due to the earlier failure.

## Code Example

```go
package main

import (
    "context"
    "fmt"
    "log"
    "sync"

    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"

    "github.com/lookatitude/beluga-ai/core"
)

var tracer = otel.Tracer("beluga.orchestration.parallel")

// GraphNode represents a node in the execution graph
type GraphNode struct {
    ID        string
    Runnable  core.Runnable
    DependsOn []string
}

// ParallelGraphExecutor executes graph nodes in parallel
type ParallelGraphExecutor struct {
    nodes         map[string]*GraphNode
    results       map[string]interface{}
    mu            sync.RWMutex
    maxConcurrent int
}

// NewParallelGraphExecutor creates a new parallel executor
func NewParallelGraphExecutor(maxConcurrent int) *ParallelGraphExecutor {
    return &ParallelGraphExecutor{
        nodes:         make(map[string]*GraphNode),
        results:       make(map[string]interface{}),
        maxConcurrent: maxConcurrent,
    }
}

// AddNode adds a node to the graph
func (pge *ParallelGraphExecutor) AddNode(node *GraphNode) {
    pge.mu.Lock()
    defer pge.mu.Unlock()
    pge.nodes[node.ID] = node
}

// Execute executes the graph with parallel node execution
func (pge *ParallelGraphExecutor) Execute(ctx context.Context) (map[string]interface{}, error) {
    ctx, span := tracer.Start(ctx, "parallel_executor.execute")
    defer span.End()

    span.SetAttributes(
        attribute.Int("node_count", len(pge.nodes)),
        attribute.Int("max_concurrent", pge.maxConcurrent),
    )

    // Build dependency graph
    dependencyGraph := pge.buildDependencyGraph()

    // Execute nodes respecting dependencies
    executed := make(map[string]bool)
    semaphore := make(chan struct{}, pge.maxConcurrent)

    for len(executed) < len(pge.nodes) {
        // Find nodes ready to execute (all dependencies satisfied)
        ready := pge.findReadyNodes(dependencyGraph, executed)

        if len(ready) == 0 && len(executed) < len(pge.nodes) {
            return nil, fmt.Errorf("circular dependency or deadlock detected")
        }

        // Execute ready nodes in parallel
        var wg sync.WaitGroup
        errCh := make(chan error, len(ready))

        for _, nodeID := range ready {
            wg.Add(1)
            go func(id string) {
                defer wg.Done()

                // Acquire semaphore
                semaphore <- struct{}{}
                defer func() { <-semaphore }()

                // Execute node
                node := pge.nodes[id]
                result, err := node.Runnable.Invoke(ctx, nil)
                if err != nil {
                    errCh <- fmt.Errorf("node %s failed: %w", id, err)
                    return
                }

                // Store result
                pge.mu.Lock()
                pge.results[id] = result
                executed[id] = true
                pge.mu.Unlock()

                span.SetAttributes(attribute.String("node.completed", id))
            }(nodeID)
        }

        wg.Wait()
        close(errCh)

        // Check for errors
        if len(errCh) > 0 {
            err := <-errCh
            span.RecordError(err)
            span.SetStatus(trace.StatusError, err.Error())
            return nil, err
        }
    }

    span.SetStatus(trace.StatusOK, "all nodes executed")
    return pge.results, nil
}

// buildDependencyGraph builds the dependency graph
func (pge *ParallelGraphExecutor) buildDependencyGraph() map[string][]string {
    graph := make(map[string][]string)
    for id, node := range pge.nodes {
        graph[id] = node.DependsOn
    }
    return graph
}

// findReadyNodes finds nodes whose dependencies are all satisfied
func (pge *ParallelGraphExecutor) findReadyNodes(dependencyGraph map[string][]string, executed map[string]bool) []string {
    ready := []string{}

    for nodeID, deps := range dependencyGraph {
        if executed[nodeID] {
            continue
        }

        // Check if all dependencies are satisfied
        allSatisfied := true
        for _, dep := range deps {
            if !executed[dep] {
                allSatisfied = false
                break
            }
        }

        if allSatisfied {
            ready = append(ready, nodeID)
        }
    }

    return ready
}

func main() {
    ctx := context.Background()

    // Create executor
    executor := NewParallelGraphExecutor(5)

    // Add nodes
    executor.AddNode(&GraphNode{
        ID:        "node1",
        Runnable:  &MockRunnable{},
        DependsOn: []string{},
    })

    executor.AddNode(&GraphNode{
        ID:        "node2",
        Runnable:  &MockRunnable{},
        DependsOn: []string{},
    })

    executor.AddNode(&GraphNode{
        ID:        "node3",
        Runnable:  &MockRunnable{},
        DependsOn: []string{"node1", "node2"},
    })

    // Execute
    results, err := executor.Execute(ctx)
    if err != nil {
        log.Fatalf("Execution failed: %v", err)
    }

    fmt.Printf("Executed %d nodes\n", len(results))
}

type MockRunnable struct{}

func (m *MockRunnable) Invoke(ctx context.Context, input any, options ...core.Option) (any, error) {
    return "result", nil
}

func (m *MockRunnable) Batch(ctx context.Context, inputs []any, options ...core.Option) ([]any, error) {
    return nil, nil
}

func (m *MockRunnable) Stream(ctx context.Context, input any, options ...core.Option) (<-chan any, error) {
    return nil, nil
}
```

## Explanation

1. **Dependency resolution enables safe parallelism** — Building an explicit dependency graph and checking which nodes are ready to execute ensures correctness while maximizing parallelism. The algorithm naturally handles complex dependency patterns: chains, diamonds, independent subgraphs. You don't need special cases for different graph shapes. The same ready-node identification works for all structures.

2. **Semaphore-based concurrency control prevents resource exhaustion** — The semaphore limits concurrent goroutines to a configured maximum, providing backpressure without blocking all parallelism. This is essential when nodes use limited resources like database connections or external API calls. Without this limit, launching hundreds of nodes simultaneously could exhaust connection pools or trigger rate limits, causing failures unrelated to business logic.

3. **Topological execution with parallelism** — Nodes execute in topological order (dependencies before dependents), but the algorithm identifies and exploits opportunities for parallelism within that ordering. When multiple nodes become ready simultaneously, they execute concurrently rather than arbitrarily sequentially. This provides the speedup benefits of parallelism while maintaining the correctness guarantees of topological sorting.

4. **Wave-based execution simplifies coordination** — Rather than continuously launching nodes as dependencies complete, the executor processes ready nodes in waves. Each wave executes all currently-ready nodes concurrently, waits for completion, then identifies the next wave. This simplification makes the code easier to reason about and naturally handles synchronization. You don't need complex coordination logic because wave boundaries provide natural synchronization points.

## Testing

```go
func TestParallelGraphExecutor_ExecutesInParallel(t *testing.T) {
    executor := NewParallelGraphExecutor(5)

    executor.AddNode(&GraphNode{ID: "node1", Runnable: &MockRunnable{}, DependsOn: []string{}})
    executor.AddNode(&GraphNode{ID: "node2", Runnable: &MockRunnable{}, DependsOn: []string{}})

    results, err := executor.Execute(context.Background())
    require.NoError(t, err)
    require.Len(t, results, 2)
}
```

## Variations

### Dynamic Concurrency

Adjust concurrency based on node characteristics:

```go
func (pge *ParallelGraphExecutor) ExecuteWithDynamicConcurrency(ctx context.Context) (map[string]interface{}, error) {
    // Adjust concurrency per node type
}
```

### Node Prioritization

Execute high-priority nodes first:

```go
type GraphNode struct {
    Priority int
    // ... other fields
}
```

## Related Recipes

- [Agents Parallel Step Execution](/cookbook/agents-parallel-execution) — Parallel agent steps
- [Workflow Checkpointing](/cookbook/workflow-checkpoints) — Save and resume workflows
