---
title: "Parallel Step Execution"
description: "Recipe for running independent agent steps concurrently in Go with dependency resolution and goroutine management to cut execution time using Beluga AI."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Beluga AI, Go parallel agents, concurrent execution, agent workflow, dependency resolution, goroutine patterns, agent performance"
---

## Problem

You need to execute multiple agent steps in parallel when they don't depend on each other, to reduce total execution time and improve responsiveness of multi-step agent workflows.

## Solution

Use `core.Parallel` for simple fan-out, or implement a dependency-aware executor that identifies independent steps and runs them concurrently. This works because many agent steps (tool calls, data retrieval) have no data dependencies on each other, and Go's concurrency primitives allow safe parallel execution.

## Why This Matters

Agent workflows often consist of steps with varying dependencies. A naive sequential approach executes all steps one after another, even when many could run concurrently. For example, if an agent needs to search a database, call a weather API, and query a calendar service, running these sequentially triples the wall-clock time compared to running them in parallel.

Go's concurrency model (goroutines, `sync.WaitGroup`) is well-suited for this pattern because goroutine creation is cheap, and `sync.RWMutex` protects shared result storage without significant overhead.

## Code Example

### Simple Parallel Execution with `core.Parallel`

For independent steps with the same input, use `core.Parallel` directly:

```go
package main

import (
	"context"
	"fmt"
	"iter"
	"log/slog"

	"github.com/lookatitude/beluga-ai/core"
)

// stepRunnable implements core.Runnable for a named processing step.
type stepRunnable struct {
	name string
}

func (s *stepRunnable) Invoke(ctx context.Context, input any, opts ...core.Option) (any, error) {
	return fmt.Sprintf("%s processed: %v", s.name, input), nil
}

func (s *stepRunnable) Stream(ctx context.Context, input any, opts ...core.Option) iter.Seq2[any, error] {
	return func(yield func(any, error) bool) {
		result, err := s.Invoke(ctx, input, opts...)
		yield(result, err)
	}
}

func main() {
	ctx := context.Background()

	// Create independent processing runnables.
	nodeA := &stepRunnable{name: "A"}
	nodeB := &stepRunnable{name: "B"}
	nodeC := &stepRunnable{name: "C"}

	// Parallel fans out to all runnables and returns []any results.
	parallel := core.Parallel(nodeA, nodeB, nodeC)

	result, err := parallel.Invoke(ctx, "hello")
	if err != nil {
		slog.Error("parallel execution failed", "error", err)
		return
	}

	results, _ := result.([]any)
	for i, r := range results {
		fmt.Printf("Node %d: %v\n", i, r)
	}
}
```

### Dependency-Aware Parallel Executor

For workflows where some steps depend on others, use a dependency graph with topological ordering:

```go
package main

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/lookatitude/beluga-ai/core"
)

// Step represents an agent step with dependencies.
type Step struct {
	ID      string
	Action  core.Runnable
	Depends []string // IDs of steps this step depends on.
}

// ParallelStepExecutor executes steps in parallel respecting dependencies.
type ParallelStepExecutor struct {
	steps   []Step
	results map[string]any
	mu      sync.RWMutex
}

// NewParallelStepExecutor creates a new executor.
func NewParallelStepExecutor(steps []Step) *ParallelStepExecutor {
	return &ParallelStepExecutor{
		steps:   steps,
		results: make(map[string]any),
	}
}

// Execute runs steps in topological order, maximizing parallelism.
func (pse *ParallelStepExecutor) Execute(ctx context.Context) (map[string]any, error) {
	executed := make(map[string]bool)

	for len(executed) < len(pse.steps) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		ready := pse.findReadySteps(executed)
		if len(ready) == 0 {
			return nil, fmt.Errorf("circular dependency detected in step graph")
		}

		var wg sync.WaitGroup
		errCh := make(chan error, len(ready))

		for _, stepID := range ready {
			wg.Add(1)
			go func(id string) {
				defer wg.Done()

				step := pse.findStep(id)
				if step == nil {
					errCh <- fmt.Errorf("step %q not found", id)
					return
				}

				result, err := step.Action.Invoke(ctx, nil)
				if err != nil {
					errCh <- fmt.Errorf("step %q failed: %w", id, err)
					return
				}

				pse.mu.Lock()
				pse.results[id] = result
				executed[id] = true
				pse.mu.Unlock()
			}(stepID)
		}

		wg.Wait()
		close(errCh)

		if err, ok := <-errCh; ok {
			return nil, err
		}
	}

	return pse.results, nil
}

// findReadySteps returns step IDs whose dependencies are all satisfied.
func (pse *ParallelStepExecutor) findReadySteps(executed map[string]bool) []string {
	pse.mu.RLock()
	defer pse.mu.RUnlock()

	var ready []string
	for _, step := range pse.steps {
		if executed[step.ID] {
			continue
		}
		allSatisfied := true
		for _, dep := range step.Depends {
			if !executed[dep] {
				allSatisfied = false
				break
			}
		}
		if allSatisfied {
			ready = append(ready, step.ID)
		}
	}
	return ready
}

// findStep finds a step by ID.
func (pse *ParallelStepExecutor) findStep(id string) *Step {
	for i := range pse.steps {
		if pse.steps[i].ID == id {
			return &pse.steps[i]
		}
	}
	return nil
}

func main() {
	ctx := context.Background()

	mkRunnable := func(name string) core.Runnable {
		return &stepRunnable{name: name}
	}

	steps := []Step{
		{ID: "step1", Action: mkRunnable("step1"), Depends: []string{}},
		{ID: "step2", Action: mkRunnable("step2"), Depends: []string{}},
		{ID: "step3", Action: mkRunnable("step3"), Depends: []string{"step1", "step2"}},
	}

	executor := NewParallelStepExecutor(steps)
	results, err := executor.Execute(ctx)
	if err != nil {
		slog.Error("execution failed", "error", err)
		return
	}

	fmt.Printf("Executed %d steps\n", len(results))
	for id, result := range results {
		fmt.Printf("  %s: %v\n", id, result)
	}
}
```

## Explanation

1. **`core.Parallel` for simple fan-out** -- When all steps receive the same input and have no dependencies, `core.Parallel` provides the simplest implementation. It runs all runnables concurrently via goroutines and returns a `[]any` slice of results in the same order as the input runnables.

2. **Dependency resolution** -- The `ParallelStepExecutor` builds a dependency graph and executes steps in topological order. Steps without unsatisfied dependencies run immediately in parallel; steps with dependencies wait for their prerequisites. Circular dependencies are detected when no ready steps are found.

3. **Result synchronization** -- A `sync.RWMutex` protects the shared results map. The read lock in `findReadySteps` allows concurrent readers while writes are serialized, minimizing contention.

4. **Error propagation** -- The error channel collects failures from any goroutine. After `wg.Wait()`, the first error (if any) is returned. The context is checked at the start of each phase to respect cancellation.

## Testing

```go
func TestParallelStepExecutor_ExecutesInParallel(t *testing.T) {
	mkRunnable := func(name string) core.Runnable {
		return &stepRunnable{name: name}
	}
	steps := []Step{
		{ID: "step1", Action: mkRunnable("step1"), Depends: []string{}},
		{ID: "step2", Action: mkRunnable("step2"), Depends: []string{}},
	}

	executor := NewParallelStepExecutor(steps)
	results, err := executor.Execute(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}
}
```

## Related Recipes

- **[Handling Tool Failures](/docs/recipes/agents/agents-tool-failures)** -- Robust error handling
- **[Parallel Node Execution](/docs/recipes/infrastructure/parallel-nodes)** -- Parallel graph execution
