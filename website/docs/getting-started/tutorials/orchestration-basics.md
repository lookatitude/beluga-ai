---
title: Orchestration Basics
sidebar_position: 6
---

# Part 6: Orchestration Basics

In this tutorial, you'll learn how to build complex workflows using Beluga AI's orchestration capabilities. Orchestration allows you to chain operations, create graphs, and manage distributed workflows.

## Learning Objectives

- ✅ Understand orchestration patterns
- ✅ Create and execute chains
- ✅ Build graph workflows
- ✅ Manage task dependencies
- ✅ Handle errors in workflows

## Prerequisites

- Completed [Part 3: Creating Your First Agent](./03-first-agent)
- Basic understanding of workflows and dependencies
- API key for an LLM provider

## What is Orchestration?

Orchestration in Beluga AI enables:
- **Chains**: Sequential execution of operations
- **Graphs**: DAG-based execution with dependencies
- **Workflows**: Long-running, distributed processes

## Step 1: Creating Chains

Chains execute steps sequentially:

```go
package main

import (
    "context"
    "fmt"

    "github.com/lookatitude/beluga-ai/pkg/core"
    "github.com/lookatitude/beluga-ai/pkg/orchestration"
)

// Define a simple runnable step
type SimpleStep struct {
    Name string
}

func (s *SimpleStep) Invoke(ctx context.Context, input map[string]any) (map[string]any, error) {
    fmt.Printf("Executing step: %s\n", s.Name)
    return map[string]any{
        "output": fmt.Sprintf("Result from %s", s.Name),
    }, nil
}

func main() {
    ctx := context.Background()

    // Create steps
    step1 := &SimpleStep{Name: "step1"}
    step2 := &SimpleStep{Name: "step2"}
    step3 := &SimpleStep{Name: "step3"}

    steps := []core.Runnable{step1, step2, step3}

    // Create chain
    chain, err := orchestration.NewChain(steps,
        orchestration.WithChainTimeout(30),
        orchestration.WithChainRetries(3),
    )
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }

    // Execute chain
    input := map[string]any{"input": "test"}
    result, err := chain.Invoke(ctx, input)
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }

    fmt.Printf("Chain result: %v\n", result)
}
```

## Step 2: Chain with LLM

Create a chain that uses an LLM:

```go
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/lookatitude/beluga-ai/pkg/core"
    "github.com/lookatitude/beluga-ai/pkg/llms"
    "github.com/lookatitude/beluga-ai/pkg/orchestration"
    "github.com/lookatitude/beluga-ai/pkg/schema"
)

// LLM Step wraps an LLM call
type LLMStep struct {
    LLM llmsiface.ChatModel
}

func (l *LLMStep) Invoke(ctx context.Context, input map[string]any) (map[string]any, error) {
    prompt := input["input"].(string)
    messages := []schema.Message{
        schema.NewHumanMessage(prompt),
    }

    response, err := l.LLM.Generate(ctx, messages)
    if err != nil {
        return nil, err
    }

    return map[string]any{
        "output": response.Content,
    }, nil
}

func main() {
    ctx := context.Background()

    // Setup LLM
    config := llms.NewConfig(
        llms.WithProvider("openai"),
        llms.WithModelName("gpt-3.5-turbo"),
        llms.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
    )
    factory := llms.NewFactory()
    llm, _ := factory.CreateProvider("openai", config)

    // Create LLM step
    llmStep := &LLMStep{LLM: llm}

    // Create chain
    chain, _ := orchestration.NewChain([]core.Runnable{llmStep})

    // Execute
    input := map[string]any{
        "input": "What is AI?",
    }
    result, _ := chain.Invoke(ctx, input)
    fmt.Printf("Result: %v\n", result)
}
```

## Step 3: Creating Graphs

Graphs allow parallel execution with dependencies:

```go
package main

import (
    "context"
    "fmt"

    "github.com/lookatitude/beluga-ai/pkg/core"
    "github.com/lookatitude/beluga-ai/pkg/orchestration"
)

func main() {
    ctx := context.Background()

    // Create orchestrator
    orchestrator := orchestration.NewOrchestrator()

    // Create graph
    graph, err := orchestrator.CreateGraph()
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }

    // Add nodes
    node1 := &SimpleStep{Name: "node1"}
    node2 := &SimpleStep{Name: "node2"}
    node3 := &SimpleStep{Name: "node3"}

    graph.AddNode("step1", node1)
    graph.AddNode("step2", node2)
    graph.AddNode("step3", node3)

    // Define dependencies
    graph.AddEdge("step1", "step2") // step2 depends on step1
    graph.AddEdge("step1", "step3") // step3 depends on step1
    // step2 and step3 can run in parallel

    // Set entry and finish points
    graph.SetEntryPoint([]string{"step1"})
    graph.SetFinishPoint([]string{"step2", "step3"})

    // Execute graph
    input := map[string]any{"input": "test"}
    result, err := graph.Invoke(ctx, input)
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }

    fmt.Printf("Graph result: %v\n", result)
}
```

