---
title: Model Switching and Fallbacks
description: Implement reliability fallbacks and cost-optimizing routing across multiple LLM providers.
---

Relying on a single LLM provider creates a single point of failure. Rate limits, outages, and cost spikes can all disrupt your application. Beluga AI provides built-in middleware for fallback chains and a Router for intelligent multi-model dispatching.

## What You Will Build

Three model switching strategies — reliability fallbacks, cost-optimizing routing, and the built-in `FailoverRouter` for automatic multi-model failover.

## Prerequisites

- Understanding of the [ChatModel interface](/guides/llm) and [middleware](/tutorials/foundation/middleware-implementation)
- API keys for at least two LLM providers

## Pattern 1: Fallback Middleware

Use `llm.WithFallback` to automatically switch to a backup model on retryable errors:

```go
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/lookatitude/beluga-ai/config"
    "github.com/lookatitude/beluga-ai/llm"
    _ "github.com/lookatitude/beluga-ai/llm/providers/openai"
    _ "github.com/lookatitude/beluga-ai/llm/providers/anthropic"
    "github.com/lookatitude/beluga-ai/schema"
)

func main() {
    ctx := context.Background()

    primary, err := llm.New("openai", config.ProviderConfig{
        "api_key": os.Getenv("OPENAI_API_KEY"),
        "model":   "gpt-4o",
    })
    if err != nil {
        fmt.Printf("Primary error: %v\n", err)
        return
    }

    backup, err := llm.New("anthropic", config.ProviderConfig{
        "api_key": os.Getenv("ANTHROPIC_API_KEY"),
        "model":   "claude-sonnet-4-5-20250929",
    })
    if err != nil {
        fmt.Printf("Backup error: %v\n", err)
        return
    }

    // Wrap primary with fallback — on retryable errors, switches to backup
    model := llm.ApplyMiddleware(primary, llm.WithFallback(backup))

    msgs := []schema.Message{
        schema.NewHumanMessage("Explain Go interfaces in 3 sentences."),
    }

    resp, err := model.Generate(ctx, msgs)
    if err != nil {
        fmt.Printf("Both models failed: %v\n", err)
        return
    }
    fmt.Println(resp.Text())
}
```

The fallback triggers on retryable errors (rate limits, server errors). Non-retryable errors (invalid API key, malformed request) are returned immediately.

## Pattern 2: FailoverRouter

The `FailoverRouter` tries models in order, moving to the next on retryable errors:

```go
func main() {
    ctx := context.Background()

    gpt4, _ := llm.New("openai", config.ProviderConfig{
        "api_key": os.Getenv("OPENAI_API_KEY"),
        "model":   "gpt-4o",
    })

    claude, _ := llm.New("anthropic", config.ProviderConfig{
        "api_key": os.Getenv("ANTHROPIC_API_KEY"),
        "model":   "claude-sonnet-4-5-20250929",
    })

    gpt35, _ := llm.New("openai", config.ProviderConfig{
        "api_key": os.Getenv("OPENAI_API_KEY"),
        "model":   "gpt-4o-mini",
    })

    // Try models in order: gpt-4o -> claude -> gpt-4o-mini
    router := llm.NewFailoverRouter(gpt4, claude, gpt35)

    msgs := []schema.Message{
        schema.NewHumanMessage("What is 2 + 2?"),
    }

    resp, err := router.Generate(ctx, msgs)
    if err != nil {
        fmt.Printf("All models failed: %v\n", err)
        return
    }
    fmt.Printf("Response from: %s\n%s\n", resp.ModelID, resp.Text())
}
```

## Pattern 3: Round-Robin Load Balancing

Distribute load evenly across models:

```go
router := llm.NewRouter(
    llm.WithModels(gpt4, claude, gpt35),
    llm.WithStrategy(&llm.RoundRobin{}), // Default strategy
)

// Each call goes to the next model in rotation
for i := 0; i < 6; i++ {
    resp, _ := router.Generate(ctx, msgs)
    fmt.Printf("Request %d handled by: %s\n", i+1, resp.ModelID)
}
```

## Pattern 4: Cost-Optimizing Router

Route simple queries to cheaper models, complex queries to more capable ones:

```go
// ComplexityRouter routes based on input complexity
type ComplexityRouter struct {
    classifier llm.ChatModel
}

func (cr *ComplexityRouter) Select(ctx context.Context, models []llm.ChatModel, msgs []schema.Message) (llm.ChatModel, error) {
    if len(models) < 2 {
        return models[0], nil
    }

    // Use a fast model to classify complexity
    classifyMsgs := []schema.Message{
        schema.NewSystemMessage("Classify this as SIMPLE or COMPLEX. Respond with one word."),
        msgs[len(msgs)-1], // Last user message
    }

    resp, err := cr.classifier.Generate(ctx, classifyMsgs, llm.WithMaxTokens(5))
    if err != nil {
        return models[0], nil // Default to first model on error
    }

    if strings.Contains(strings.ToUpper(resp.Text()), "COMPLEX") {
        return models[0], nil // Expensive model for complex queries
    }
    return models[len(models)-1], nil // Cheap model for simple queries
}
```

Usage:

```go
classifier, _ := llm.New("openai", config.ProviderConfig{
    "api_key": os.Getenv("OPENAI_API_KEY"),
    "model":   "gpt-4o-mini", // Cheap classifier
})

router := llm.NewRouter(
    llm.WithModels(gpt4, gpt35), // gpt4 first (expensive), gpt35 last (cheap)
    llm.WithStrategy(&ComplexityRouter{classifier: classifier}),
)
```

## Verification

1. Mock a rate limit error on the primary model — verify the fallback triggers.
2. Send 10 requests through a round-robin router — verify even distribution.
3. Send both "What is 2+2?" (simple) and "Design a microservices architecture" (complex) — verify the cost router selects appropriate models.

## Next Steps

- [Multi-provider Chat](/tutorials/agents/multi-provider) — Unified interface across providers
- [Advanced Inference](/tutorials/providers/advanced-inference) — Per-call generation options
