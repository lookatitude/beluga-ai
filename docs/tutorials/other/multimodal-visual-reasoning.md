# Visual Reasoning with Pixtral

<!--
Persona: Pair Programmer Colleague
- Treat the reader as a competent peer
- Be helpful, direct, and slightly informal but professional
- Focus on getting results quickly
- Keep it concise and functional
- Provide runnable code examples
-->

## What you will build
In this tutorial, we'll use Pixtral (Mistral's multimodal model) to analyze images and extract information. You'll build a "Receipt Scanner" agent capable of reading visual content and reasoning about what it sees.

## Learning Objectives
- ✅ Configure Pixtral multimodal provider
- ✅ Send images to the model (URL and Base64)
- ✅ Perform visual Q&A
- ✅ Implement a "Receipt Scanner" agent

## Introduction
Welcome, colleague! Modern AI isn't just about text anymore. Multimodal models like Pixtral can "see" and reason about images. Let's look at how to integrate visual reasoning into our agents to automate tasks like scanning documents or analyzing photos.

## Prerequisites

- Mistral API Key
- Go 1.24+
- `pkg/multimodal` package

## Step 1: Initialize Pixtral Provider
```go
package main

import (
    "context"
    "github.com/lookatitude/beluga-ai/pkg/multimodal"
    "github.com/lookatitude/beluga-ai/pkg/multimodal/providers/pixtral"
)

func main() {
    config := &pixtral.Config{
        APIKey: os.Getenv("MISTRAL_API_KEY"),
        Model:  "pixtral-12b",
    }
    
    provider, _ := pixtral.NewProvider(config)
}
```

## Step 2: Preparing Multimodal Input

Beluga AI uses a structured `Content` slice for multimodal requests.
```text
go
go
    // Mix text and image
    content := []multimodal.Content{
        multimodal.NewTextContent("What is written on this receipt?"),
        multimodal.NewImageURLContent("https://example.com/receipt.jpg"),
    }
```

## Step 3: Generating Response
```go
    res, err := provider.Generate(context.Background(), content)
    if err != nil {
        log.Fatal(err)
    }

    
    fmt.Println("Analysis:", res.Text)
```

## Step 4: Local Image Handling (Base64)
```text
go
go
    imageData, _ := os.ReadFile("local_image.png")
    content := []multimodal.Content{
        multimodal.NewTextContent("Describe this image."),
        multimodal.NewImageBase64Content(imageData, "image/png"),
    }
```

## Step 5: Advanced Visual Reasoning

Ask the model to perform complex tasks, like "Compare these two floor plans and list differences".
```text
go
go
    content := []multimodal.Content{
        multimodal.NewTextContent("Compare image A and B:"),
        multimodal.NewImageURLContent(urlA),
        multimodal.NewImageURLContent(urlB),
    }
```

## Verification

1. Run the script with an image of a simple object (e.g., a coffee mug).
2. Ask "What color is the object?".
3. Verify the model answers correctly.

## Next Steps

- **[Audio Analysis with Gemini](./multimodal-audio-analysis.md)** - Process sound and video.
- **[RAG Multimodal](../../guides/rag-multimodal.md)** - Index images for retrieval.
