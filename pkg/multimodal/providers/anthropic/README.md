# Anthropic Claude Multimodal Provider

This provider implements multimodal model support for Anthropic's Claude models.

## Overview

The Anthropic provider supports:
- **Text processing**: Full support
- **Image processing**: Base64 encoded images
- **Audio processing**: Limited support (may require transcription)
- **Video processing**: Limited support (may require frame extraction)

## Configuration

### Basic Configuration

```go
import (
    "github.com/lookatitude/beluga-ai/pkg/multimodal"
    "github.com/lookatitude/beluga-ai/pkg/multimodal/providers/anthropic"
)

config := &multimodal.Config{
    Provider: "anthropic",
    Model:   "claude-3-5-sonnet-20241022",
    APIKey:  os.Getenv("ANTHROPIC_API_KEY"),
}
```

### Advanced Configuration

```go
config := &multimodal.Config{
    Provider: "anthropic",
    Model:   "claude-3-5-sonnet-20241022",
    APIKey:  os.Getenv("ANTHROPIC_API_KEY"),
    BaseURL: "https://api.anthropic.com/v1", // Optional
    Timeout: 30 * time.Second,
    MaxRetries: 3,
    ProviderSpecific: map[string]any{
        "api_version": "2023-06-01",
    },
}
```

## Usage

### Text Processing

```go
ctx := context.Background()

model, err := multimodal.NewModel(ctx, "anthropic", config)
if err != nil {
    log.Fatal(err)
}

input := &types.MultimodalInput{
    ID: "input-1",
    ContentBlocks: []*types.ContentBlock{
        {
            Type:     "text",
            Data:     []byte("What is the capital of France?"),
            MIMEType: "text/plain",
        },
    },
}

output, err := model.Process(ctx, input)
if err != nil {
    log.Fatal(err)
}

fmt.Println(string(output.ContentBlocks[0].Data))
```

### Text + Image Processing

```go
// Read image file
imageData, err := os.ReadFile("image.png")
if err != nil {
    log.Fatal(err)
}

input := &types.MultimodalInput{
    ID: "input-2",
    ContentBlocks: []*types.ContentBlock{
        {
            Type:     "text",
            Data:     []byte("What's in this image?"),
            MIMEType: "text/plain",
        },
        {
            Type:     "image",
            Data:     imageData,
            MIMEType: "image/png",
        },
    },
}

output, err := model.Process(ctx, input)
if err != nil {
    log.Fatal(err)
}
```

### Streaming

```go
outputChan, err := model.ProcessStream(ctx, input)
if err != nil {
    log.Fatal(err)
}

for output := range outputChan {
    for _, block := range output.ContentBlocks {
        fmt.Print(string(block.Data))
    }
}
```

## Environment Variables

- `ANTHROPIC_API_KEY`: Your Anthropic API key (required)
- `ANTHROPIC_MODEL`: Default model to use
- `ANTHROPIC_BASE_URL`: Custom API base URL
- `ANTHROPIC_TIMEOUT`: Request timeout (default: 30s)
- `ANTHROPIC_MAX_RETRIES`: Maximum retry attempts (default: 3)
- `ANTHROPIC_API_VERSION`: API version header (default: 2023-06-01)

## Supported Models

- `claude-3-5-sonnet-20241022` (recommended)
- `claude-3-opus-20240229`
- `claude-3-sonnet-20240229`
- `claude-3-haiku-20240307`

## Capabilities

- **Text**: ✅ Full support
- **Image**: ✅ Base64 encoded images (PNG, JPEG, GIF, WebP)
- **Audio**: ⚠️ Limited (may require transcription)
- **Video**: ⚠️ Limited (may require frame extraction)

## Error Handling

The provider includes automatic retry logic for transient errors:
- Rate limits (429)
- Server errors (500, 502, 503)
- Timeout errors
- Network errors

## Observability

All operations are instrumented with OpenTelemetry:
- Traces for request/response cycles
- Metrics for latency and success rates
- Structured logging with trace IDs

## Limitations

- Image URLs are not directly supported; use base64 encoded data
- Audio and video support may require preprocessing
- Maximum image size: 20MB
- Maximum audio size: 25MB
- Maximum video size: 100MB
