# Pattern Examples in Practice

This document provides real-world examples of design patterns used throughout the Beluga AI Framework, showing how they work in practice.

## Factory Pattern

### LLM Factory Example

```go
// Create factory
factory := llms.NewFactory()

// Create provider with configuration
config := llms.NewConfig(
    llms.WithProvider("openai"),
    llms.WithModelName("gpt-3.5-turbo"),
    llms.WithAPIKey(apiKey),
)

llm, err := factory.CreateProvider("openai", config)
```

### Agent Factory Example

```go
// Create agent factory
agentFactory := agents.NewAgentFactory(&agents.Config{
    MaxRetries: 3,
    Timeout: 30 * time.Second,
})

// Create agent
agent, err := agentFactory.CreateBaseAgent(ctx, "my-agent", llm, tools)
```

## Global Registry Pattern

### Embedding Provider Registry

```go
// Register a custom embedder
embeddings.RegisterGlobal("custom", func(ctx context.Context, config embeddings.Config) (embeddingsiface.Embedder, error) {
    return NewCustomEmbedder(config), nil
})

// Create using global registry
embedder, err := embeddings.NewEmbedder(ctx, "custom", config)
```

### Agent Type Registry

```go
// Register custom agent type
agents.RegisterAgentType("custom", func(ctx context.Context, name string, llm any, tools []tools.Tool, config agents.Config) (iface.CompositeAgent, error) {
    return NewCustomAgent(name, llm, tools, config), nil
})

// Create using registry
agent, err := agents.CreateAgent(ctx, "custom", "my-agent", llm, tools, config)
```

## OTEL Metrics Pattern

### Metrics Implementation

```go
// Create metrics
metrics, err := llms.NewMetrics(meter, tracer)
if err != nil {
    return err
}

// Record operation
metrics.RecordRequest(ctx, "openai", "gpt-3.5-turbo", duration)
metrics.RecordTokenUsage(ctx, "openai", "gpt-3.5-turbo", inputTokens, outputTokens)
```

### Metrics in Agent Operations

```go
func (a *BaseAgent) Plan(ctx context.Context, steps []IntermediateStep, inputs map[string]any) (AgentAction, AgentFinish, error) {
    start := time.Now()
    
    // ... planning logic ...
    
    if a.metrics != nil {
        a.metrics.RecordPlanningCall(ctx, a.name, time.Since(start), err == nil)
    }
    
    return action, finish, err
}
```

## Error Handling Pattern

### Custom Error Types

```go
// Define error
type LLMError struct {
    Op   string
    Err  error
    Code string
}

func (e *LLMError) Error() string {
    return fmt.Sprintf("llm %s: %v", e.Op, e.Err)
}

// Create error
err := llms.NewLLMError("generate", llms.ErrCodeRateLimit, fmt.Errorf("rate limit exceeded"))

// Check error type
if llms.IsLLMError(err) {
    code := llms.GetLLMErrorCode(err)
    if code == llms.ErrCodeRateLimit {
        // Handle rate limit
    }
}
```

## Configuration Pattern

### Configuration Struct

```go
type Config struct {
    APIKey      string        `mapstructure:"api_key" yaml:"api_key" env:"API_KEY" validate:"required"`
    Model       string        `mapstructure:"model" yaml:"model" env:"MODEL" default:"gpt-3.5-turbo"`
    Timeout     time.Duration `mapstructure:"timeout" yaml:"timeout" env:"TIMEOUT" default:"30s"`
    MaxRetries  int           `mapstructure:"max_retries" yaml:"max_retries" env:"MAX_RETRIES" default:"3"`
}

// Create with functional options
config := llms.NewConfig(
    llms.WithProvider("openai"),
    llms.WithModelName("gpt-3.5-turbo"),
    llms.WithAPIKey(apiKey),
    llms.WithTimeout(60*time.Second),
)
```

## Testing Pattern

### Advanced Mock

```go
// Create mock with options
mockAgent := agents.NewAdvancedMockAgent("test-agent", "base",
    agents.WithMockError(false, nil),
    agents.WithMockDelay(100*time.Millisecond),
)

// Use in tests
result, err := mockAgent.Invoke(ctx, input)
```

### Table-Driven Tests

```go
func TestAgentPlanning(t *testing.T) {
    tests := []struct {
        name          string
        agent         *BaseAgent
        inputs        map[string]any
        expectedError bool
    }{
        {
            name:          "valid input",
            agent:         createTestAgent(),
            inputs:        map[string]any{"input": "test"},
            expectedError: false,
        },
        // ... more test cases
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            _, _, err := tt.agent.Plan(ctx, nil, tt.inputs)
            if (err != nil) != tt.expectedError {
                t.Errorf("expected error: %v, got: %v", tt.expectedError, err != nil)
            }
        })
    }
}
```

## Composition Pattern

### Embedding BaseAgent

```go
type CustomAgent struct {
    *base.BaseAgent
    customField string
}

func NewCustomAgent(name string, llm llmsiface.LLM, tools []tools.Tool) (*CustomAgent, error) {
    baseAgent, err := base.NewBaseAgent(name, llm, tools)
    if err != nil {
        return nil, err
    }
    
    return &CustomAgent{
        BaseAgent:   baseAgent,
        customField: "custom value",
    }, nil
}

// CustomAgent automatically implements Agent interface through BaseAgent
```

## Options Pattern

### Functional Options

```go
type Option func(*Agent)

func WithMaxRetries(maxRetries int) Option {
    return func(a *Agent) {
        a.maxRetries = maxRetries
    }
}

func WithTimeout(timeout time.Duration) Option {
    return func(a *Agent) {
        a.timeout = timeout
    }
}

// Usage
agent, err := NewAgent(llm, tools,
    WithMaxRetries(5),
    WithTimeout(60*time.Second),
)
```

## Related Documentation

- [Package Design Patterns](../../package_design_patterns.md) - Complete pattern reference
- [Cross-Package Patterns](./cross-package-patterns.md) - How patterns work together
- [Pattern Decision Guide](./pattern-decision-guide.md) - When to use which pattern
