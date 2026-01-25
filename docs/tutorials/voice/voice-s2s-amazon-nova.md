# Native S2S with Amazon Nova

<!--
Persona: Pair Programmer Colleague
- Treat the reader as a competent peer
- Be helpful, direct, and slightly informal but professional
- Focus on getting results quickly
- Keep it concise and functional
- Provide runnable code examples
-->

## What you will build
In this tutorial, we'll use Amazon Nova's Speech-to-Speech (S2S) capabilities for end-to-end, ultra-low-latency voice conversations. You'll learn how to configure the provider, handle real-time audio streams, and manage S2S session lifecycles.

## Learning Objectives
- ✅ Understand S2S (Speech-to-Speech) vs. traditional STT+LLM+TTS
- ✅ Configure the Amazon Nova S2S provider
- ✅ Handle real-time audio streams
- ✅ Manage S2S session lifecycle

## Introduction
Welcome, colleague! Traditional voice agents use a complex pipeline of STT, LLM, and TTS, which often introduces latency and loses emotional nuance. Native S2S models like Amazon Nova change the game by processing audio-to-audio directly. Let's look at how to build an ultra-responsive voice experience.

## Prerequisites

- AWS Credentials with Bedrock/Nova access
- Go 1.24+
- `pkg/voice/s2s` package

## What is S2S?

Traditional voice agents use a pipeline:
`Audio` -> **STT** -> `Text` -> **LLM** -> `Text` -> **TTS** -> `Audio`

Native S2S (like Amazon Nova or OpenAI Realtime) does it in one step:
`Audio` -> **S2S Model** -> `Audio`

**Benefits**:
- **Latency**: No "waiting for transcript" steps.
- **Emotion**: The model hears your tone and responds with appropriate emotion directly.

## Step 1: Initialize the S2S Provider
```go
package main

import (
    "context"
    "log"

    "github.com/lookatitude/beluga-ai/pkg/voice/s2s"
    "github.com/lookatitude/beluga-ai/pkg/voice/s2s/providers/amazon_nova"
)

func main() {
    ctx := context.Background()

    // 1. Configure Amazon Nova
    config := &amazon_nova.Config{
        Region: "us-east-1",
        ModelID: "amazon.nova-reel-v1:0", // S2S model
        VoiceID: "Brian",
    }

    // 2. Create provider
    provider, err := amazon_nova.NewProvider(ctx, config)
    if err != nil {
        log.Fatal(err)
    }
}
```

## Step 2: Start an S2S Session

S2S requires a bidirectional stream.
```text
go
go
    // 3. Start S2S Session
    session, err := provider.NewSession(ctx)
    if err != nil {
        log.Fatal(err)
    }
    defer session.Close()
```

## Step 3: Processing Streams

You must handle incoming audio and transcripts (if the model provides them) simultaneously.
```text
go
go
    // 4. Handle incoming model response
    go func() {
        for event := range session.Events() {
            switch e := event.(type) {
            case *s2s.AudioEvent:
                // Play audio back to user
                playAudio(e.Audio)
            case *s2s.TranscriptEvent:
                // Optional: show user what model is saying
                fmt.Println("AI:", e.Text)
            }
        }
    }()
```

## Step 4: Sending User Audio
```text
go
go
    // 5. Pipe user microphone to model
    for audioChunk := range micStream {
        err := session.SendAudio(audioChunk)
        if err != nil {
            break
        }
    }
```

## Step 5: Handling Interruptions

S2S models handle interruptions naturally because they "hear" you while they are "speaking".
```
    // If the user starts talking, the model will automatically 
    // stop generating audio and start listening.

## Verification

1. Run the example.
2. Say "Hello, who are you?".
3. Verify the model responds with voice almost instantly (\<500ms).
4. Try interrupting the model while it's talking.

## Next Steps

- **[Configuring Voice Reasoning Modes](./voice-s2s-reasoning-modes.md)** - Optimize for speed or quality.
- **[Voice Transport Management](../transport/README.md)** - Setup WebRTC for production.
