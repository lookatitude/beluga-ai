---
title: Multimodal Recipes
description: Practical patterns for working with images, audio, and video in AI applications, including preprocessing, provider fallbacks, and batch classification.
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
    "os"

    "github.com/lookatitude/beluga-ai/pkg/llms"
    "github.com/lookatitude/beluga-ai/pkg/schema"
)

func CompareMultipleImages(ctx context.Context, llm llms.LLM, imagePaths []string, question string) (string, error) {
    // Load all images
    var parts []schema.ContentPart
    parts = append(parts, schema.TextPart(question))

    for _, path := range imagePaths {
        imageData, err := os.ReadFile(path)
        if err != nil {
            return "", fmt.Errorf("read image %s: %w", path, err)
        }

        parts = append(parts, schema.ImagePart(imageData, "image/jpeg"))
    }

    messages := []schema.Message{
        schema.NewUserMessage(parts...),
    }

    response, err := llm.Generate(ctx, messages)
    if err != nil {
        return "", err
    }

    return response.Content, nil
}

// Example: Compare product photos
func main() {
    ctx := context.Background()

    config := llms.NewConfig(
        llms.WithProvider("openai"),
        llms.WithModelName("gpt-4o"),
        llms.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
    )

    factory := llms.NewFactory()
    llm, _ := factory.CreateLLM("openai", config)

    images := []string{
        "product_v1.jpg",
        "product_v2.jpg",
        "product_v3.jpg",
    }

    result, err := CompareMultipleImages(
        ctx,
        llm,
        images,
        "Which of these product designs is most appealing? Explain your reasoning.",
    )
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(result)
}
```

## Capability-Based Provider Fallbacks

Not all LLM providers support the same modalities. GPT-4o handles images but not audio natively. Gemini supports images, audio, and video. Claude supports images but not video. When building multimodal applications that need to work across providers, you need a routing layer that matches requests to capable models and falls back gracefully when the primary model cannot handle a given input type.

```go
type ProviderCapabilities struct {
    SupportsImages bool
    SupportsAudio  bool
    SupportsVideo  bool
    MaxImageSize   int // bytes
    MaxAudioDuration time.Duration
}

var providerCapabilities = map[string]ProviderCapabilities{
    "gpt-4o": {
        SupportsImages: true,
        SupportsAudio:  false,
        MaxImageSize:   20 * 1024 * 1024, // 20MB
    },
    "claude-3-5-sonnet-20241022": {
        SupportsImages: true,
        SupportsAudio:  false,
        MaxImageSize:   10 * 1024 * 1024, // 10MB
    },
    "gemini-1.5-pro": {
        SupportsImages: true,
        SupportsAudio:  true,
        SupportsVideo:  true,
        MaxImageSize:   20 * 1024 * 1024,
        MaxAudioDuration: 2 * time.Hour,
    },
}

type MultimodalRequest struct {
    Text        string
    Images      [][]byte
    Audio       []byte
    VideoFrames [][]byte
}

func ProcessWithFallback(ctx context.Context, req MultimodalRequest) (string, error) {
    // Determine required capabilities
    needsImages := len(req.Images) > 0
    needsAudio := len(req.Audio) > 0
    needsVideo := len(req.VideoFrames) > 0

    // Find compatible providers in priority order
    providers := []string{"gpt-4o", "claude-3-5-sonnet-20241022", "gemini-1.5-pro"}

    var lastErr error
    for _, providerName := range providers {
        caps := providerCapabilities[providerName]

        // Check compatibility
        if needsImages && !caps.SupportsImages {
            continue
        }
        if needsAudio && !caps.SupportsAudio {
            continue
        }
        if needsVideo && !caps.SupportsVideo {
            continue
        }

        // Check size limits
        if needsImages {
            for _, img := range req.Images {
                if len(img) > caps.MaxImageSize {
                    continue
                }
            }
        }

        // Try this provider
        result, err := tryProvider(ctx, providerName, req)
        if err == nil {
            return result, nil
        }

        lastErr = err
        log.Printf("Provider %s failed: %v, trying next", providerName, err)
    }

    return "", fmt.Errorf("all providers failed, last error: %w", lastErr)
}

func tryProvider(ctx context.Context, providerName string, req MultimodalRequest) (string, error) {
    config := llms.NewConfig(
        llms.WithProvider(getProviderType(providerName)),
        llms.WithModelName(providerName),
    )

    factory := llms.NewFactory()
    llm, err := factory.CreateLLM(getProviderType(providerName), config)
    if err != nil {
        return "", err
    }

    // Build message
    var parts []schema.ContentPart
    parts = append(parts, schema.TextPart(req.Text))

    for _, img := range req.Images {
        parts = append(parts, schema.ImagePart(img, "image/jpeg"))
    }

    if len(req.Audio) > 0 {
        parts = append(parts, schema.AudioPart(req.Audio, "audio/mp3"))
    }

    messages := []schema.Message{
        schema.NewUserMessage(parts...),
    }

    response, err := llm.Generate(ctx, messages)
    if err != nil {
        return "", err
    }

    return response.Content, nil
}

