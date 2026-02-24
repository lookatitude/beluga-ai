---
title: "AWS Bedrock LLM Provider"
description: "Integrate AWS Bedrock models with Beluga AI. Access Claude, Llama, and Mistral via AWS with IAM auth, VPC endpoints, and streaming support in Go."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "AWS Bedrock, LLM provider, Go AWS SDK, IAM auth, Claude on AWS, streaming, Beluga AI"
---

The AWS Bedrock provider connects Beluga AI to Amazon Bedrock's multi-provider model catalog using the AWS SDK v2 Converse API. It supports models from Anthropic, Meta, Mistral, Cohere, Amazon, and others through a unified interface with native AWS authentication.

Choose Bedrock when you need AWS-native integration with IAM roles, VPC endpoints, and CloudTrail auditing. Bedrock consolidates access to models from multiple providers under a single billing and governance layer, which simplifies procurement and compliance in AWS-centric environments.

## Installation

```bash
go get github.com/lookatitude/beluga-ai/llm/providers/bedrock
```

## Configuration

| Field     | Required | Default       | Description                                       |
|-----------|----------|---------------|---------------------------------------------------|
| `Model`   | Yes      | —             | Bedrock model ID (e.g. `"us.anthropic.claude-sonnet-4-5-20250929-v1:0"`) |
| `APIKey`  | No       | AWS default   | AWS Access Key ID (optional, uses default credentials if unset) |
| `BaseURL` | No       | AWS default   | Override Bedrock endpoint                         |
| `Timeout` | No       | `30s`         | Request timeout                                   |

**Provider-specific options (via `Options` map):**

| Key          | Default       | Description                       |
|--------------|---------------|-----------------------------------|
| `region`     | `"us-east-1"` | AWS region                        |
| `secret_key` | —             | AWS Secret Access Key (if using static credentials) |

**Environment variables (standard AWS SDK):**

| Variable                | Description             |
|-------------------------|-------------------------|
| `AWS_ACCESS_KEY_ID`     | AWS access key          |
| `AWS_SECRET_ACCESS_KEY` | AWS secret key          |
| `AWS_REGION`            | AWS region              |
| `AWS_PROFILE`           | Named profile           |

The provider uses the standard AWS SDK credential chain. If `APIKey` is set in the config, it creates static credentials using `APIKey` + `secret_key`. Otherwise, it falls back to the default credential chain (environment, shared config, IAM role, etc.).

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
    _ "github.com/lookatitude/beluga-ai/llm/providers/bedrock"
)

func main() {
    model, err := llm.New("bedrock", config.ProviderConfig{
        Model: "us.anthropic.claude-sonnet-4-5-20250929-v1:0",
        Options: map[string]any{
            "region": "us-east-1",
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

Bedrock streaming uses the Converse Stream API, which provides content block start/delta/stop events, message stop events with finish reason, and usage metadata.

## Advanced Features

### Tool Calling

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
if err != nil {
    log.Fatal(err)
}

for _, tc := range resp.ToolCalls {
    fmt.Printf("Tool: %s, Args: %s\n", tc.Name, tc.Arguments)
}
```

Bedrock tool choice mapping:

| Beluga ToolChoice         | Bedrock Equivalent     |
|---------------------------|------------------------|
| `llm.ToolChoiceAuto`     | `AutoToolChoice`       |
| `llm.ToolChoiceNone`     | Omit tool config       |
| `llm.ToolChoiceRequired` | `AnyToolChoice`        |
| `llm.WithSpecificTool()` | `SpecificToolChoice`   |

### Vision (Multimodal)

For models that support vision (e.g. Claude on Bedrock):

```go
msgs := []schema.Message{
    schema.NewHumanMessageWithParts(
        schema.TextPart{Text: "Describe this image."},
        schema.ImagePart{
            Data:     imageBytes,
            MimeType: "image/png",
        },
    ),
}

resp, err := model.Generate(ctx, msgs)
```

Supported image formats: PNG, JPEG, GIF, WebP.

### Generation Options

```go
resp, err := model.Generate(ctx, msgs,
    llm.WithTemperature(0.7),
    llm.WithMaxTokens(4096),
    llm.WithTopP(0.9),
    llm.WithStopSequences("END"),
)
```

## Error Handling

```go
resp, err := model.Generate(ctx, msgs)
if err != nil {
    // Errors are wrapped with the "bedrock:" prefix
    log.Fatal(err)
}
```

The response includes Bedrock-specific metadata:

```go
fmt.Printf("Input: %d, Output: %d, Total: %d\n",
    resp.Usage.InputTokens,
    resp.Usage.OutputTokens,
    resp.Usage.TotalTokens,
)

// Stop reason is available in metadata
fmt.Println("Stop reason:", resp.Metadata["stop_reason"])
```

## Direct Construction

```go
import "github.com/lookatitude/beluga-ai/llm/providers/bedrock"

model, err := bedrock.New(config.ProviderConfig{
    Model: "us.anthropic.claude-sonnet-4-5-20250929-v1:0",
    Options: map[string]any{"region": "us-west-2"},
})
```

For testing with a mock client:

```go
model := bedrock.NewWithClient(mockClient, "test-model")
```

## Available Models

| Model ID                                    | Provider  | Description              |
|---------------------------------------------|-----------|--------------------------|
| `us.anthropic.claude-sonnet-4-5-20250929-v1:0` | Anthropic | Claude Sonnet 4.5        |
| `us.anthropic.claude-haiku-3-5-20241022-v1:0`  | Anthropic | Claude Haiku 3.5         |
| `us.meta.llama3-3-70b-instruct-v1:0`       | Meta      | Llama 3.3 70B            |
| `mistral.mistral-large-2407-v1:0`           | Mistral   | Mistral Large            |
| `amazon.nova-pro-v1:0`                      | Amazon    | Amazon Nova Pro          |

Refer to the [AWS Bedrock documentation](https://docs.aws.amazon.com/bedrock/latest/userguide/models-supported.html) for the full model catalog.
