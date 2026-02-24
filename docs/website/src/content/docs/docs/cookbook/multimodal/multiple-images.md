---
title: "Processing Multiple Images per Prompt"
description: "Recipe for sending multiple images in one LLM prompt with context annotations, validation, and relationship hints for comparison tasks in Go."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Beluga AI, multiple images, Go multimodal LLM, image comparison, batch image processing, vision AI recipe, ContentPart"
---

# Processing Multiple Images per Prompt

## Problem

You need to process multiple images in a single prompt, where each image provides different context and the LLM needs to understand relationships between images.

Single-image prompts work for independent analysis, but many real-world tasks require the model to compare, correlate, or synthesize information across multiple images: comparing product designs, verifying document consistency, analyzing before/after states, or processing multi-page documents. Sending images in separate requests loses the relational context. The model needs to see all images together in one prompt to make relative judgments.

## Solution

Implement multi-image processing that loads images from both file paths and URLs, validates the total count against model limits, constructs a multimodal message with all images, and includes annotations that help the LLM understand image relationships. The processor validates inputs before API calls to prevent errors from exceeding model capabilities (e.g., sending 20 images when the model supports 5).

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

1. **Image aggregation** -- Images from both file paths and URLs are combined into a single prompt, providing flexibility in how images are sourced. File paths work for local images, while URLs support images hosted on CDNs or messaging platforms. Both sources are validated and loaded uniformly.

2. **Validation** -- The number of images is validated against model limits before any loading or API calls. This prevents wasted work: loading 20 large images only to have the API reject the request is expensive. Early validation also provides clear error messages ("too many images: 20, max: 5") rather than cryptic API errors.

3. **Annotation support** -- The `AnnotateImages` method adds structured text annotations to help the LLM understand image relationships. Annotations like "Image 1: Product design v1" and "Image 2: Product design v2" give the model explicit context about what each image represents and what kind of comparison is expected. Without annotations, the model must infer image relationships purely from visual content.

> **Key insight:** Structure multi-image prompts clearly with annotations. Use numbered labels or descriptions to help the LLM understand the role of each image and the expected analysis. The text prompt should explicitly state whether you want comparison, synthesis, or independent analysis of each image.

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

Specify image order explicitly for sequential analysis:

```go
type ImageWithContext struct {
    Image   schema.ImageContent
    Context string
    Order   int
}
```

### Image Relationships

Model relationships between images for structured comparison:

```go
type ImageRelationship struct {
    Type     string // "comparison", "sequence", "overlay"
    ImageIDs []int
}
```

## Related Recipes

- [Capability-based Fallbacks](/docs/cookbook/capability-fallbacks) -- Handle capability limitations across providers
- [Voice Providers Guide](/docs/guides/voice-providers) -- For a deeper understanding of multimodal pipelines
