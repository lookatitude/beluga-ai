---
title: Agent Recipes
sidebar_position: 1
---

# Agent Recipes

Common agent patterns and recipes.

## Basic Agent

```go
tools := []tools.Tool{
    tools.NewCalculatorTool(),
    tools.NewEchoTool(),
}
agent, _ := agents.NewBaseAgent("assistant", llm, tools)
result, _ := agent.Invoke(ctx, input)
```

## Agent with Memory

```go
mem, _ := memory.NewMemory(memory.MemoryTypeBuffer)
agent.Initialize(map[string]interface{}{
    "memory": mem,
})
```

## Multi-Agent System

```go
agent1, _ := agents.NewBaseAgent("researcher", llm1, tools1)
agent2, _ := agents.NewBaseAgent("writer", llm2, tools2)

chain := orchestration.NewChain([]core.Runnable{agent1, agent2})
result, _ := chain.Invoke(ctx, input)
```

---

**More Recipes:** [Tool Recipes](./tool-recipes) | [Memory Recipes](./memory-recipes)

