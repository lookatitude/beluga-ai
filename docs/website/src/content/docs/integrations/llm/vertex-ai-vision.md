---
title: Google Vertex AI Vision
description: Integrate Google Vertex AI with Beluga AI for enterprise-grade multimodal processing using Gemini models, including text, image, audio, and video understanding.
---

Organizations already invested in Google Cloud often need their AI workloads to run within the same security perimeter -- VPC Service Controls, IAM policies, audit logging, and CMEK encryption. Vertex AI provides Gemini model access through Google Cloud's managed platform, giving you enterprise controls that the public Gemini API does not offer.

Choose Vertex AI over the public Gemini API when you need data residency guarantees, network isolation via VPC, or when your security team requires IAM-based access control rather than API keys.

## Overview

Vertex AI is Google Cloud's managed AI platform. When paired with Beluga AI's LLM package, it provides:

- **Gemini model access** through the Vertex AI API (separate from the public Gemini API)
- **Multi-modality** across text, image, audio, and video inputs
- **Enterprise features** including VPC Service Controls, CMEK, and IAM-based access
- **Regional deployment** for data residency and latency requirements

## Prerequisites

- Go 1.23 or later
- A Google Cloud project with the Vertex AI API enabled
- Google Cloud credentials (service account or Application Default Credentials)
- Beluga AI framework installed

## Installation

Install the Google Cloud AI Platform client library:

```bash
go get cloud.google.com/go/aiplatform/apiv1
```

Configure authentication using one of these methods:

```bash
# Option 1: Service account key file
export GOOGLE_APPLICATION_CREDENTIALS="/path/to/service-account.json"

# Option 2: Application Default Credentials (recommended for development)
gcloud auth application-default login
```

## Configuration

### Basic Setup

Create a Vertex AI multimodal provider using the Google LLM provider with Vertex AI mode enabled:

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/lookatitude/beluga-ai/llm/providers/google"
)

func main() {
    ctx := context.Background()

    // Create Vertex AI configuration
    config := &google.Config{
        ProjectID:  os.Getenv("GOOGLE_CLOUD_PROJECT"),
        Location:   "us-central1",
        Model:      "gemini-1.5-pro",
        UseVertexAI: true,
    }

    provider, err := google.New(config)
    if err != nil {
        log.Fatalf("failed to create Vertex AI provider: %v", err)
    }

    // Use provider for multimodal processing
    _ = provider
    fmt.Println("Vertex AI provider created successfully")
}
```

Verify the setup:

```bash
export GOOGLE_CLOUD_PROJECT="your-project-id"
export GOOGLE_APPLICATION_CREDENTIALS="/path/to/credentials.json"
go run main.go
```

### Configuration Reference

| Option | Description | Default | Required |
|--------|-------------|---------|----------|
| `ProjectID` | Google Cloud project ID | - | Yes |
| `Location` | GCP region for Vertex AI | `us-central1` | No |
| `Model` | Gemini model name | `gemini-1.5-pro` | No |
| `UseVertexAI` | Use Vertex AI API instead of Gemini API | `false` | No |
| `APIKey` | API key (alternative to ADC) | - | No |
| `Endpoint` | Custom Vertex AI endpoint | auto-detected | No |

## Usage

### Multi-Modality Support

Vertex AI with Gemini models supports multiple input modalities. Use Beluga AI's `schema` package to construct multimodal messages:

```go
import (
    "github.com/lookatitude/beluga-ai/schema"
)

// Text + Image
msg := schema.UserMessage{
    Content: []schema.ContentPart{
        schema.TextPart{Text: "Analyze this image"},
        schema.ImagePart{Data: imageData, MIMEType: "image/png"},
    },
}

// Text + Audio
msg = schema.UserMessage{
    Content: []schema.ContentPart{
        schema.TextPart{Text: "Transcribe this audio"},
        schema.AudioPart{Data: audioData, MIMEType: "audio/wav"},
    },
}

