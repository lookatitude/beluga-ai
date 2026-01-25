# Pixtral Mistral Integration

Welcome, colleague! In this integration guide, we're going to integrate Pixtral (Mistral AI's vision-language model) with Beluga AI's multimodal package. Pixtral provides powerful vision-language capabilities for image understanding and generation.

## What you will build

You will configure Beluga AI to use Pixtral for multimodal processing (text + image), enabling vision-language understanding, image analysis, and multimodal reasoning.

## Learning Objectives

- ✅ Configure Pixtral with Beluga AI multimodal
- ✅ Process text and image inputs
- ✅ Use Pixtral for vision-language tasks
- ✅ Understand Pixtral-specific features

## Prerequisites

- Go 1.24 or later installed
- Beluga AI Framework installed
- Pixtral API access (Mistral AI)
- API key

## Step 1: Setup and Installation

Get Pixtral API key from https://mistral.ai

Set environment variable:
bash
```bash
export MISTRAL_API_KEY="your-api-key"
```

## Step 2: Basic Pixtral Configuration

Create a Pixtral multimodal provider:
```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/lookatitude/beluga-ai/pkg/multimodal"
    "github.com/lookatitude/beluga-ai/pkg/multimodal/providers/pixtral"
    "github.com/lookatitude/beluga-ai/pkg/multimodal/types"
)

func main() {
    ctx := context.Background()

    // Create Pixtral configuration
    config := &pixtral.Config{
        APIKey:  os.Getenv("MISTRAL_API_KEY"),
        Model:   "pixtral-12b",
        Timeout: 30 * time.Second,
    }

    // Create provider
    provider, err := pixtral.NewPixtralProvider(config)
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
        log.Fatalf("Processing failed: %v", err)
    }


    fmt.Printf("Response: %s\n", output.Text)
}
```

### Verification

Run the example:
bash
```bash
export MISTRAL_API_KEY="your-api-key"
go run main.go
```

You should see the multimodal response.

## Step 3: Vision-Language Tasks

Use Pixtral for various tasks:
```go
// Image captioning
input := &types.MultimodalInput{
    Images: []types.ImageData{{Data: imageData, Format: "png"}},
}

// Visual question answering
input := &types.MultimodalInput{
    Text: "What color is the car?",
    Images: []types.ImageData{{Data: imageData}},
}

// Image analysis
input := &types.MultimodalInput{
    Text: "Analyze this image and describe the scene.",
    Images: []types.ImageData{{Data: imageData}},
}
```

## Step 4: Use with Beluga AI Agents

Integrate with agents:
```go
func main() {
    ctx := context.Background()

    // Create multimodal model
    config := &pixtral.Config{
        APIKey: os.Getenv("MISTRAL_API_KEY"),
    }
    multimodalModel, _ := pixtral.NewPixtralProvider(config)

    // Create agent with multimodal capabilities
    agent, err := agents.NewAgent(ctx, agents.Config{
        LLMProvider: "pixtral",
        MultimodalModel: multimodalModel,
    })
    if err != nil {
        log.Fatalf("Failed: %v", err)
    }


    // Agent can now process multimodal inputs
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

    "github.com/lookatitude/beluga-ai/pkg/multimodal/providers/pixtral"
    "github.com/lookatitude/beluga-ai/pkg/multimodal/types"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"
)

func main() {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    config := &pixtral.Config{
        APIKey:  os.Getenv("MISTRAL_API_KEY"),
        Model:   "pixtral-12b",
        Timeout: 30 * time.Second,
    }

    tracer := otel.Tracer("beluga.multimodal.pixtral")
    ctx, span := tracer.Start(ctx, "pixtral.process",
        trace.WithAttributes(
            attribute.String("model", config.Model),
        ),
    )
    defer span.End()

    provider, err := pixtral.NewPixtralProvider(config)
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
        attribute.Int("image_count", len(input.Images)),
    )

    fmt.Printf("Response: %s\n", output.Text)
}
```

## Configuration Options

| Option | Description | Default | Required |
|--------|-------------|---------|----------|
| `APIKey` | Mistral API key | - | Yes |
| `Model` | Pixtral model | `pixtral-12b` | No |
| `Timeout` | Request timeout | `30s` | No |
| `MaxImageSize` | Max image size | `20MB` | No |

## Common Issues

### "API key invalid"

**Problem**: Wrong or missing API key.

**Solution**: Verify API key:export MISTRAL_API_KEY="your-api-key"
```

### "Image format not supported"

**Problem**: Unsupported image format.

**Solution**: Use supported formats (PNG, JPEG, WebP).

## Production Considerations

When using Pixtral in production:

- **Image optimization**: Optimize images before sending
- **Cost management**: Monitor API usage
- **Error handling**: Handle API failures gracefully
- **Format support**: Verify supported formats
- **Size limits**: Respect image size limits

## Next Steps

Congratulations! You've integrated Pixtral with Beluga AI. Next, learn how to:

- **[Google Vertex AI Vision](./google-vertex-ai-vision.md)** - Google Vertex AI integration
- **Multimodal Package Documentation** - Deep dive into multimodal package
- **[Multimodal Guide](../../guides/rag-multimodal.md)** - Multimodal patterns

---

**Ready for more?** Check out the Integrations Index for more integration guides!
