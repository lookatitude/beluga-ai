# ChatModel Embeds Runnable

ChatModel interface embeds core.Runnable for framework composability.

```go
// ChatModel is composable (chains, graphs, orchestration)
type ChatModel interface {
    MessageGenerator
    StreamMessageHandler
    ModelInfoProvider
    HealthChecker
    core.Runnable  // Invoke/Batch/Stream for framework integration
}

// LLM is lower-level (llms/iface)
type LLM interface {
    Invoke(ctx context.Context, prompt string) (string, error)
    GetModelName() string
    GetProviderName() string
    // No Runnable embedding
}
```

## Deliberate Separation
- **LLMs**: Low-level, single-call abstractions
- **ChatModels**: Higher-level, meant for chains/graphs/orchestration

## When to Use Which
| Use Case | Interface |
|----------|-----------|
| Direct API call | LLM |
| Chain/Graph step | ChatModel |
| Agent backbone | ChatModel |
| Simple text generation | LLM |

## Implication
ChatModels can be used directly in `Chain.AddStep()` or `Graph.AddNode()`.
LLMs need an adapter wrapper.
