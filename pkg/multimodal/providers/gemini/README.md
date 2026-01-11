# Google Gemini Multimodal Provider

This package provides Google Gemini provider implementation for multimodal models.

## Features

- **Text Processing**: Full support for text inputs and outputs
- **Image Processing**: Support for image inputs (PNG, JPEG, WebP)
- **Audio Processing**: Support for audio inputs (MP3, WAV, M4A, OGG)
- **Video Processing**: Support for video inputs (MP4, WebM, MOV)
- **Streaming**: Real-time streaming support for incremental responses
- **Retry Logic**: Automatic retry with exponential backoff for transient errors

## Configuration

```go
import (
    "github.com/lookatitude/beluga-ai/pkg/multimodal"
    "github.com/lookatitude/beluga-ai/pkg/multimodal/providers/gemini"
)

config := multimodal.Config{
    Provider: "gemini",
    Model:    "gemini-1.5-pro",
    APIKey:   "your-gemini-api-key",
    BaseURL:  "https://generativelanguage.googleapis.com/v1beta", // Optional
    Timeout:  30 * time.Second,
    MaxRetries: 3,
}
```

## Usage

### Basic Text Processing

```go
ctx := context.Background()

model, err := multimodal.NewMultimodalModel(ctx, "gemini", config)
if err != nil {
    log.Fatal(err)
}

textBlock, _ := types.NewContentBlock("text", []byte("What is machine learning?"))
input, _ := types.NewMultimodalInput([]*types.ContentBlock{textBlock})

output, err := model.Process(ctx, input)
if err != nil {
    log.Fatal(err)
}

fmt.Println(string(output.ContentBlocks[0].Data))
```

### Text + Image Processing

```go
textBlock, _ := types.NewContentBlock("text", []byte("Describe this image"))
imageBlock, _ := types.NewContentBlockFromURL(ctx, "image", "https://example.com/image.png")

input, _ := types.NewMultimodalInput([]*types.ContentBlock{textBlock, imageBlock})
output, err := model.Process(ctx, input)
```

### Streaming

```go
outputChan, err := model.ProcessStream(ctx, input)
if err != nil {
    log.Fatal(err)
}

for output := range outputChan {
    if len(output.ContentBlocks) > 0 {
        fmt.Print(string(output.ContentBlocks[0].Data))
    }
}
```

## Capabilities

The Gemini provider supports:

- **Text**: ✅ Full support
- **Image**: ✅ PNG, JPEG, WebP (up to 20MB)
- **Audio**: ✅ MP3, WAV, M4A, OGG (up to 25MB)
- **Video**: ✅ MP4, WebM, MOV (up to 100MB)

## Error Handling

The provider includes automatic retry logic for:
- Rate limit errors (429)
- Server errors (500, 502, 503)
- Network timeouts
- Connection errors

## Observability

All operations include:
- OTEL tracing with span attributes
- Structured logging with trace/span IDs
- Error recording in spans

## API Reference

See the [main multimodal README](../../README.md) for the complete API documentation.
