---
title: LLM Guard
description: Prompt and output scanning using LLM Guard.
---

The LLM Guard provider implements the `guard.Guard` interface using the [LLM Guard](https://llm-guard.com/) API. LLM Guard is an open-source toolkit that provides prompt injection detection, toxicity filtering, and sensitive data detection through a collection of configurable scanners.

Choose LLM Guard when you need a self-hosted, open-source safety solution with no external API dependencies. LLM Guard runs locally and provides both prompt and output scanning with configurable scanners. It also supports content sanitization (e.g., masking sensitive data in outputs). For a cloud-hosted prompt injection specialist, consider [Lakera Guard](/providers/guard/lakera). For programmable conversational rules, consider [NeMo Guardrails](/providers/guard/nemo).

## Installation

```bash
go get github.com/lookatitude/beluga-ai/guard/providers/llmguard
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/lookatitude/beluga-ai/guard"
    "github.com/lookatitude/beluga-ai/guard/providers/llmguard"
)

func main() {
    g, err := llmguard.New(
        llmguard.WithBaseURL("http://localhost:8000"),
    )
    if err != nil {
        log.Fatal(err)
    }

    result, err := g.Validate(context.Background(), guard.GuardInput{
        Content: "What is the capital of France?",
        Role:    "input",
    })
    if err != nil {
        log.Fatal(err)
    }

    if result.Allowed {
        fmt.Println("Content passed all scanners")
    } else {
        fmt.Printf("Blocked: %s\n", result.Reason)
    }
}
```

## Configuration

| Option | Type | Default | Description |
|---|---|---|---|
| `WithBaseURL(url)` | `string` | `http://localhost:8000` | LLM Guard API endpoint |
| `WithAPIKey(key)` | `string` | `""` | API key for authentication (optional) |
| `WithTimeout(d)` | `time.Duration` | `15s` | HTTP request timeout |

## Role-Based Endpoints

The provider automatically selects the appropriate LLM Guard endpoint based on the `GuardInput.Role`:

| Role | Endpoint | Description |
|---|---|---|
| `"input"` or `"tool"` | `/analyze/prompt` | Scans prompts and tool inputs |
| `"output"` | `/analyze/output` | Scans model responses |

```go
// Prompt scanning
result, err := g.Validate(ctx, guard.GuardInput{
    Content: userPrompt,
    Role:    "input",
})

// Output scanning
result, err := g.Validate(ctx, guard.GuardInput{
    Content: modelOutput,
    Role:    "output",
})
```

## Scanner Results

When content is blocked, the `GuardResult.Reason` identifies the failing scanner with its score and threshold:

```
"scanner PromptInjection failed (score=0.95, threshold=0.50)"
```

## Content Sanitization

LLM Guard can return sanitized versions of content (e.g., with sensitive data masked). The guard populates `GuardResult.Modified` when sanitized content is available:

```go
result, err := g.Validate(ctx, input)
if err != nil {
    log.Fatal(err)
}
if result.Modified != "" {
    // Use sanitized content instead of original
    processedContent = result.Modified
}
```

The sanitized content source depends on the role:
- For input/tool: `sanitized_prompt` from the API response
- For output: `sanitized_output` from the API response

## Pipeline Integration

LLM Guard works well for both input and output stages:

```go
g, err := llmguard.New(
    llmguard.WithBaseURL("http://localhost:8000"),
)
if err != nil {
    log.Fatal(err)
}

pipeline := guard.NewPipeline(
    guard.Input(g),   // Scan prompts
    guard.Output(g),  // Scan responses
)
```

## Guard Name

The guard reports its name as `"llm_guard"` in `GuardResult.GuardName`.

## Running LLM Guard

LLM Guard can be deployed as a standalone API server:

```bash
pip install llm-guard[api]
llm_guard_api
```

Or using Docker:

```bash
docker run -p 8000:8000 laiyer/llm-guard-api:latest
```

## Error Handling

```go
result, err := g.Validate(ctx, input)
if err != nil {
    // Possible errors:
    // - "llmguard: validate: ..." (API request failure)
    log.Fatal(err)
}
```
