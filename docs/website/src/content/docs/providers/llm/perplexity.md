---
title: "Perplexity"
description: "Integration guide for Perplexity's search-augmented models with Beluga AI."
---

The Perplexity provider connects Beluga AI to Perplexity's search-augmented language models. Perplexity models combine LLM reasoning with real-time web search, making them well-suited for tasks requiring up-to-date information. Perplexity exposes an OpenAI-compatible API, so this provider supports all standard features including streaming.

## Installation

```bash
go get github.com/lookatitude/beluga-ai/llm/providers/perplexity
```

## Configuration

| Field    | Required | Default                          | Description                        |
|----------|----------|----------------------------------|------------------------------------|
| `Model`  | Yes      | —                                | Model ID (e.g. `"sonar-pro"`)     |
| `APIKey` | Yes      | —                                | Perplexity API key (`pplx-...`)   |
| `BaseURL`| No       | `https://api.perplexity.ai`     | Override API endpoint              |
| `Timeout`| No       | `30s`                            | Request timeout                    |

**Environment variables:**

| Variable              | Maps to  |
|-----------------------|----------|
| `PERPLEXITY_API_KEY`  | `APIKey` |

## Basic Usage

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/lookatitude/beluga-ai/config"
    "github.com/lookatitude/beluga-ai/llm"
    "github.com/lookatitude/beluga-ai/schema"
    _ "github.com/lookatitude/beluga-ai/llm/providers/perplexity"
)

func main() {
    model, err := llm.New("perplexity", config.ProviderConfig{
        Model:  "sonar-pro",
        APIKey: os.Getenv("PERPLEXITY_API_KEY"),
    })
    if err != nil {
        log.Fatal(err)
    }

    msgs := []schema.Message{
        schema.NewSystemMessage("You are a helpful research assistant."),
        schema.NewHumanMessage("What are the latest developments in Go 1.24?"),
    }

    resp, err := model.Generate(context.Background(), msgs)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(resp.Text())
}
```

## Streaming

```go
for chunk, err := range model.Stream(context.Background(), msgs) {
    if err != nil {
        log.Fatal(err)
    }
    fmt.Print(chunk.Delta)
}
fmt.Println()
```

## Advanced Features

### Generation Options

```go
resp, err := model.Generate(ctx, msgs,
    llm.WithTemperature(0.2),
    llm.WithMaxTokens(2048),
    llm.WithTopP(0.9),
)
```

Lower temperatures are often preferred for factual search queries.

## Error Handling

```go
resp, err := model.Generate(ctx, msgs)
if err != nil {
    log.Fatal(err)
}
```

## Direct Construction

```go
import "github.com/lookatitude/beluga-ai/llm/providers/perplexity"

model, err := perplexity.New(config.ProviderConfig{
    Model:  "sonar-pro",
    APIKey: os.Getenv("PERPLEXITY_API_KEY"),
})
```

## Available Models

| Model ID       | Description                              |
|----------------|------------------------------------------|
| `sonar-pro`    | Advanced search-augmented model          |
| `sonar`        | Standard search-augmented model          |

Refer to [Perplexity's documentation](https://docs.perplexity.ai/docs/model-cards) for the latest model list.
