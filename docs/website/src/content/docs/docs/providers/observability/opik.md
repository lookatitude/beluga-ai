---
title: "Opik Observability Provider"
description: "Export LLM traces to Opik in Beluga AI. Workspace-level analytics with trace visualization, cost tracking, and experiment management in Go."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Opik, LLM tracing, observability, workspace analytics, trace visualization, AI monitoring, Go, Beluga AI"
---

The Opik provider exports LLM call data to [Opik](https://www.comet.com/site/products/opik/), Comet's LLM observability platform. It implements the `o11y.TraceExporter` interface and sends trace data through the Opik REST API with workspace-level routing.

Choose Opik when you want workspace-level trace organization integrated with Comet's experiment tracking platform. Opik routes traces to workspaces via HTTP headers, making it easy to separate team, project, or environment data. It also supports self-hosted deployments. For an open-source self-hostable alternative, consider [Langfuse](/docs/providers/observability/langfuse). For OTel-native local debugging, consider [Phoenix](/docs/providers/observability/phoenix).

## Installation

```bash
go get github.com/lookatitude/beluga-ai/o11y/providers/opik
```

## Configuration

| Option | Type | Default | Description |
|---|---|---|---|
| `WithAPIKey(key)` | `string` | (required) | Opik API key (`opik-...`) |
| `WithWorkspace(name)` | `string` | `"default"` | Opik workspace name |
| `WithBaseURL(url)` | `string` | `https://www.comet.com/opik/api` | API endpoint |
| `WithTimeout(d)` | `time.Duration` | `10s` | HTTP request timeout |

**Environment variables:**

| Variable | Maps to |
|---|---|
| `OPIK_API_KEY` | `WithAPIKey` |

## Basic Usage

```go
package main

import (
    "context"
    "log"
    "os"
    "time"

    "github.com/lookatitude/beluga-ai/o11y"
    "github.com/lookatitude/beluga-ai/o11y/providers/opik"
)

func main() {
    exporter, err := opik.New(
        opik.WithAPIKey(os.Getenv("OPIK_API_KEY")),
        opik.WithWorkspace("my-workspace"),
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
            "experiment": "baseline-v2",
        },
    })
    if err != nil {
        log.Fatal(err)
    }
}
```

## Workspace Management

Opik organizes traces into workspaces. Use the `WithWorkspace` option to route traces:

```go
// Team workspace
teamExporter, err := opik.New(
    opik.WithAPIKey(os.Getenv("OPIK_API_KEY")),
    opik.WithWorkspace("ml-team"),
)

// Personal workspace
personalExporter, err := opik.New(
    opik.WithAPIKey(os.Getenv("OPIK_API_KEY")),
    opik.WithWorkspace("my-experiments"),
)
```

The workspace name is sent via the `Comet-Workspace` HTTP header on every request.

## Trace Structure

Each `ExportLLMCall` creates an Opik trace with:

- **Name**: Formatted as `"provider/model"` (e.g., `"openai/gpt-4o"`)
- **Input**: The message history wrapped as `{"messages": ...}`
- **Output**: The model's response wrapped as `{"response": ...}`
- **Metadata**: Includes model, provider, token counts, cost, and any additional metadata
- **Error info**: Populated when the `Error` field in `LLMCallData` is non-empty
- **Timestamps**: Start and end times derived from the call duration

## With MultiExporter

Combine Opik with other observability providers:

```go
import (
    "github.com/lookatitude/beluga-ai/o11y"
    "github.com/lookatitude/beluga-ai/o11y/providers/opik"
    "github.com/lookatitude/beluga-ai/o11y/providers/langfuse"
)

opikExporter, err := opik.New(
    opik.WithAPIKey(os.Getenv("OPIK_API_KEY")),
    opik.WithWorkspace("production"),
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

multi := o11y.NewMultiExporter(opikExporter, lfExporter)
```

## Authentication

Opik uses bearer token authentication. The API key is sent in the `Authorization` header as `Bearer <key>`. API keys are prefixed with `opik-`.

## Self-Hosted Opik

Point the exporter to a self-hosted Opik instance:

```go
exporter, err := opik.New(
    opik.WithAPIKey(os.Getenv("OPIK_API_KEY")),
    opik.WithWorkspace("default"),
    opik.WithBaseURL("https://opik.internal.example.com/api"),
)
```

## Error Handling

```go
err = exporter.ExportLLMCall(ctx, data)
if err != nil {
    // Error format: "opik: export: <underlying error>"
    log.Printf("Opik export failed: %v", err)
}
```

The `Flush` method is a no-op since the provider sends data synchronously via HTTP.
