# Amazon Transcribe Audio Websockets

Welcome, colleague! In this integration guide, we're going to integrate AWS Transcribe for streaming speech transcription with Beluga AI's STT package. AWS Transcribe provides real-time transcription via WebSocket with automatic language detection.

## What you will build

You will configure Beluga AI to use AWS Transcribe for streaming speech-to-text transcription via WebSocket, enabling real-time transcription with automatic language detection and speaker identification.

## Learning Objectives

- ✅ Configure AWS Transcribe with Beluga AI STT
- ✅ Use WebSocket streaming for real-time transcription
- ✅ Handle AWS credentials and regions
- ✅ Understand Transcribe-specific features

## Prerequisites

- Go 1.24 or later installed
- Beluga AI Framework installed
- AWS account with Transcribe access
- AWS credentials configured

## Step 1: Setup and Installation

Install AWS SDK:
bash
```bash
go get github.com/aws/aws-sdk-go-v2/service/transcribestreaming
```

Configure AWS credentials:
bash
```bash
export AWS_ACCESS_KEY_ID="your-access-key"
export AWS_SECRET_ACCESS_KEY="your-secret-key"
export AWS_REGION="us-east-1"
```

## Step 2: Basic Transcribe Configuration

Create an AWS Transcribe provider:
```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/service/transcribestreaming"
    "github.com/lookatitude/beluga-ai/pkg/voice/stt"
)

func main() {
    ctx := context.Background()

    // Load AWS config
    cfg, err := config.LoadDefaultConfig(ctx,
        config.WithRegion(os.Getenv("AWS_REGION")),
    )
    if err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }

    // Create Transcribe client
    client := transcribestreaming.NewFromConfig(cfg)

    // Create STT provider wrapper
    provider := NewAWSTranscribeSTT(client, "en-US")

    // Transcribe audio
    audio := []byte{/* audio data */}
    transcript, err := provider.Transcribe(ctx, audio)
    if err != nil {
        log.Fatalf("Transcription failed: %v", err)
    }

    fmt.Printf("Transcript: %s\n", transcript)
}
```

## Step 3: Streaming Transcription

Implement WebSocket streaming:
```go
type AWSTranscribeSTT struct {
    client   *transcribestreaming.Client
    language string
}

func NewAWSTranscribeSTT(client *transcribestreaming.Client, language string) *AWSTranscribeSTT {
    return &AWSTranscribeSTT{
        client:   client,
        language: language,
    }
}

func (t *AWSTranscribeSTT) Transcribe(ctx context.Context, audio []byte) (string, error) {
    // Start streaming session
    // AWS Transcribe uses WebSocket for streaming
    // Implementation handles WebSocket connection and audio streaming
    
    // This is a simplified example
    // Full implementation would handle WebSocket connection,
    // audio chunking, and result parsing
    return "", nil
}
```

## Step 4: Use with Beluga AI

Integrate with Beluga AI STT interface:
```go
func main() {
    ctx := context.Background()
    
    // Setup AWS
    cfg, _ := config.LoadDefaultConfig(ctx)
    client := transcribestreaming.NewFromConfig(cfg)
    
    // Create provider
    provider := NewAWSTranscribeSTT(client, "en-US")
    
    // Use in voice session
    // The voice session will use this provider
}
```

## Configuration Options

| Option | Description | Default | Required |
|--------|-------------|---------|----------|
| `Region` | AWS region | `us-east-1` | No |
| `Language` | Language code | `en-US` | No |
| `SampleRate` | Audio sample rate | `16000` | No |

## Common Issues

### "Credentials not found"

**Problem**: AWS credentials not configured.

**Solution**: Configure credentials:aws configure
```

### "Region not supported"

**Problem**: Transcribe not available in region.

**Solution**: Use supported region (e.g., us-east-1, us-west-2).

## Production Considerations

When using AWS Transcribe in production:

- **IAM roles**: Use IAM roles instead of access keys
- **Cost optimization**: Monitor usage and optimize settings
- **Region selection**: Choose region close to users
- **Streaming**: Use WebSocket for real-time transcription
- **Error handling**: Handle connection failures gracefully

## Next Steps

Congratulations! You've integrated AWS Transcribe with Beluga AI. Next, learn how to:

- **[Deepgram Live Streams](./deepgram-live-streams.md)** - Deepgram integration
- **[STT Package Documentation](../../../api/packages/voice/stt.md)** - Deep dive into STT package
- **[Voice Providers Guide](../../../guides/voice-providers.md)** - Voice provider patterns

---

**Ready for more?** Check out the [Integrations Index](./README.md) for more integration guides!
