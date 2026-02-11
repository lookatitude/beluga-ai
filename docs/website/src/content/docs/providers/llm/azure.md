---
title: "Azure OpenAI"
description: "Integration guide for Azure OpenAI Service with Beluga AI."
---

The Azure OpenAI provider connects Beluga AI to OpenAI models hosted on Microsoft Azure. It uses the same OpenAI-compatible API format with Azure-specific authentication (api-key header) and URL structure (per-deployment endpoints with api-version query parameter).

Choose Azure OpenAI when your organization requires enterprise compliance controls, Azure Active Directory integration, virtual network isolation, or data residency guarantees. Azure provides the same OpenAI models with enterprise-grade SLAs and private networking.

## Installation

```bash
go get github.com/lookatitude/beluga-ai/llm/providers/azure
```

## Configuration

| Field     | Required | Default        | Description                                       |
|-----------|----------|----------------|---------------------------------------------------|
| `Model`   | No       | `"gpt-4o"`    | Model identifier                                  |
| `APIKey`  | Yes      | —              | Azure API key                                     |
| `BaseURL` | Yes      | —              | Azure deployment endpoint (see below)             |
| `Timeout` | No       | `30s`          | Request timeout                                   |

**Provider-specific options (via `Options` map):**

| Key           | Default        | Description                |
|---------------|----------------|----------------------------|
| `api_version` | `"2024-10-21"` | Azure API version string   |

**BaseURL format:**

```
https://{resource-name}.openai.azure.com/openai/deployments/{deployment-name}
```

**Environment variables:**

| Variable               | Maps to   |
|------------------------|-----------|
| `AZURE_OPENAI_API_KEY` | `APIKey`  |

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
    _ "github.com/lookatitude/beluga-ai/llm/providers/azure"
)

func main() {
    model, err := llm.New("azure", config.ProviderConfig{
        Model:   "gpt-4o",
        APIKey:  os.Getenv("AZURE_OPENAI_API_KEY"),
        BaseURL: "https://myresource.openai.azure.com/openai/deployments/my-gpt4o",
        Options: map[string]any{
            "api_version": "2024-10-21",
        },
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

### Tool Calling

Azure OpenAI supports the same tool calling capabilities as OpenAI:

```go
tools := []schema.ToolDefinition{
    {
        Name:        "get_weather",
        Description: "Get current weather for a location",
        InputSchema: map[string]any{
            "type": "object",
            "properties": map[string]any{
                "location": map[string]any{
                    "type":        "string",
                    "description": "City name",
                },
            },
            "required": []any{"location"},
        },
    },
}

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

## How Azure Authentication Works

Azure OpenAI uses a different authentication mechanism than standard OpenAI. The provider automatically handles this by:

1. Setting the `api-key` HTTP header (instead of `Authorization: Bearer`)
2. Appending the `api-version` query parameter to every request
3. Removing the default `Authorization` header set by the OpenAI SDK

This is transparent to the caller. You only need to provide the correct `APIKey` and `BaseURL`.

## Error Handling

```go
resp, err := model.Generate(ctx, msgs)
if err != nil {
    // Errors are wrapped with the "openaicompat:" prefix
    log.Fatal(err)
}
```

## Direct Construction

```go
import "github.com/lookatitude/beluga-ai/llm/providers/azure"

model, err := azure.New(config.ProviderConfig{
    Model:   "gpt-4o",
    APIKey:  os.Getenv("AZURE_OPENAI_API_KEY"),
    BaseURL: "https://myresource.openai.azure.com/openai/deployments/my-gpt4o",
})
```
