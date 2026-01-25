# Audio Analysis with Gemini

<!--
Persona: Pair Programmer Colleague
- Treat the reader as a competent peer
- Be helpful, direct, and slightly informal but professional
- Focus on getting results quickly
- Keep it concise and functional
- Provide runnable code examples
-->

## What you will build
In this tutorial, we'll use Google Gemini to analyze audio files beyond simple transcription. You'll build an "Audio Summarizer" capable of detecting sentiment, summarizing meetings, and answering questions about specific timestamps in an audio recording.

## Learning Objectives
- ✅ Configure Gemini multimodal provider
- ✅ Send audio files to the model
- ✅ Extract insights from audio
- ✅ Build an "Audio Summarizer"

## Introduction
Welcome, colleague! Audio is a rich data source, but it's often trapped in large files. Models like Gemini 1.5 Pro allow us to treat audio as a first-class citizen, letting us "chat" with our recordings to find key decisions in a meeting or gauge a speaker's tone without manual listening.

## Prerequisites

- Google Cloud API Key (Vertex AI or AI Studio)
- Go 1.24+
- `pkg/multimodal` package

## Step 1: Initialize Gemini Provider
```go
package main

import (
    "context"
    "github.com/lookatitude/beluga-ai/pkg/multimodal/providers/gemini"
)

func main() {
    config := &gemini.Config{
        APIKey: os.Getenv("GOOGLE_API_KEY"),
        Model:  "gemini-1.5-pro",
    }
    
    provider, _ := gemini.NewProvider(config)
}
```

## Step 2: Preparing Audio Input
```go
    audioData, _ := os.ReadFile("meeting_recording.mp3")
    
    content := []multimodal.Content{
        multimodal.NewTextContent("Summarize the key decisions made in this meeting."),
        multimodal.NewAudioContent(audioData, "audio/mp3"),
    }
```

## Step 3: Reasoning over Audio

Gemini can do more than just transcribe. It can answer questions about the *vibe* or specific timestamps.
```go
    content := []multimodal.Content{
        multimodal.NewTextContent("What was the tone of the speaker at the 2-minute mark?"),
        multimodal.NewAudioContent(audioData, "audio/mp3"),
    }
    
    res, _ := provider.Generate(ctx, content)
    fmt.Println("Sentiment:", res.Text)
```

## Step 4: Long Audio (File URI)

For large files, it's better to use a URI (GCS bucket).
```text
go
go
    content := []multimodal.Content{
        multimodal.NewTextContent("Transcribe this entire 1-hour lecture."),
        multimodal.NewFileURIContent("gs://my-bucket/lecture.mp3", "audio/mp3"),
    }
```

## Verification

1. Use a short audio clip of someone talking.
2. Ask "What is the main topic?".
3. Verify Gemini provides a coherent summary.

## Next Steps

- **[Visual Reasoning with Pixtral](./multimodal-visual-reasoning.md)** - Analyze images.
- **[Real-time STT Streaming](../voice/voice-stt-realtime-streaming.md)** - Low latency alternatives.
