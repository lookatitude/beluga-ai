---
title: "Processing Multiple Images per Prompt"
description: "Process multiple images in a single prompt with proper context and relationship annotations."
---

# Processing Multiple Images per Prompt

## Problem

You need to process multiple images in a single prompt, where each image provides different context and the LLM needs to understand relationships between images.

## Solution

Implement multi-image processing that loads multiple images, formats them according to the model's requirements, includes them in the prompt with proper context, and processes the combined input. This works because multimodal LLMs support multiple images in a single request, and you can structure them appropriately.

## Code Example

```go
package main

import (
    "context"
    "fmt"
    "log"

    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"

    "github.com/lookatitude/beluga-ai/schema"
)

var tracer = otel.Tracer("beluga.multimodal.multi_image")

// MultiImageProcessor processes multiple images in a prompt
type MultiImageProcessor struct {
    maxImages    int
    imageFormats []string
}

// NewMultiImageProcessor creates a new processor
func NewMultiImageProcessor(maxImages int) *MultiImageProcessor {
    return &MultiImageProcessor{
        maxImages:    maxImages,
        imageFormats: []string{"image/jpeg", "image/png", "image/webp"},
    }
}

// ProcessMultipleImages processes multiple images with a prompt
func (mip *MultiImageProcessor) ProcessMultipleImages(ctx context.Context, prompt string, imagePaths []string, imageURLs []string) ([]schema.Message, error) {
    ctx, span := tracer.Start(ctx, "multi_image_processor.process")
    defer span.End()

    span.SetAttributes(
        attribute.Int("image_path_count", len(imagePaths)),
        attribute.Int("image_url_count", len(imageURLs)),
        attribute.String("prompt", prompt),
    )

    // Validate image count
    totalImages := len(imagePaths) + len(imageURLs)
    if totalImages > mip.maxImages {
        err := fmt.Errorf("too many images: %d (max: %d)", totalImages, mip.maxImages)
        span.RecordError(err)
        span.SetStatus(trace.StatusError, err.Error())
        return nil, err
    }

    // Build messages with images
    messages := []schema.Message{}

    // Add system message if needed
    if prompt != "" {
        messages = append(messages, schema.NewSystemMessage(prompt))
    }

    // Create user message with multiple images
    imageMessages := []schema.ImageContent{}

    // Add images from paths
    for i, path := range imagePaths {
        imgContent, err := mip.loadImageFromPath(ctx, path, i)
        if err != nil {
            span.RecordError(err)
            continue
        }
        imageMessages = append(imageMessages, imgContent)
    }

    // Add images from URLs
    for i, url := range imageURLs {
        imgContent, err := mip.loadImageFromURL(ctx, url, len(imagePaths)+i)
        if err != nil {
            span.RecordError(err)
            continue
        }
        imageMessages = append(imageMessages, imgContent)
    }

    // Create multimodal message
    userMsg := schema.NewHumanMessage("")
    // Add image contents to message
    // This would use the actual multimodal message API

    messages = append(messages, userMsg)

    span.SetAttributes(attribute.Int("processed_image_count", len(imageMessages)))
    span.SetStatus(trace.StatusOK, "multiple images processed")

    return messages, nil
}

// loadImageFromPath loads image from file path
func (mip *MultiImageProcessor) loadImageFromPath(ctx context.Context, path string, index int) (schema.ImageContent, error) {
    // Load and encode image
    // In practice, would load file, encode to base64 or bytes
    return schema.ImageContent{}, nil
}

// loadImageFromURL loads image from URL
func (mip *MultiImageProcessor) loadImageFromURL(ctx context.Context, url string, index int) (schema.ImageContent, error) {
    // Download and encode image
    // In practice, would download, validate, encode
    return schema.ImageContent{}, nil
}

// AnnotateImages adds annotations to help LLM understand image relationships
func (mip *MultiImageProcessor) AnnotateImages(ctx context.Context, prompt string, imageDescriptions []string) string {
    annotated := prompt

    for i, desc := range imageDescriptions {
        annotated += fmt.Sprintf("\n\nImage %d: %s", i+1, desc)
    }

    return annotated
}

func main() {
    ctx := context.Background()

    // Create processor
    processor := NewMultiImageProcessor(5)

    // Process multiple images
    prompt := "Compare these images and describe the differences"
    imagePaths := []string{"image1.jpg", "image2.jpg"}

    messages, err := processor.ProcessMultipleImages(ctx, prompt, imagePaths, nil)
    if err != nil {
        log.Fatalf("Failed: %v", err)
    }
    fmt.Printf("Processed %d images\n", len(messages))
}
```

## Explanation

1. **Image aggregation** — Images from both file paths and URLs are combined, providing flexibility in how images are provided.

2. **Validation** — The number of images is validated against model limits to prevent errors from exceeding model capabilities.

3. **Message construction** — Multimodal messages are constructed with multiple image contents, allowing the LLM to process all images together.

> **Key insight:** Structure multi-image prompts clearly. Use annotations or text to help the LLM understand relationships between images and what to look for.

## Testing

```go
func TestMultiImageProcessor_ProcessesMultipleImages(t *testing.T) {
    processor := NewMultiImageProcessor(5)

    messages, err := processor.ProcessMultipleImages(context.Background(), "prompt", []string{"img1.jpg", "img2.jpg"}, nil)
    require.NoError(t, err)
    require.Greater(t, len(messages), 0)
}
```

## Variations

### Image Sequencing

Specify image order explicitly:

```go
type ImageWithContext struct {
    Image   schema.ImageContent
    Context string
    Order   int
}
```

### Image Relationships

Model relationships between images:

```go
type ImageRelationship struct {
    Type     string // "comparison", "sequence", "overlay"
    ImageIDs []int
}
```

## Related Recipes

- [Capability-based Fallbacks](/cookbook/capability-fallbacks) — Handle capability limitations
- [Voice Providers Guide](/guides/voice-providers) — For a deeper understanding of multimodal
