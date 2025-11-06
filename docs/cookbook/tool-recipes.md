# Tool Recipes

Common tool integration patterns.

## Custom Tool

```go
type MyTool struct {
    tools.BaseTool
}

func (t *MyTool) Execute(ctx context.Context, input any) (any, error) {
    // Implementation
    return result, nil
}
```

## Tool Registry

```go
registry := tools.NewInMemoryToolRegistry()
registry.RegisterTool(tools.NewCalculatorTool())
tool, _ := registry.GetTool("calculator")
```

---

**More Recipes:** [Memory Recipes](./memory-recipes.md) | [Integration Recipes](./integration-recipes.md)