func getProviderType(modelName string) string {
    if strings.Contains(modelName, "gpt") {
        return "openai"
    }
    if strings.Contains(modelName, "claude") {
        return "anthropic"
    }
    if strings.Contains(modelName, "gemini") {
        return "google"
    }
    return "openai"
}
```

## Image Preprocessing Pipeline

Sending raw, high-resolution images to vision models wastes tokens and money. A 4000x3000 photo may cost 10x more tokens than a resized 1000x750 version, with minimal quality difference for most analysis tasks. Preprocessing (resize, compress, format conversion) reduces API costs and improves response latency while preserving sufficient visual information for the model.

```go
import (
    "image"
    "image/jpeg"
    "image/png"
    "bytes"

    "github.com/nfnt/resize"
)

type ImageProcessor struct {
    MaxWidth       int
    MaxHeight      int
    Quality        int // JPEG quality 1-100
    Format         string // "jpeg" or "png"
}

func NewImageProcessor() *ImageProcessor {
    return &ImageProcessor{
        MaxWidth:  2000,
        MaxHeight: 2000,
        Quality:   85,
        Format:    "jpeg",
    }
}

func (ip *ImageProcessor) Process(imageData []byte) ([]byte, string, error) {
    // Decode image
    img, format, err := image.Decode(bytes.NewReader(imageData))
    if err != nil {
        return nil, "", fmt.Errorf("decode image: %w", err)
    }

    // Resize if needed
    bounds := img.Bounds()
    width := bounds.Dx()
    height := bounds.Dy()

    if width > ip.MaxWidth || height > ip.MaxHeight {
        // Calculate new dimensions maintaining aspect ratio
        ratio := float64(width) / float64(height)

        var newWidth, newHeight uint
        if width > height {
            newWidth = uint(ip.MaxWidth)
            newHeight = uint(float64(ip.MaxWidth) / ratio)
        } else {
            newHeight = uint(ip.MaxHeight)
            newWidth = uint(float64(ip.MaxHeight) * ratio)
        }

        img = resize.Resize(newWidth, newHeight, img, resize.Lanczos3)
    }

    // Encode to target format
    var buf bytes.Buffer

    switch ip.Format {
    case "jpeg":
        err = jpeg.Encode(&buf, img, &jpeg.Options{Quality: ip.Quality})
    case "png":
        err = png.Encode(&buf, img)
    default:
        return nil, "", fmt.Errorf("unsupported format: %s", ip.Format)
    }

    if err != nil {
        return nil, "", fmt.Errorf("encode image: %w", err)
    }

    mimeType := "image/" + ip.Format
    return buf.Bytes(), mimeType, nil
}

// Usage in vision pipeline
func AnalyzeImageOptimized(ctx context.Context, llm llms.LLM, imagePath string) (string, error) {
    // Read image
    rawData, err := os.ReadFile(imagePath)
    if err != nil {
        return "", err
    }

    // Process image
    processor := NewImageProcessor()
    processedData, mimeType, err := processor.Process(rawData)
    if err != nil {
        return "", fmt.Errorf("process image: %w", err)
    }

    log.Printf("Image optimized: %d bytes -> %d bytes (%.1f%% reduction)",
        len(rawData), len(processedData),
        (1.0-float64(len(processedData))/float64(len(rawData)))*100)

    // Analyze
    messages := []schema.Message{
        schema.NewUserMessage(
            schema.TextPart("Describe this image."),
            schema.ImagePart(processedData, mimeType),
        ),
    }

    response, err := llm.Generate(ctx, messages)
    if err != nil {
        return "", err
    }

    return response.Content, nil
}
```

## Video Frame Analysis

Video models are expensive and not universally supported. A practical alternative is to extract key frames at regular intervals and analyze them as individual images. This approach works with any vision model and provides a structured timeline of what happens in the video. The frame extraction interval controls the tradeoff between coverage and API cost.

```go
import (
    "os/exec"
    "path/filepath"
)

type VideoFrameExtractor struct {
    FFmpegPath string
    TempDir    string
}

func NewVideoFrameExtractor() *VideoFrameExtractor {
    return &VideoFrameExtractor{
        FFmpegPath: "ffmpeg",
        TempDir:    os.TempDir(),
    }
}

