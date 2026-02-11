---
title: Anthropic Claude Enterprise
description: Configure Anthropic Claude Enterprise with Beluga AI for enhanced security, priority access, and enterprise-grade support in production AI applications.
---

Production AI deployments typically require guarantees beyond what standard API tiers provide: higher rate limits for sustained throughput, priority routing during peak demand, extended context windows for processing large documents, and enterprise-grade data handling commitments. Anthropic Claude Enterprise provides these capabilities.

This guide covers configuring Beluga AI to use Claude Enterprise endpoints. The same `anthropic` provider handles both standard and enterprise configurations -- enterprise features are enabled through configuration options, not a separate provider.

## Overview

The Anthropic provider in Beluga AI supports both standard and enterprise Claude configurations through the same `llm.ChatModel` interface. Enterprise features are enabled through configuration options such as custom base URLs, extended context windows, and enterprise API keys.

Key enterprise capabilities:
- Extended context windows (200K tokens for supported models)
- Priority API routing and higher rate limits
- Enhanced security and data handling guarantees
- Dedicated enterprise API endpoints

## Prerequisites

- Go 1.23 or later
- A Beluga AI project initialized with `go mod init`
- An Anthropic Enterprise API key
- Enterprise account access with model permissions configured

## Installation

Install the Anthropic provider:

```bash
go get github.com/lookatitude/beluga-ai/llm/providers/anthropic
```

Set your enterprise API key:

```bash
export ANTHROPIC_API_KEY="sk-ant-enterprise-..."
```

## Configuration

### Basic Enterprise Setup

Create a ChatModel using the Anthropic provider with enterprise credentials:

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/llm"
	_ "github.com/lookatitude/beluga-ai/llm/providers/anthropic"
	"github.com/lookatitude/beluga-ai/schema"
)

func main() {
	ctx := context.Background()

	// Create the Anthropic model via the registry.
	model, err := llm.New("anthropic", config.ProviderConfig{
		Model:   "claude-sonnet-4-5-20250929",
		APIKey:  os.Getenv("ANTHROPIC_API_KEY"),
		BaseURL: "https://api.anthropic.com", // Enterprise endpoint
	})
	if err != nil {
		log.Fatalf("Failed to create model: %v", err)
	}

	// Build a conversation with a system prompt and user message.
	msgs := []schema.Message{
		schema.NewSystemMessage("You are an enterprise AI assistant."),
		schema.NewHumanMessage("Analyze this enterprise data..."),
	}

	resp, err := model.Generate(ctx, msgs)
	if err != nil {
		log.Fatalf("Generate failed: %v", err)
	}

	fmt.Printf("Response: %s\n", resp.Text())
}
```

### Extended Context and Temperature

Use `GenerateOption` functional options to control model behavior per request:

```go
// Use the extended 200K context window with a lower temperature.
resp, err := model.Generate(ctx, msgs,
	llm.WithMaxTokens(4096),
	llm.WithTemperature(0.3),
)
if err != nil {
	log.Fatalf("Generate failed: %v", err)
}
```

### Custom Enterprise Endpoint

If your enterprise account uses a dedicated API endpoint, configure it via `BaseURL`:

```go
model, err := llm.New("anthropic", config.ProviderConfig{
	Model:   "claude-opus-4-20250514",
	APIKey:  os.Getenv("ANTHROPIC_API_KEY"),
	BaseURL: "https://enterprise.api.anthropic.com",
	Timeout: 60 * time.Second,
})
```

## Usage

### Streaming Responses

Enterprise accounts benefit from priority streaming. Use the `Stream` method for real-time output:

```go
for chunk, err := range model.Stream(ctx, msgs) {
	if err != nil {
		log.Fatalf("Stream error: %v", err)
	}
	fmt.Print(chunk.Delta)
}
fmt.Println()
```

### Tool Binding

Bind tools to the enterprise model for function calling:

```go
toolModel := model.BindTools([]schema.ToolDefinition{
	{
		Name:        "lookup_customer",
		Description: "Look up a customer record by ID.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"customer_id": map[string]any{"type": "string"},
			},
			"required": []any{"customer_id"},
		},
	},
})

resp, err := toolModel.Generate(ctx, msgs)
if err != nil {
	log.Fatalf("Generate failed: %v", err)
}
```

### Adding Resilience

Wrap the enterprise model with Beluga's resilience middleware for production use:

```go
import "github.com/lookatitude/beluga-ai/resilience"

policy := resilience.RetryPolicy{
	MaxAttempts:    5,
	InitialBackoff: 500 * time.Millisecond,
	MaxBackoff:     30 * time.Second,
	BackoffFactor:  2.0,
	Jitter:         true,
}

// Use resilience.Do to wrap calls with retry logic.
var resp *schema.AIMessage
err := resilience.Do(ctx, policy, func(ctx context.Context) error {
	var genErr error
	resp, genErr = model.Generate(ctx, msgs)
	return genErr
})
if err != nil {
	log.Fatalf("Generate failed after retries: %v", err)
}
```

## Advanced Topics

### Observability

Integrate OpenTelemetry tracing to monitor enterprise API calls:

```go
import (
	"github.com/lookatitude/beluga-ai/llm"
	"go.opentelemetry.io/otel"
)

tracer := otel.Tracer("beluga.llm.anthropic")

ctx, span := tracer.Start(ctx, "anthropic.generate")
defer span.End()

resp, err := model.Generate(ctx, msgs)
if err != nil {
	span.RecordError(err)
	log.Fatalf("Generate failed: %v", err)
}
```

Beluga's `o11y` package provides built-in LLM middleware that automatically records `gen_ai.*` attributes on spans when configured.

### Middleware Composition

Apply cross-cutting concerns using Beluga's middleware pattern:

```go
// Wrap with logging and fallback middleware.
model = llm.ApplyMiddleware(model,
	llm.WithLogging(slog.Default()),
)
```

## Configuration Reference

| Option | Description | Default | Required |
|--------|-------------|---------|----------|
| `Model` | Claude model ID (e.g., `claude-sonnet-4-5-20250929`) | -- | Yes |
| `APIKey` | Anthropic Enterprise API key | `ANTHROPIC_API_KEY` env var | Yes |
| `BaseURL` | API endpoint URL | `https://api.anthropic.com` | No |
| `Timeout` | Maximum request duration | `30s` | No |

## Troubleshooting

### "anthropic: model is required"

The `Model` field in `config.ProviderConfig` is empty. Specify a valid Claude model identifier.

### "authentication_error"

The API key is invalid or does not have enterprise permissions. Verify that:
1. The `ANTHROPIC_API_KEY` environment variable is set correctly.
2. The key is an enterprise-tier key, not a standard API key.
3. The key has not been revoked or expired.

### Rate Limit Errors

Enterprise accounts have higher rate limits, but they can still be exceeded under heavy load. Use the `resilience` package retry policy shown above to handle transient rate limit responses.

## Related Resources

- [LLM Providers Overview](/integrations/llm-providers) -- All supported LLM providers
- [AWS Bedrock Integration](/integrations/bedrock-integration) -- Alternative cloud LLM provider
- [Resilience Package](/guides/resilience) -- Retry, circuit breaker, and rate limiting
