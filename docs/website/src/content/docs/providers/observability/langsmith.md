---
title: "LangSmith"
description: "Export LLM traces to LangSmith for observability and debugging."
---

The LangSmith provider exports LLM call data to [LangSmith](https://smith.langchain.com/), LangChain's platform for debugging, testing, and monitoring LLM applications. It implements the `o11y.TraceExporter` interface and sends run data through the LangSmith batch API.

## Installation

```bash
go get github.com/lookatitude/beluga-ai/o11y/providers/langsmith
```

## Configuration

| Option | Type | Default | Description |
|---|---|---|---|
| `WithAPIKey(key)` | `string` | (required) | LangSmith API key (`lsv2_...`) |
| `WithProject(name)` | `string` | `"default"` | LangSmith project name |
| `WithBaseURL(url)` | `string` | `https://api.smith.langchain.com` | API endpoint |
| `WithTimeout(d)` | `time.Duration` | `10s` | HTTP request timeout |

**Environment variables:**

| Variable | Maps to |
|---|---|
| `LANGSMITH_API_KEY` | `WithAPIKey` |

## Basic Usage

```go
package main

import (
    "context"
    "log"
    "os"
    "time"

    "github.com/lookatitude/beluga-ai/o11y"
    "github.com/lookatitude/beluga-ai/o11y/providers/langsmith"
)

func main() {
    exporter, err := langsmith.New(
        langsmith.WithAPIKey(os.Getenv("LANGSMITH_API_KEY")),
        langsmith.WithProject("my-project"),
    )
    if err != nil {
        log.Fatal(err)
    }

    err = exporter.ExportLLMCall(context.Background(), o11y.LLMCallData{
        Model:        "gpt-4o",
        Provider:     "openai",
        InputTokens:  500,
        OutputTokens: 150,
        Duration:     1200 * time.Millisecond,
        Cost:         0.003,
        Response:     "The capital of France is Paris.",
        Metadata: map[string]any{
            "temperature": 0.7,
            "max_tokens":  2048,
        },
    })
    if err != nil {
        log.Fatal(err)
    }
}
```

## Project Organization

LangSmith organizes traces into projects. Use the `WithProject` option to route traces to a specific project:

```go
// Development traces
devExporter, err := langsmith.New(
    langsmith.WithAPIKey(os.Getenv("LANGSMITH_API_KEY")),
    langsmith.WithProject("my-app-dev"),
)

// Production traces
prodExporter, err := langsmith.New(
    langsmith.WithAPIKey(os.Getenv("LANGSMITH_API_KEY")),
    langsmith.WithProject("my-app-prod"),
)
```

## Run Model

Each `ExportLLMCall` creates a LangSmith "run" with:

- **Run type**: `"llm"`
- **Name**: Formatted as `"provider/model"` (e.g., `"openai/gpt-4o"`)
- **Inputs**: The message history sent to the model
- **Outputs**: The model's response
- **Extras**: Token counts, cost, duration, and any additional metadata
- **Session name**: The configured project name

## With MultiExporter

Combine LangSmith with other observability providers:

```go
import (
    "github.com/lookatitude/beluga-ai/o11y"
    "github.com/lookatitude/beluga-ai/o11y/providers/langsmith"
    "github.com/lookatitude/beluga-ai/o11y/providers/langfuse"
)

lsExporter, err := langsmith.New(
    langsmith.WithAPIKey(os.Getenv("LANGSMITH_API_KEY")),
)
if err != nil {
    log.Fatal(err)
}

lfExporter, err := langfuse.New(
    langfuse.WithPublicKey(os.Getenv("LANGFUSE_PUBLIC_KEY")),
    langfuse.WithSecretKey(os.Getenv("LANGFUSE_SECRET_KEY")),
)
if err != nil {
    log.Fatal(err)
}

multi := o11y.NewMultiExporter(lsExporter, lfExporter)
```

## Authentication

LangSmith uses API key authentication via the `x-api-key` HTTP header. API keys are prefixed with `lsv2_`.

## Error Handling

```go
err = exporter.ExportLLMCall(ctx, data)
if err != nil {
    // Errors include authentication failures, network issues, and API errors
    log.Printf("LangSmith export failed: %v", err)
}
```

The `Flush` method is a no-op since the provider sends data synchronously via HTTP.
