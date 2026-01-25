# Cross-Package Pattern Integration

This document explains how design patterns work together across different packages in the Beluga AI Framework.

## Factory Pattern Enables Provider Registration

### How Factories Work Together

The factory pattern is used consistently across packages to enable provider registration and creation:

```go
// LLM Factory
llmFactory := llms.NewFactory()
llm, err := llmFactory.CreateProvider("openai", config)

// Embedding Factory
embeddingFactory := embeddings.NewFactory()
embedder, err := embeddingFactory.NewEmbedder("openai", config)

// Vector Store Factory
vectorStore, err := vectorstores.NewVectorStore(ctx, "inmemory", opts...)

// Agent Factory
agentFactory := agents.NewAgentFactory(&agents.Config{})
agent, err := agentFactory.CreateBaseAgent(ctx, "my-agent", llm, tools)
```

### Global Registry Integration

Global registries allow cross-package provider registration:

```go
// Register across packages
embeddings.RegisterGlobal("custom", customEmbedderCreator)
vectorstores.RegisterGlobal("custom", customVectorStoreCreator)
agents.RegisterAgentType("custom", customAgentCreator)

// Use registered providers
embedder := embeddings.NewEmbedder(ctx, "custom", config)
store := vectorstores.NewVectorStore(ctx, "custom", opts...)
agent := agents.CreateAgent(ctx, "custom", "my-agent", llm, tools, config)
```

## OTEL Patterns Provide Observability

### Consistent Metrics Across Packages

All packages use the same OTEL metrics pattern:

```go
// LLM metrics
llmMetrics.RecordRequest(ctx, provider, model, duration)

// Agent metrics
agentMetrics.RecordPlanningCall(ctx, agentName, duration, success)

// Memory metrics
memoryMetrics.RecordOperation(ctx, "save_context", memoryType, duration, success)

// Orchestration metrics
orchestrationMetrics.RecordChainExecution(ctx, chainName, duration, success)
```

### Distributed Tracing

Tracing spans propagate across package boundaries:

```go
// LLM creates span
ctx, span := llmTracer.Start(ctx, "llm.generate")
defer span.End()

// Agent uses LLM (span context propagated)
agentResult, err := agent.Invoke(ctx, input) // Context includes LLM span

// Orchestration coordinates (spans linked)
chainResult, err := chain.Invoke(ctx, input) // All spans in trace
```

## Error Patterns Enable Consistent Error Handling

### Error Code Consistency

All packages use similar error code patterns:

```go
// LLM errors
llms.ErrCodeRateLimit
llms.ErrCodeAuthentication
llms.ErrCodeInvalidRequest

// Agent errors
agents.ErrCodeInitialization
agents.ErrCodeExecutionFailed
agents.ErrCodeToolNotFound

// Memory errors
memory.ErrCodeInvalidConfig
memory.ErrCodeSaveFailed
memory.ErrCodeLoadFailed
```

### Error Wrapping

Errors are wrapped to preserve context:

```go
// LLM error wrapped in agent error
agentErr := agents.NewAgentError(
    "execute",
    agentName,
    agents.ErrCodeExecutionFailed,
    llmErr, // Wrapped LLM error
)

// Can unwrap to get original error
if llmErr := errors.Unwrap(agentErr); llmErr != nil {
    // Handle LLM-specific error
}
```

## Config Patterns Enable Dynamic Configuration

### Configuration Loading

All packages support the same configuration loading pattern:

```go
// Load from YAML
config := config.LoadFromFile("config.yaml")

// LLM config
llmConfig := llms.Config{
    Provider: config.GetString("llms.provider"),
    Model:    config.GetString("llms.model"),
}

// Agent config
agentConfig := agents.Config{
    MaxRetries: config.GetInt("agents.max_retries"),
    Timeout:    config.GetDuration("agents.timeout"),
}

// Memory config
memoryConfig := memory.Config{
    Type:  config.GetString("memory.type"),
    Enabled: config.GetBool("memory.enabled"),
}
```

### Environment Variable Support

All packages support environment variable configuration:

```go
// Set via environment
export BELUGA_LLMS_PROVIDER=openai
export BELUGA_LLMS_MODEL=gpt-3.5-turbo
export BELUGA_AGENTS_MAX_RETRIES=3

// Config automatically loads from environment
config := config.Load()
```

## Integration Example

### Complete Pattern Integration

Here's how all patterns work together in a complete example:

```go
// 1. Configuration (Config Pattern)
config := config.LoadFromFile("config.yaml")

// 2. Factory Pattern
llmFactory := llms.NewFactory()
llm, _ := llmFactory.CreateProvider("openai", config.LLM)

embeddingFactory := embeddings.NewFactory()
embedder, _ := embeddingFactory.NewEmbedder("openai", config.Embedding)

// 3. Global Registry Pattern
vectorStore, _ := vectorstores.NewVectorStore(ctx, "inmemory",
    vectorstores.WithEmbedder(embedder),
)

// 4. Factory Pattern
agentFactory := agents.NewAgentFactory(&config.Agent)
agent, _ := agentFactory.CreateBaseAgent(ctx, "my-agent", llm, tools)

// 5. OTEL Observability
ctx, span := tracer.Start(ctx, "complete.workflow")
defer span.End()

// 6. Error Handling Pattern
result, err := agent.Invoke(ctx, input)
if err != nil {
    if agents.IsAgentError(err) {
        code := agents.GetAgentErrorCode(err)
        // Handle agent-specific error
    }
    span.RecordError(err)
    return err
}

// 7. Metrics Recording
metrics.RecordOperation(ctx, "complete.workflow", duration, err == nil)
```

## Pattern Benefits

### Consistency

- All packages follow the same patterns
- Easy to understand and use
- Predictable behavior

### Extensibility

- Easy to add new providers
- Simple to extend functionality
- Clear extension points

### Observability

- Consistent metrics across packages
- Distributed tracing support
- Error tracking and reporting

### Configuration

- Unified configuration approach
- Environment variable support
- Dynamic configuration loading

## Related Documentation

- [Pattern Examples](./pattern-examples.md) - Real-world pattern examples
- [Pattern Decision Guide](./pattern-decision-guide.md) - When to use which pattern
- [Package Design Patterns](../../package_design_patterns.md) - Complete pattern reference
