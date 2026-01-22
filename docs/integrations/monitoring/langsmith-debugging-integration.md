# LangSmith Debugging Integration

Welcome, colleague! In this integration guide, we're going to integrate LangSmith for debugging and monitoring LLM calls in Beluga AI applications. LangSmith provides powerful debugging tools for tracing LLM operations.

## What you will build

You will configure LangSmith to capture and debug LLM calls from Beluga AI, enabling you to trace requests, inspect prompts, analyze responses, and debug issues in your AI applications.

## Learning Objectives

- ✅ Configure LangSmith with Beluga AI
- ✅ Trace LLM calls to LangSmith
- ✅ Debug prompts and responses
- ✅ Analyze LLM performance

## Prerequisites

- Go 1.24 or later installed
- Beluga AI Framework installed
- LangSmith account and API key
- Understanding of LLM debugging

## Step 1: Setup and Installation

Install LangSmith SDK:
bash
```bash
go get github.com/langchain-ai/langsmith-go
```

Get your LangSmith API key from https://smith.langchain.com

## Step 2: Basic LangSmith Integration

Create a LangSmith tracer:
```go
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/langchain-ai/langsmith-go"
    "github.com/lookatitude/beluga-ai/pkg/llms"
    "github.com/lookatitude/beluga-ai/pkg/schema"
)

type LangSmithTracer struct {
    client *langsmith.Client
}

func NewLangSmithTracer() (*LangSmithTracer, error) {
    apiKey := os.Getenv("LANGSMITH_API_KEY")
    if apiKey == "" {
        return nil, fmt.Errorf("LANGSMITH_API_KEY environment variable is required")
    }
    
    client, err := langsmith.NewClient(
        langsmith.WithAPIKey(apiKey),
    )
    if err != nil {
        return nil, fmt.Errorf("failed to create LangSmith client: %w", err)
    }
    
    return &LangSmithTracer{client: client}, nil
}

func (t *LangSmithTracer) TraceLLMCall(ctx context.Context, provider llms.ChatModel, messages []schema.Message) (*langsmith.Run, error) {
    // Create a run in LangSmith
    run, err := t.client.CreateRun(ctx, &langsmith.CreateRunRequest{
        Name:        "llm_call",
        RunType:     "llm",
        Inputs:      map[string]interface{}{"messages": messages},
        ProjectName: "beluga-ai",
    })
    if err != nil {
        return nil, fmt.Errorf("failed to create run: %w", err)
    }

    
    return run, nil
}
```

## Step 3: Wrap LLM Provider

Wrap Beluga AI LLM provider with LangSmith tracing:
```go
type TracedLLMProvider struct \{
    provider llms.ChatModel
    tracer   *LangSmithTracer
}
go
func NewTracedLLMProvider(provider llms.ChatModel, tracer *LangSmithTracer) *TracedLLMProvider {
    return &TracedLLMProvider{
        provider: provider,
        tracer:   tracer,
    }
}

func (t *TracedLLMProvider) Generate(ctx context.Context, messages []schema.Message) (*schema.AIMessage, error) {
    // Start trace
    run, err := t.tracer.TraceLLMCall(ctx, t.provider, messages)
    if err != nil {
        // Continue even if tracing fails
        return t.provider.Generate(ctx, messages)
    }
    
    // Make LLM call
    startTime := time.Now()
    response, err := t.provider.Generate(ctx, messages)
    duration := time.Since(startTime)

    

    // Update trace with results
    t.tracer.client.UpdateRun(ctx, run.ID, &langsmith.UpdateRunRequest{
        Outputs: map[string]interface{}{
            "response": response.Content,
        },
        EndTime: time.Now(),
        Extra: map[string]interface{}{
            "duration_ms": duration.Milliseconds(),
            "error":       err != nil,
        },
    })
    
    return response, err
}
```

## Step 4: Complete Integration

Here's a complete example:
```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"
    "time"

    "github.com/langchain-ai/langsmith-go"
    "github.com/lookatitude/beluga-ai/pkg/llms"
    "github.com/lookatitude/beluga-ai/pkg/schema"
)

func main() {
    // Create LangSmith tracer
    tracer, err := NewLangSmithTracer()
    if err != nil {
        log.Fatalf("Failed to create tracer: %v", err)
    }
    
    // Create LLM provider
    config := llms.NewConfig(
        llms.WithProvider("openai"),
        llms.WithModelName("gpt-3.5-turbo"),
        llms.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
    )
    
    factory := llms.NewFactory()
    provider, err := factory.CreateProvider("openai", config)
    if err != nil {
        log.Fatalf("Failed to create provider: %v", err)
    }
    
    // Wrap with tracing
    tracedProvider := NewTracedLLMProvider(provider, tracer)
    
    // Make a call
    ctx := context.Background()
    messages := []schema.Message{
        schema.NewHumanMessage("What is the capital of France?"),
    }
    
    response, err := tracedProvider.Generate(ctx, messages)
    if err != nil {
        log.Fatalf("Failed to generate: %v", err)
    }

    
    fmt.Printf("Response: %s\n", response.Content)
    fmt.Println("Check LangSmith dashboard for trace details")
}
```

## Configuration Options

| Option | Description | Default | Required |
|--------|-------------|---------|----------|
| `LANGSMITH_API_KEY` | LangSmith API key | - | Yes |
| `LANGSMITH_PROJECT` | Project name | `beluga-ai` | No |
| `LANGSMITH_ENDPOINT` | LangSmith endpoint | `https://api.smith.langchain.com` | No |

## Common Issues

### "API key not found"

**Problem**: Missing LangSmith API key.

**Solution**: Set environment variable:export LANGSMITH_API_KEY="your-api-key"
```

### "Traces not appearing"

**Problem**: Traces not being sent to LangSmith.

**Solution**: Check API key and endpoint:client, err := langsmith.NewClient(
    langsmith.WithAPIKey(apiKey),
    langsmith.WithEndpoint("https://api.smith.langchain.com"),
)
```

## Production Considerations

When using LangSmith in production:

- **Sample traces**: Don't trace every call to reduce costs
- **Filter sensitive data**: Remove PII from traces
- **Monitor costs**: Track trace volume
- **Use projects**: Organize traces by project
- **Set up alerts**: Alert on errors or latency

## Next Steps

Congratulations! You've integrated LangSmith with Beluga AI. Next, learn how to:

- **[Datadog Dashboard Templates](./datadog-dashboard-templates.md)** - Monitoring dashboards
- **[Monitoring Package Documentation](../../api/packages/monitoring.md)** - Deep dive into monitoring
- **[LLM Providers Guide](../../guides/llm-providers.md)** - LLM configuration

---

**Ready for more?** Check out the [Integrations Index](./README.md) for more integration guides!
