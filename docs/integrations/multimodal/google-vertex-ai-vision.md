# Google Vertex AI Vision

Welcome, colleague! In this integration guide, we're going to integrate Google Vertex AI Vision with Beluga AI's multimodal package. Vertex AI provides enterprise-grade multimodal capabilities with Gemini models.

## What you will build

You will configure Beluga AI to use Google Vertex AI for multimodal processing, enabling enterprise-grade vision-language understanding with Google's Gemini models through Vertex AI.

## Learning Objectives

- ✅ Configure Vertex AI with Beluga AI multimodal
- ✅ Use Gemini models via Vertex AI
- ✅ Process text, image, audio, and video
- ✅ Understand Vertex AI-specific features

## Prerequisites

- Go 1.24 or later installed
- Beluga AI Framework installed
- Google Cloud project with Vertex AI enabled
- Google Cloud credentials

## Step 1: Setup and Installation

Install Google Cloud SDK:
bash
```bash
go get cloud.google.com/go/aiplatform/apiv1
```

Configure Google Cloud credentials:
```bash
export GOOGLE_APPLICATION_CREDENTIALS="path/to/service-account.json"
```
# Or use gcloud auth application-default login
```

## Step 2: Basic Vertex AI Configuration

Create a Vertex AI multimodal provider:
```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/lookatitude/beluga-ai/pkg/multimodal/providers/google"
    "github.com/lookatitude/beluga-ai/pkg/multimodal/types"
)

func main() {
    ctx := context.Background()

    // Create Vertex AI configuration
    config := &google.Config{
        ProjectID: os.Getenv("GOOGLE_CLOUD_PROJECT"),
        Location:  "us-central1",
        Model:     "gemini-1.5-pro",
        APIKey:    os.Getenv("GOOGLE_API_KEY"), // Optional, can use ADC
    }

    // Create provider
    provider, err := google.NewGoogleProvider(config)
    if err != nil {
        log.Fatalf("Failed: %v", err)
    }

    // Process multimodal input
    input := &types.MultimodalInput{
        Text: "What's in this image?",
        Images: []types.ImageData{
            {Data: []byte{/* image data */}, Format: "png"},
        },
    }

    output, err := provider.Process(ctx, input)
    if err != nil {
        log.Fatalf("Failed: %v", err)
    }

    fmt.Printf("Response: %s\n", output.Text)
}
```

### Verification

Run the example:
bash
```bash
export GOOGLE_CLOUD_PROJECT="your-project"
export GOOGLE_APPLICATION_CREDENTIALS="path/to/credentials.json"
go run main.go
```

You should see the multimodal response.

## Step 3: Multi-Modality Support

Use Vertex AI for various modalities:
```go
// Text + Image
input := &types.MultimodalInput{
    Text: "Analyze this image",
    Images: []types.ImageData{{Data: imageData}},
}

// Text + Audio
input := &types.MultimodalInput{
    Text: "Transcribe this audio",
    Audio: []types.AudioData{{Data: audioData}},
}

// Text + Video
input := &types.MultimodalInput{
    Text: "Describe this video",
    Video: []types.VideoData{{Data: videoData}},
}
```

## Step 4: Enterprise Features

Use Vertex AI enterprise features:
```text
go
go
config := &google.Config{
    ProjectID:    "your-project",
    Location:     "us-central1",
    Model:        "gemini-1.5-pro",
    Endpoint:     "us-central1-aiplatform.googleapis.com",
    UseVertexAI:  true, // Use Vertex AI instead of Gemini API
}
```

## Step 5: Complete Integration

Here's a complete, production-ready example:
```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"
    "time"

    "github.com/lookatitude/beluga-ai/pkg/multimodal/providers/google"
    "github.com/lookatitude/beluga-ai/pkg/multimodal/types"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"
)

func main() {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    config := &google.Config{
        ProjectID: os.Getenv("GOOGLE_CLOUD_PROJECT"),
        Location:  "us-central1",
        Model:     "gemini-1.5-pro",
        UseVertexAI: true,
    }

    tracer := otel.Tracer("beluga.multimodal.vertex_ai")
    ctx, span := tracer.Start(ctx, "vertex_ai.process",
        trace.WithAttributes(
            attribute.String("model", config.Model),
            attribute.String("project", config.ProjectID),
        ),
    )
    defer span.End()

    provider, err := google.NewGoogleProvider(config)
    if err != nil {
        span.RecordError(err)
        log.Fatalf("Failed: %v", err)
    }

    input := &types.MultimodalInput{
        Text: "Describe this image in detail.",
        Images: []types.ImageData{
            {Data: loadImage("image.png"), Format: "png"},
        },
    }

    output, err := provider.Process(ctx, input)
    if err != nil {
        span.RecordError(err)
        log.Fatalf("Failed: %v", err)
    }



    span.SetAttributes(
        attribute.String("response", output.Text),
        attribute.Int("modalities", 2), // text + image
    )

    fmt.Printf("Response: %s\n", output.Text)
}
```

## Configuration Options

| Option | Description | Default | Required |
|--------|-------------|---------|----------|
| `ProjectID` | GCP project ID | - | Yes |
| `Location` | GCP region | `us-central1` | No |
| `Model` | Gemini model | `gemini-1.5-pro` | No |
| `UseVertexAI` | Use Vertex AI API | `false` | No |

## Common Issues

### "Project not found"

**Problem**: Invalid project ID.

**Solution**: Verify project ID:export GOOGLE_CLOUD_PROJECT="your-project-id"
```

### "Authentication failed"

**Problem**: Missing or invalid credentials.

**Solution**: Configure credentials:gcloud auth application-default login
```

## Production Considerations

When using Vertex AI in production:

- **IAM roles**: Use service accounts with proper roles
- **Regional deployment**: Choose appropriate region
- **Cost optimization**: Monitor usage and optimize
- **Enterprise features**: Leverage Vertex AI enterprise features
- **Error handling**: Handle API failures gracefully

## Next Steps

Congratulations! You've integrated Vertex AI with Beluga AI. Next, learn how to:

- **[Pixtral Mistral Integration](./pixtral-mistral-integration.md)** - Pixtral integration
- **[Multimodal Package Documentation](../../api/packages/multimodal.md)** - Deep dive into multimodal package
- **[Multimodal Guide](../../guides/rag-multimodal.md)** - Multimodal patterns

---

**Ready for more?** Check out the [Integrations Index](./README.md) for more integration guides!
