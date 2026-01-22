# Amazon Nova Bedrock Streaming

Welcome, colleague! In this integration guide, we're going to integrate Amazon Nova 2 Sonic for speech-to-speech with Beluga AI's S2S package. Amazon Nova provides end-to-end voice conversations via AWS Bedrock.

## What you will build

You will configure Beluga AI to use Amazon Nova 2 Sonic for speech-to-speech conversations, enabling natural voice interactions with AWS Bedrock integration.

## Learning Objectives

- ✅ Configure Amazon Nova with Beluga AI S2S
- ✅ Use Bedrock streaming for S2S
- ✅ Handle AWS credentials and regions
- ✅ Understand Nova-specific features

## Prerequisites

- Go 1.24 or later installed
- Beluga AI Framework installed
- AWS account with Bedrock access
- Nova 2 Sonic model enabled
- AWS credentials configured

## Step 1: Setup and Installation

Configure AWS credentials:
bash
```bash
export AWS_ACCESS_KEY_ID="your-access-key"
export AWS_SECRET_ACCESS_KEY="your-secret-key"
export AWS_REGION="us-east-1"
```

Enable Nova 2 Sonic model in Bedrock console.

## Step 2: Basic Nova Configuration

Create an Amazon Nova S2S provider:
```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/lookatitude/beluga-ai/pkg/voice/s2s"
    "github.com/lookatitude/beluga-ai/pkg/voice/s2s/providers/amazon_nova"
)

func main() {
    ctx := context.Background()

    // Create Amazon Nova provider
    config := amazon_nova.Config{
        Region: os.Getenv("AWS_REGION"),
        Model:  "amazon.nova-pro-v1:0",
    }
    
    provider, err := amazon_nova.NewAmazonNovaS2S(ctx, config)
    if err != nil {
        log.Fatalf("Failed: %v", err)
    }

    // Start streaming
    stream, err := provider.StartStreaming(ctx, &s2s.ConversationContext{
        SystemPrompt: "You are a helpful assistant.",
    })
    if err != nil {
        log.Fatalf("Failed: %v", err)
    }
    defer stream.Close()

    // Handle audio streaming
    fmt.Println("Nova streaming started")
}
```

## Step 3: Streaming Audio

Handle bidirectional streaming:
```go
func handleNovaStream(ctx context.Context, stream s2s.Stream) error {
    // Send audio to Nova
    audioIn := make(chan []byte)
    go func() {
        for audio := range audioIn {
            stream.SendAudio(ctx, audio)
        }
    }()
    
    // Receive audio from Nova
    for chunk := range stream.ReceiveAudio() {
        if chunk.Error != nil {
            return chunk.Error
        }
        processAudio(chunk.Audio)
    }

    
    return nil
}
```

## Step 4: Complete Integration

Here's a complete, production-ready example:
```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"
    "time"

    "github.com/lookatitude/beluga-ai/pkg/voice/s2s"
    "github.com/lookatitude/beluga-ai/pkg/voice/s2s/providers/amazon_nova"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"
)

func main() {
    ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
    defer cancel()

    config := amazon_nova.Config{
        Region: os.Getenv("AWS_REGION"),
        Model:  "amazon.nova-pro-v1:0",
    }
    
    tracer := otel.Tracer("beluga.voice.s2s.amazon_nova")
    ctx, span := tracer.Start(ctx, "amazon_nova.start",
        trace.WithAttributes(
            attribute.String("model", config.Model),
            attribute.String("region", config.Region),
        ),
    )
    defer span.End()
    
    provider, err := amazon_nova.NewAmazonNovaS2S(ctx, config)
    if err != nil {
        span.RecordError(err)
        log.Fatalf("Failed: %v", err)
    }

    stream, err := provider.StartStreaming(ctx, &s2s.ConversationContext{
        SystemPrompt: "You are a helpful assistant.",
    })
    if err != nil {
        span.RecordError(err)
        log.Fatalf("Failed: %v", err)
    }
    defer stream.Close()


    span.SetAttributes(attribute.Bool("streaming", true))
    fmt.Println("Nova streaming started")
}
```

## Configuration Options

| Option | Description | Default | Required |
|--------|-------------|---------|----------|
| `Region` | AWS region | `us-east-1` | No |
| `Model` | Nova model ID | `amazon.nova-pro-v1:0` | No |
| `Voice` | Voice selection | - | No |

## Common Issues

### "Model not enabled"

**Problem**: Nova model not enabled in Bedrock.

**Solution**: Enable model in AWS Bedrock console.

### "Access denied"

**Problem**: Missing Bedrock permissions.

**Solution**: Add Bedrock permissions to IAM role.

## Production Considerations

When using Amazon Nova in production:

- **Model access**: Ensure Nova model is enabled
- **IAM roles**: Use IAM roles for authentication
- **Cost management**: Monitor Bedrock usage
- **Regional deployment**: Choose appropriate region
- **Error handling**: Handle API failures gracefully

## Next Steps

Congratulations! You've integrated Amazon Nova with Beluga AI. Next, learn how to:

- **[OpenAI Realtime API](./openai-realtime-api.md)** - OpenAI Realtime integration
- **[S2S Package Documentation](../../../api/packages/voice/s2s.md)** - Deep dive into S2S package
- **[Voice Providers Guide](../../../guides/voice-providers.md)** - Voice provider patterns

---

**Ready for more?** Check out the [Integrations Index](./README.md) for more integration guides!
