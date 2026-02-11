---
title: LangChain Hub Prompt Loading
description: Load community prompt templates from LangChain Hub and convert them to Beluga AI format for use in your applications.
---

Writing effective prompts from scratch is time-consuming. LangChain Hub hosts a community-curated library of prompt templates for common tasks -- RAG, summarization, classification, extraction -- that have been tested and refined by thousands of developers. By loading these templates into Beluga AI's prompt system, you can bootstrap new features with battle-tested prompts and customize them for your domain. This guide shows how to fetch prompts from the Hub API, convert them to Beluga AI templates, and use them in your applications.

## Overview

LangChain Hub provides a repository of reusable prompt templates. By building a thin loader on top of Beluga AI's prompt system, you can:

- Load prompts from the LangChain Hub REST API
- Convert them to Beluga AI `PromptTemplate` format
- Use them alongside locally defined templates
- Cache loaded prompts for production use

## Prerequisites

- Go 1.23 or later
- Beluga AI framework installed
- LangChain Hub API key (optional, required for private prompts)
- Familiarity with the `prompt` package

## Installation

The loader uses only the Go standard library for HTTP and Beluga AI's prompt package:

```bash
go get github.com/lookatitude/beluga-ai
```

The LangChain Hub API endpoint is `https://api.hub.langchain.com`.

## Usage

### Loading a Prompt from LangChain Hub

Define a loader that fetches prompts from the Hub API and parses the response:

```go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/lookatitude/beluga-ai/prompt"
)

// LangChainHubPrompt represents the response format from the Hub API.
type LangChainHubPrompt struct {
	Messages []struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"messages"`
	InputVariables []string `json:"input_variables"`
}

// LangChainHubLoader fetches prompt templates from LangChain Hub.
type LangChainHubLoader struct {
	baseURL    string
	httpClient *http.Client
}