func (vfe *VideoFrameExtractor) ExtractFrames(
    videoPath string,
    intervalSeconds int,
) ([]string, error) {
    // Create temp directory for frames
    frameDir := filepath.Join(vfe.TempDir, fmt.Sprintf("frames_%d", time.Now().Unix()))
    if err := os.MkdirAll(frameDir, 0755); err != nil {
        return nil, err
    }

    // Extract frames using ffmpeg
    outputPattern := filepath.Join(frameDir, "frame_%04d.jpg")

    cmd := exec.Command(
        vfe.FFmpegPath,
        "-i", videoPath,
        "-vf", fmt.Sprintf("fps=1/%d", intervalSeconds),
        "-q:v", "2", // Quality
        outputPattern,
    )

    if err := cmd.Run(); err != nil {
        return nil, fmt.Errorf("extract frames: %w", err)
    }

    // Find extracted frames
    matches, err := filepath.Glob(filepath.Join(frameDir, "frame_*.jpg"))
    if err != nil {
        return nil, err
    }

    return matches, nil
}

func AnalyzeVideo(ctx context.Context, llm llms.LLM, videoPath string) ([]string, error) {
    extractor := NewVideoFrameExtractor()

    // Extract frames every 10 seconds
    framePaths, err := extractor.ExtractFrames(videoPath, 10)
    if err != nil {
        return nil, fmt.Errorf("extract frames: %w", err)
    }
    defer cleanupFrames(framePaths)

    log.Printf("Extracted %d frames from video", len(framePaths))

    // Analyze each frame
    var analyses []string
    for i, framePath := range framePaths {
        frameData, err := os.ReadFile(framePath)
        if err != nil {
            return nil, err
        }

        messages := []schema.Message{
            schema.NewUserMessage(
                schema.TextPart(fmt.Sprintf("Describe what's happening in this frame (timestamp: %ds)", i*10)),
                schema.ImagePart(frameData, "image/jpeg"),
            ),
        }

        response, err := llm.Generate(ctx, messages)
        if err != nil {
            return nil, fmt.Errorf("analyze frame %d: %w", i, err)
        }

        analyses = append(analyses, response.Content)
    }

    return analyses, nil
}

func cleanupFrames(framePaths []string) {
    if len(framePaths) == 0 {
        return
    }

    frameDir := filepath.Dir(framePaths[0])
    os.RemoveAll(frameDir)
}
```

## Audio Segmentation

Long audio files exceed most STT and LLM input limits. Splitting audio into segments at natural boundaries (silence, fixed intervals) allows processing files of any length. Each segment is transcribed independently, then results are concatenated. The segment duration should match the model's maximum input length while being short enough to produce timely results.

```go
type AudioSegmenter struct {
    FFmpegPath     string
    SegmentDuration time.Duration
}

func NewAudioSegmenter(segmentDuration time.Duration) *AudioSegmenter {
    return &AudioSegmenter{
        FFmpegPath:     "ffmpeg",
        SegmentDuration: segmentDuration,
    }
}

func (as *AudioSegmenter) Segment(audioPath string) ([]string, error) {
    tempDir, err := os.MkdirTemp("", "audio_segments_*")
    if err != nil {
        return nil, err
    }

    outputPattern := filepath.Join(tempDir, "segment_%03d.mp3")

    cmd := exec.Command(
        as.FFmpegPath,
        "-i", audioPath,
        "-f", "segment",
        "-segment_time", fmt.Sprintf("%d", int(as.SegmentDuration.Seconds())),
        "-c", "copy",
        outputPattern,
    )

    if err := cmd.Run(); err != nil {
        return nil, fmt.Errorf("segment audio: %w", err)
    }

    segments, err := filepath.Glob(filepath.Join(tempDir, "segment_*.mp3"))
    if err != nil {
        return nil, err
    }

    return segments, nil
}

func TranscribeLongAudio(ctx context.Context, llm llms.LLM, audioPath string) (string, error) {
    segmenter := NewAudioSegmenter(5 * time.Minute)

    segments, err := segmenter.Segment(audioPath)
    if err != nil {
        return "", err
    }
    defer cleanupSegments(segments)

    var transcripts []string
    for i, segmentPath := range segments {
        audioData, err := os.ReadFile(segmentPath)
        if err != nil {
            return "", err
        }

        messages := []schema.Message{
            schema.NewUserMessage(
                schema.TextPart(fmt.Sprintf("Transcribe this audio segment (part %d)", i+1)),
                schema.AudioPart(audioData, "audio/mp3"),
            ),
        }

        response, err := llm.Generate(ctx, messages)
        if err != nil {
            return "", fmt.Errorf("transcribe segment %d: %w", i, err)
        }

        transcripts = append(transcripts, response.Content)
    }

    // Combine transcripts
    return strings.Join(transcripts, "\n\n"), nil
}