// Text + Video
msg = schema.UserMessage{
    Content: []schema.ContentPart{
        schema.TextPart{Text: "Describe this video"},
        schema.VideoPart{Data: videoData, MIMEType: "video/mp4"},
    },
}
```

### Enterprise Features

Enable Vertex AI-specific enterprise features by setting the endpoint and project configuration:

```go
config := &google.Config{
    ProjectID:   os.Getenv("GOOGLE_CLOUD_PROJECT"),
    Location:    "us-central1",
    Model:       "gemini-1.5-pro",
    UseVertexAI: true,
    Endpoint:    "us-central1-aiplatform.googleapis.com",
}
```

This routes requests through the Vertex AI API, which supports VPC Service Controls, audit logging, and organization-level policies.

## Advanced Topics

### OpenTelemetry Instrumentation

Add tracing to Vertex AI calls using Beluga AI's observability package:

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"
    "time"

    "github.com/lookatitude/beluga-ai/llm/providers/google"
    "github.com/lookatitude/beluga-ai/schema"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"
)

func main() {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    config := &google.Config{
        ProjectID:   os.Getenv("GOOGLE_CLOUD_PROJECT"),
        Location:    "us-central1",
        Model:       "gemini-1.5-pro",
        UseVertexAI: true,
    }

    tracer := otel.Tracer("beluga.multimodal.vertex_ai")
    ctx, span := tracer.Start(ctx, "vertex_ai.process",
        trace.WithAttributes(
            attribute.String("gen_ai.system", "vertex_ai"),
            attribute.String("gen_ai.request.model", config.Model),
            attribute.String("gcp.project_id", config.ProjectID),
        ),
    )
    defer span.End()

    provider, err := google.New(config)
    if err != nil {
        span.RecordError(err)
        log.Fatalf("failed to create provider: %v", err)
    }

    messages := []schema.Message{
        &schema.UserMessage{
            Content: []schema.ContentPart{
                schema.TextPart{Text: "Describe this image in detail."},
                schema.ImagePart{Data: loadImage("image.png"), MIMEType: "image/png"},
            },
        },
    }

    response, err := provider.Generate(ctx, messages)
    if err != nil {
        span.RecordError(err)
        log.Fatalf("generation failed: %v", err)
    }

    span.SetAttributes(
        attribute.Int("gen_ai.response.input_tokens", response.Usage.InputTokens),
        attribute.Int("gen_ai.response.output_tokens", response.Usage.OutputTokens),
    )

    fmt.Printf("Response: %s\n", response.Content)
}

func loadImage(path string) []byte {
    data, err := os.ReadFile(path)
    if err != nil {
        log.Fatalf("failed to load image: %v", err)
    }
    return data
}
```

### Production Considerations

When deploying Vertex AI integrations to production:

- **IAM roles**: Use dedicated service accounts with the `roles/aiplatform.user` role. Avoid using owner or editor roles.
- **Regional deployment**: Choose a region that matches your data residency requirements and provides acceptable latency.
- **Cost optimization**: Monitor usage through Cloud Billing. Use Gemini Flash models for latency-sensitive or cost-sensitive workloads.
- **VPC Service Controls**: Configure a service perimeter around your Vertex AI resources to prevent data exfiltration.
- **Error handling**: Handle quota errors (HTTP 429) with exponential backoff using Beluga AI's `resilience` package.

## Troubleshooting

### "Project not found"

The Google Cloud project ID is incorrect or the Vertex AI API is not enabled.

```bash
# Verify the project exists and you have access
gcloud projects describe your-project-id

# Enable the Vertex AI API
gcloud services enable aiplatform.googleapis.com --project=your-project-id
```

### "Authentication failed"

Credentials are missing or invalid.

```bash
# Check current credentials
gcloud auth application-default print-access-token

# Re-authenticate
gcloud auth application-default login
```

## Related Resources

- [Pixtral (Mistral) Integration](/integrations/pixtral-mistral) -- Mistral AI vision-language integration
- [LLM Providers](/integrations/llm-providers) -- All supported LLM providers
- [Monitoring](/integrations/monitoring) -- Observability and tracing setup
