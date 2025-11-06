---
title: Memory Recipes
sidebar_position: 1
---

# Memory Recipes

Common memory management patterns.

## Buffer Memory

```go
mem, _ := memory.NewMemory(memory.MemoryTypeBuffer)
mem.SaveContext(ctx, inputs, outputs)
vars, _ := mem.LoadMemoryVariables(ctx, map[string]any{})
```

## Window Memory

```go
mem, _ := memory.NewMemory(
    memory.MemoryTypeWindow,
    memory.WithWindowSize(10),
)
```

## Memory Persistence

```go
// Save
vars, _ := mem.LoadMemoryVariables(ctx, map[string]any{})
data, _ := json.Marshal(vars)
os.WriteFile("memory.json", data, 0644)

// Load
data, _ := os.ReadFile("memory.json")
json.Unmarshal(data, &vars)
```

---

**More Recipes:** [Integration Recipes](./integration-recipes) | [Quick Solutions](./quick-solutions)

