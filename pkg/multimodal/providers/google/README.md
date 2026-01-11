# Google Vertex AI Multimodal Provider

This provider implements the MultimodalModel interface for Google Vertex AI, providing access to Gemini models through Google Cloud's Vertex AI platform.

## Features

- **Text Processing**: Full support for text-based multimodal inputs
- **Image Processing**: Support for image inputs via base64 encoding
- **Audio Processing**: Support for audio inputs via base64 encoding
- **Video Processing**: Support for video inputs via base64 encoding
- **Streaming**: Real-time streaming support for incremental responses
- **OTEL Integration**: Full observability with OpenTelemetry tracing and metrics

## Configuration

### Required Fields

- `project_id`: Google Cloud project ID (required)
- `model`: Vertex AI model name (e.g., "gemini-1.5-pro")

### Optional Fields

- `location`: Google Cloud location/region (default: "us-central1")
- `api_key`: API key for authentication (optional if using service account)
- `base_url`: Vertex AI API base URL (default: location-specific endpoint)
- `timeout`: Request timeout (default: 30s)
- `max_retries`: Maximum retry attempts (default: 3)

### Example Configuration

```go
import (
    "context"
    "time"
    "github.com/lookatitude/beluga-ai/pkg/multimodal"
    "github.com/lookatitude/beluga-ai/pkg/multimodal/providers/google"
)

// Create config
config := &google.Config{
    ProjectID: "my-gcp-project",
    Location:  "us-central1",
    Model:     "gemini-1.5-pro",
    APIKey:     "your-api-key", // Optional if using service account
    Timeout:    30 * time.Second,
    MaxRetries: 3,
}

// Create provider
provider, err := google.NewGoogleProvider(config)
if err != nil {
    log.Fatal(err)
}

// Use with multimodal package
multimodalConfig := multimodal.Config{
    Provider: "google",
    Model:    "gemini-1.5-pro",
    ProviderSpecific: map[string]any{
        "project_id": "my-gcp-project",
        "location":   "us-central1",
    },
}

model, err := multimodal.NewMultimodalModel(ctx, "google", multimodalConfig)
```

## Authentication

The provider supports two authentication methods:

1. **API Key**: Set the `api_key` field in the configuration
2. **Service Account**: Use Google Cloud Application Default Credentials (ADC)

For service account authentication, ensure your environment has the appropriate credentials configured:

```bash
export GOOGLE_APPLICATION_CREDENTIALS="/path/to/service-account-key.json"
```

## Capabilities

| Modality | Supported | Max Size | Formats |
|----------|-----------|----------|---------|
| Text     | ✅ Yes    | N/A      | text/plain |
| Image    | ✅ Yes    | 20MB     | PNG, JPEG, JPG, WEBP |
| Audio    | ✅ Yes    | 25MB     | MP3, WAV, M4A, OGG |
| Video    | ✅ Yes    | 100MB    | MP4, WEBM, MOV |

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

## Differences from Gemini Provider

The Google provider uses **Vertex AI** endpoints, which are different from the direct Gemini API:

- **Vertex AI**: Enterprise-focused, uses project ID and location
- **Gemini API**: Direct API access, uses API key only

Choose the Google provider for:
- Enterprise deployments
- Integration with Google Cloud services
- Service account authentication
- Advanced IAM and security controls

Choose the Gemini provider for:
- Simpler setup with API key
- Direct API access
- Development and testing

## References

- [Vertex AI Documentation](https://cloud.google.com/vertex-ai/docs)
- [Gemini API Reference](https://cloud.google.com/vertex-ai/docs/generative-ai/model-reference/gemini)
