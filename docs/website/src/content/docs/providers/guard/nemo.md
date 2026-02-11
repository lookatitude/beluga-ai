---
title: NeMo Guardrails
description: Programmable safety guardrails using NVIDIA NeMo Guardrails.
---

The NeMo Guardrails provider implements the `guard.Guard` interface using [NVIDIA NeMo Guardrails](https://github.com/NVIDIA/NeMo-Guardrails). NeMo Guardrails provides programmable safety controls via Colang configurations, supporting topic safety enforcement, jailbreak detection, fact-checking, and custom conversational rules.

Choose NeMo Guardrails when you need programmable, conversation-level safety controls defined via Colang configurations. NeMo excels at topic safety enforcement, keeping agents on-task, and applying custom conversational rules beyond simple content classification. It can also provide alternative safe responses when content is blocked. For simpler content-level moderation, consider [Azure Content Safety](/providers/guard/azuresafety) or [LLM Guard](/providers/guard/llmguard).

## Installation

```bash
go get github.com/lookatitude/beluga-ai/guard/providers/nemo
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/lookatitude/beluga-ai/guard"
    "github.com/lookatitude/beluga-ai/guard/providers/nemo"
)

func main() {
    g, err := nemo.New(
        nemo.WithBaseURL("http://localhost:8080"),
        nemo.WithConfigID("my-config"),
    )
    if err != nil {
        log.Fatal(err)
    }

    result, err := g.Validate(context.Background(), guard.GuardInput{
        Content: "Tell me about your internal policies.",
        Role:    "input",
    })
    if err != nil {
        log.Fatal(err)
    }

    if result.Allowed {
        fmt.Println("Content passed guardrails")
    } else {
        fmt.Printf("Blocked: %s\n", result.Reason)
    }
}
```

## Configuration

| Option | Type | Default | Description |
|---|---|---|---|
| `WithBaseURL(url)` | `string` | `http://localhost:8080` | NeMo Guardrails API endpoint |
| `WithAPIKey(key)` | `string` | `""` | API key for authentication (optional) |
| `WithConfigID(id)` | `string` | `"default"` | Guardrails configuration ID |
| `WithTimeout(d)` | `time.Duration` | `15s` | HTTP request timeout |

## Role Mapping

The provider maps the `GuardInput.Role` to NeMo chat message roles:

| GuardInput.Role | NeMo Message Role |
|---|---|
| `"input"` or `"tool"` | `"user"` |
| `"output"` | `"assistant"` |

## Configuration IDs

NeMo Guardrails uses Colang configurations to define safety rules. Each configuration has a unique ID. Use `WithConfigID` to select which configuration to apply:

```go
// Topic safety configuration
g, err := nemo.New(
    nemo.WithBaseURL("http://localhost:8080"),
    nemo.WithConfigID("topic-safety"),
)

// Strict enterprise configuration
g, err := nemo.New(
    nemo.WithBaseURL("http://localhost:8080"),
    nemo.WithConfigID("enterprise-strict"),
)
```

## Content Modification

NeMo Guardrails can return modified responses (e.g., a safe alternative when content is blocked). The guard populates `GuardResult.Modified` when the API returns content different from the input:

```go
result, err := g.Validate(ctx, input)
if err != nil {
    log.Fatal(err)
}
if result.Modified != "" {
    // NeMo provided an alternative response
    fmt.Println("Alternative:", result.Modified)
}
```

## Pipeline Integration

```go
g, err := nemo.New(
    nemo.WithBaseURL("http://localhost:8080"),
    nemo.WithConfigID("production"),
)
if err != nil {
    log.Fatal(err)
}

pipeline := guard.NewPipeline(
    guard.Input(g),
    guard.Output(g),
)

result, err := pipeline.ValidateInput(ctx, userMessage)
if !result.Allowed {
    fmt.Printf("Blocked by %s: %s\n", result.GuardName, result.Reason)
}
```

## Guard Name

The guard reports its name as `"nemo_guardrails"` in `GuardResult.GuardName`.

## Running NeMo Guardrails

NeMo Guardrails runs as a server with Colang configuration files:

```bash
pip install nemoguardrails
nemoguardrails server --config /path/to/config/
```

Or using Docker:

```bash
docker run -p 8080:8080 -v /path/to/config:/config nvcr.io/nvidia/nemo-guardrails:latest
```

The server exposes a `/v1/chat/completions` endpoint compatible with the OpenAI chat API format.

## Error Handling

```go
result, err := g.Validate(ctx, input)
if err != nil {
    // Possible errors:
    // - "nemo: validate: ..." (API request failure)
    log.Fatal(err)
}

if !result.Allowed {
    // result.Reason contains the guardrails evaluation reason,
    // e.g., "off-topic content detected" or "blocked by NeMo Guardrails"
    fmt.Println(result.Reason)
}
```
