# Pixtral Multimodal Provider

This provider implements the MultimodalModel interface for Pixtral, Mistral AI's open-source multimodal model.

## Features

- **Text Processing**: Full support for text-based multimodal inputs
- **Image Processing**: Support for image inputs via base64 encoding or URLs
- **Audio Processing**: Support for audio inputs (converted to text descriptions)
- **Video Processing**: Support for video inputs (converted to text descriptions)
- **Streaming**: Real-time streaming support for incremental responses
- **OTEL Integration**: Full observability with OpenTelemetry tracing and metrics

## Configuration

### Required Fields

- `api_key`: Pixtral API key (required)
- `model`: Pixtral model name (e.g., "pixtral-12b")

### Optional Fields

- `base_url`: API base URL (default: "https://api.mistral.ai/v1")
- `timeout`: Request timeout (default: 30s)
- `max_retries`: Maximum retry attempts (default: 3)

### Example Configuration

```go
import (
    "context"
    "time"
    "github.com/lookatitude/beluga-ai/pkg/multimodal"
    "github.com/lookatitude/beluga-ai/pkg/multimodal/providers/pixtral"
)

// Create config
config := &pixtral.Config{
    APIKey:  "your-api-key",
    Model:   "pixtral-12b",
    BaseURL: "https://api.mistral.ai/v1",
    Timeout: 30 * time.Second,
    MaxRetries: 3,
}

// Create provider
provider, err := pixtral.NewPixtralProvider(config)
if err != nil {
    log.Fatal(err)
}

// Use with multimodal package
multimodalConfig := multimodal.Config{
    Provider: "pixtral",
    Model:    "pixtral-12b",
    APIKey:   "your-api-key",
}

model, err := multimodal.NewMultimodalModel(ctx, "pixtral", multimodalConfig)
```

## Authentication

The provider uses API key authentication. Set your API key in the configuration:

```go
config := &pixtral.Config{
    APIKey: os.Getenv("PIXTRAL_API_KEY"),
    Model:  "pixtral-12b",
}
```

## Capabilities

| Modality | Supported | Max Size | Formats |
|----------|-----------|----------|---------|
| Text     | ✅ Yes    | N/A      | text/plain |
| Image    | ✅ Yes    | 20MB     | PNG, JPEG, JPG, GIF, WEBP |
| Audio    | ✅ Yes*   | 25MB     | MP3, WAV, M4A, OGG (*converted to text) |
| Video    | ✅ Yes*   | 100MB    | MP4, WEBM, MOV (*converted to text) |

*Note: Audio and video are currently converted to text descriptions. Full native support may be available in future model versions.

## Usage Examples

### Basic Text Processing

```go
input := &types.MultimodalInput{
    ID: "input-1",
    ContentBlocks: []*types.ContentBlock{
        {
            Type:     "text",
            Data:     []byte("What is in this image?"),
            MIMEType: "text/plain",
        },
    },
}

output, err := provider.Process(ctx, input)
```

### Image Processing

```go
// Read image file
imageData, _ := os.ReadFile("image.png")

input := &types.MultimodalInput{
    ID: "input-2",
    ContentBlocks: []*types.ContentBlock{
        {
            Type:     "text",
            Data:     []byte("Describe this image"),
            MIMEType: "text/plain",
        },
        {
            Type:     "image",
            Data:     imageData,
            MIMEType: "image/png",
        },
    },
}

output, err := provider.Process(ctx, input)
```

### Image from URL

```go
input := &types.MultimodalInput{
    ID: "input-3",
    ContentBlocks: []*types.ContentBlock{
        {
            Type:     "text",
            Data:     []byte("What is in this image?"),
            MIMEType: "text/plain",
        },
        {
            Type:     "image",
            URL:      "https://example.com/image.png",
            MIMEType: "image/png",
        },
    },
}

output, err := provider.Process(ctx, input)
```

### Streaming

```go
outputChan, err := provider.ProcessStream(ctx, input)
if err != nil {
    log.Fatal(err)
}

for output := range outputChan {
    for _, block := range output.ContentBlocks {
        if block.Type == "text" {
            fmt.Print(string(block.Data))
        }
    }
}
```

## Error Handling

The provider implements comprehensive error handling with retry logic for transient failures:

- **Rate Limits**: Automatically retried with exponential backoff
- **Network Errors**: Retried up to `max_retries` times
- **Authentication Errors**: Returned immediately without retry

## Observability

All operations are instrumented with OpenTelemetry:

- **Traces**: Full request/response tracing
- **Metrics**: Request counts, durations, error rates
- **Logging**: Structured logging with trace context

## Open Source

Pixtral is an open-source multimodal model from Mistral AI. This provider enables you to use Pixtral models through Mistral's API.

## References

- [Mistral AI Documentation](https://docs.mistral.ai/)
- [Pixtral Model Information](https://mistral.ai/news/pixtral-12b/)
