---
title: Guardrails AI
description: Content validation using Guardrails AI validators.
---

The Guardrails AI provider implements the `guard.Guard` interface using the [Guardrails AI](https://www.guardrailsai.com/) platform. Guardrails AI provides a library of validators for PII detection, toxicity filtering, hallucination detection, prompt injection prevention, and custom rules defined via RAIL specifications.

## Installation

```bash
go get github.com/lookatitude/beluga-ai/guard/providers/guardrailsai
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/lookatitude/beluga-ai/guard"
    "github.com/lookatitude/beluga-ai/guard/providers/guardrailsai"
)

func main() {
    g, err := guardrailsai.New(
        guardrailsai.WithBaseURL("http://localhost:8000"),
        guardrailsai.WithGuardName("my-guard"),
    )
    if err != nil {
        log.Fatal(err)
    }

    result, err := g.Validate(context.Background(), guard.GuardInput{
        Content: "Please process this request.",
        Role:    "input",
    })
    if err != nil {
        log.Fatal(err)
    }

    if result.Allowed {
        fmt.Println("Content passed validation")
    } else {
        fmt.Printf("Blocked: %s\n", result.Reason)
    }
}
```

## Configuration

| Option | Type | Default | Description |
|---|---|---|---|
| `WithBaseURL(url)` | `string` | `http://localhost:8000` | Guardrails AI API endpoint |
| `WithAPIKey(key)` | `string` | `""` | API key for authentication (optional) |
| `WithGuardName(name)` | `string` | `"default"` | Guard name to invoke on the server |
| `WithTimeout(d)` | `time.Duration` | `15s` | HTTP request timeout |

## Role-Based Validation

The provider maps the `GuardInput.Role` to the appropriate API field:

| Role | API Field | Description |
|---|---|---|
| `"input"` | `prompt` | Validates as user input / prompt |
| `"output"` or `"tool"` | `llmOutput` | Validates as LLM or tool output |

```go
// Validate user input
result, err := g.Validate(ctx, guard.GuardInput{
    Content: userMessage,
    Role:    "input",
})

// Validate model output
result, err := g.Validate(ctx, guard.GuardInput{
    Content: modelResponse,
    Role:    "output",
})
```

## Content Modification

When Guardrails AI returns a sanitized version of the content (e.g., with PII redacted), the guard populates `GuardResult.Modified`:

```go
result, err := g.Validate(ctx, input)
if err != nil {
    log.Fatal(err)
}
if result.Modified != "" {
    // Use the sanitized version
    fmt.Println("Sanitized:", result.Modified)
}
```

## Pipeline Integration

```go
g, err := guardrailsai.New(
    guardrailsai.WithBaseURL("http://localhost:8000"),
    guardrailsai.WithGuardName("production-guard"),
)
if err != nil {
    log.Fatal(err)
}

pipeline := guard.NewPipeline(
    guard.Input(g),
    guard.Output(g),
)
```

## Guard Name

The guard reports its name as `"guardrails_ai"` in `GuardResult.GuardName`.

## Running Guardrails AI Server

Guardrails AI can be run as a local server:

```bash
pip install guardrails-ai
guardrails start --config my-guard.rail
```

Or using Docker:

```bash
docker run -p 8000:8000 guardrails/api:latest
```

For the hosted Guardrails Hub, use the cloud endpoint with an API key:

```go
g, err := guardrailsai.New(
    guardrailsai.WithBaseURL("https://api.guardrailsai.com"),
    guardrailsai.WithAPIKey(os.Getenv("GUARDRAILS_API_KEY")),
    guardrailsai.WithGuardName("my-guard"),
)
```

## Error Handling

```go
result, err := g.Validate(ctx, input)
if err != nil {
    // Possible errors:
    // - "guardrailsai: guard name is required" (empty guard name)
    // - "guardrailsai: validate: ..." (API request failure)
    log.Fatal(err)
}

if !result.Allowed {
    // result.Reason contains the first failing validator message,
    // e.g., "Toxicity detected" or "failed validator: PIIValidator"
    fmt.Println(result.Reason)
}
```
