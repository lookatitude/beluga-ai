---
title: Azure Content Safety
description: Content moderation using Microsoft Azure AI Content Safety.
---

The Azure Content Safety provider implements the `guard.Guard` interface using the [Azure AI Content Safety](https://azure.microsoft.com/en-us/products/ai-services/ai-content-safety) API. It evaluates text across four harm categories (Hate, SelfHarm, Sexual, Violence) with configurable severity thresholds.

## Installation

```bash
go get github.com/lookatitude/beluga-ai/guard/providers/azuresafety
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
    "github.com/lookatitude/beluga-ai/guard/providers/azuresafety"
)

func main() {
    g, err := azuresafety.New(
        azuresafety.WithEndpoint(os.Getenv("AZURE_SAFETY_ENDPOINT")),
        azuresafety.WithAPIKey(os.Getenv("AZURE_SAFETY_KEY")),
    )
    if err != nil {
        log.Fatal(err)
    }

    result, err := g.Validate(context.Background(), guard.GuardInput{
        Content: "Hello, how are you?",
        Role:    "input",
    })
    if err != nil {
        log.Fatal(err)
    }

    if result.Allowed {
        fmt.Println("Content is safe")
    } else {
        fmt.Printf("Blocked: %s\n", result.Reason)
    }
}
```

## Configuration

| Option | Type | Default | Description |
|---|---|---|---|
| `WithEndpoint(url)` | `string` | (required) | Azure Content Safety endpoint URL |
| `WithAPIKey(key)` | `string` | (required) | Azure Content Safety subscription key |
| `WithThreshold(n)` | `int` | `2` | Severity threshold (0-6); content at or above is blocked |
| `WithTimeout(d)` | `time.Duration` | `15s` | HTTP request timeout |

## Harm Categories

Azure Content Safety evaluates text across four categories, each scored from 0 (safe) to 6 (severe):

| Category | Description |
|---|---|
| Hate | Hate speech and discriminatory content |
| SelfHarm | Content related to self-harm or suicide |
| Sexual | Sexually explicit content |
| Violence | Violent content or threats |

Content is blocked when **any** category severity meets or exceeds the configured threshold.

## Threshold Configuration

The threshold controls sensitivity. Lower values are stricter:

```go
// Strict: block anything above safe (severity >= 1)
g, err := azuresafety.New(
    azuresafety.WithEndpoint(endpoint),
    azuresafety.WithAPIKey(apiKey),
    azuresafety.WithThreshold(1),
)

// Lenient: only block severe content (severity >= 4)
g, err := azuresafety.New(
    azuresafety.WithEndpoint(endpoint),
    azuresafety.WithAPIKey(apiKey),
    azuresafety.WithThreshold(4),
)
```

## Pipeline Integration

Use the guard in a safety pipeline:

```go
pipeline := guard.NewPipeline(
    guard.Input(azureGuard),
    guard.Output(azureGuard),
)

result, err := pipeline.ValidateInput(ctx, userMessage)
if !result.Allowed {
    fmt.Printf("Blocked by %s: %s\n", result.GuardName, result.Reason)
    // result.Reason: "flagged: Hate(severity=4), Violence(severity=2)"
}
```

## Guard Name

The guard reports its name as `"azure_content_safety"` in `GuardResult.GuardName`.

## Azure Setup

1. Create an Azure Content Safety resource in the [Azure Portal](https://portal.azure.com)
2. Copy the endpoint URL and subscription key from the resource's "Keys and Endpoint" page
3. The endpoint format is: `https://<resource-name>.cognitiveservices.azure.com`

## Error Handling

```go
result, err := g.Validate(ctx, input)
if err != nil {
    // Possible errors:
    // - "azuresafety: endpoint is required" (missing endpoint)
    // - "azuresafety: API key is required" (missing key)
    // - "azuresafety: validate: ..." (API request failure)
    log.Fatal(err)
}
```