func cleanupSegments(segments []string) {
    if len(segments) == 0 {
        return
    }

    segmentDir := filepath.Dir(segments[0])
    os.RemoveAll(segmentDir)
}
```

## Batch Image Classification

Classify multiple images efficiently with rate limiting. Batch classification is common in content moderation, product categorization, and media library organization. The rate limiter prevents API throttling, while concurrent processing maximizes throughput within the rate limit.

```go
type ImageClassifier struct {
    llm        llms.LLM
    processor  *ImageProcessor
    rateLimiter *rate.Limiter
    categories []string
}

func NewImageClassifier(llm llms.LLM, categories []string, rateLimit float64) *ImageClassifier {
    return &ImageClassifier{
        llm:        llm,
        processor:  NewImageProcessor(),
        rateLimiter: rate.NewLimiter(rate.Limit(rateLimit), 1),
        categories: categories,
    }
}

type ClassificationResult struct {
    Path       string
    Category   string
    Confidence float64
    Error      error
}

func (ic *ImageClassifier) ClassifyBatch(ctx context.Context, imagePaths []string) ([]ClassificationResult, error) {
    results := make([]ClassificationResult, len(imagePaths))
    var wg sync.WaitGroup

    for i, path := range imagePaths {
        wg.Add(1)

        go func(idx int, imagePath string) {
            defer wg.Done()

            // Rate limit
            if err := ic.rateLimiter.Wait(ctx); err != nil {
                results[idx] = ClassificationResult{
                    Path:  imagePath,
                    Error: err,
                }
                return
            }

            // Classify
            category, confidence, err := ic.classify(ctx, imagePath)
            results[idx] = ClassificationResult{
                Path:       imagePath,
                Category:   category,
                Confidence: confidence,
                Error:      err,
            }
        }(i, path)
    }

    wg.Wait()
    return results, nil
}

func (ic *ImageClassifier) classify(ctx context.Context, imagePath string) (string, float64, error) {
    // Load and process image
    rawData, err := os.ReadFile(imagePath)
    if err != nil {
        return "", 0, err
    }

    processedData, mimeType, err := ic.processor.Process(rawData)
    if err != nil {
        return "", 0, err
    }

    // Build prompt
    prompt := fmt.Sprintf("Classify this image into one of these categories: %s\nRespond with only the category name.",
        strings.Join(ic.categories, ", "))

    messages := []schema.Message{
        schema.NewUserMessage(
            schema.TextPart(prompt),
            schema.ImagePart(processedData, mimeType),
        ),
    }

    response, err := ic.llm.Generate(ctx, messages,
        llms.WithTemperature(0.0),
        llms.WithMaxTokens(50),
    )
    if err != nil {
        return "", 0, err
    }

    category := strings.TrimSpace(response.Content)

    // Simple confidence based on exact match
    confidence := 0.8
    if !contains(ic.categories, category) {
        confidence = 0.5 // Low confidence if not exact match
    }

    return category, confidence, nil
}

func contains(slice []string, item string) bool {
    for _, s := range slice {
        if strings.EqualFold(s, item) {
            return true
        }
    }
    return false
}
```

## Best Practices

When working with multimodal AI:

1. **Preprocess images** -- Resize and compress before API calls to reduce token costs and latency
2. **Use appropriate formats** -- JPEG for photos (lossy compression is fine), PNG for screenshots and diagrams (lossless preserves text)
3. **Batch similar requests** -- Group images by type for efficiency, using rate limiters to stay within provider limits
4. **Implement fallbacks** -- Have backup providers for capabilities; not all models support all modalities
5. **Cache results** -- Identical images produce identical analyses; cache by content hash to avoid duplicate API calls
6. **Monitor costs** -- Multimodal tokens are significantly more expensive than text tokens; track usage per request type
7. **Handle rate limits** -- Use token buckets and exponential backoff; multimodal endpoints often have stricter rate limits
8. **Validate inputs** -- Check file sizes, formats, and dimensions before sending to prevent API errors
9. **Extract key frames** -- For video, analyze representative frames rather than every frame to control costs
10. **Segment long audio** -- Break audio into manageable chunks that fit within model input limits

## Next Steps

- Learn about [Document Processing](/guides/document-processing) for OCR workflows
- Explore [Voice Recipes](/cookbook/voice-recipes) for real-time audio
- Read [Multimodal AI](/guides/multimodal) for comprehensive patterns