## Step 4: Error Handling in Chains

```go
chain, err := orchestration.NewChain(steps,
    orchestration.WithChainRetries(3),
    orchestration.WithChainRetryDelay(2*time.Second),
    orchestration.WithChainErrorHandler(func(err error) error {
        // Custom error handling
        fmt.Printf("Chain error: %v\n", err)
        return err
    }),
)
```

## Step 5: Chain with Memory

Add memory to maintain context across chain steps:

```go
import "github.com/lookatitude/beluga-ai/pkg/memory"

// Create memory
mem, _ := memory.NewMemory(memory.MemoryTypeBuffer)

// Create chain with memory
chain, err := orchestration.NewChain(steps,
    orchestration.WithChainMemory(mem),
)
```

## Step 6: Complete RAG Chain Example

```go
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/lookatitude/beluga-ai/pkg/core"
    "github.com/lookatitude/beluga-ai/pkg/embeddings"
    "github.com/lookatitude/beluga-ai/pkg/llms"
    "github.com/lookatitude/beluga-ai/pkg/orchestration"
    "github.com/lookatitude/beluga-ai/pkg/schema"
    "github.com/lookatitude/beluga-ai/pkg/vectorstores"
)

// RetrievalStep retrieves relevant documents
type RetrievalStep struct {
    Store    vectorstoresiface.VectorStore
    Embedder embeddingsiface.Embedder
}

func (r *RetrievalStep) Invoke(ctx context.Context, input map[string]any) (map[string]any, error) {
    query := input["input"].(string)
    docs, _, _ := r.Store.SimilaritySearchByQuery(ctx, query, 3, r.Embedder)
    
    context := ""
    for _, doc := range docs {
        context += doc.GetContent() + "\n"
    }

    return map[string]any{
        "context": context,
        "query": query,
    }, nil
}

// GenerationStep generates answer using context
type GenerationStep struct {
    LLM llmsiface.ChatModel
}

func (g *GenerationStep) Invoke(ctx context.Context, input map[string]any) (map[string]any, error) {
    context := input["context"].(string)
    query := input["query"].(string)

    messages := []schema.Message{
        schema.NewSystemMessage("Answer using this context: " + context),
        schema.NewHumanMessage(query),
    }

    response, _ := g.LLM.Generate(ctx, messages)
    return map[string]any{
        "output": response.Content,
    }, nil
}

func main() {
    ctx := context.Background()

    // Setup components
    embedder, _ := setupEmbedder(ctx)
    store, _ := setupVectorStore(ctx, embedder)
    llm, _ := setupLLM(ctx)

    // Create steps
    retrievalStep := &RetrievalStep{Store: store, Embedder: embedder}
    generationStep := &GenerationStep{LLM: llm}

    // Create chain
    chain, _ := orchestration.NewChain(
        []core.Runnable{retrievalStep, generationStep},
    )

    // Execute
    input := map[string]any{
        "input": "What is machine learning?",
    }
    result, _ := chain.Invoke(ctx, input)
    fmt.Printf("Answer: %v\n", result)
}
```

## Step 7: Workflow Orchestration

For long-running, distributed workflows:

```go
import "github.com/lookatitude/beluga-ai/pkg/orchestration"

// Create workflow
orchestrator := orchestration.NewOrchestrator()

workflow, err := orchestrator.CreateWorkflow(
    myWorkflowFunction,
    orchestration.WithWorkflowID("my-workflow"),
)

// Execute workflow
workflowID, runID, err := workflow.Execute(ctx, input)

// Get result
result, err := workflow.GetResult(ctx, workflowID, runID)
```

## Exercises

1. **Multi-step chain**: Create a chain with 5+ steps
2. **Parallel execution**: Build a graph with parallel nodes
3. **Error recovery**: Implement retry logic in chains
4. **Conditional execution**: Create a graph with conditional paths
5. **Workflow integration**: Build a distributed workflow

## Next Steps

Congratulations! You've learned orchestration basics. Next, learn how to:

- **[Part 7: Production Deployment](./production-deployment)** - Deploy your applications
- **[Concepts: Orchestration](../../concepts/orchestration)** - Deep dive into orchestration
- **[Best Practices](../../guides/best-practices)** - Production best practices

---

**Ready for the next step?** Continue to [Part 7: Production Deployment](./production-deployment)!

