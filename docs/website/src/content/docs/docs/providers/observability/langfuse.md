---
title: "Langfuse Observability Provider"
description: "Export LLM traces to Langfuse in Beluga AI. Open-source tracing with cost tracking, prompt management, and evaluation analytics in Go."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Langfuse, LLM tracing, observability, cost tracking, open-source, AI analytics, Go, Beluga AI"
---

The Langfuse provider exports LLM call data to [Langfuse](https://langfuse.com/), an open-source LLM observability platform. It implements the `o11y.TraceExporter` interface and sends trace and generation events through the Langfuse batch ingestion API.

Choose Langfuse when you want an open-source observability platform that can be self-hosted or used as a managed cloud service. Langfuse provides trace and generation analytics with a dashboard for cost tracking and latency monitoring. For LangChain ecosystem integration, consider [LangSmith](/providers/observability/langsmith). For OTel-native local debugging, consider [Phoenix](/providers/observability/phoenix).

## Installation

```bash
go get github.com/lookatitude/beluga-ai/o11y/providers/langfuse
```

## Configuration

| Option | Type | Default | Description |
|---|---|---|---|
| `WithPublicKey(key)` | `string` | (required) | Langfuse public key |
| `WithSecretKey(key)` | `string` | (required) | Langfuse secret key |
| `WithBaseURL(url)` | `string` | `https://cloud.langfuse.com` | Langfuse API endpoint |
| `WithTimeout(d)` | `time.Duration` | `10s` | HTTP request timeout |

**Environment variables:**

| Variable | Maps to |
|---|---|
| `LANGFUSE_PUBLIC_KEY` | `WithPublicKey` |
| `LANGFUSE_SECRET_KEY` | `WithSecretKey` |

## Basic Usage

```go
package main

import (
    "context"
    "log"
    "os"
    "time"

    "github.com/lookatitude/beluga-ai/o11y"
    "github.com/lookatitude/beluga-ai/o11y/providers/langfuse"
)

func main() {
    exporter, err := langfuse.New(
        langfuse.WithPublicKey(os.Getenv("LANGFUSE_PUBLIC_KEY")),
        langfuse.WithSecretKey(os.Getenv("LANGFUSE_SECRET_KEY")),
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
            "user_id":    "user-123",
            "session_id": "sess-456",
        },
    })
    if err != nil {
        log.Fatal(err)
    }
}
```

## Self-Hosted Langfuse

Point the exporter to a self-hosted Langfuse instance:

```go
exporter, err := langfuse.New(
    langfuse.WithPublicKey(os.Getenv("LANGFUSE_PUBLIC_KEY")),
    langfuse.WithSecretKey(os.Getenv("LANGFUSE_SECRET_KEY")),
    langfuse.WithBaseURL("https://langfuse.internal.example.com"),
)
```

## With MultiExporter

Combine Langfuse with other observability providers:

```go
import (
    "github.com/lookatitude/beluga-ai/o11y"
    "github.com/lookatitude/beluga-ai/o11y/providers/langfuse"
    "github.com/lookatitude/beluga-ai/o11y/providers/phoenix"
)

lfExporter, err := langfuse.New(
    langfuse.WithPublicKey(os.Getenv("LANGFUSE_PUBLIC_KEY")),
    langfuse.WithSecretKey(os.Getenv("LANGFUSE_SECRET_KEY")),
)
if err != nil {
    log.Fatal(err)
}

pxExporter, err := phoenix.New(
    phoenix.WithBaseURL("http://localhost:6006"),
)
if err != nil {
    log.Fatal(err)
}

multi := o11y.NewMultiExporter(lfExporter, pxExporter)
```

## Trace Structure

Each `ExportLLMCall` creates two events in Langfuse's batch ingestion API:

1. **trace-create** -- A top-level trace capturing the operation name, metadata, and timestamps
2. **generation-create** -- A child generation event linked to the trace, containing the model, token usage, cost, and duration

This structure provides both a high-level view of operations and detailed per-generation analytics in the Langfuse dashboard.

## Authentication

Langfuse uses Basic authentication. The provider encodes the public key and secret key as `base64(publicKey:secretKey)` and sends it in the `Authorization` header.

## Error Handling

```go
err = exporter.ExportLLMCall(ctx, data)
if err != nil {
    // Errors include authentication failures, network issues, and API errors
    log.Printf("Langfuse export failed: %v", err)
}
```

The `Flush` method is a no-op since the provider sends data synchronously via HTTP.
