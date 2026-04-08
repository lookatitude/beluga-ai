---
title: Multimodal Recipes
description: "Go recipes for multimodal AI: process images, audio, and video with provider fallbacks, batch classification, and format detection in Beluga AI."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Beluga AI, multimodal AI recipes, Go image processing, audio AI, video processing, provider fallbacks, batch classification, multimodal LLM"
sidebar:
  order: 0
---

Multimodal AI extends LLM capabilities beyond text to process images, audio, and video. Working with multimodal inputs introduces challenges that text-only applications do not face: varying file sizes and formats, provider-specific capability differences, cost optimization through preprocessing, and the need for fallback strategies when a model does not support a required modality. These recipes provide production-tested patterns for handling these challenges.

Each recipe is copy-paste ready with complete error handling. The patterns follow Beluga AI's standard conventions: `context.Context` as the first parameter, explicit error returns, and functional options for configuration.

## Processing Multiple Images Per Prompt

Analyze multiple images in a single LLM call for comparison, synthesis, or batch processing. This is useful for product comparison, document verification, or visual quality assessment where the model needs to see all images together to make relative judgments.

```go
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/llm"
	"github.com/lookatitude/beluga-ai/schema"
	_ "github.com/lookatitude/beluga-ai/llm/providers/openai"
)

// CompareMultipleImages sends multiple images in a single prompt.
// Each image is an schema.ImagePart with Data and MimeType set.
func CompareMultipleImages(ctx context.Context, model llm.ChatModel, imagePaths []string, question string) (string, error) {
	parts := []schema.ContentPart{
		schema.TextPart{Text: question},
	}

	for _, path := range imagePaths {
		imageData, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("read image %s: %w", path, err)
		}
		parts = append(parts, schema.ImagePart{
			Data:     imageData,
			MimeType: "image/jpeg",
		})
	}

	// Build a HumanMessage with all content parts.
	msg := &schema.HumanMessage{Parts: parts}
	response, err := model.Generate(ctx, []schema.Message{msg})
	if err != nil {
		return "", err
	}

	return response.Text(), nil
}

func main() {
	ctx := context.Background()

	model, err := llm.New("openai", config.ProviderConfig{
		APIKey: os.Getenv("OPENAI_API_KEY"),
		Model:  "gpt-4o",
	})
	if err != nil {
		slog.Error("model creation failed", "error", err)
		return
	}

	images := []string{
		"product_v1.jpg",
		"product_v2.jpg",
		"product_v3.jpg",
	}

	result, err := CompareMultipleImages(
		ctx,
		model,
		images,
		"Which of these product designs is most appealing? Explain your reasoning.",
	)
	if err != nil {
		slog.Error("comparison failed", "error", err)
		return
	}

	fmt.Println(result)
}
```

## Capability-Based Provider Fallbacks

Not all LLM providers support the same modalities. When building multimodal applications that need to work across providers, you need a routing layer that matches requests to capable models and falls back gracefully when the primary model cannot handle a given input type.

```go
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/llm"
	"github.com/lookatitude/beluga-ai/schema"
	_ "github.com/lookatitude/beluga-ai/llm/providers/openai"
	_ "github.com/lookatitude/beluga-ai/llm/providers/anthropic"
)

// ProviderCapabilities describes what a given model supports.
type ProviderCapabilities struct {
	SupportsImages   bool
	SupportsAudio    bool
	MaxImageSize     int // bytes
	MaxAudioDuration time.Duration
}

var providerCapabilities = map[string]ProviderCapabilities{
	"gpt-4o": {
		SupportsImages: true,
		SupportsAudio:  false,
		MaxImageSize:   20 * 1024 * 1024,
	},
	"claude-3-5-sonnet-20241022": {
		SupportsImages: true,
		SupportsAudio:  false,
		MaxImageSize:   10 * 1024 * 1024,
	},
}

// MultimodalRequest describes a request that may contain multiple modalities.
type MultimodalRequest struct {
	Text   string
	Images [][]byte
	Audio  []byte
}

// ProcessWithFallback tries providers in priority order until one succeeds.
func ProcessWithFallback(ctx context.Context, req MultimodalRequest) (string, error) {
	needsImages := len(req.Images) > 0
	needsAudio := len(req.Audio) > 0

	type candidate struct {
		providerName string
		model        string
	}
	candidates := []candidate{
		{"openai", "gpt-4o"},
		{"anthropic", "claude-3-5-sonnet-20241022"},
	}

	var lastErr error
	for _, c := range candidates {
		caps := providerCapabilities[c.model]

		if needsImages && !caps.SupportsImages {
			continue
		}
		if needsAudio && !caps.SupportsAudio {
			continue
		}
		for _, img := range req.Images {
			if len(img) > caps.MaxImageSize {
				goto next
			}
		}

		if result, err := tryProvider(ctx, c.providerName, c.model, req); err == nil {
			return result, nil
		} else {
			lastErr = err
			slog.Warn("provider failed, trying next", "provider", c.providerName, "error", err)
		}
	next:
	}

	return "", fmt.Errorf("all providers failed: %w", lastErr)
}

func tryProvider(ctx context.Context, providerName, modelName string, req MultimodalRequest) (string, error) {
	model, err := llm.New(providerName, config.ProviderConfig{
		APIKey: os.Getenv(providerName + "_API_KEY"),
		Model:  modelName,
	})
	if err != nil {
		return "", err
	}

	parts := []schema.ContentPart{schema.TextPart{Text: req.Text}}
	for _, img := range req.Images {
		parts = append(parts, schema.ImagePart{Data: img, MimeType: "image/jpeg"})
	}
	if len(req.Audio) > 0 {
		parts = append(parts, schema.AudioPart{Data: req.Audio, Format: "mp3"})
	}

	msg := &schema.HumanMessage{Parts: parts}
	response, err := model.Generate(ctx, []schema.Message{msg})
	if err != nil {
		return "", err
	}
	return response.Text(), nil
}

func main() {
	result, err := ProcessWithFallback(context.Background(), MultimodalRequest{
		Text:   "What is in this image?",
		Images: [][]byte{},
	})
	if err != nil {
		slog.Error("processing failed", "error", err)
		return
	}
	fmt.Println(result)
}
```

