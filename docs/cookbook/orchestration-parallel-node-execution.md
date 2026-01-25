---
title: "Parallel Node Execution in Graphs"
package: "orchestration"
category: "performance"
complexity: "advanced"
---

# Parallel Node Execution in Graphs

## Problem

You need to execute independent nodes in an orchestration graph in parallel to reduce total execution time, while respecting dependencies and ensuring thread-safe execution.

## Solution

Implement a parallel graph executor that analyzes node dependencies, identifies independent nodes, executes them concurrently, and synchronizes dependent nodes. This works because graph nodes with no dependencies can run simultaneously, and Go's concurrency primitives allow safe parallel execution.

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
    
    "github.com/lookatitude/beluga-ai/pkg/core"
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
    nodes     map[string]*GraphNode
    results   map[string]interface{}
    mu        sync.RWMutex
    maxConcurrent int
}

// NewParallelGraphExecutor creates a new parallel executor
func NewParallelGraphExecutor(maxConcurrent int) *ParallelGraphExecutor {
    return &ParallelGraphExecutor{
        nodes:        make(map[string]*GraphNode),
        results:      make(map[string]interface{}),
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
        ID:       "node1",
        Runnable: &MockRunnable{},
        DependsOn: []string{},
    })
    
    executor.AddNode(&GraphNode{
        ID:       "node2",
        Runnable: &MockRunnable{},
        DependsOn: []string{},
    })
    
    executor.AddNode(&GraphNode{
        ID:       "node3",
        Runnable: &MockRunnable{},
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

Let's break down what's happening:

1. **Dependency resolution** - Notice how we build a dependency graph and identify nodes ready to execute (all dependencies satisfied). This ensures dependencies are respected while maximizing parallelism.

2. **Concurrent execution** - Ready nodes execute concurrently using a semaphore to limit concurrency. This prevents resource exhaustion while utilizing available parallelism.

3. **Topological execution** - We execute nodes in topological order (dependencies before dependents), but execute independent nodes in parallel. This maximizes speed while maintaining correctness.

```go
**Key insight:** Identify independent nodes and execute them in parallel. Even with dependencies, you can often parallelize significant portions of graph execution.

## Testing

```
Here's how to test this solution:
```go
func TestParallelGraphExecutor_ExecutesInParallel(t *testing.T) {
    executor := NewParallelGraphExecutor(5)
    
    executor.AddNode(&GraphNode{ID: "node1", Runnable: &MockRunnable{}, DependsOn: []string{}})
    executor.AddNode(&GraphNode{ID: "node2", Runnable: &MockRunnable{}, DependsOn: []string{}})
    
    results, err := executor.Execute(context.Background())
    require.NoError(t, err)
    require.Len(t, results, 2)
}

## Variations

### Dynamic Concurrency

Adjust concurrency based on node characteristics:
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

- **[Agents Parallel Step Execution](./agents-parallel-step-execution.md)** - Parallel agent steps
- **[Orchestration Workflow Checkpointing](./orchestration-workflow-checkpointing.md)** - Save and resume workflows
- **[Orchestration Package Guide](../package_design_patterns.md)** - For a deeper understanding of orchestration
