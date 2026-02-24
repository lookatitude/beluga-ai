---
title: AWS Transcribe WebSocket STT
description: "Integrate AWS Transcribe for real-time streaming speech-to-text via WebSocket with speaker diarization in Beluga AI pipelines."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "AWS Transcribe, speech-to-text, WebSocket STT, Beluga AI, real-time transcription, speaker diarization, Go STT provider"
---

For organizations already operating within the AWS ecosystem, AWS Transcribe provides real-time speech transcription that integrates natively with IAM, CloudWatch, and other AWS services. Its automatic language detection and speaker diarization capabilities make it well suited for multi-speaker, multi-language scenarios such as contact center transcription. This guide covers integrating AWS Transcribe as an STT provider within Beluga AI.

## Overview

The Amazon Transcribe integration wraps the AWS Transcribe Streaming API to implement the Beluga AI STT provider interface. Audio is streamed over WebSocket for low-latency transcription with support for multiple languages and speaker diarization.

## Prerequisites

- Go 1.23 or later
- AWS account with Transcribe access
- AWS credentials configured (IAM role or access keys)

## Installation

```bash
go get github.com/lookatitude/beluga-ai
go get github.com/aws/aws-sdk-go-v2/service/transcribestreaming
```

Configure AWS credentials:

```bash
export AWS_ACCESS_KEY_ID="your-access-key"
export AWS_SECRET_ACCESS_KEY="your-secret-key"
export AWS_REGION="us-east-1"
```

## Configuration

Create an AWS Transcribe STT provider:

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/service/transcribestreaming"
)

func main() {
    ctx := context.Background()

    cfg, err := config.LoadDefaultConfig(ctx,
        config.WithRegion(os.Getenv("AWS_REGION")),
    )
    if err != nil {
        log.Fatalf("Failed to load AWS config: %v", err)
    }

    client := transcribestreaming.NewFromConfig(cfg)

    provider := NewAWSTranscribeSTT(client, "en-US")

    audio := loadAudioData()
    transcript, err := provider.Transcribe(ctx, audio)
    if err != nil {
        log.Fatalf("Transcription failed: %v", err)
    }

    fmt.Printf("Transcript: %s\n", transcript)
}
```

## Usage

### Custom Provider Wrapper

Wrap the AWS Transcribe client to implement the Beluga AI STT interface:

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
    // Implementation handles WebSocket connection,
    // audio chunking, and result parsing via the
    // AWS Transcribe Streaming API.
    return "", nil
}
```

### Voice Session Integration

Use the Transcribe provider within a Beluga AI voice session:

```go
import "github.com/lookatitude/beluga-ai/voice/session"

sess, err := session.NewVoiceSession(ctx,
    session.WithSTTProvider(provider),
    session.WithConfig(session.DefaultConfig()),
)
if err != nil {
    log.Fatalf("Failed to create session: %v", err)
}
```

## Configuration Reference

| Option       | Description         | Default    | Required |
|-------------|---------------------|------------|----------|
| `Region`     | AWS region          | `us-east-1`| No      |
| `Language`   | Language code       | `en-US`    | No       |
| `SampleRate` | Audio sample rate   | `16000`    | No       |

## Troubleshooting

### Credentials not found

Ensure AWS credentials are configured via environment variables, shared credentials file, or IAM role. Run `aws configure` to set up the default profile.

### Region not supported

AWS Transcribe Streaming is not available in all regions. Use a supported region such as `us-east-1`, `us-west-2`, or `eu-west-1`.

## Advanced Topics

### Production Deployment

- Use IAM roles instead of access keys for authentication
- Monitor usage through AWS CloudWatch and Cost Explorer
- Choose an AWS region close to your users for lower latency
- Use WebSocket streaming for all real-time transcription scenarios
- Implement reconnection logic for WebSocket failures

## Related Resources

- [Deepgram Live Streams](/integrations/deepgram-streams)
- [Voice Services Overview](/integrations/voice-services)
