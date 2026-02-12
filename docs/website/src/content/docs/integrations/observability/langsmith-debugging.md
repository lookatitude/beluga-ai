---
title: LangSmith LLM Debugging
description: "Integrate LangSmith with Beluga AI to trace and debug LLM calls with full prompt visibility, response analysis, and token tracking."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "LangSmith, LLM debugging, Beluga AI, prompt tracing, AI observability, LLM analytics Go, response analysis"
---

## Overview

General-purpose APM tools show you that an LLM call took 800ms, but LangSmith shows you the exact prompt that was sent, the full model response, and how both changed across iterations. This level of detail is essential during development when you are iterating on prompts, debugging unexpected agent behavior, or evaluating response quality. This guide shows how to integrate LangSmith with Beluga AI to capture detailed traces of every LLM call, including input prompts, model responses, latency, and token usage.

## Prerequisites

- Go 1.23 or later
- A Beluga AI application
- A LangSmith account and API key (from [smith.langchain.com](https://smith.langchain.com))

## Installation

Install the LangSmith Go SDK:

```bash
go get github.com/langchain-ai/langsmith-go
```

Set the required environment variables:

```bash
export LANGSMITH_API_KEY="your-api-key"
export LANGSMITH_PROJECT="beluga-ai"  # optional, defaults to "default"
```

## Configuration

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `LANGSMITH_API_KEY` | LangSmith API key | - | Yes |
| `LANGSMITH_PROJECT` | Project name for organizing traces | `default` | No |
| `LANGSMITH_ENDPOINT` | LangSmith API endpoint | `https://api.smith.langchain.com` | No |

## Usage

### Create a LangSmith Tracer

The tracer wraps the LangSmith client and provides methods for creating trace runs:

```go
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/langchain-ai/langsmith-go"
	"github.com/lookatitude/beluga-ai/schema"
)

// LangSmithTracer captures LLM calls as LangSmith runs.
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
		return nil, fmt.Errorf("create LangSmith client: %w", err)
	}

	return &LangSmithTracer{client: client}, nil
}

func (t *LangSmithTracer) CreateRun(ctx context.Context, name string, messages []schema.Message) (*langsmith.Run, error) {
	run, err := t.client.CreateRun(ctx, &langsmith.CreateRunRequest{
		Name:        name,
		RunType:     "llm",
		Inputs:      map[string]interface{}{"messages": messages},
		ProjectName: os.Getenv("LANGSMITH_PROJECT"),
	})
	if err != nil {
		return nil, fmt.Errorf("create run: %w", err)
	}

	return run, nil
}
```

### Build an LLM Middleware

Wrap a Beluga AI `ChatModel` with LangSmith tracing using the middleware pattern. This approach traces every call without modifying the underlying provider:

```go
import (
	"context"
	"time"

	"github.com/lookatitude/beluga-ai/llm"
	"github.com/lookatitude/beluga-ai/schema"
)

// TracedChatModel wraps a ChatModel to emit LangSmith traces.
type TracedChatModel struct {
	model  llm.ChatModel
	tracer *LangSmithTracer
}

func NewTracedChatModel(model llm.ChatModel, tracer *LangSmithTracer) *TracedChatModel {
	return &TracedChatModel{
		model:  model,
		tracer: tracer,
	}
}

func (t *TracedChatModel) Generate(ctx context.Context, messages []schema.Message) (*schema.AIMessage, error) {
	run, err := t.tracer.CreateRun(ctx, "llm_call", messages)
	if err != nil {
		// If tracing fails, continue without it.
		return t.model.Generate(ctx, messages)
	}

	start := time.Now()
	response, genErr := t.model.Generate(ctx, messages)
	duration := time.Since(start)

	// Record the result in LangSmith.
	updateReq := &langsmith.UpdateRunRequest{
		EndTime: time.Now(),
		Extra: map[string]interface{}{
			"duration_ms": duration.Milliseconds(),
		},
	}
	if genErr != nil {
		updateReq.Error = genErr.Error()
	} else {
		updateReq.Outputs = map[string]interface{}{
			"response": response.Content,
		}
	}

	_ = t.tracer.client.UpdateRun(ctx, run.ID, updateReq)

	return response, genErr
}
```

### Complete Example

Wire the tracer into a Beluga AI application:

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/lookatitude/beluga-ai/llm"
	"github.com/lookatitude/beluga-ai/schema"
)

func main() {
	ctx := context.Background()

	// Create the LangSmith tracer.
	tracer, err := NewLangSmithTracer()
	if err != nil {
		log.Fatalf("failed to create tracer: %v", err)
	}

	// Create an LLM provider via the registry.
	model, err := llm.New("openai", llm.ProviderConfig{
		APIKey: os.Getenv("OPENAI_API_KEY"),
		Model:  "gpt-4o",
	})
	if err != nil {
		log.Fatalf("failed to create model: %v", err)
	}

	// Wrap with LangSmith tracing.
	traced := NewTracedChatModel(model, tracer)

	messages := []schema.Message{
		schema.NewHumanMessage("What is the capital of France?"),
	}

	response, err := traced.Generate(ctx, messages)
	if err != nil {
		log.Fatalf("generation failed: %v", err)
	}

	fmt.Printf("Response: %s\n", response.Content)
	fmt.Println("View the trace at https://smith.langchain.com")
}
```

## Advanced Topics

### Trace Sampling

In production, tracing every LLM call can generate significant volume. Implement sampling to trace a percentage of calls:

```go
import "math/rand"

func (t *TracedChatModel) shouldTrace() bool {
	return rand.Float64() < 0.1 // Trace 10% of calls
}
```

### Sensitive Data Filtering

Remove personally identifiable information (PII) from traces before sending them to LangSmith. Filter message content in the `CreateRun` inputs to strip sensitive fields.

### Project Organization

Use separate LangSmith projects for different environments:

```bash
# Development
export LANGSMITH_PROJECT="beluga-ai-dev"

# Staging
export LANGSMITH_PROJECT="beluga-ai-staging"

# Production
export LANGSMITH_PROJECT="beluga-ai-prod"
```

### Combining with OpenTelemetry

LangSmith tracing complements Beluga AI's built-in OpenTelemetry instrumentation. Use LangSmith for detailed LLM-specific debugging and Datadog or another backend for infrastructure-level observability. Both can run simultaneously without conflict.

## Troubleshooting

### "API key not found"

Verify that `LANGSMITH_API_KEY` is set in the environment. API keys are available from the [LangSmith Settings](https://smith.langchain.com/settings) page.

### Traces not appearing

Check that the API key is valid and the endpoint is reachable. If using a custom endpoint, verify the `LANGSMITH_ENDPOINT` variable:

```bash
export LANGSMITH_ENDPOINT="https://api.smith.langchain.com"
```

## Related Resources

- [Datadog Dashboards](/integrations/datadog-dashboards) -- Metrics and dashboard monitoring
- [Observability and Tracing](/guides/observability-tracing) -- Beluga AI observability setup
- [LLM Providers](/providers/llm/openai) -- Configuring LLM providers
