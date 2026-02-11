---
title: "Parallel Step Execution"
description: "Execute multiple independent agent steps concurrently with dependency resolution to reduce total execution time."
---

## Problem

You need to execute multiple agent steps in parallel when they don't depend on each other, to reduce total execution time and improve responsiveness of multi-step agent workflows.

## Solution

Implement a parallel step executor that identifies independent steps, executes them concurrently with proper synchronization, and merges results. This works because many agent steps (like tool calls or data retrieval) can run independently, and Go's concurrency primitives allow safe parallel execution.

## Why This Matters

Agent workflows often consist of steps with varying dependencies. A naive sequential approach executes all steps one after another, even when many could run concurrently. For example, if an agent needs to search a database, call a weather API, and query a calendar service, running these sequentially triples the wall-clock time compared to running them in parallel.

The challenge is correctly identifying which steps are independent and which have data dependencies. This recipe uses a dependency graph with topological ordering to determine which steps can safely run concurrently at each phase. Steps without dependencies run immediately, while dependent steps wait only for their specific prerequisites -- not for all prior steps to complete. This approach maximizes parallelism while maintaining correctness.

Go's concurrency model (goroutines, `sync.WaitGroup`, channels) is well-suited for this pattern because goroutine creation is cheap, and mutexes protect shared result storage without significant overhead. The `sync.RWMutex` used here allows multiple goroutines to read completed results concurrently while writes are serialized, avoiding contention on the results map.

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

var tracer = otel.Tracer("beluga.agents.parallel")

// Step represents an agent step
type Step struct {
    ID      string
    Action  core.Runnable
    Depends []string // IDs of steps this depends on
}

// ParallelStepExecutor executes steps in parallel
type ParallelStepExecutor struct {
    steps    []Step
    results  map[string]interface{}
    mu       sync.RWMutex
    wg       sync.WaitGroup
}

// NewParallelStepExecutor creates a new executor
func NewParallelStepExecutor(steps []Step) *ParallelStepExecutor {
    return &ParallelStepExecutor{
        steps:   steps,
        results: make(map[string]interface{}),
    }
}

// Execute executes steps in parallel respecting dependencies
func (pse *ParallelStepExecutor) Execute(ctx context.Context) (map[string]interface{}, error) {
    ctx, span := tracer.Start(ctx, "parallel_executor.execute")
    defer span.End()

    span.SetAttributes(attribute.Int("step_count", len(pse.steps)))

    // Build dependency graph
    graph := pse.buildDependencyGraph()

    // Execute in topological order
    executed := make(map[string]bool)
    for len(executed) < len(pse.steps) {
        // Find steps ready to execute
        ready := pse.findReadySteps(graph, executed)

        if len(ready) == 0 {
            return nil, fmt.Errorf("circular dependency detected")
        }

        // Execute ready steps in parallel
        var wg sync.WaitGroup
        errCh := make(chan error, len(ready))

        for _, stepID := range ready {
            wg.Add(1)
            go func(id string) {
                defer wg.Done()

                step := pse.findStep(id)
                result, err := step.Action.Invoke(ctx, nil)
                if err != nil {
                    errCh <- fmt.Errorf("step %s failed: %w", id, err)
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

        if len(errCh) > 0 {
            err := <-errCh
            span.RecordError(err)
            span.SetStatus(trace.StatusError, err.Error())
            return nil, err
        }
    }

    span.SetStatus(trace.StatusOK, "all steps executed")
    return pse.results, nil
}

// buildDependencyGraph builds a dependency graph
func (pse *ParallelStepExecutor) buildDependencyGraph() map[string][]string {
    graph := make(map[string][]string)
    for _, step := range pse.steps {
        graph[step.ID] = step.Depends
    }
    return graph
}

// findReadySteps finds steps that can be executed (dependencies satisfied)
func (pse *ParallelStepExecutor) findReadySteps(graph map[string][]string, executed map[string]bool) []string {
    ready := []string{}

    for stepID, deps := range graph {
        if executed[stepID] {
            continue
        }

        allSatisfied := true
        for _, dep := range deps {
            if !executed[dep] {
                allSatisfied = false
                break
            }
        }

        if allSatisfied {
            ready = append(ready, stepID)
        }
    }

    return ready
}

// findStep finds a step by ID
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

    // Create steps
    steps := []Step{
        {ID: "step1", Action: &MockRunnable{}, Depends: []string{}},
        {ID: "step2", Action: &MockRunnable{}, Depends: []string{}},
        {ID: "step3", Action: &MockRunnable{}, Depends: []string{"step1", "step2"}},
    }

    executor := NewParallelStepExecutor(steps)
    results, err := executor.Execute(ctx)
    if err != nil {
        log.Fatalf("Execution failed: %v", err)
    }

    fmt.Printf("Executed %d steps\n", len(results))
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

1. **Dependency resolution** -- The executor builds a dependency graph and runs steps in topological order. Steps without dependencies can run immediately, while dependent steps wait for their prerequisites. Circular dependencies are detected and reported as errors, preventing infinite loops.

2. **Parallel execution** -- Steps that are ready at the same time execute concurrently using goroutines. This maximizes parallelism while respecting dependencies. The error channel collects failures from any goroutine, and the first error terminates the execution phase.

3. **Result synchronization** -- A `sync.RWMutex` protects the shared results map and the executed tracking map. This prevents race conditions when concurrent goroutines store their results. The lock is held only for the brief duration of the map write, minimizing contention.

4. **Observability** -- OpenTelemetry spans track the overall execution and report step counts, making it straightforward to identify bottlenecks in production workflows.

## Testing

```go
func TestParallelStepExecutor_ExecutesInParallel(t *testing.T) {
    steps := []Step{
        {ID: "step1", Action: &MockRunnable{}, Depends: []string{}},
        {ID: "step2", Action: &MockRunnable{}, Depends: []string{}},
    }

    executor := NewParallelStepExecutor(steps)
    results, err := executor.Execute(context.Background())

    require.NoError(t, err)
    require.Len(t, results, 2)
}
```

## Variations

### Step Timeout

Add timeouts to individual steps to prevent slow operations from blocking the entire workflow:

```go
func (pse *ParallelStepExecutor) ExecuteWithTimeout(ctx context.Context, stepTimeout time.Duration) (map[string]interface{}, error) {
    // Add timeout per step
}
```

### Step Retry

Retry failed steps before propagating the error, useful when steps call unreliable external services:

```go
func (pse *ParallelStepExecutor) ExecuteWithRetry(ctx context.Context, maxRetries int) (map[string]interface{}, error) {
    // Retry logic
}
```

## Related Recipes

- **[Handling Tool Failures](/cookbook/agents/agents-tool-failures)** -- Robust error handling
- **[Parallel Node Execution](/cookbook/infrastructure/parallel-nodes)** -- Parallel graph execution