## Image Preprocessing Pipeline

Sending raw, high-resolution images to vision models wastes tokens and money. Preprocessing (resize, compress, format conversion) reduces API costs and improves response latency while preserving sufficient visual information for the model.

This example uses only the Go standard library; the `image/jpeg` and `image/png` packages handle encoding, and `golang.org/x/image/draw` handles resizing without external dependencies:

```go
package main

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	_ "image/png"
	"log/slog"
	"os"

	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/llm"
	"github.com/lookatitude/beluga-ai/schema"
	_ "github.com/lookatitude/beluga-ai/llm/providers/openai"
)

// ImageProcessor resizes and reencodes images before sending to vision models.
type ImageProcessor struct {
	MaxWidth  int
	MaxHeight int
	Quality   int // JPEG quality 1-100.
}

// NewImageProcessor returns an ImageProcessor with sensible defaults.
func NewImageProcessor() *ImageProcessor {
	return &ImageProcessor{MaxWidth: 2000, MaxHeight: 2000, Quality: 85}
}

// Process resizes the image if it exceeds the configured dimensions and
// reencodes it as JPEG. Returns the processed bytes and the MIME type.
func (ip *ImageProcessor) Process(imageData []byte) ([]byte, string, error) {
	img, _, err := image.Decode(bytes.NewReader(imageData))
	if err != nil {
		return nil, "", fmt.Errorf("decode image: %w", err)
	}

	b := img.Bounds()
	if b.Dx() > ip.MaxWidth || b.Dy() > ip.MaxHeight {
		img = resize(img, ip.MaxWidth, ip.MaxHeight)
	}

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: ip.Quality}); err != nil {
		return nil, "", fmt.Errorf("encode image: %w", err)
	}
	return buf.Bytes(), "image/jpeg", nil
}

// resize scales img to fit within maxW x maxH while preserving aspect ratio.
func resize(src image.Image, maxW, maxH int) image.Image {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	scaleW := float64(maxW) / float64(w)
	scaleH := float64(maxH) / float64(h)
	scale := scaleW
	if scaleH < scale {
		scale = scaleH
	}
	newW := int(float64(w) * scale)
	newH := int(float64(h) * scale)

	dst := image.NewRGBA(image.Rect(0, 0, newW, newH))
	draw.BiLinear.Scale(dst, dst.Bounds(), src, b, draw.Over, nil)
	return dst
}

// AnalyzeImageOptimized preprocesses then sends an image for analysis.
func AnalyzeImageOptimized(ctx context.Context, model llm.ChatModel, imagePath string) (string, error) {
	rawData, err := os.ReadFile(imagePath)
	if err != nil {
		return "", fmt.Errorf("read image: %w", err)
	}

	processor := NewImageProcessor()
	processedData, mimeType, err := processor.Process(rawData)
	if err != nil {
		return "", fmt.Errorf("process image: %w", err)
	}

	slog.Info("image optimized",
		"original_bytes", len(rawData),
		"processed_bytes", len(processedData),
	)

	msg := &schema.HumanMessage{
		Parts: []schema.ContentPart{
			schema.TextPart{Text: "Describe this image."},
			schema.ImagePart{Data: processedData, MimeType: mimeType},
		},
	}

	response, err := model.Generate(ctx, []schema.Message{msg})
	if err != nil {
		return "", err
	}
	return response.Text(), nil
}

func main() {
	ctx := context.Background()

	model, err := llm.New("openai", config.ProviderConfig{
		APIKey: os.Getenv("OPENAI_API_KEY"),
		Model:  "gpt-4o",
	})
	if err != nil {
		slog.Error("model creation failed", "error", err)
		return
	}

	result, err := AnalyzeImageOptimized(ctx, model, "photo.jpg")
	if err != nil {
		slog.Error("analysis failed", "error", err)
		return
	}
	fmt.Println(result)
}
```

