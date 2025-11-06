---
title: Orchestration
sidebar_position: 1
---

# Orchestration Concepts

This document explains orchestration in Beluga AI, including chains, graphs, workflows, and task scheduling.

## Orchestration Patterns

Beluga AI supports three orchestration patterns:

1. **Chains**: Sequential execution
2. **Graphs**: DAG-based execution with dependencies
3. **Workflows**: Long-running, distributed processes

## Chains

Chains execute steps sequentially.

### Creating Chains

```go
steps := []core.Runnable{step1, step2, step3}

chain, err := orchestration.NewChain(steps,
    orchestration.WithChainTimeout(30),
    orchestration.WithChainRetries(3),
)
```

### Chain Execution

```go
input := map[string]any{"input": "data"}
result, err := chain.Invoke(ctx, input)
```

### Chain with Memory

```go
mem, _ := memory.NewMemory(memory.MemoryTypeBuffer)

chain, err := orchestration.NewChain(steps,
    orchestration.WithChainMemory(mem),
)
```

## Graphs

Graphs enable parallel execution with dependencies.

### Creating Graphs

```go
orchestrator := orchestration.NewOrchestrator()
graph, err := orchestrator.CreateGraph()
```

### Adding Nodes

```go
graph.AddNode("step1", runnable1)
graph.AddNode("step2", runnable2)
graph.AddNode("step3", runnable3)
```

### Defining Dependencies

```go
graph.AddEdge("step1", "step2") // step2 depends on step1
graph.AddEdge("step1", "step3") // step3 depends on step1
// step2 and step3 can run in parallel
```

### Entry and Finish Points

```go
graph.SetEntryPoint([]string{"step1"})
graph.SetFinishPoint([]string{"step2", "step3"})
```

## Workflows

Workflows are long-running, distributed processes.

### Creating Workflows

```go
workflow, err := orchestrator.CreateWorkflow(
    workflowFunction,
    orchestration.WithWorkflowID("my-workflow"),
)
```

### Workflow Execution

```go
workflowID, runID, err := workflow.Execute(ctx, input)
```

### Getting Results

```go
result, err := workflow.GetResult(ctx, workflowID, runID)
```

## Task Scheduling

### Dependency Management

Graphs automatically handle dependencies:
- Execute nodes only when dependencies are met
- Parallel execution when possible
- Error propagation

### Retry Mechanisms

```go
chain, err := orchestration.NewChain(steps,
    orchestration.WithChainRetries(3),
    orchestration.WithChainRetryDelay(2*time.Second),
)
```

### Circuit Breakers

Prevent cascading failures:

```go
orchestration.WithCircuitBreaker(
    maxFailures: 5,
    timeout: 60*time.Second,
)
```

## Worker Pools

For concurrent execution:

```go
scheduler := orchestration.NewEnhancedScheduler(10) // 10 workers
scheduler.Start()
defer scheduler.Stop()
```

## Error Handling

### Chain Errors

```go
chain, err := orchestration.NewChain(steps,
    orchestration.WithChainErrorHandler(func(err error) error {
        // Custom error handling
        return err
    }),
)
```

### Graph Errors

Graphs handle errors by:
- Stopping dependent nodes
- Propagating errors
- Allowing error recovery

## Best Practices

1. **Design dependencies carefully**: Minimize dependencies for parallelism
2. **Set timeouts**: Always configure timeouts
3. **Handle errors**: Implement proper error handling
4. **Monitor execution**: Use observability tools
5. **Test thoroughly**: Test with various scenarios

## Related Concepts

- [Core Concepts](./core) - Runnable interface
- [Agent Concepts](./agents) - Agent orchestration
- [Getting Started: Orchestration](../../getting-started/tutorials/orchestration-basics) - Tutorial

---

**Next:** Review [Best Practices](../../guides/best-practices) or explore [Use Cases](../../use-cases/)

