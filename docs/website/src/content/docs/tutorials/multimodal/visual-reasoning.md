---
title: Visual Reasoning with Multimodal Models
description: Analyze images and extract information using multimodal LLMs with Beluga AI's ContentPart system for URL-based and base64-encoded image input.
---

Modern AI models can "see" and reason about images alongside text. Beluga AI's `schema.ContentPart` system provides a unified interface for sending multimodal content to any provider that supports vision capabilities. This tutorial demonstrates how to build a visual reasoning pipeline using image parts.

## What You Will Build

A visual analysis pipeline that sends images (both URL-based and base64-encoded) to a multimodal LLM, performs visual question answering, and compares multiple images. You will build a receipt scanner that extracts structured data from photographs.

## Prerequisites

- An LLM provider API key for a model with vision support (OpenAI GPT-4o, Google Gemini, Anthropic Claude)
- Basic familiarity with the `llm` and `schema` packages

## Core Concepts

### Multimodal Content Parts

Every message in Beluga AI contains a slice of `ContentPart` values. The `schema` package provides typed parts for different media:

```go
import "github.com/lookatitude/beluga-ai/schema"

// Text content.
textPart := schema.TextPart{Text: "Describe this image."}

// Image from URL.
imagePart := schema.ImagePart{
    URL:      "https://example.com/photo.jpg",
    MimeType: "image/jpeg",
}

// Image from raw bytes (base64 encoded internally by providers).
imageBytes := schema.ImagePart{
    Data:     rawBytes,
    MimeType: "image/png",
}
```

### Messages with Mixed Content

A `HumanMessage` can contain both text and image parts:

```go
msg := &schema.HumanMessage{
    Parts: []schema.ContentPart{
        schema.TextPart{Text: "What is shown in this image?"},
        schema.ImagePart{URL: "https://example.com/photo.jpg", MimeType: "image/jpeg"},
    },
}
```

## Step 1: Initialize a Vision-Capable Model

```go
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/lookatitude/beluga-ai/llm"
    "github.com/lookatitude/beluga-ai/schema"

    _ "github.com/lookatitude/beluga-ai/llm/providers/openai"
)

func main() {
    model, err := llm.New("openai", llm.ProviderConfig{
        Options: map[string]any{
            "api_key": os.Getenv("OPENAI_API_KEY"),
            "model":   "gpt-4o",
        },
    })
    if err != nil {
        fmt.Printf("model creation failed: %v\n", err)
        return
    }

    ctx := context.Background()

    // Continue with visual reasoning examples...
    _ = model
    _ = ctx
}
```

## Step 2: Analyze an Image from URL

Send an image URL alongside a text question:

```go
func analyzeImageURL(ctx context.Context, model llm.ChatModel) {
    msgs := []schema.Message{
        &schema.HumanMessage{
            Parts: []schema.ContentPart{
                schema.TextPart{Text: "What items are visible in this receipt? List them with prices."},
                schema.ImagePart{
                    URL:      "https://example.com/receipt.jpg",
                    MimeType: "image/jpeg",
                },
            },
        },
    }

    aiMsg, err := model.Generate(ctx, msgs)
    if err != nil {
        fmt.Printf("error: %v\n", err)
        return
    }

    fmt.Println("Analysis:", aiMsg.Text())
}
```

## Step 3: Analyze a Local Image (Base64)

Load a local image file and send it as raw bytes:

```go
func analyzeLocalImage(ctx context.Context, model llm.ChatModel) {
    imageData, err := os.ReadFile("receipt.png")
    if err != nil {
        fmt.Printf("read error: %v\n", err)
        return
    }

    msgs := []schema.Message{
        &schema.HumanMessage{
            Parts: []schema.ContentPart{
                schema.TextPart{Text: "Extract the total amount and date from this receipt."},
                schema.ImagePart{
                    Data:     imageData,
                    MimeType: "image/png",
                },
            },
        },
    }

    aiMsg, err := model.Generate(ctx, msgs)
    if err != nil {
        fmt.Printf("error: %v\n", err)
        return
    }

    fmt.Println("Extracted:", aiMsg.Text())
}
```

## Step 4: Compare Multiple Images

Send multiple images in a single message for comparison:

```go
func compareImages(ctx context.Context, model llm.ChatModel) {
    msgs := []schema.Message{
        &schema.HumanMessage{
            Parts: []schema.ContentPart{
                schema.TextPart{Text: "Compare these two floor plans. List the differences."},
                schema.ImagePart{
                    URL:      "https://example.com/floorplan-v1.png",
                    MimeType: "image/png",
                },
                schema.ImagePart{
                    URL:      "https://example.com/floorplan-v2.png",
                    MimeType: "image/png",
                },
            },
        },
    }

    aiMsg, err := model.Generate(ctx, msgs)
    if err != nil {
        fmt.Printf("error: %v\n", err)
        return
    }

    fmt.Println("Differences:", aiMsg.Text())
}
```

## Step 5: Build a Receipt Scanner

Combine visual reasoning with structured output extraction:

```go
func scanReceipt(ctx context.Context, model llm.ChatModel, imagePath string) {
    imageData, err := os.ReadFile(imagePath)
    if err != nil {
        fmt.Printf("read error: %v\n", err)
        return
    }

    msgs := []schema.Message{
        schema.NewSystemMessage(
            "You are a receipt scanner. Extract structured data from receipt images. " +
                "Return JSON with fields: store_name, date, items (array of {name, price}), " +
                "subtotal, tax, and total.",
        ),
        &schema.HumanMessage{
            Parts: []schema.ContentPart{
                schema.TextPart{Text: "Extract the receipt data as JSON."},
                schema.ImagePart{
                    Data:     imageData,
                    MimeType: "image/jpeg",
                },
            },
        },
    }

    aiMsg, err := model.Generate(ctx, msgs)
    if err != nil {
        fmt.Printf("error: %v\n", err)
        return
    }

    fmt.Println("Receipt JSON:", aiMsg.Text())
}
```

## Step 6: Streaming Visual Analysis

Stream the analysis for large or complex images:

```go
func streamAnalysis(ctx context.Context, model llm.ChatModel) {
    msgs := []schema.Message{
        &schema.HumanMessage{
            Parts: []schema.ContentPart{
                schema.TextPart{Text: "Provide a detailed description of everything in this image."},
                schema.ImagePart{
                    URL:      "https://example.com/complex-scene.jpg",
                    MimeType: "image/jpeg",
                },
            },
        },
    }

    for chunk, err := range model.Stream(ctx, msgs) {
        if err != nil {
            fmt.Printf("\nstream error: %v\n", err)
            return
        }
        fmt.Print(chunk.Delta)
    }
    fmt.Println()
}
```

## Provider Compatibility

Vision capabilities vary by provider:

| Provider | Models | URL Support | Base64 Support |
|----------|--------|-------------|----------------|
| OpenAI | GPT-4o, GPT-4o-mini | Yes | Yes |
| Anthropic | Claude 3.5, Claude 4 | Yes | Yes |
| Google | Gemini 1.5 Pro, Gemini 2 | Yes | Yes |
| Ollama | LLaVA, Llama 3.2 Vision | No | Yes |

## Verification

1. Run the image URL analysis with a public image. Verify the model describes the content.
2. Run the local image analysis with a screenshot. Verify the model reads text from the image.
3. Test the receipt scanner with a photograph of a receipt. Verify the JSON output matches.

## Next Steps

- [Audio Analysis](/tutorials/multimodal/audio-analysis) -- Process audio files with multimodal models
- [Content Moderation](/tutorials/safety/content-moderation) -- Validate image descriptions before returning to users
