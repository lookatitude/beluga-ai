---
title: "Parallel Node Execution in Graphs"
description: "Recipe for executing independent orchestration graph nodes in parallel with Go dependency resolution and concurrency control using Beluga AI."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Beluga AI, parallel graph nodes, Go orchestration, dependency resolution, concurrency control, DAG execution, graph parallelism"
---

## Problem

Orchestration graphs often contain nodes that are independent of each other. If you have five independent nodes that each take one second, sequential execution takes five seconds. Parallel execution takes one second. The challenge is identifying which nodes can run simultaneously while respecting dependencies between others.

Additionally, unbounded parallelism can overwhelm resources — running 100 nodes simultaneously might exhaust connection pool limits or trigger rate limits. You need concurrency control that limits simultaneous execution while still achieving significant speedup.

## Solution

Build an explicit dependency map and execute ready nodes (all dependencies satisfied) in concurrent waves, using a semaphore to cap parallelism. Each wave launches all currently-ready nodes concurrently, waits for completion, then identifies the next wave.

## Why This Matters

Even in graphs with dependencies, most structures have significant parallelizable sections. Identifying and exploiting this parallelism dramatically reduces total execution time. Error handling is fail-fast by design: if any node in a wave fails, the entire execution terminates immediately, preventing wasted work on nodes whose outputs will be discarded.

## Code Example

```go
package main

import (
	"context"
	"fmt"
	"iter"
	"log/slog"
	"sync"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/lookatitude/beluga-ai/core"
)

var tracer = otel.Tracer("beluga.orchestration.parallel")

// GraphNode represents a node in the execution graph.
type GraphNode struct {
	ID        string
	Runnable  core.Runnable
	DependsOn []string // IDs of nodes that must complete before this one runs.
}

// ParallelGraphExecutor executes graph nodes in topological order with parallelism.
type ParallelGraphExecutor struct {
	nodes         map[string]*GraphNode
	mu            sync.RWMutex
	maxConcurrent int
}

// NewParallelGraphExecutor creates a new executor limited to maxConcurrent goroutines.
func NewParallelGraphExecutor(maxConcurrent int) *ParallelGraphExecutor {
	return &ParallelGraphExecutor{
		nodes:         make(map[string]*GraphNode),
		maxConcurrent: maxConcurrent,
	}
}

// AddNode registers a node in the graph.
func (pge *ParallelGraphExecutor) AddNode(node *GraphNode) {
	pge.mu.Lock()
	defer pge.mu.Unlock()
	pge.nodes[node.ID] = node
}

// Execute runs all nodes respecting dependencies, returning results keyed by node ID.
func (pge *ParallelGraphExecutor) Execute(ctx context.Context) (map[string]any, error) {
	ctx, span := tracer.Start(ctx, "parallel_executor.execute")
	defer span.End()

	span.SetAttributes(
		attribute.Int("node_count", len(pge.nodes)),
		attribute.Int("max_concurrent", pge.maxConcurrent),
	)

	results := make(map[string]any)
	executed := make(map[string]bool)
	sem := make(chan struct{}, pge.maxConcurrent)

	for len(executed) < len(pge.nodes) {
		ready := pge.readyNodes(executed)
		if len(ready) == 0 {
			return nil, fmt.Errorf("circular dependency or missing node detected")
		}

		var (
			wg    sync.WaitGroup
			mu    sync.Mutex
			errCh = make(chan error, len(ready))
		)

		for _, nodeID := range ready {
			wg.Add(1)
			go func(id string) {
				defer wg.Done()

				sem <- struct{}{}
				defer func() { <-sem }()

				node := pge.nodes[id]
				result, err := node.Runnable.Invoke(ctx, nil)
				if err != nil {
					errCh <- fmt.Errorf("node %s: %w", id, err)
					return
				}

				mu.Lock()
				results[id] = result
				executed[id] = true
				mu.Unlock()

				span.SetAttributes(attribute.String("node.completed", id))
			}(nodeID)
		}

		wg.Wait()
		close(errCh)

		if err := <-errCh; err != nil {
			span.RecordError(err)
			span.SetStatus(trace.StatusError, err.Error())
			return nil, err
		}
	}

	span.SetStatus(trace.StatusOK, "all nodes executed")
	return results, nil
}

// readyNodes returns IDs of nodes whose dependencies are all in executed.
func (pge *ParallelGraphExecutor) readyNodes(executed map[string]bool) []string {
	var ready []string
	for id, node := range pge.nodes {
		if executed[id] {
			continue
		}
		allDone := true
		for _, dep := range node.DependsOn {
			if !executed[dep] {
				allDone = false
				break
			}
		}
		if allDone {
			ready = append(ready, id)
		}
	}
	return ready
}

// --- Example node implementation ---

// echoRunnable returns its input unchanged.
type echoRunnable struct {
	label string
}

func (r *echoRunnable) Invoke(ctx context.Context, input any, opts ...core.Option) (any, error) {
	slog.Info("node executed", "label", r.label)
	return r.label + ":done", nil
}

func (r *echoRunnable) Stream(ctx context.Context, input any, opts ...core.Option) iter.Seq2[any, error] {
	return func(yield func(any, error) bool) {
		result, err := r.Invoke(ctx, input, opts...)
		yield(result, err)
	}
}

func main() {
	ctx := context.Background()
	executor := NewParallelGraphExecutor(5)

	executor.AddNode(&GraphNode{ID: "node1", Runnable: &echoRunnable{label: "node1"}, DependsOn: []string{}})
	executor.AddNode(&GraphNode{ID: "node2", Runnable: &echoRunnable{label: "node2"}, DependsOn: []string{}})
	executor.AddNode(&GraphNode{ID: "node3", Runnable: &echoRunnable{label: "node3"}, DependsOn: []string{"node1", "node2"}})

	results, err := executor.Execute(ctx)
	if err != nil {
		slog.Error("execution failed", "error", err)
		return
	}
	fmt.Printf("Executed %d nodes\n", len(results))
	for id, result := range results {
		fmt.Printf("  %s -> %v\n", id, result)
	}
}
```