func NewLangChainHubLoader() *LangChainHubLoader {
	return &LangChainHubLoader{
		baseURL: "https://api.hub.langchain.com",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (l *LangChainHubLoader) LoadPrompt(ctx context.Context, owner, repo, promptID string) (*LangChainHubPrompt, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/prompts/%s", l.baseURL, owner, repo, promptID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := l.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch prompt: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var hubPrompt LangChainHubPrompt
	if err := json.Unmarshal(body, &hubPrompt); err != nil {
		return nil, fmt.Errorf("failed to parse prompt: %w", err)
	}

	return &hubPrompt, nil
}
```

### Converting to Beluga AI Format

Once loaded, convert the Hub prompt into a Beluga AI `PromptTemplate`:

```go
func (l *LangChainHubLoader) ConvertToBelugaTemplate(ctx context.Context, hubPrompt *LangChainHubPrompt) (prompt.PromptTemplate, error) {
	manager, err := prompt.NewPromptManager()
	if err != nil {
		return nil, fmt.Errorf("failed to create manager: %w", err)
	}

	// Concatenate message content into a single template string
	var templateStr string
	for _, msg := range hubPrompt.Messages {
		templateStr += fmt.Sprintf("%s: %s\n", msg.Role, msg.Content)
	}

	template, err := manager.NewStringTemplate(templateStr, hubPrompt.InputVariables)
	if err != nil {
		return nil, fmt.Errorf("failed to create template: %w", err)
	}

	return template, nil
}
```

### Using the Loaded Prompt

```go
func main() {
	ctx := context.Background()

	loader := NewLangChainHubLoader()
	hubPrompt, err := loader.LoadPrompt(ctx, "langchain-ai", "prompt-hub", "prompts/rag")
	if err != nil {
		log.Fatalf("Failed to load prompt: %v", err)
	}

	template, err := loader.ConvertToBelugaTemplate(ctx, hubPrompt)
	if err != nil {
		log.Fatalf("Failed to convert prompt: %v", err)
	}

	result, err := template.Format(ctx, map[string]any{
		"context":  "Machine learning is...",
		"question": "What is ML?",
	})
	if err != nil {
		log.Fatalf("Failed to format: %v", err)
	}

	fmt.Printf("Formatted prompt: %s\n", result.ToString())
}
```

## Advanced Topics

### Production-Ready Loader with Observability

For production use, add OpenTelemetry tracing to track API calls and conversion latency:

```go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/lookatitude/beluga-ai/prompt"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type LangChainHubLoader struct {
	baseURL    string
	httpClient *http.Client
	tracer     trace.Tracer
}

func NewLangChainHubLoader() *LangChainHubLoader {
	return &LangChainHubLoader{
		baseURL: "https://api.hub.langchain.com",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		tracer: otel.Tracer("beluga.prompt.langchain_hub"),
	}
}

func (l *LangChainHubLoader) LoadAndConvert(ctx context.Context, owner, repo, promptID string) (prompt.PromptTemplate, error) {
	ctx, span := l.tracer.Start(ctx, "langchain_hub.load",
		trace.WithAttributes(
			attribute.String("owner", owner),
			attribute.String("repo", repo),
			attribute.String("prompt_id", promptID),
		),
	)
	defer span.End()

	// Fetch from Hub API
	url := fmt.Sprintf("%s/repos/%s/%s/prompts/%s", l.baseURL, owner, repo, promptID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := l.httpClient.Do(req)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("unexpected status: %d", resp.StatusCode)
		span.RecordError(err)
		return nil, err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to read: %w", err)
	}

	var hubPrompt struct {
		Messages       []map[string]interface{} `json:"messages"`
		InputVariables []string                 `json:"input_variables"`
	}
	if err := json.Unmarshal(body, &hubPrompt); err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to parse: %w", err)
	}

	// Convert to Beluga AI template
	manager, err := prompt.NewPromptManager()
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to create manager: %w", err)
	}

	templateStr := ""
	for _, msg := range hubPrompt.Messages {
		if content, ok := msg["content"].(string); ok {
			templateStr += content + "\n"
		}
	}

	template, err := manager.NewStringTemplate(templateStr, hubPrompt.InputVariables)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to create template: %w", err)
	}

	span.SetAttributes(attribute.Int("input_variables", len(hubPrompt.InputVariables)))
	return template, nil
}

func main() {
	ctx := context.Background()

	loader := NewLangChainHubLoader()
	template, err := loader.LoadAndConvert(ctx, "langchain-ai", "prompt-hub", "prompts/rag")
	if err != nil {
		log.Fatalf("Failed to load prompt: %v", err)
	}

	result, err := template.Format(ctx, map[string]any{
		"context":  "AI is transforming technology.",
		"question": "What is AI?",
	})
	if err != nil {
		log.Fatalf("Failed to format: %v", err)
	}

	fmt.Printf("Prompt: %s\n", result.ToString())
}
```

## Configuration

| Option | Description | Default | Required |
|--------|-------------|---------|----------|
| `BaseURL` | LangChain Hub API URL | `https://api.hub.langchain.com` | No |
| `Timeout` | HTTP request timeout | `30s` | No |
| `APIKey` | API key for private prompts | - | No |

## Troubleshooting

### Prompt not found

The owner, repo, or prompt ID may be incorrect. Verify the prompt path exists by querying the Hub API directly:

```bash
curl https://api.hub.langchain.com/repos/langchain-ai/prompt-hub/prompts/rag
```

### Format conversion failed

LangChain prompt formats can vary. When encountering incompatible formats, inspect the raw JSON response and handle the specific message structure before passing it to the template converter.

## Production Considerations

- **Cache prompts** locally to reduce API calls and improve startup time
- **Pin prompt versions** in production to avoid unexpected changes
- **Validate loaded prompts** before use to catch format mismatches early
- **Provide local fallbacks** in case the Hub API is unreachable
- **Handle network failures** gracefully with retries and timeouts

## Related Resources

- [Filesystem Template Store](/integrations/filesystem-templates) -- Local prompt storage
- [Prompt Package Reference](/api-reference/prompt) -- Prompt management API
