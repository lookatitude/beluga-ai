---
title: Pixtral (Mistral AI Vision)
description: Integrate Pixtral, Mistral AI's vision-language model, with Beluga AI for multimodal image understanding, visual question answering, and image analysis.
---

Pixtral is Mistral AI's vision-language model that enables image understanding and multimodal reasoning. This guide covers integrating Pixtral with Beluga AI for tasks such as visual question answering, image captioning, and scene analysis.

## Overview

Pixtral provides vision-language capabilities through the Mistral AI API. When integrated with Beluga AI, it enables:

- **Visual question answering** -- ask questions about image content
- **Image captioning** -- generate descriptions from images
- **Scene analysis** -- detailed analysis of complex visual scenes
- **Document understanding** -- extract information from documents and screenshots

Pixtral uses the same Mistral API as other Mistral models, with multimodal input support through content parts.

## Prerequisites

- Go 1.23 or later
- A Mistral AI API key (obtain from [mistral.ai](https://mistral.ai))
- Beluga AI framework installed

## Installation

The Mistral provider is included in Beluga AI. No additional dependencies are required beyond the framework itself.

Set your API key:

```bash
export MISTRAL_API_KEY="your-api-key"
```

## Configuration

### Basic Setup

Create a Mistral provider configured for the Pixtral model:

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/lookatitude/beluga-ai/llm/providers/mistral"
    "github.com/lookatitude/beluga-ai/schema"
)

func main() {
    ctx := context.Background()

    config := &mistral.Config{
        APIKey: os.Getenv("MISTRAL_API_KEY"),
        Model:  "pixtral-12b",
    }

    provider, err := mistral.New(config)
    if err != nil {
        log.Fatalf("failed to create Pixtral provider: %v", err)
    }

    messages := []schema.Message{
        &schema.UserMessage{
            Content: []schema.ContentPart{
                schema.TextPart{Text: "What is in this image?"},
                schema.ImagePart{Data: []byte{/* image data */}, MIMEType: "image/png"},
            },
        },
    }

    response, err := provider.Generate(ctx, messages)
    if err != nil {
        log.Fatalf("generation failed: %v", err)
    }

    fmt.Printf("Response: %s\n", response.Content)
}
```

Verify the setup:

```bash
export MISTRAL_API_KEY="your-api-key"
go run main.go
```

### Configuration Reference

| Option | Description | Default | Required |
|--------|-------------|---------|----------|
| `APIKey` | Mistral AI API key | - | Yes |
| `Model` | Pixtral model name | `pixtral-12b` | No |
| `Timeout` | Request timeout | `30s` | No |
| `MaxImageSize` | Maximum image size in bytes | `20MB` | No |

## Usage

### Vision-Language Tasks

Pixtral supports several vision-language task patterns:

```go
import "github.com/lookatitude/beluga-ai/schema"

// Image captioning (image only, no text prompt)
messages := []schema.Message{
    &schema.UserMessage{
        Content: []schema.ContentPart{
            schema.ImagePart{Data: imageData, MIMEType: "image/png"},
        },
    },
}

// Visual question answering
messages = []schema.Message{
    &schema.UserMessage{
        Content: []schema.ContentPart{
            schema.TextPart{Text: "What color is the car in this image?"},
            schema.ImagePart{Data: imageData, MIMEType: "image/jpeg"},
        },
    },
}

// Detailed scene analysis
messages = []schema.Message{
    &schema.UserMessage{
        Content: []schema.ContentPart{
            schema.TextPart{Text: "Analyze this image and describe the scene in detail."},
            schema.ImagePart{Data: imageData, MIMEType: "image/png"},
        },
    },
}
```

### Using Pixtral with Agents

Integrate Pixtral as the backing model for a Beluga AI agent that can process images:

```go
package main

import (
    "context"
    "log"
    "os"

    "github.com/lookatitude/beluga-ai/agent"
    "github.com/lookatitude/beluga-ai/llm/providers/mistral"
)

func main() {
    ctx := context.Background()

    config := &mistral.Config{
        APIKey: os.Getenv("MISTRAL_API_KEY"),
        Model:  "pixtral-12b",
    }

    model, err := mistral.New(config)
    if err != nil {
        log.Fatalf("failed to create model: %v", err)
    }

    a, err := agent.New(
        agent.WithName("vision-agent"),
        agent.WithModel(model),
        agent.WithInstructions("You are an image analysis assistant."),
    )
    if err != nil {
        log.Fatalf("failed to create agent: %v", err)
    }

    // Agent can now process multimodal inputs
    _ = a
    _ = ctx
}
```

## Advanced Topics

### OpenTelemetry Instrumentation

Add tracing to Pixtral calls:

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"
    "time"

    "github.com/lookatitude/beluga-ai/llm/providers/mistral"
    "github.com/lookatitude/beluga-ai/schema"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"
)

func main() {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    config := &mistral.Config{
        APIKey: os.Getenv("MISTRAL_API_KEY"),
        Model:  "pixtral-12b",
    }

    tracer := otel.Tracer("beluga.llm.pixtral")
    ctx, span := tracer.Start(ctx, "pixtral.generate",
        trace.WithAttributes(
            attribute.String("gen_ai.system", "mistral"),
            attribute.String("gen_ai.request.model", config.Model),
        ),
    )
    defer span.End()

    provider, err := mistral.New(config)
    if err != nil {
        span.RecordError(err)
        log.Fatalf("failed to create provider: %v", err)
    }

    messages := []schema.Message{
        &schema.UserMessage{
            Content: []schema.ContentPart{
                schema.TextPart{Text: "Describe this image in detail."},
                schema.ImagePart{Data: loadImage("image.png"), MIMEType: "image/png"},
            },
        },
    }

    response, err := provider.Generate(ctx, messages)
    if err != nil {
        span.RecordError(err)
        log.Fatalf("generation failed: %v", err)
    }

    span.SetAttributes(
        attribute.Int("gen_ai.response.input_tokens", response.Usage.InputTokens),
        attribute.Int("gen_ai.response.output_tokens", response.Usage.OutputTokens),
    )

    fmt.Printf("Response: %s\n", response.Content)
}

func loadImage(path string) []byte {
    data, err := os.ReadFile(path)
    if err != nil {
        log.Fatalf("failed to load image: %v", err)
    }
    return data
}
```

### Production Considerations

When deploying Pixtral integrations to production:

- **Image optimization**: Resize and compress images before sending to reduce latency and cost. Pixtral supports PNG, JPEG, and WebP formats.
- **Cost management**: Monitor API usage through the Mistral AI dashboard. Image inputs consume more tokens than text.
- **Size limits**: Respect the maximum image size (20MB default). Implement client-side validation before sending requests.
- **Format validation**: Verify image format compatibility before sending. Unsupported formats return errors.
- **Error handling**: Use Beluga AI's `resilience` package for automatic retry on transient API failures.

## Troubleshooting

### "API key invalid"

The API key is incorrect or has been revoked.

```bash
# Verify the key is set
echo $MISTRAL_API_KEY

# Re-set from the Mistral AI dashboard
export MISTRAL_API_KEY="your-new-api-key"
```

### "Image format not supported"

The provided image format is not recognized by Pixtral. Supported formats are PNG, JPEG, and WebP. Convert the image before sending:

```go
// Ensure images are in a supported format before creating the content part
// Supported: image/png, image/jpeg, image/webp
```

## Related Resources

- [Google Vertex AI Vision](/integrations/vertex-ai-vision) -- Google Vertex AI multimodal integration
- [LLM Providers](/integrations/llm-providers) -- All supported LLM providers
- [Monitoring](/integrations/monitoring) -- Observability and tracing setup