## Explanation

1. **Wave-based execution** — On each iteration, `readyNodes` scans all nodes and returns those whose dependencies are all marked complete. All ready nodes launch concurrently in a wave. This naturally handles chains, diamonds, and independent subgraphs without special cases.

2. **Semaphore-based concurrency control** — The `sem` channel limits concurrent goroutines to `maxConcurrent`. A goroutine blocks on `sem <- struct{}{}` until a slot is available. This provides backpressure without forcing fully sequential processing.

3. **Fail-fast error handling** — `errCh` collects the first error from any node in the wave. After `wg.Wait()`, one error is drained and returned immediately. Subsequent waves are never started after a failure.

4. **Cycle detection** — If `readyNodes` returns empty but nodes remain unexecuted, a circular dependency exists. The executor returns an error rather than looping forever.

## Testing

```go
func TestParallelGraphExecutor_ExecutesInParallel(t *testing.T) {
	executor := NewParallelGraphExecutor(5)
	executor.AddNode(&GraphNode{ID: "n1", Runnable: &echoRunnable{label: "n1"}, DependsOn: []string{}})
	executor.AddNode(&GraphNode{ID: "n2", Runnable: &echoRunnable{label: "n2"}, DependsOn: []string{}})
	executor.AddNode(&GraphNode{ID: "n3", Runnable: &echoRunnable{label: "n3"}, DependsOn: []string{"n1", "n2"}})

	results, err := executor.Execute(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 3 {
		t.Errorf("expected 3 results, got %d", len(results))
	}
}

func TestParallelGraphExecutor_DetectsCycle(t *testing.T) {
	executor := NewParallelGraphExecutor(2)
	executor.AddNode(&GraphNode{ID: "a", Runnable: &echoRunnable{label: "a"}, DependsOn: []string{"b"}})
	executor.AddNode(&GraphNode{ID: "b", Runnable: &echoRunnable{label: "b"}, DependsOn: []string{"a"}})

	_, err := executor.Execute(context.Background())
	if err == nil {
		t.Fatal("expected circular dependency error, got nil")
	}
}
```

## Variations

### Dynamic Concurrency

Adjust the semaphore size based on node characteristics (e.g., CPU-bound vs. IO-bound nodes use different limits):

```go
type WeightedNode struct {
	GraphNode
	ConcurrencyWeight int // Tokens this node consumes from the semaphore.
}
```

## Related Recipes

- **[Agents Parallel Step Execution](/docs/recipes/agents/agents-parallel-execution)** — Parallel agent steps using `core.Parallel`
- **[Workflow Checkpointing](./workflow-checkpoints)** — Save and resume workflows
