---
title: "Bifrost"
description: "Integration guide for Bifrost LLM gateway with Beluga AI."
---

The Bifrost provider connects Beluga AI to a [Bifrost](https://github.com/maxthomas/bifrost) gateway. Bifrost is an OpenAI-compatible proxy that routes requests to multiple LLM providers with load balancing and failover. This provider is a thin wrapper that points to your Bifrost deployment endpoint.

## Installation

```bash
go get github.com/lookatitude/beluga-ai/llm/providers/bifrost
```

**Prerequisites:** A running Bifrost gateway instance.

## Configuration

| Field    | Required | Default | Description                              |
|----------|----------|---------|------------------------------------------|
| `Model`  | Yes      | —       | Model ID to route through Bifrost        |
| `APIKey` | No       | —       | API key (if Bifrost requires authentication) |
| `BaseURL`| Yes      | —       | Bifrost gateway endpoint (e.g. `http://localhost:8080/v1`) |
| `Timeout`| No       | `30s`   | Request timeout                          |

Both `Model` and `BaseURL` are required. The provider will return an error if either is missing.

## Basic Usage

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/lookatitude/beluga-ai/config"
    "github.com/lookatitude/beluga-ai/llm"
    "github.com/lookatitude/beluga-ai/schema"
    _ "github.com/lookatitude/beluga-ai/llm/providers/bifrost"
)

func main() {
    model, err := llm.New("bifrost", config.ProviderConfig{
        Model:   "gpt-4o",
        APIKey:  "sk-...",
        BaseURL: "http://localhost:8080/v1",
    })
    if err != nil {
        log.Fatal(err)
    }

    msgs := []schema.Message{
        schema.NewSystemMessage("You are a helpful assistant."),
        schema.NewHumanMessage("What is the capital of France?"),
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

Since Bifrost is an OpenAI-compatible proxy, all standard features are supported:

### Tool Calling

```go
modelWithTools := model.BindTools(tools)
resp, err := modelWithTools.Generate(ctx, msgs, llm.WithToolChoice(llm.ToolChoiceAuto))
```

### Structured Output

```go
resp, err := model.Generate(ctx, msgs,
    llm.WithResponseFormat(llm.ResponseFormat{Type: "json_object"}),
)
```

### Generation Options

```go
resp, err := model.Generate(ctx, msgs,
    llm.WithTemperature(0.7),
    llm.WithMaxTokens(2048),
    llm.WithTopP(0.9),
)
```

## Error Handling

```go
resp, err := model.Generate(ctx, msgs)
if err != nil {
    log.Fatal(err)
}
```

## Direct Construction

```go
import "github.com/lookatitude/beluga-ai/llm/providers/bifrost"

model, err := bifrost.New(config.ProviderConfig{
    Model:   "gpt-4o",
    APIKey:  "sk-...",
    BaseURL: "http://localhost:8080/v1",
})
```
