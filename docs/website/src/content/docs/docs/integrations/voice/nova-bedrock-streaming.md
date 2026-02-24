---
title: Amazon Nova S2S on Bedrock
description: "Integrate Amazon Nova 2 Sonic for real-time speech-to-speech voice conversations via AWS Bedrock with Beluga AI in Go."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Amazon Nova, AWS Bedrock S2S, speech-to-speech, Beluga AI, voice AI AWS, real-time audio, Go voice provider"
---

For teams already running infrastructure on AWS, Amazon Nova provides end-to-end voice conversations through AWS Bedrock without introducing a separate vendor. This avoids additional billing relationships, keeps audio data within your AWS account boundary, and leverages existing IAM policies for access control. This guide covers integrating the Amazon Nova S2S provider with Beluga AI for real-time streaming voice interactions.

## Overview

The Amazon Nova integration uses the `voice/s2s` package to enable bidirectional audio streaming via AWS Bedrock. Nova 2 Sonic supports natural voice conversations with configurable models and regions.

## Prerequisites

- Go 1.23 or later
- AWS account with Bedrock access
- Nova 2 Sonic model enabled in Bedrock console
- AWS credentials configured (IAM role or access keys)

## Installation

```bash
go get github.com/lookatitude/beluga-ai
```

Configure AWS credentials:

```bash
export AWS_ACCESS_KEY_ID="your-access-key"
export AWS_SECRET_ACCESS_KEY="your-secret-key"
export AWS_REGION="us-east-1"
```

Enable the Nova 2 Sonic model in the AWS Bedrock console before proceeding.

## Configuration

Create an Amazon Nova S2S provider with the desired region and model:

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/lookatitude/beluga-ai/voice/s2s"
    "github.com/lookatitude/beluga-ai/voice/s2s/providers/amazon_nova"
)

func main() {
    ctx := context.Background()

    config := amazon_nova.Config{
        Region: os.Getenv("AWS_REGION"),
        Model:  "amazon.nova-pro-v1:0",
    }

    provider, err := amazon_nova.NewAmazonNovaS2S(ctx, config)
    if err != nil {
        log.Fatalf("Failed to create Nova provider: %v", err)
    }

    stream, err := provider.StartStreaming(ctx, &s2s.ConversationContext{
        SystemPrompt: "You are a helpful assistant.",
    })
    if err != nil {
        log.Fatalf("Failed to start streaming: %v", err)
    }
    defer stream.Close()

    fmt.Println("Nova streaming started")
}
```

## Usage

### Bidirectional Audio Streaming

Handle sending and receiving audio concurrently:

```go
func handleNovaStream(ctx context.Context, stream s2s.Stream) error {
    audioIn := make(chan []byte)
    go func() {
        for audio := range audioIn {
            if err := stream.SendAudio(ctx, audio); err != nil {
                log.Printf("Send error: %v", err)
                return
            }
        }
    }()

    for chunk := range stream.ReceiveAudio() {
        if chunk.Error != nil {
            return chunk.Error
        }
        processAudio(chunk.Audio)
    }

    return nil
}
```

### Observability

Add OpenTelemetry tracing to monitor Nova streaming sessions:

```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"
)

tracer := otel.Tracer("beluga.voice.s2s.amazon_nova")
ctx, span := tracer.Start(ctx, "amazon_nova.start",
    trace.WithAttributes(
        attribute.String("model", config.Model),
        attribute.String("region", config.Region),
    ),
)
defer span.End()
```

## Configuration Reference

| Option   | Description       | Default                | Required |
|----------|-------------------|------------------------|----------|
| `Region` | AWS region        | `us-east-1`           | No       |
| `Model`  | Nova model ID     | `amazon.nova-pro-v1:0`| No       |
| `Voice`  | Voice selection   | -                      | No       |

## Troubleshooting

### Model not enabled

Verify the Nova model is enabled in the AWS Bedrock console for your region. Navigate to Bedrock > Model access and confirm Nova 2 Sonic is available.

### Access denied

Ensure your IAM role or user has the required Bedrock permissions. Add `bedrock:InvokeModelWithResponseStream` to your IAM policy.

## Advanced Topics

### Production Deployment

- Use IAM roles instead of access keys for authentication in production
- Monitor Bedrock usage through AWS Cost Explorer and CloudWatch
- Choose an AWS region close to your users for lower latency
- Implement connection retry logic for transient Bedrock errors
- Set context timeouts to prevent indefinite streaming sessions

## Related Resources

- [OpenAI Realtime API](/docs/integrations/openai-realtime)
- [Voice Services Overview](/docs/integrations/voice-services)
