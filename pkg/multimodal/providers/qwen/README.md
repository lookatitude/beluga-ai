# Qwen (Alibaba) Multimodal Provider

This provider implements multimodal model support for Alibaba's Qwen models via DashScope API.

## Overview

The Qwen provider supports:
- **Text processing**: Full support
- **Image processing**: Base64 encoded images via data URLs
- **Audio processing**: Limited support (may require transcription)
- **Video processing**: Limited support (may require frame extraction)

## Configuration

### Basic Configuration

```go
import (
    "github.com/lookatitude/beluga-ai/pkg/multimodal"
)

config := &multimodal.Config{
    Provider: "qwen",
    Model:   "qwen-vl-max",
    APIKey:  os.Getenv("QWEN_API_KEY"),
}
```

### Advanced Configuration

```go
config := &multimodal.Config{
    Provider: "qwen",
    Model:   "qwen-vl-max",
    APIKey:  os.Getenv("QWEN_API_KEY"),
    BaseURL: "https://dashscope.aliyuncs.com/api/v1", // Optional
    Timeout: 30 * time.Second,
    MaxRetries: 3,
}
```

## Usage

### Text Processing

```go
ctx := context.Background()

model, err := multimodal.NewModel(ctx, "qwen", config)
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

- `QWEN_API_KEY`: Your Qwen/DashScope API key (required)
- `QWEN_MODEL`: Default model to use
- `QWEN_BASE_URL`: Custom API base URL (default: https://dashscope.aliyuncs.com/api/v1)
- `QWEN_TIMEOUT`: Request timeout (default: 30s)
- `QWEN_MAX_RETRIES`: Maximum retry attempts (default: 3)

## Supported Models

- `qwen-vl-max` (recommended for multimodal)
- `qwen-vl-plus`
- `qwen-turbo`
- `qwen-plus`
- Other Qwen models as available

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

- Image URLs are supported via data URLs (base64 encoded)
- Audio and video support may require preprocessing
- Maximum image size: 20MB
- Maximum audio size: 25MB
- Maximum video size: 100MB

## Getting API Key

1. Visit https://dashscope.console.aliyun.com/
2. Create an account and API key
3. Store the API key securely
