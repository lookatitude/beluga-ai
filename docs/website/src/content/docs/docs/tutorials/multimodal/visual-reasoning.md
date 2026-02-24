---
title: Visual Reasoning with Multimodal Models
description: "Analyze images and extract structured data using multimodal LLMs in Go — URL-based and base64 image input with Beluga AI's ContentPart system."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Beluga AI, Go, tutorial, visual reasoning, multimodal, image analysis, ContentPart, receipt scanner"
---

Modern AI models can "see" and reason about images alongside text. Beluga AI's `schema.ContentPart` system provides a unified interface for sending multimodal content to any provider that supports vision capabilities. The `ContentPart` interface is the foundation of Beluga AI's multimodal design -- `TextPart`, `ImagePart`, and `AudioPart` all implement it, which means a single `HumanMessage` can carry any combination of media types. Provider implementations handle the encoding details (base64 for raw bytes, URL references for remote images) transparently, so your application code stays provider-agnostic.

## What You Will Build

A visual analysis pipeline that sends images (both URL-based and base64-encoded) to a multimodal LLM, performs visual question answering, and compares multiple images. You will build a receipt scanner that extracts structured data from photographs.

## Prerequisites

- An LLM provider API key for a model with vision support (OpenAI GPT-4o, Google Gemini, Anthropic Claude)
- Basic familiarity with the `llm` and `schema` packages

## Core Concepts

### Multimodal Content Parts

Every message in Beluga AI contains a slice of `ContentPart` values. The `schema` package provides typed parts for different media. `ImagePart` supports two modes: URL-based (the provider fetches the image from the URL) and data-based (raw bytes are base64-encoded and sent inline). URL-based is more efficient for large images because the provider downloads them directly, while data-based is necessary for local files or private images that the provider cannot reach.

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

A `HumanMessage` can contain both text and image parts. The text part provides the instruction and the image part provides the visual data. This separation allows the same image to be analyzed with different questions without re-uploading -- just change the text part and reuse the image part.

```go
msg := &schema.HumanMessage{
    Parts: []schema.ContentPart{
        schema.TextPart{Text: "What is shown in this image?"},
        schema.ImagePart{URL: "https://example.com/photo.jpg", MimeType: "image/jpeg"},
    },
}
```

## Step 1: Initialize a Vision-Capable Model

Create a vision-capable model via the registry pattern. GPT-4o is used here because it supports both URL and base64 image input with strong visual reasoning capabilities. The same `llm.ChatModel` interface applies -- you can switch to Gemini or Claude without changing the visual analysis code.

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

Send an image URL alongside a text question. URL-based image input is the preferred approach when the image is publicly accessible because the provider fetches it directly, avoiding the overhead of base64 encoding (which increases payload size by approximately 33%). The `MimeType` field helps the provider handle the image correctly, though most providers can auto-detect the format.

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

Load a local image file and send it as raw bytes. The provider implementation handles base64 encoding transparently -- your code works with `[]byte` from `os.ReadFile` directly. This approach is necessary for local files, screenshots, or images from private sources that the provider API cannot fetch via URL.

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

Send multiple images in a single message for comparison. The `Parts` slice can contain any number of `ImagePart` values, and the model will reason about all of them in relation to the text prompt. This is useful for before/after comparisons, version comparisons, or any task where the model needs to see multiple images simultaneously to produce a meaningful analysis.

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

Combine visual reasoning with structured output extraction. The system message instructs the model to return JSON with specific fields, creating a structured extraction pipeline from unstructured image data. For production use, consider combining this with the `llm.StructuredOutput` facility to validate the JSON against a Go struct schema, ensuring the model's output conforms to the expected format.

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

Stream the analysis for large or complex images. Streaming uses `iter.Seq2[StreamChunk, error]`, Beluga AI's standard streaming pattern, providing real-time output as the model processes the visual content. This is particularly useful for detailed image descriptions where the output is long and the user benefits from seeing partial results immediately.

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

- [Audio Analysis](/docs/tutorials/multimodal/audio-analysis) -- Process audio files with multimodal models
- [Content Moderation](/docs/tutorials/safety/content-moderation) -- Validate image descriptions before returning to users
