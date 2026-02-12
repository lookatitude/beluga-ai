---
title: "Phoenix Observability Provider"
description: "Export LLM traces to Phoenix in Beluga AI. OpenTelemetry-native observability with trace analysis, embeddings viewer, and local dashboards in Go."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Phoenix, LLM tracing, OpenTelemetry, observability, trace analysis, embeddings viewer, Go, Beluga AI"
---

The Phoenix provider exports LLM call data to [Phoenix](https://phoenix.arize.com/), Arize AI's open-source LLM observability platform. It implements the `o11y.TraceExporter` interface and sends OTel-compatible span data through the Phoenix traces API.

Choose Phoenix when you want a local-first, OTel-native observability tool for development and debugging. Phoenix runs locally with no API keys required, making it ideal for rapid iteration. It uses OpenTelemetry-compatible span formats, aligning with Beluga's OTel GenAI conventions. For production observability with team features, consider [Langfuse](/providers/observability/langfuse) or [LangSmith](/providers/observability/langsmith).

## Installation

```bash
go get github.com/lookatitude/beluga-ai/o11y/providers/phoenix
```

## Configuration

| Option | Type | Default | Description |
|---|---|---|---|
| `WithBaseURL(url)` | `string` | `http://localhost:6006` | Phoenix server endpoint |
| `WithAPIKey(key)` | `string` | — | Optional bearer token for authentication |
| `WithTimeout(d)` | `time.Duration` | `10s` | HTTP request timeout |

## Basic Usage

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/lookatitude/beluga-ai/o11y"
    "github.com/lookatitude/beluga-ai/o11y/providers/phoenix"
)

func main() {
    exporter, err := phoenix.New(
        phoenix.WithBaseURL("http://localhost:6006"),
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
        },
    })
    if err != nil {
        log.Fatal(err)
    }
}
```

## Local Development

Phoenix is commonly run locally for development. Start a Phoenix server and point the exporter at it:

```bash
# Start Phoenix (Python)
pip install arize-phoenix
phoenix serve

# Phoenix UI available at http://localhost:6006
```

```go
exporter, err := phoenix.New(
    phoenix.WithBaseURL("http://localhost:6006"),
)
```

No API key is needed for local Phoenix instances.

## Cloud Phoenix

For hosted Phoenix deployments, provide an API key:

```go
exporter, err := phoenix.New(
    phoenix.WithBaseURL("https://phoenix.example.com"),
    phoenix.WithAPIKey(os.Getenv("PHOENIX_API_KEY")),
)
```

The API key is sent as a bearer token in the `Authorization` header.

## OTel-Native Span Format

Phoenix uses an OpenTelemetry-compatible span format. Each `ExportLLMCall` creates a span with:

- **Span kind**: `"LLM"`
- **Trace and span IDs**: Generated as random hex strings (32-char trace ID, 16-char span ID)
- **Status**: `"OK"` on success, `"ERROR"` with a message on failure
- **Attributes**: Mapped to OTel GenAI conventions

### Attribute Mapping

| Attribute | Description |
|---|---|
| `llm.model_name` | Model identifier |
| `llm.provider` | Provider name |
| `llm.token_count.prompt` | Input token count |
| `llm.token_count.completion` | Output token count |
| `llm.token_count.total` | Total token count |
| `llm.cost` | Estimated cost |
| `input.value` | Serialized input messages |
| `output.value` | Model response text |
| `metadata.*` | Custom metadata fields (prefixed) |

## With MultiExporter

Combine Phoenix with other observability providers:

```go
import (
    "github.com/lookatitude/beluga-ai/o11y"
    "github.com/lookatitude/beluga-ai/o11y/providers/phoenix"
    "github.com/lookatitude/beluga-ai/o11y/providers/langsmith"
)

pxExporter, err := phoenix.New(
    phoenix.WithBaseURL("http://localhost:6006"),
)
if err != nil {
    log.Fatal(err)
}

lsExporter, err := langsmith.New(
    langsmith.WithAPIKey(os.Getenv("LANGSMITH_API_KEY")),
)
if err != nil {
    log.Fatal(err)
}

multi := o11y.NewMultiExporter(pxExporter, lsExporter)
```

## Error Handling

```go
err = exporter.ExportLLMCall(ctx, data)
if err != nil {
    // Error format: "phoenix: export trace: <underlying error>"
    log.Printf("Phoenix export failed: %v", err)
}
```

The `Flush` method is a no-op since the provider sends data synchronously via HTTP.
