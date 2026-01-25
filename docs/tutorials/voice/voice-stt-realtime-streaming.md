# Real-time STT Streaming

<!--
Persona: Pair Programmer Colleague
- Treat the reader as a competent peer
- Be helpful, direct, and slightly informal but professional
- Focus on getting results quickly
- Keep it concise and functional
- Provide runnable code examples
-->

## What you will build
In this tutorial, we'll implement real-time Speech-to-Text (STT) streaming using the Deepgram provider. You'll learn how to manage WebSocket-based audio sessions and process interim and final transcripts for ultra-low-latency applications.

## Learning Objectives
- ✅ Configure a streaming STT provider
- ✅ Open a streaming session
- ✅ Process audio chunks in real-time
- ✅ Handle interim and final transcripts

## Introduction
Welcome, colleague! For voice assistants, latency is everything. If you wait for the whole audio file to finish before transcribing, the user experience will suffer. Let's look at how to stream audio chunks and get transcripts back in milliseconds using Deepgram.

## Prerequisites

- Deepgram API Key
- Go 1.24+
- Basic understanding of audio buffers

## Step 1: Initialize the Deepgram Provider

We'll use Deepgram's WebSocket-based streaming for the best performance.
```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/lookatitude/beluga-ai/pkg/voice/stt"
    "github.com/lookatitude/beluga-ai/pkg/voice/stt/providers/deepgram"
)

func main() {
    ctx := context.Background()

    // 1. Configure Deepgram
    config := &deepgram.Config{
        APIKey: os.Getenv("DEEPGRAM_API_KEY"),
        Model:  "nova-2",
        Language: "en-US",
        // Enable interim results for low latency
        InterimResults: true,
    }

    // 2. Create provider
    provider, err := deepgram.NewProvider(config)
    if err != nil {
        log.Fatal(err)
    }
}
```

## Step 2: Start a Streaming Session

A streaming session allows you to send a continuous stream of audio data.
```text
go
go
    // 3. Create a streaming session
    session, err := provider.CreateStream(ctx)
    if err != nil {
        log.Fatal(err)
    }
    defer session.Close()
```

## Step 3: Handling Transcripts

The session provides a channel where transcripts are delivered as they are processed.
```go
    // 4. Process transcripts in a goroutine
    go func() {
        for result := range session.Transcripts() {
            if result.Error != nil {
                fmt.Printf("Error: %v\n", result.Error)
                continue
            }


            // Interim results are partial and might change
text
            if result.IsInterim \{
                fmt.Printf("\rInterim: %s", result.Text)
            } else {
                fmt.Printf("\nFinal: %s\n", result.Text)
            }
        }
    }()
```

## Step 4: Streaming Audio Data

Now, you can pipe audio data from a microphone or a file into the session.
```text
go
go
    // 5. Send audio chunks (e.g., 20ms of PCM data)
    // For this example, we'll simulate sending one chunk
    mockAudio := make([]byte, 640) // Placeholder for real audio data
    err = session.SendAudio(mockAudio)
    if err != nil {
        log.Fatal(err)
    }
```

## Step 5: Finalizing

Always remember to close the session to signal the end of the stream.
```
    err = session.Stop()
    if err != nil {
        log.Fatal(err)
    }

## Verification

1. Run the script with a valid Deepgram key.
2. (Optional) Connect a microphone library to `session.SendAudio`.
3. Verify that `Final` transcripts appear when you stop speaking.

## Next Steps

- **[Fine-tuning Whisper for Industry Terms](./voice-stt-whisper-finetuning.md)** - Improve accuracy for specialized vocabulary.
- **[Voice Session Management](./voice-session-interruptions.md)** - Combine STT, TTS, and Agents.