## Batch Image Classification

Classify multiple images efficiently with concurrency limiting. Batch classification is common in content moderation, product categorization, and media library organization. The semaphore limits concurrent API calls to avoid rate limiting.

```go
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"

	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/llm"
	"github.com/lookatitude/beluga-ai/schema"
	_ "github.com/lookatitude/beluga-ai/llm/providers/openai"
)

// ClassificationResult holds the outcome for a single image.
type ClassificationResult struct {
	Path     string
	Category string
	Error    error
}

// ClassifyBatch classifies a batch of images concurrently with a concurrency cap.
func ClassifyBatch(ctx context.Context, model llm.ChatModel, categories []string, imagePaths []string, maxConcurrent int) []ClassificationResult {
	results := make([]ClassificationResult, len(imagePaths))
	sem := make(chan struct{}, maxConcurrent)
	var wg sync.WaitGroup

	prompt := fmt.Sprintf(
		"Classify this image into one of these categories: %s\nRespond with only the category name.",
		strings.Join(categories, ", "),
	)

	for i, path := range imagePaths {
		wg.Add(1)
		go func(idx int, imagePath string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			rawData, err := os.ReadFile(imagePath)
			if err != nil {
				results[idx] = ClassificationResult{Path: imagePath, Error: err}
				return
			}

			msg := &schema.HumanMessage{
				Parts: []schema.ContentPart{
					schema.TextPart{Text: prompt},
					schema.ImagePart{Data: rawData, MimeType: "image/jpeg"},
				},
			}

			response, err := model.Generate(ctx, []schema.Message{msg})
			if err != nil {
				results[idx] = ClassificationResult{Path: imagePath, Error: err}
				return
			}

			results[idx] = ClassificationResult{
				Path:     imagePath,
				Category: strings.TrimSpace(response.Text()),
			}
		}(i, path)
	}

	wg.Wait()
	return results
}

func main() {
	ctx := context.Background()

	model, err := llm.New("openai", config.ProviderConfig{
		APIKey: os.Getenv("OPENAI_API_KEY"),
		Model:  "gpt-4o",
	})
	if err != nil {
		slog.Error("model creation failed", "error", err)
		return
	}

	categories := []string{"nature", "architecture", "people", "food", "other"}
	imagePaths := []string{"img1.jpg", "img2.jpg", "img3.jpg"}

	results := ClassifyBatch(ctx, model, categories, imagePaths, 3)
	for _, r := range results {
		if r.Error != nil {
			slog.Error("classification error", "path", r.Path, "error", r.Error)
		} else {
			fmt.Printf("%s -> %s\n", r.Path, r.Category)
		}
	}
}
```

## Best Practices

When working with multimodal AI in Beluga AI:

1. **Use `schema.ImagePart{Data: bytes, MimeType: "image/jpeg"}`** -- The `Data` field holds raw bytes; `MimeType` tells the provider how to decode them. Use `URL` instead of `Data` when the image is publicly accessible to reduce request payload size.
2. **Use `schema.AudioPart{Data: bytes, Format: "mp3"}`** -- The `Format` field (not `MimeType`) specifies the encoding. Check provider documentation for supported formats.
3. **Build multimodal messages with `&schema.HumanMessage{Parts: []schema.ContentPart{...}}`** -- Mix `TextPart`, `ImagePart`, `AudioPart` freely in the `Parts` slice.
4. **Preprocess images before sending** -- Resize to provider-recommended dimensions. Most vision models work well at 1000-2000px and cost less than oversized inputs.
5. **Check provider capabilities** -- Not all providers support all modalities. Build fallback routing that checks capabilities before attempting a call.
6. **Monitor costs** -- Multimodal tokens are significantly more expensive than text tokens; track usage per request type using `response.Usage`.
7. **Validate inputs** -- Check file sizes and formats before sending to prevent API errors.

## Related Recipes

- **[Streaming Chunks with Metadata](/docs/cookbook/llm/streaming-metadata)** -- Capture token counts from multimodal streaming calls
- **[LLM Error Handling](/docs/cookbook/llm/llm-error-handling)** -- Handle provider errors for multimodal requests
